package main

import (
	"log"
	"os"

	"github.com/RiceSafe/rice-safe-backend/internal/auth"
	"github.com/RiceSafe/rice-safe-backend/internal/community"
	"github.com/RiceSafe/rice-safe-backend/internal/config"
	"github.com/RiceSafe/rice-safe-backend/internal/dashboard"
	"github.com/RiceSafe/rice-safe-backend/internal/diagnosis"
	"github.com/RiceSafe/rice-safe-backend/internal/disease"
	"github.com/RiceSafe/rice-safe-backend/internal/notification"
	"github.com/RiceSafe/rice-safe-backend/internal/outbreak"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/ai_client"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	_ "github.com/RiceSafe/rice-safe-backend/docs"
	"github.com/gofiber/swagger"
)

// @title RiceSafe API
// @version 1.0
// @description Backend API for RiceSafe Application
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

	// Connect Database
	database.ConnectDB(cfg.DBSource)
	defer database.CloseDB()

	// Initialize Infrastructure
	storageService, err := storage.NewGCSService(cfg.GCSBucketName, cfg.GCSCredentialsFile)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	aiClient := ai_client.NewClient(cfg.AIServiceURL)
	weatherClient := dashboard.NewWeatherClient(cfg.OpenWeatherMapKey, cfg.WeatherAPIURL)

	// Initialize Modules
	authRepo := auth.NewRepository()
	authService := auth.NewService(authRepo, cfg.JWTSecret, storageService)
	auth.NewHandler(authService)

	diseaseRepo := disease.NewRepository()
	diseaseService := disease.NewService(diseaseRepo, storageService)

	outbreakRepo := outbreak.NewRepository()
	outbreakService := outbreak.NewService(outbreakRepo, storageService)

	notificationRepo := notification.NewRepository()
	notificationService := notification.NewService(notificationRepo)
	notificationHandler := notification.NewHandler(notificationService)

	diagnosisRepo := diagnosis.NewRepository()
	diagnosisService := diagnosis.NewService(diagnosisRepo, diseaseRepo, outbreakRepo, storageService, aiClient, notificationService)

	communityRepo := community.NewRepository()
	communityService := community.NewService(communityRepo, storageService)

	// Setup Fiber
	app := fiber.New()
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New())

	// Health Check
	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "env": os.Getenv("ENV")})
	})

	// Public Routes
	auth.RegisterRoutes(app, authService)
	disease.RegisterRoutes(app, diseaseService)
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Protected Routes
	api := app.Group("/api")
	api.Use(auth.Protected())

	// Register Diagnosis Routes
	diagnosis.RegisterRoutes(api, diagnosisService)
	// Register Outbreak Routes
	outbreak.RegisterRoutes(api, outbreakService)
	// Register Community Routes
	community.RegisterRoutes(api, communityService)
	// Register Notification Routes
	notificationHandler.RegisterRoutes(api)
	// Register Dashboard Routes
	dashboard.RegisterRoutes(api, weatherClient)

	// Start Server
	log.Fatal(app.Listen(":" + cfg.Port))
}
