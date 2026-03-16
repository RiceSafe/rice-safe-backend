package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RiceSafe/rice-safe-backend/internal/auth"
	"github.com/RiceSafe/rice-safe-backend/internal/config"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/email"
	"github.com/RiceSafe/rice-safe-backend/internal/server"
	"github.com/RiceSafe/rice-safe-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthIntegration(t *testing.T) {
	// Setup Database Infrastructure
	ctx := context.Background()
	db, err := testutil.SetupTestDB(ctx)
	require.NoError(t, err)
	defer db.Teardown(ctx)

	// Setup Mocks and Config
	mockStorage := &testutil.MockStorageService{}
	mockAI := &testutil.MockAIService{}
	cfg := &config.Config{
		JWTSecret: "test-secret",
		Port:      "8080",
	}

	// Mount the Fiber App
	mockEmail := &email.MockEmailService{}
	app := server.SetupApp(cfg, mockStorage, mockAI, nil, mockEmail)

	// --- TEST SUITE ENTRANCE ---

	// Clear DB before we begin
	err = db.TruncateAll(ctx)
	require.NoError(t, err)

	var registeredToken string

	t.Run("Register a new user (Farmer)", func(t *testing.T) {
		payload := auth.RegisterRequest{
			Username: "testfarmer",
			Email:    "farmer@test.com",
			Password: "securepassword",
			Role:     "FARMER",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, int(2*time.Second.Milliseconds())) // Wait max 2 seconds
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var resBody auth.AuthResponse
		err = json.NewDecoder(resp.Body).Decode(&resBody)
		require.NoError(t, err)

		assert.NotEmpty(t, resBody.Token)
		assert.Equal(t, "testfarmer", resBody.User.Username)
		assert.Equal(t, "farmer@test.com", resBody.User.Email)
		assert.Equal(t, "FARMER", resBody.User.Role)

		registeredToken = resBody.Token
	})

	t.Run("Prevent duplicate registration", func(t *testing.T) {
		payload := auth.RegisterRequest{
			Username: "farmer2",
			Email:    "farmer@test.com", // Same email!
			Password: "password123",
			Role:     "FARMER",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode) // 409 Conflict
	})

	t.Run("Login with correct credentials", func(t *testing.T) {
		payload := auth.LoginRequest{
			Email:    "farmer@test.com",
			Password: "securepassword",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resBody auth.AuthResponse
		err = json.NewDecoder(resp.Body).Decode(&resBody)
		require.NoError(t, err)

		assert.NotEmpty(t, resBody.Token)
	})

	t.Run("Access a protected route with a valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+registeredToken)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var resUser auth.User
		err = json.NewDecoder(resp.Body).Decode(&resUser)
		require.NoError(t, err)
		assert.Equal(t, "testfarmer", resUser.Username)
	})

	t.Run("Protect route from unauthorized access", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		// Missing Authorization header entirely

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
