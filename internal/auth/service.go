package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"mime/multipart"
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
	ListUsers(ctx context.Context, role string) ([]*UserListItem, error)
	UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error
}

type service struct {
	repo      Repository
	jwtSecret string
	storage   storage.Service
	email     email.Service
}

func NewService(repo Repository, jwtSecret string, storage storage.Service, emailSvc email.Service) Service {
	return &service{repo: repo, jwtSecret: jwtSecret, storage: storage, email: emailSvc}
}

func (s *service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	hashedPwd, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPwd,
		Role:         req.Role,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if err := checkPassword(req.Password, user.PasswordHash); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

// GetProfile returns the user profile
func (s *service) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert stored Avatar Path to Signed URL
	if user.AvatarURL != "" {
		signedURL, err := s.storage.GetFileUrl(user.AvatarURL)
		if err == nil {
			user.AvatarURL = signedURL
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

	if err := checkPassword(req.OldPassword, user.PasswordHash); err != nil {
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

	// Generate a cryptographically secure 6-digit OTP
	resetToken, err := generateOTP()
	if err != nil {
		return err
	}
	expiry := time.Now().Add(15 * time.Minute)

	if err := s.repo.SaveResetToken(ctx, user.Email, resetToken, expiry); err != nil {
		return err
	}

	return s.email.SendPasswordReset(ctx, user.Email, resetToken)
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
		user.AvatarURL = filename // Store key/path in DB
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	// Generate signed URL for response
	if user.AvatarURL != "" {
		signedURL, err := s.storage.GetFileUrl(user.AvatarURL)
		if err == nil {
			user.AvatarURL = signedURL
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
