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
	"github.com/RiceSafe/rice-safe-backend/internal/community"
	"github.com/RiceSafe/rice-safe-backend/internal/config"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/email"
	"github.com/RiceSafe/rice-safe-backend/internal/server"
	"github.com/RiceSafe/rice-safe-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommunityIntegration(t *testing.T) {
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

	// --- Helper: Register & Login to get a token ---
	func() {
		reqBody, _ := json.Marshal(auth.RegisterRequest{
			Username: "postermaker",
			Email:    "poster@test.com",
			Password: "securepassword",
			Role:     "FARMER",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}()

	loginBody, _ := json.Marshal(auth.LoginRequest{
		Email:    "poster@test.com",
		Password: "securepassword",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, _ := app.Test(loginReq)

	var authData auth.AuthResponse
	json.NewDecoder(loginResp.Body).Decode(&authData)
	token := authData.Token
	require.NotEmpty(t, token, "Need a valid token to test community features")

	// --- TEST SUITE ENTRANCE ---

	t.Run("Create a Post without an image", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("content", "Hello from the test suite!")
		writer.WriteField("type", "GENERAL")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/community/posts", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var post community.Post
		json.NewDecoder(resp.Body).Decode(&post)
		assert.Equal(t, "Hello from the test suite!", post.Content)
		assert.NotEmpty(t, post.ID)
	})

	t.Run("Fetch the Feed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/community/posts?limit=10&offset=0", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var posts []community.PostResponse
		err = json.NewDecoder(resp.Body).Decode(&posts)
		require.NoError(t, err)

		assert.Len(t, posts, 1, "There should be exactly one post in the feed")
		assert.Equal(t, "Hello from the test suite!", posts[0].Content)
		assert.Equal(t, 0, posts[0].LikeCount)
		assert.Equal(t, 0, posts[0].CommentCount)
	})

	var postID string
	t.Run("Get first post ID from feed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/community/posts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ := app.Test(req)
		var posts []community.PostResponse
		json.NewDecoder(resp.Body).Decode(&posts)
		require.NotEmpty(t, posts)
		postID = posts[0].ID.String()
	})

	t.Run("Toggle Like on a post", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/community/posts/"+postID+"/like", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var res map[string]bool
		json.NewDecoder(resp.Body).Decode(&res)
		assert.True(t, res["liked"], "Post should be liked")

		// Verify like count in feed
		reqFeed := httptest.NewRequest(http.MethodGet, "/api/community/posts", nil)
		reqFeed.Header.Set("Authorization", "Bearer "+token)
		respFeed, _ := app.Test(reqFeed)
		var posts []community.PostResponse
		json.NewDecoder(respFeed.Body).Decode(&posts)
		assert.Equal(t, 1, posts[0].LikeCount)
	})

	t.Run("Add a comment to the post", func(t *testing.T) {
		body, _ := json.Marshal(community.CreateCommentRequest{
			Content: "Great post!",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/community/posts/"+postID+"/comments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var comment community.Comment
		json.NewDecoder(resp.Body).Decode(&comment)
		assert.Equal(t, "Great post!", comment.Content)
	})

	t.Run("Get post details with comments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/community/posts/"+postID, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, int(2*time.Second.Milliseconds()))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var res struct {
			Post     community.PostResponse `json:"post"`
			Comments []community.Comment    `json:"comments"`
		}
		json.NewDecoder(resp.Body).Decode(&res)

		assert.Equal(t, postID, res.Post.ID.String())
		assert.Equal(t, 1, res.Post.LikeCount)
		assert.Len(t, res.Comments, 1)
		assert.Equal(t, "Great post!", res.Comments[0].Content)
	})
}
