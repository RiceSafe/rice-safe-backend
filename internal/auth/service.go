package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/email"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error
	ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, username string, avatar *multipart.FileHeader) (*User, error)
	OAuthLogin(ctx context.Context, req *OAuthRequest) (*AuthResponse, error)
	ListUsers(ctx context.Context, role string) ([]*UserListItem, error)
	UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error
}

type service struct {
	repo            Repository
	jwtSecret       string
	storage         storage.Service
	email           email.Service
	googleClientIDs []string // iOS + Android client IDs for token verification
	lineChannelID   string
}

func NewService(repo Repository, jwtSecret string, storage storage.Service, emailSvc email.Service, googleClientIDs []string, lineChannelID string) Service {
	return &service{
		repo:            repo,
		jwtSecret:       jwtSecret,
		storage:         storage,
		email:           emailSvc,
		googleClientIDs: googleClientIDs,
		lineChannelID:   lineChannelID,
	}
}

func (s *service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	hashedPwd, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		Username:     req.Username,
		Email:        &req.Email,
		PasswordHash: &hashedPwd,
		Role:         req.Role,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: *user}, nil
}

func (s *service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	// OAuth-only accounts have no password
	if user.PasswordHash == nil {
		return nil, errors.New("invalid email or password")
	}

	if err := checkPassword(req.Password, *user.PasswordHash); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: *user}, nil
}

// GetProfile returns the user profile
func (s *service) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert stored Avatar Path to Signed URL
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		signedURL, err := s.storage.GetFileUrl(*user.AvatarURL)
		if err == nil {
			user.AvatarURL = &signedURL
		}
	}

	return user, nil
}

// ChangePassword updates the user's password
func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// OAuth-only accounts have no password to change
	if user.PasswordHash == nil {
		return errors.New("บัญชีนี้ใช้การเข้าสู่ระบบผ่าน Google หรือ LINE ไม่สามารถเปลี่ยนรหัสผ่านได้")
	}

	if err := checkPassword(req.OldPassword, *user.PasswordHash); err != nil {
		return errors.New("invalid old password")
	}

	hashedPwd, err := hashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, userID, hashedPwd)
}

// ForgotPassword handles the password reset request
func (s *service) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Return nil even if email not found to prevent enumeration
		return nil
	}

	// OAuth-only accounts have no password to reset
	if user.PasswordHash == nil {
		return errors.New("บัญชีนี้ใช้การเข้าสู่ระบบผ่าน Google หรือ LINE ไม่สามารถรีเซ็ตรหัสผ่านได้")
	}

	// Generate a cryptographically secure 6-digit OTP
	resetToken, err := generateOTP()
	if err != nil {
		return err
	}
	expiry := time.Now().Add(15 * time.Minute)

	if err := s.repo.SaveResetToken(ctx, req.Email, resetToken, expiry); err != nil {
		return err
	}

	return s.email.SendPasswordReset(ctx, req.Email, resetToken)
}

// ResetPassword resets the user's password using the token
func (s *service) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	user, err := s.repo.GetUserByResetToken(ctx, req.Token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	hashedPwd, err := hashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	if err := s.repo.UpdatePassword(ctx, user.ID, hashedPwd); err != nil {
		return err
	}

	return s.repo.ClearResetToken(ctx, user.ID)
}

// OAuthLogin handles social login via Google or LINE.
// It verifies the id_token with the provider, then finds or creates a user.
func (s *service) OAuthLogin(ctx context.Context, req *OAuthRequest) (*AuthResponse, error) {
	info, err := s.verifyOAuthToken(req.Provider, req.IDToken)
	if err != nil {
		return nil, fmt.Errorf("invalid %s token: %w", req.Provider, err)
	}

	// Try to find existing user by provider identity
	user, err := s.repo.GetUserByProviderID(ctx, req.Provider, info.ProviderUID)
	if err == nil {
		// Existing OAuth user — return JWT
		token, err := s.generateToken(user)
		if err != nil {
			return nil, err
		}
		return &AuthResponse{Token: token, User: *user}, nil
	}

	// New user — create account and identity
	newUser := &User{
		Username:  info.Name,
		Email:     nil, // We keep email NULL for OAuth users to allow separate accounts & reduce security risks
		AvatarURL: &info.Picture,
		Role:      "FARMER",
	}

	if info.Picture == "" {
		newUser.AvatarURL = nil
	}

	if err := s.repo.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	if err := s.repo.CreateUserIdentity(ctx, newUser.ID, req.Provider, info.ProviderUID); err != nil {
		return nil, err
	}

	token, err := s.generateToken(newUser)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: *newUser}, nil
}

