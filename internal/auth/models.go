package auth

import (
	"time"

	"github.com/google/uuid"
)

// User represents the user entity in the database
type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never send password hash in JSON
	Role         string    `json:"role"`
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RegisterRequest is the payload for registration
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"oneof=FARMER EXPERT"`
}

// LoginRequest is the payload for login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// ChangePasswordRequest is the payload for changing password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// ForgotPasswordRequest is the payload for requesting a reset
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest is the payload for reseting password
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// AuthResponse is returned after successful auth
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// UserListItem is returned by the list-users endpoint (Admin view)
type UserListItem struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateRoleRequest is the payload for changing a user's role
type UpdateRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=FARMER EXPERT ADMIN"`
}
