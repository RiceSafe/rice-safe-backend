package server

import (
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
	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/gofiber/swagger"
)

// SetupApp wires up all dependencies and returns the Fiber app.
// It is extracted from main.go so we can use it cleanly during integration tests.
func SetupApp(
	cfg *config.Config,
	storageService storage.Service,
	aiClient ai_client.Client,
	weatherClient dashboard.WeatherClient,
) *fiber.App {

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
	auth.RegisterRoutes(app, authService, cfg.JWTSecret)
	disease.RegisterRoutes(app, cfg.JWTSecret, diseaseService)
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Protected Routes
	api := app.Group("/api")
	api.Use(auth.Protected(cfg.JWTSecret))

	// Register Diagnosis Routes
	diagnosis.RegisterRoutes(api, diagnosisService)
	// Register Outbreak Routes
	outbreak.RegisterRoutes(api, outbreakService)
	// Register Community Routes
	community.RegisterRoutes(api, communityService)
	// Register Notification Routes
	notificationHandler.RegisterRoutes(api, cfg.JWTSecret)

	// Weather client is optional in some tests
	if weatherClient != nil {
		dashboard.RegisterRoutes(api, weatherClient, cfg.JWTSecret)
	}

	return app
}
