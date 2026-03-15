package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DBSource           string
	JWTSecret          string
	GCSBucketName      string
	GCSCredentialsFile string
	AIServiceURL       string
	OpenWeatherMapKey  string
	WeatherAPIURL      string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	config := &Config{
		Port:               getEnv("PORT", "8080"),
		DBSource:           getEnv("DB_SOURCE", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		GCSBucketName:      getEnv("GCS_BUCKET_NAME", ""),
		GCSCredentialsFile: getEnv("GCS_CREDENTIALS_FILE", "service-account.json"),
		AIServiceURL:       getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		OpenWeatherMapKey:  getEnv("OPEN_WEATHER_MAP_KEY", ""),
		WeatherAPIURL:      getEnv("WEATHER_API_URL", "https://api.openweathermap.org/data/2.5/weather"),
	}

	// Validate required variables
	if config.DBSource == "" {
		log.Fatal("DB_SOURCE environment variable is not set")
	}
	if config.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	return config, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
