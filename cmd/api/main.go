package main

import (
	"log"
	"os"

	_ "github.com/RiceSafe/rice-safe-backend/docs"
	"github.com/RiceSafe/rice-safe-backend/internal/auth"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title RiceSafe Backend API
// @version 1.0
// @description Backend API for RiceSafe Mobile Application
// @host localhost:8080
// @BasePath /api
func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to Database
	database.ConnectDB()
	defer database.CloseDB()

	app := fiber.New()

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())

	// Initialize Auth Module
	authRepo := auth.NewRepository()
	authService := auth.NewService(authRepo)
	auth.RegisterRoutes(app, authService)

	// Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "RiceSafe Backend is running",
			"db":      "connected",
		})
	})

	// Swagger
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
