package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RiceSafe/rice-safe-backend/internal/auth"
	"github.com/RiceSafe/rice-safe-backend/internal/config"
	"github.com/RiceSafe/rice-safe-backend/internal/diagnosis"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/ai_client"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/email"
	"github.com/RiceSafe/rice-safe-backend/internal/server"
	"github.com/RiceSafe/rice-safe-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiagnosisE2EIntegration(t *testing.T) {
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

	// Inject essential disease data for the test to succeed
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO diseases (id, alias, name, category, image_url, description, spread_details, match_weather, symptoms, prevention, treatment)
		VALUES (gen_random_uuid(), 'rice_blast', 'Rice Blast', 'Fungal', 'http://example.com/img.jpg', 'Lesions on leaves', 'Wind-borne spores', '["High Humidity"]'::jsonb, '[{"title": "Leaves", "description": "Diamond shaped lesions"}]'::jsonb, '[{"title": "Fungicide", "description": "Apply early"}]'::jsonb, '[{"title": "Treatment", "description": "Use X"}]'::jsonb)
	`)
	require.NoError(t, err)

	// --- Helper: Register & Login to get a token ---
	func() {
		reqBody, _ := json.Marshal(auth.RegisterRequest{
			Username: "diagnosisfarmer",
			Email:    "farmer3@test.com",
			Password: "securepassword",
			Role:     "FARMER",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}()

	loginBody, _ := json.Marshal(auth.LoginRequest{
		Email:    "farmer3@test.com",
		Password: "securepassword",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, _ := app.Test(loginReq)

	var authData auth.AuthResponse
	json.NewDecoder(loginResp.Body).Decode(&authData)
	token := authData.Token
	require.NotEmpty(t, token, "Need a valid token to test diagnosis")

	t.Run("Perform E2E Diagnosis and Trigger Outbreak", func(t *testing.T) {
		// Mock the AI to definitively predict rice_blast
		mockAI.PredictFunc = func(image []byte, filename, description string) (*ai_client.PredictionResponse, error) {
			return &ai_client.PredictionResponse{
				Prediction: "rice_blast",
				Confidence: "98.50%",
			}, nil
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		
		// Create dummy image file part
		part, err := writer.CreateFormFile("image", "test_leaf.jpg")
		require.NoError(t, err)
		part.Write([]byte("fake image data"))

		writer.WriteField("latitude", "13.7563")
		writer.WriteField("longitude", "100.5018")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/diagnosis", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)

		// This endpoint does a lot: GCS, AI, DB inserts, Notification triggers. Wait 5 seconds to be safe.
		resp, err := app.Test(req, int(5*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var diagRes diagnosis.DiagnosisResponse
		err = json.NewDecoder(resp.Body).Decode(&diagRes)
		require.NoError(t, err)

		// Assertions
		assert.NotNil(t, diagRes.DiseaseResult)
		assert.Equal(t, "Rice Blast", diagRes.DiseaseResult.Name)
		assert.Equal(t, float64(98.50), diagRes.Confidence)

		// Verify that an Outbreak record was actually created in the DB
		var outbreakCount int
		err = db.Pool.QueryRow(ctx, "SELECT count(*) FROM outbreaks WHERE disease_id = $1", diagRes.DiseaseResult.ID).Scan(&outbreakCount)
		require.NoError(t, err)
		assert.Equal(t, 1, outbreakCount, "An outbreak record should have been created for a confident diagnosis")
	})
}
