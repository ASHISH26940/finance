package routes

import (
	"finance/controllers"
	"finance/docs"
	"finance/middlewares"
	"finance/models"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	docs.Register(app)
	app.Use(middlewares.GlobalRateLimit())

	auth := app.Group("/auth")
	auth.Post("/signup", middlewares.SignupRateLimit(), controllers.Signup)
	auth.Post("/login", middlewares.AuthRateLimit(), controllers.Login)
	auth.Post("/logout", middlewares.Protected(), controllers.Logout)

	d := app.Group("/dashboard", middlewares.Protected())
	d.Get("/summary",
		middlewares.Authorize(func(p models.Permission) bool { return p.CanViewDashboard }),
		controllers.GetDashboardSummary,
	)
	d.Get("/trends",
		middlewares.Authorize(func(p models.Permission) bool { return p.CanViewInsights }),
		controllers.GetDashboardTrends,
	)

	r := app.Group("/records", middlewares.Protected())

	r.Post("/",
		middlewares.Authorize(func(p models.Permission) bool { return p.CanCreateRecords }),
		middlewares.Idempotency(),
		controllers.CreateRecord,
	)

	r.Get("/",
		middlewares.Authorize(func(p models.Permission) bool { return p.CanViewRecords }),
		controllers.GetRecords,
	)

	r.Put("/:id",
		middlewares.Authorize(func(p models.Permission) bool { return p.CanUpdateRecords }),
		controllers.UpdateRecord,
	)

	r.Delete("/:id",
		middlewares.Authorize(func(p models.Permission) bool { return p.CanDeleteRecords }),
		controllers.DeleteRecord,
	)
}
