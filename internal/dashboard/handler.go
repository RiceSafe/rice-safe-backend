package dashboard

import (
	"github.com/RiceSafe/rice-safe-backend/internal/auth"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	weatherClient WeatherClient
}

func NewHandler(weatherClient WeatherClient) *Handler {
	return &Handler{
		weatherClient: weatherClient,
	}
}

func RegisterRoutes(router fiber.Router, weatherClient WeatherClient, jwtSecret string) {
	h := NewHandler(weatherClient)

	group := router.Group("/dashboard")
	group.Use(auth.Protected(jwtSecret))
	group.Get("/weather", h.GetWeather)
}

// GetWeather godoc
// @Summary      Get current weather
// @Description  Get real-time weather data for a given location using OpenWeatherMap
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Param        lat        query     number  true "Latitude"
// @Param        long       query     number  true "Longitude"
// @Success      200  {object}  WeatherResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /dashboard/weather [get]
func (h *Handler) GetWeather(c *fiber.Ctx) error {
	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("long", 0)

	if c.Query("lat") == "" || c.Query("long") == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Latitude and longitude are required"})
	}

	weather, err := h.weatherClient.GetWeather(lat, lon)
	if err != nil {
		// Return 503 instead of 500 to clearly indicate the service is unavailable (e.g. missing API key)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Weather data is currently unavailable"})
	}

	return c.JSON(weather)
}
