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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpertRoleIntegration(t *testing.T) {
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

	// --- Helper: Register Users (FARMER and EXPERT) ---
	registerUser := func(username, email, role string) string {
		reqBody, _ := json.Marshal(auth.RegisterRequest{
			Username: username,
			Email:    email,
			Password: "securepassword",
			Role:     role,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		var authData auth.AuthResponse
		json.NewDecoder(resp.Body).Decode(&authData)
		return authData.Token
	}

	farmerToken := registerUser("farmerjoe", "joe@farm.com", "FARMER")
	expertToken := registerUser("drrice", "doc@rice.com", "EXPERT")
	require.NotEmpty(t, farmerToken)
	require.NotEmpty(t, expertToken)

	// First, inject a dummy disease
	diseaseID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO diseases (id, alias, name, category, image_url, description, spread_details, match_weather, symptoms, prevention, treatment)
		VALUES ($1, 'rice_blast', 'Rice Blast', 'Fungal', 'http://example.com/img.jpg', 'Lesions on leaves', 'Wind-borne spores', '["High Humidity"]'::jsonb, '[{"title": "Leaves", "description": "Diamond shaped lesions"}]'::jsonb, '[{"title": "Fungicide", "description": "Apply early"}]'::jsonb, '[{"title": "Treatment", "description": "Use X"}]'::jsonb)
	`, diseaseID)
	require.NoError(t, err, "Failed to insert test disease")

	// Inject an unverified outbreak straight into the database
	outbreakID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO outbreaks (id, disease_id, latitude, longitude, is_verified)
		VALUES ($1, $2, 13.0, 100.0, false)
	`, outbreakID, diseaseID)
	require.NoError(t, err, "Failed to insert test outbreak")

	t.Run("Farmer cannot verify an outbreak", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/outbreaks/"+outbreakID.String()+"/verify", nil)
		req.Header.Set("Authorization", "Bearer "+farmerToken)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "Farmers should be forbidden from verifying")
	})

	t.Run("Expert can verify an outbreak", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/outbreaks/"+outbreakID.String()+"/verify", nil)
		req.Header.Set("Authorization", "Bearer "+expertToken)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Experts should be able to verify")

		var res map[string]string
		err = json.NewDecoder(resp.Body).Decode(&res)
		require.NoError(t, err)
		assert.Equal(t, "Outbreak successfully verified", res["message"])

		// Verify in the database directly
		var isVerified bool
		var verifiedBy *uuid.UUID
		err = db.Pool.QueryRow(ctx, "SELECT is_verified, verified_by FROM outbreaks WHERE id = $1", outbreakID).Scan(&isVerified, &verifiedBy)
		require.NoError(t, err)

		assert.True(t, isVerified)
		assert.NotNil(t, verifiedBy)
	})
}
