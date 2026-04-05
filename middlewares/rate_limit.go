package middlewares

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

const (
	globalRateLimitMax    = 120
	globalRateLimitWindow = time.Minute
	authRateLimitMax      = 10
	authRateLimitWindow   = 10 * time.Minute
	signupRateLimitMax    = 5
	signupRateLimitWindow = 15 * time.Minute
)

func GlobalRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        globalRateLimitMax,
		Expiration: globalRateLimitWindow,
		Next: func(c *fiber.Ctx) bool {
			return strings.HasPrefix(c.Path(), "/auth")
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  "error",
				"message": "Too many requests. Please slow down and try again shortly.",
			})
		},
	})
}

func AuthRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        authRateLimitMax,
		Expiration: authRateLimitWindow,
		KeyGenerator: func(c *fiber.Ctx) string {
			return strings.ToLower(c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  "error",
				"message": "Too many authentication attempts. Please try again later.",
			})
		},
	})
}

func SignupRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        signupRateLimitMax,
		Expiration: signupRateLimitWindow,
		KeyGenerator: func(c *fiber.Ctx) string {
			return strings.ToLower(c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  "error",
				"message": "Too many signup attempts. Please try again later.",
			})
		},
	})
}
