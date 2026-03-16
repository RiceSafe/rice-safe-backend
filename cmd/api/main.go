package main

import (
	"log"

	"github.com/RiceSafe/rice-safe-backend/internal/config"
	"github.com/RiceSafe/rice-safe-backend/internal/dashboard"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/ai_client"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/RiceSafe/rice-safe-backend/internal/server"
	_ "github.com/RiceSafe/rice-safe-backend/docs"
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

	app := server.SetupApp(cfg, storageService, aiClient, weatherClient)

	// Start Server
	log.Fatal(app.Listen(":" + cfg.Port))
}
