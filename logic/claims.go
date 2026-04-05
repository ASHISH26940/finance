package logic

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
)

func ExtractUserID(c *fiber.Ctx) (uint64, bool) {
	claims, ok := c.Locals("claims").(jwt.MapClaims)
	if !ok {
		return 0, false
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, false
	}

	return uint64(userIDFloat), true
}
