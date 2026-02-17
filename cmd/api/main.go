package main

import (
	"log"

	_ "github.com/RiceSafe/rice-safe-backend/docs"
	"github.com/RiceSafe/rice-safe-backend/internal/auth"
	"github.com/RiceSafe/rice-safe-backend/internal/config"
	"github.com/RiceSafe/rice-safe-backend/internal/disease"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title RiceSafe Backend API
// @version 1.0
// @description Backend API for RiceSafe Mobile Application
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Database
	database.ConnectDB(cfg.DBSource)
	defer database.CloseDB()

	app := fiber.New()

	// Middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())

	// Initialize Storage Module
	storageService, err := storage.NewGCSService(cfg.GCSBucketName, cfg.GCSCredentialsFile)
	if err != nil {
		log.Printf("Failed to initialize GCS storage: %v", err)
	}

	// Initialize Auth Module
	authRepo := auth.NewRepository()
	authService := auth.NewService(authRepo, cfg.JWTSecret, storageService)
	auth.RegisterRoutes(app, authService)

	// Initialize Disease Module
	diseaseRepo := disease.NewRepository()
	diseaseService := disease.NewService(diseaseRepo, storageService)
	disease.RegisterRoutes(app, diseaseService)

	// Upload Endpoint (Utility)
	storageHandler := storage.NewHandler(storageService)
	app.Post("/api/upload", storageHandler.UploadFile)

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

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
