package auth

import (
	"context"
	"errors"
	"log"
	"mime/multipart"
	"time"

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
}

type service struct {
	repo      Repository
	jwtSecret string
	storage   storage.Service
}

func NewService(repo Repository, jwtSecret string, storage storage.Service) Service {
	return &service{repo: repo, jwtSecret: jwtSecret, storage: storage}
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

	// Generate a 6-digit code (Mock)
	// In production, use crypto/rand
	resetToken := "123456"
	expiry := time.Now().Add(15 * time.Minute)

	if err := s.repo.SaveResetToken(ctx, user.Email, resetToken, expiry); err != nil {
		return err
	}

	// Mock Email Sending
	log.Printf("EMAIL SENT: To %s, Code: %s", user.Email, resetToken)
	return nil
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
