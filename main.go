package main

import (
	"finance/config"
	"finance/database"
	"finance/routes"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal(err)
	}

	database.MySqlConnectDb(cfg)
	if err := database.RedisConnectDb(cfg); err != nil {
		log.Printf("redis unavailable, falling back to in-memory storage where supported: %v", err)
	}

	app := fiber.New()

	routes.SetupRoutes(app)
	err = app.Listen(":3000")
	if err != nil {
		log.Fatal(err)
	}
}
