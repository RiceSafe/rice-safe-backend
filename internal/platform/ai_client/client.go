package ai_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type Client interface {
	Predict(imageBytes []byte, filename string, description string) (*PredictionResponse, error)
}

type client struct {
	baseURL string
	timeout time.Duration
}

type PredictionResponse struct {
	Prediction string `json:"prediction"`
	Confidence string `json:"confidence"`
}

func NewClient(baseURL string) Client {
	if baseURL == "" {
		baseURL = os.Getenv("AI_SERVICE_URL")
	}
	if baseURL == "" {
		baseURL = "http://rice-safe-ai:8000" // Default in Docker
	}
	return &client{
		baseURL: baseURL,
		timeout: 90 * time.Second,
	}
}

func (c *client) Predict(imageBytes []byte, filename string, description string) (*PredictionResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add Image
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		return nil, fmt.Errorf("failed to write image bytes: %w", err)
	}

	// Add Description
	if err := writer.WriteField("description", description); err != nil {
		return nil, fmt.Errorf("failed to write description: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Make Request
	req, err := http.NewRequest("POST", c.baseURL+"/predict/", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	httpClient := &http.Client{Timeout: c.timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai service error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result PredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
