package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestClearAuthCookie(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		clearAuthCookie(c)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "token" {
		t.Fatalf("expected token cookie, got %q", cookie.Name)
	}

	if cookie.Value != "" {
		t.Fatalf("expected cleared cookie value, got %q", cookie.Value)
	}

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite Lax, got %v", cookie.SameSite)
	}
}
