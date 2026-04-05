package docs

import (
	"embed"
	"io/fs"

	"github.com/gofiber/fiber/v2"
)

//go:embed openapi.yaml swagger.html
var embeddedDocs embed.FS

func Register(app *fiber.App) {
	app.Get("/docs", func(c *fiber.Ctx) error {
		content, err := fs.ReadFile(embeddedDocs, "swagger.html")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("failed to load swagger ui")
		}

		c.Type("html", "utf-8")
		return c.Send(content)
	})

	app.Get("/docs/", func(c *fiber.Ctx) error {
		return c.Redirect("/docs", fiber.StatusTemporaryRedirect)
	})

	app.Get("/docs/openapi.yaml", func(c *fiber.Ctx) error {
		content, err := fs.ReadFile(embeddedDocs, "openapi.yaml")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("failed to load openapi spec")
		}

		c.Type("yaml", "utf-8")
		return c.Send(content)
	})
}