// oauthUserInfo holds the normalized info extracted from a provider token
type oauthUserInfo struct {
	ProviderUID string
	Name        string
	Email       *string // nil if not provided
	Picture     string
}

// verifyOAuthToken verifies an id_token with the given provider and returns normalized user info
func (s *service) verifyOAuthToken(provider, idToken string) (*oauthUserInfo, error) {
	switch provider {
	case "google":
		return s.verifyGoogleToken(idToken)
	case "line":
		return s.verifyLINEToken(idToken)
	default:
		return nil, errors.New("unsupported provider")
	}
}

// verifyGoogleToken validates a Google id_token using Google's tokeninfo endpoint
func (s *service) verifyGoogleToken(idToken string) (*oauthUserInfo, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid Google token")
	}

	var payload struct {
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
		Aud     string `json:"aud"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	// Verify audience — token must be issued for one of our registered client IDs
	validAudience := false
	for _, clientID := range s.googleClientIDs {
		if payload.Aud == clientID {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return nil, errors.New("Google token audience mismatch")
	}

	info := &oauthUserInfo{
		ProviderUID: payload.Sub,
		Name:        payload.Name,
		Picture:     payload.Picture,
	}
	if payload.Email != "" {
		info.Email = &payload.Email
	}

	return info, nil
}

// verifyLINEToken validates a LINE id_token using LINE's verify endpoint
func (s *service) verifyLINEToken(idToken string) (*oauthUserInfo, error) {
	body := strings.NewReader("id_token=" + idToken + "&client_id=" + s.lineChannelID)
	resp, err := http.Post("https://api.line.me/oauth2/v2.1/verify", "application/x-www-form-urlencoded", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid LINE token")
	}

	var payload struct {
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, err
	}

	info := &oauthUserInfo{
		ProviderUID: payload.Sub,
		Name:        payload.Name,
		Picture:     payload.Picture,
	}
	if payload.Email != "" {
		info.Email = &payload.Email
	}

	return info, nil
}

// Admin Methods

var allowedRoles = map[string]bool{
	"FARMER": true,
	"EXPERT": true,
	"ADMIN":  true,
}

func (s *service) ListUsers(ctx context.Context, role string) ([]*UserListItem, error) {
	if role != "" && !allowedRoles[role] {
		return nil, errors.New("invalid role filter")
	}
	return s.repo.ListUsers(ctx, role)
}

func (s *service) UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error {
	if !allowedRoles[role] {
		return errors.New("invalid role: must be FARMER, EXPERT, or ADMIN")
	}
	return s.repo.UpdateUserRole(ctx, userID, role)
}

// Helper Functions

func (s *service) UpdateProfile(ctx context.Context, userID uuid.UUID, username string, avatar *multipart.FileHeader) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if username != "" {
		user.Username = username
	}

	// Handle Avatar Upload if provided
	if avatar != nil {
		// Upload to "avatars" folder
		filename, err := s.storage.UploadFile(avatar, "avatars")
		if err != nil {
			return nil, err
		}
		user.AvatarURL = &filename
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	// Generate signed URL for response
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		signedURL, err := s.storage.GetFileUrl(*user.AvatarURL)
		if err == nil {
			user.AvatarURL = &signedURL
		}
	}

	return user, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func checkPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *service) generateToken(user *User) (string, error) {
	if s.jwtSecret == "" {
		return "", errors.New("JWT_SECRET is not set")
	}

	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 72).Unix(), // 3 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// generateOTP returns a cryptographically secure random 6-digit code (e.g. "047291").
func generateOTP() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Convert to a number 0-999999 and zero-pad to 6 digits
	n := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}
