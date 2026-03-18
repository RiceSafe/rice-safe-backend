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

func TestAdminAPIIntegration(t *testing.T) {
	ctx := context.Background()
	db, err := testutil.SetupTestDB(ctx)
	require.NoError(t, err)
	defer db.Teardown(ctx)

	mockStorage := &testutil.MockStorageService{}
	mockAI := &testutil.MockAIService{}
	mockEmail := &email.MockEmailService{}
	cfg := &config.Config{JWTSecret: "test-secret"}
	app := server.SetupApp(cfg, mockStorage, mockAI, nil, mockEmail)

	err = db.TruncateAll(ctx)
	require.NoError(t, err)

	// Helper: register user and return token
	registerUser := func(username, emailAddr, role string) string {
		body, _ := json.Marshal(auth.RegisterRequest{
			Username: username,
			Email:    emailAddr,
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

	farmerToken := registerUser("farmerjoe3", "farmer3@admin.com", "FARMER")
	expertToken := registerUser("drrice3", "expert3@admin.com", "EXPERT")

	// Promote a user to ADMIN via SQL + re-login to get ADMIN JWT
	registerUser("adminuser2", "admin2@admin.com", "FARMER")
	_, err = db.Pool.Exec(ctx, `UPDATE users SET role = 'ADMIN' WHERE email = 'admin2@admin.com'`)
	require.NoError(t, err)

	reloginBody, _ := json.Marshal(map[string]string{"email": "admin2@admin.com", "password": "securepassword"})
	reloginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(reloginBody))
	reloginReq.Header.Set("Content-Type", "application/json")
	reloginResp, err := app.Test(reloginReq, int(2*time.Second.Milliseconds()))
	require.NoError(t, err)
	var adminAuth auth.AuthResponse
	json.NewDecoder(reloginResp.Body).Decode(&adminAuth)
	adminToken := adminAuth.Token

	require.NotEmpty(t, farmerToken)
	require.NotEmpty(t, expertToken)
	require.NotEmpty(t, adminToken)

	// --- User Management ---

	t.Run("Farmer CANNOT access admin user list (403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+farmerToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Expert CANNOT access admin user list (403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+expertToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Admin CAN list all users (200)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var users []map[string]any
		json.NewDecoder(resp.Body).Decode(&users)
		assert.GreaterOrEqual(t, len(users), 3, "should have at least 3 registered users")
	})

	t.Run("Admin CAN filter users by role (200)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users?role=EXPERT", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var users []map[string]any
		json.NewDecoder(resp.Body).Decode(&users)
		assert.Equal(t, 1, len(users), "should have exactly 1 EXPERT user")
	})

	// --- Role Promotion ---

	var farmerUserID string
	t.Run("Admin CAN get farmer user ID from list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users?role=FARMER", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, _ := app.Test(req, int(2*time.Second.Milliseconds()))
		var users []map[string]any
		json.NewDecoder(resp.Body).Decode(&users)
		require.NotEmpty(t, users)
		// Pick the first FARMER in the list
		farmerUserID = users[0]["id"].(string)
	})

	t.Run("Admin CAN promote farmer to Expert (200)", func(t *testing.T) {
		if farmerUserID == "" {
			t.Skip("No farmer user ID found")
		}
		body, _ := json.Marshal(map[string]string{"role": "EXPERT"})
		req := httptest.NewRequest(http.MethodPut, "/api/users/"+farmerUserID+"/role", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify in DB
		var role string
		db.Pool.QueryRow(ctx, "SELECT role FROM users WHERE id = $1", farmerUserID).Scan(&role)
		assert.Equal(t, "EXPERT", role)
	})

	t.Run("Admin gets 400 on invalid role value", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"role": "SUPERUSER"})
		req := httptest.NewRequest(http.MethodPut, "/api/users/"+uuid.New().String()+"/role", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// --- Outbreak Management ---

	// Inject test disease + outbreak directly
	diseaseID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO diseases (id, alias, name, category, image_url, description, spread_details, match_weather, symptoms, prevention, treatment)
		VALUES ($1, 'test_disease', 'Test Disease', 'Fungal', '', '', '', '[]'::jsonb, '[]'::jsonb, '[]'::jsonb, '[]'::jsonb)
	`, diseaseID)
	require.NoError(t, err)

	outbreakID := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO outbreaks (id, disease_id, latitude, longitude, is_verified) VALUES ($1, $2, 13.0, 100.0, false)
	`, outbreakID, diseaseID)
	require.NoError(t, err)

	t.Run("Admin CAN list all outbreaks (200)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/outbreaks", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var outbreaks []map[string]any
		json.NewDecoder(resp.Body).Decode(&outbreaks)
		assert.Equal(t, 1, len(outbreaks))
	})

	t.Run("Admin CAN delete outbreak (200)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/outbreaks/"+outbreakID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify it's gone from DB
		var count int
		db.Pool.QueryRow(ctx, "SELECT count(*) FROM outbreaks WHERE id = $1", outbreakID).Scan(&count)
		assert.Equal(t, 0, count)
	})

	// --- Community Post Moderation ---

	// Inject a post directly
	postID := uuid.New()
	var farmerDBID uuid.UUID
	db.Pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'farmer3@admin.com'").Scan(&farmerDBID)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO posts (id, user_id, content) VALUES ($1, $2, 'test post')
	`, postID, farmerDBID)
	require.NoError(t, err)

	t.Run("Admin CAN delete community post (200)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/community/posts/"+postID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var count int
		db.Pool.QueryRow(ctx, "SELECT count(*) FROM posts WHERE id = $1", postID).Scan(&count)
		assert.Equal(t, 0, count)
	})
}
