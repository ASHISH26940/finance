package middlewares

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"finance/config"
	"finance/database"
	"finance/models"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
)

var blacklistedTokens = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tokenString string

		cookieToken := c.Cookies("token")
		if cookieToken != "" {
			tokenString = cookieToken
		} else {
			authHeader := c.Get("Authorization")
			if authHeader == "" {
				c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"status":  "error",
					"message": "Missing Authorization header and cookie",
				})
				return nil
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"status":  "error",
					"message": "Invalid token format",
				})
				return nil
			}

			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenString == "" {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Missing token",
			})
			return nil
		}

		if config.GlobalConfig.Secret == "" {
			log.Fatal("Secret Key is empty")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method : %v", token.Header["alg"])
			}
			return []byte(config.GlobalConfig.Secret), nil
		})

		if err != nil {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid or expired token",
			})
			return nil
		}

		if !token.Valid {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid token",
			})
			return nil
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid token claims",
			})
			return nil
		}

		userId, ok := claims["user_id"].(float64)
		if !ok {
			c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "User ID not found in token claims",
			})
			return nil
		}

		if !isUserActive(uint(userId)) {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid or expired token",
			})
			return nil
		}

		if isTokenBlacklisted(tokenString) {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token is blacklisted, please login again",
			})
			return nil
		}

		pua, ok := claims["password_updated_at"].(float64)
		if !ok {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Session expired, Please login again to continue",
			})
			return nil
		}

		if !validatePassExpiry(uint(userId), int64(pua)) {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Password update detected, please login again",
			})
			return nil
		}

		c.Locals("claims", claims)
		c.Locals("token", tokenString)
		return c.Next()
	}
}

func AddToBlacklist(token string, userId float64) {
	if database.RedisAvailable() {
		expiresIn := blacklistTTL(token)
		if err := database.RedisDb.Db.Set(blacklistTokenKey(token), strconv.FormatFloat(userId, 'f', -1, 64), expiresIn).Err(); err == nil {
			return
		}
	}

	blacklistedTokens.Lock()
	blacklistedTokens.m[blacklistTokenKey(token)] = token
	blacklistedTokens.Unlock()
}

func RemoveFromBlacklist(token string) {
	key := blacklistTokenKey(token)

	if database.RedisAvailable() {
		if err := database.RedisDb.Db.Del(key).Err(); err != nil {
			log.Printf("redis blacklist delete failed, falling back to in-memory blacklist cleanup: %v", err)
		}
	}

	blacklistedTokens.Lock()
	delete(blacklistedTokens.m, key)
	blacklistedTokens.Unlock()
}

func validatePassExpiry(userId uint, tokenTime int64) bool {
	var user struct {
		PasswordUpdatedAt int64 `gorm:"column:password_updated_at"`
	}

	if err := database.Database.Db.
		Table("users").
		Select("password_updated_at").
		Where("id = ?", userId).
		Scan(&user).Error; err != nil {
		return false
	}

	if user.PasswordUpdatedAt == 0 {
		return true
	}

	const skew int64 = 2

	if user.PasswordUpdatedAt > tokenTime {
		return user.PasswordUpdatedAt-tokenTime <= skew
	}

	return tokenTime-user.PasswordUpdatedAt <= skew
}

func isUserActive(userId uint) bool {
	var user struct {
		Active bool `gorm:"column:active"`
	}

	if err := database.Database.Db.
		Table("users").
		Select("active").
		Where("id = ?", userId).
		Scan(&user).Error; err != nil {
		return false
	}

	return user.Active
}

func isTokenBlacklisted(token string) bool {
	key := blacklistTokenKey(token)

	if database.RedisAvailable() {
		_, err := database.RedisDb.Db.Get(key).Result()
		if err == nil {
			return true
		}

		if err != nil && err != redis.Nil {
			log.Printf("redis blacklist lookup failed, falling back to in-memory blacklist: %v", err)
		}
	}

	blacklistedTokens.RLock()
	_, found := blacklistedTokens.m[key]
	blacklistedTokens.RUnlock()
	return found
}

func blacklistTokenKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "blacklist:jwt:" + hex.EncodeToString(sum[:])
}

func blacklistTTL(token string) time.Duration {
	if config.GlobalConfig == nil || strings.TrimSpace(config.GlobalConfig.Secret) == "" {
		return 24 * time.Hour
	}

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method : %v", token.Header["alg"])
		}
		return []byte(config.GlobalConfig.Secret), nil
	})
	if err != nil {
		return 24 * time.Hour
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return 24 * time.Hour
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return 24 * time.Hour
	}

	expiresIn := time.Until(time.Unix(int64(expFloat), 0))
	if expiresIn <= 0 {
		return time.Minute
	}

	return expiresIn
}

func Authorize(check func(models.Permission) bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claimsVal := c.Locals("claims")
		claims, ok := claimsVal.(jwt.MapClaims)
		if !ok {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Invalid claims",
			})
			return nil
		}

		roleIDFloat, ok := claims["role_id"].(float64)
		if !ok {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Role not found in token",
			})
			return nil
		}

		role := models.Role{}
		if err := database.Database.Db.First(&role, uint(roleIDFloat)).Error; err != nil {
			c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Role not found",
			})
			return nil
		}

		var perm models.Permission
		if err := json.Unmarshal([]byte(role.Permissions), &perm); err != nil {
			c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Permission parse failed",
			})
			return nil
		}

		if !check(perm) {
			c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Access denied",
			})
			return nil
		}

		return c.Next()
	}
}
