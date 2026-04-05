package middlewares

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"finance/database"
	"finance/logic"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/gofiber/fiber/v2"
)

const (
	idempotencyHeader = "Idempotency-Key"
	idempotencyTTL    = 24 * time.Hour
	maxIdempotencyKey = 255
)

type idempotencyEntry struct {
	BodyHash    string
	StatusCode  int
	Response    []byte
	ContentType string
	ExpiresAt   time.Time
	InProgress  bool
}

var idempotencyStore = struct {
	sync.Mutex
	Entries map[string]idempotencyEntry
}{
	Entries: make(map[string]idempotencyEntry),
}

func Idempotency() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := strings.TrimSpace(c.Get(idempotencyHeader))
		if key == "" {
			return c.Next()
		}

		if len(key) > maxIdempotencyKey {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Idempotency-Key must be 255 characters or fewer",
			})
		}

		if !isASCIIPrintable(key) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Idempotency-Key contains invalid characters",
			})
		}

		userID, ok := logic.ExtractUserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid token claims",
			})
		}

		storageKey := buildIdempotencyStorageKey(userID, c.Path(), key)
		bodyHash := hashRequestBody(c.Body())

		if database.RedisAvailable() {
			return handleRedisIdempotency(c, storageKey, bodyHash)
		}

		idempotencyStore.Lock()
		pruneExpiredIdempotencyEntries(time.Now())
		if entry, found := idempotencyStore.Entries[storageKey]; found {
			idempotencyStore.Unlock()
			if entry.BodyHash != bodyHash {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"status":  "error",
					"message": "Idempotency-Key has already been used with a different request body",
				})
			}

			if entry.InProgress {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"status":  "error",
					"message": "A request with this Idempotency-Key is already being processed",
				})
			}

			c.Set(fiber.HeaderContentType, entry.ContentType)
			return c.Status(entry.StatusCode).Send(entry.Response)
		}
		idempotencyStore.Entries[storageKey] = idempotencyEntry{
			BodyHash:   bodyHash,
			ExpiresAt:  time.Now().Add(idempotencyTTL),
			InProgress: true,
		}
		idempotencyStore.Unlock()

		if err := c.Next(); err != nil {
			idempotencyStore.Lock()
			delete(idempotencyStore.Entries, storageKey)
			idempotencyStore.Unlock()
			return err
		}

		statusCode := c.Response().StatusCode()
		if statusCode >= fiber.StatusInternalServerError {
			idempotencyStore.Lock()
			delete(idempotencyStore.Entries, storageKey)
			idempotencyStore.Unlock()
			return nil
		}

		responseBody := append([]byte(nil), c.Response().Body()...)
		contentType := string(c.Response().Header.ContentType())
		if contentType == "" {
			contentType = fiber.MIMEApplicationJSON
		}

		idempotencyStore.Lock()
		idempotencyStore.Entries[storageKey] = idempotencyEntry{
			BodyHash:    bodyHash,
			StatusCode:  statusCode,
			Response:    responseBody,
			ContentType: contentType,
			ExpiresAt:   time.Now().Add(idempotencyTTL),
			InProgress:  false,
		}
		idempotencyStore.Unlock()

		return nil
	}
}

func handleRedisIdempotency(c *fiber.Ctx, storageKey, bodyHash string) error {
	createdAt := time.Now()
	entry := idempotencyEntry{
		BodyHash:   bodyHash,
		ExpiresAt:  createdAt.Add(idempotencyTTL),
		InProgress: true,
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not initialize idempotency state",
		})
	}

	reserved, err := database.RedisDb.Db.SetNX(storageKey, entryJSON, idempotencyTTL).Result()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not initialize idempotency state",
		})
	}

	if !reserved {
		return replayRedisIdempotentResponse(c, storageKey, bodyHash)
	}

	if err := c.Next(); err != nil {
		_ = database.RedisDb.Db.Del(storageKey).Err()
		return err
	}

	statusCode := c.Response().StatusCode()
	if statusCode >= fiber.StatusInternalServerError {
		_ = database.RedisDb.Db.Del(storageKey).Err()
		return nil
	}

	finalEntry := idempotencyEntry{
		BodyHash:    bodyHash,
		StatusCode:  statusCode,
		Response:    append([]byte(nil), c.Response().Body()...),
		ContentType: string(c.Response().Header.ContentType()),
		ExpiresAt:   time.Now().Add(idempotencyTTL),
		InProgress:  false,
	}
	if finalEntry.ContentType == "" {
		finalEntry.ContentType = fiber.MIMEApplicationJSON
	}

	finalJSON, err := json.Marshal(finalEntry)
	if err != nil {
		_ = database.RedisDb.Db.Del(storageKey).Err()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not finalize idempotency state",
		})
	}

	if err := database.RedisDb.Db.Set(storageKey, finalJSON, idempotencyTTL).Err(); err != nil {
		_ = database.RedisDb.Db.Del(storageKey).Err()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not finalize idempotency state",
		})
	}

	return nil
}

func replayRedisIdempotentResponse(c *fiber.Ctx, storageKey, bodyHash string) error {
	value, err := database.RedisDb.Db.Get(storageKey).Result()
	if err != nil {
		if err == redis.Nil {
			return c.Next()
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not read idempotency state",
		})
	}

	var entry idempotencyEntry
	if err := json.Unmarshal([]byte(value), &entry); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Could not parse idempotency state",
		})
	}

	if entry.BodyHash != bodyHash {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"status":  "error",
			"message": "Idempotency-Key has already been used with a different request body",
		})
	}

	if entry.InProgress {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"status":  "error",
			"message": "A request with this Idempotency-Key is already being processed",
		})
	}

	c.Set(fiber.HeaderContentType, entry.ContentType)
	return c.Status(entry.StatusCode).Send(entry.Response)
}

func buildIdempotencyStorageKey(userID uint64, path, key string) string {
	return strings.Join([]string{"idempotency", "records", path, formatUserID(userID), key}, ":")
}

func hashRequestBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func pruneExpiredIdempotencyEntries(now time.Time) {
	for key, entry := range idempotencyStore.Entries {
		if now.After(entry.ExpiresAt) {
			delete(idempotencyStore.Entries, key)
		}
	}
}

func isASCIIPrintable(value string) bool {
	for _, r := range value {
		if r < 33 || r > 126 {
			return false
		}
	}

	return true
}

func formatUserID(userID uint64) string {
	return strconv.FormatUint(userID, 10)
}
