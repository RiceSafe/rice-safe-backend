package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WeatherClient interface {
	GetWeather(lat, lon float64) (*WeatherResponse, error)
}

type weatherClient struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

func NewWeatherClient(apiKey, apiURL string) WeatherClient {
	return &weatherClient{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// OpenWeatherMap Response structs
type owmResponse struct {
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Name string `json:"name"`
}

// WeatherResponse is our internal clean struct to return to the mobile app
type WeatherResponse struct {
	LocationName string  `json:"location_name"`
	Temperature  float64 `json:"temperature"`
	Condition    string  `json:"condition"`
	Description  string  `json:"description"`
	Humidity     int     `json:"humidity"`
	IconURL      string  `json:"icon_url"`
}

func (c *weatherClient) GetWeather(lat, lon float64) (*WeatherResponse, error) {
	// Return error if no API key is provided, so the handler can process it gracefully
	if c.apiKey == "" {
		return nil, fmt.Errorf("weather API key is not configured")
	}

	url := fmt.Sprintf("%s?lat=%f&lon=%f&appid=%s&units=metric", c.apiURL, lat, lon, c.apiKey)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather api returned status: %d", resp.StatusCode)
	}

	var owm owmResponse
	if err := json.NewDecoder(resp.Body).Decode(&owm); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Create safe response
	res := &WeatherResponse{
		LocationName: owm.Name,
		Temperature:  owm.Main.Temp,
		Humidity:     owm.Main.Humidity,
	}

	if len(owm.Weather) > 0 {
		res.Condition = owm.Weather[0].Main
		res.Description = owm.Weather[0].Description
		res.IconURL = fmt.Sprintf("http://openweathermap.org/img/wn/%s@2x.png", owm.Weather[0].Icon)
	}

	return res, nil
}
