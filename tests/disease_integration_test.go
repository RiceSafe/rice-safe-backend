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
	"github.com/RiceSafe/rice-safe-backend/internal/disease"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/email"
	"github.com/RiceSafe/rice-safe-backend/internal/server"
	"github.com/RiceSafe/rice-safe-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiseaseRouteGuard(t *testing.T) {
	ctx := context.Background()
	db, err := testutil.SetupTestDB(ctx)
	require.NoError(t, err)
	defer db.Teardown(ctx)

	mockStorage := &testutil.MockStorageService{}
	mockAI := &testutil.MockAIService{}
	cfg := &config.Config{JWTSecret: "test-secret"}
	mockEmail := &email.MockEmailService{}
	app := server.SetupApp(cfg, mockStorage, mockAI, nil, mockEmail)

	err = db.TruncateAll(ctx)
	require.NoError(t, err)

	// Helper: register user and return token
	registerUser := func(username, email, role string) string {
		body, _ := json.Marshal(auth.RegisterRequest{
			Username: username,
			Email:    email,
			Password: "securepassword",
			Role:     role,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		var res auth.AuthResponse
		json.NewDecoder(resp.Body).Decode(&res)
		return res.Token
	}

	farmerToken := registerUser("testfarmer2", "farmer2@test.com", "FARMER")
	expertToken := registerUser("drrice2", "expert2@test.com", "EXPERT")
	// Register admin as FARMER first (ADMIN role is blocked from public register for security),
	// then promote via SQL and re-login to get a JWT with role=ADMIN.
	registerUser("adminuser", "admin@test.com", "FARMER")
	_, err = db.Pool.Exec(ctx, `UPDATE users SET role = 'ADMIN' WHERE email = 'admin@test.com'`)
	require.NoError(t, err)

	// Re-login to get a JWT with role=ADMIN
	reloginBody, _ := json.Marshal(map[string]string{"email": "admin@test.com", "password": "securepassword"})
	reloginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(reloginBody))
	reloginReq.Header.Set("Content-Type", "application/json")
	reloginResp, err := app.Test(reloginReq, int(2*time.Second.Milliseconds()))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, reloginResp.StatusCode)
	var adminLoginData auth.AuthResponse
	json.NewDecoder(reloginResp.Body).Decode(&adminLoginData)
	adminToken := adminLoginData.Token

	require.NotEmpty(t, farmerToken)
	require.NotEmpty(t, expertToken)
	require.NotEmpty(t, adminToken)

	diseasePayload := disease.Disease{
		Alias:       "brown_spot",
		Name:        "Brown Spot",
		Category:    "Fungal",
		Description: "Brown lesions on leaves",
	}

	t.Run("Public: anyone can GET disease list (no auth)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/diseases", nil)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Public: anyone can GET disease categories (no auth)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/diseases/categories", nil)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Farmer CANNOT create disease (403 Forbidden)", func(t *testing.T) {
		body, _ := json.Marshal(diseasePayload)
		req := httptest.NewRequest(http.MethodPost, "/api/diseases", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+farmerToken)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Unauthenticated request CANNOT create disease (401 Unauthorized)", func(t *testing.T) {
		body, _ := json.Marshal(diseasePayload)
		req := httptest.NewRequest(http.MethodPost, "/api/diseases", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	var createdID string

	t.Run("Expert CAN create disease (201 Created)", func(t *testing.T) {
		body, _ := json.Marshal(diseasePayload)
		req := httptest.NewRequest(http.MethodPost, "/api/diseases", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+expertToken)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created disease.Disease
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)
		assert.Equal(t, "Brown Spot", created.Name)
		createdID = created.ID.String()
	})

	t.Run("Farmer CANNOT update disease (403 Forbidden)", func(t *testing.T) {
		if createdID == "" {
			t.Skip("Skipping: no disease was created in prior test")
		}
		update := disease.Disease{Name: "Brown Spot Updated"}
		body, _ := json.Marshal(update)
		req := httptest.NewRequest(http.MethodPut, "/api/diseases/"+createdID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+farmerToken)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Admin CAN update disease (200 OK)", func(t *testing.T) {
		if createdID == "" {
			t.Skip("Skipping: no disease was created in prior test")
		}
		update := disease.Disease{Name: "Brown Spot (Admin Updated)", Alias: "brown_spot", Category: "Fungal", Description: "Updated desc"}
		body, _ := json.Marshal(update)
		req := httptest.NewRequest(http.MethodPut, "/api/diseases/"+createdID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
