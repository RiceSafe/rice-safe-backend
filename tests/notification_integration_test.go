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
	"github.com/RiceSafe/rice-safe-backend/internal/notification"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/email"
	"github.com/RiceSafe/rice-safe-backend/internal/server"
	"github.com/RiceSafe/rice-safe-backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationIntegration(t *testing.T) {
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

	// --- Helper: Register & Login ---
	registerUser := func(username, email, role string) (string, uuid.UUID) {
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
		
		id, _ := uuid.Parse(authData.User.ID.String())
		return authData.Token, id
	}

	token, userID := registerUser("notifuser", "notif@test.com", "FARMER")
	require.NotEmpty(t, token)

	// Inject a couple of notifications directly into the DB
	notifID1 := uuid.New()
	notifID2 := uuid.New()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO notifications (id, user_id, title, body, type, is_read, created_at)
		VALUES 
		($1, $3, 'Alert 1', 'Near outbreak', 'OUTBREAK_NEARBY', false, NOW()),
		($2, $3, 'Alert 2', 'Community update', 'COMMUNITY', false, NOW() - INTERVAL '1 hour')
	`, notifID1, notifID2, userID)
	require.NoError(t, err)

	t.Run("Get unread count", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var res map[string]int
		json.NewDecoder(resp.Body).Decode(&res)
		assert.Equal(t, 2, res["unread_count"])
	})

	t.Run("List notifications", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/notifications?limit=10&offset=0", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var notifs []notification.Notification
		json.NewDecoder(resp.Body).Decode(&notifs)
		assert.Len(t, notifs, 2)
		assert.Equal(t, "Alert 1", notifs[0].Title)
	})

	t.Run("Mark notification as read", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/notifications/"+notifID1.String()+"/read", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify unread count decreased
		reqCount := httptest.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
		reqCount.Header.Set("Authorization", "Bearer "+token)
		respCount, _ := app.Test(reqCount)
		var res map[string]int
		json.NewDecoder(respCount.Body).Decode(&res)
		assert.Equal(t, 1, res["unread_count"])
	})

	t.Run("Mark all notifications as read", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/notifications/read-all", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify unread count is zero
		reqCount := httptest.NewRequest(http.MethodGet, "/api/notifications/unread-count", nil)
		reqCount.Header.Set("Authorization", "Bearer "+token)
		respCount, _ := app.Test(reqCount)
		var res map[string]int
		json.NewDecoder(respCount.Body).Decode(&res)
		assert.Equal(t, 0, res["unread_count"])
	})
}
