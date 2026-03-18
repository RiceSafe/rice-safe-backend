package auth

import (
	"context"
	"errors"
	"time"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateUser(ctx context.Context, user *User) error

	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	SaveResetToken(ctx context.Context, email string, token string, expiry time.Time) error
	GetUserByResetToken(ctx context.Context, token string) (*User, error)
	ClearResetToken(ctx context.Context, id uuid.UUID) error

	// Admin Actions
	ListUsers(ctx context.Context, role string) ([]*UserListItem, error)
	UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error
}

type repository struct{}

// NewRepository creates a new auth repository
func NewRepository() Repository {
	return &repository{}
}

// CreateUser inserts a new user into the database
func (r *repository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	if user.Role == "" {
		user.Role = "FARMER"
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	err := database.DB.QueryRow(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.AvatarURL,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			return errors.New("email already exists")
		}
		return err
	}

	return nil
}

// GetUserByEmail finds a user by email
func (r *repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, username, email, password_hash, role, avatar_url, created_at, updated_at FROM users WHERE email = $1`

	row := database.DB.QueryRow(ctx, query, email)
	return scanUser(row)
}

// GetUserByID finds a user by ID
func (r *repository) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `SELECT id, username, email, password_hash, role, avatar_url, created_at, updated_at FROM users WHERE id = $1`

	row := database.DB.QueryRow(ctx, query, id)
	return scanUser(row)
}

// UpdateUser updates user profile
func (r *repository) UpdateUser(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET username = $1, email = $2, password_hash = $3, avatar_url = $4, updated_at = $5
		WHERE id = $6
	`
	_, err := database.DB.Exec(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.AvatarURL,
		time.Now(),
		user.ID,
	)
	return err
}

// UpdatePassword updates the user's password
func (r *repository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`

	_, err := database.DB.Exec(ctx, query, passwordHash, time.Now(), id)
	return err
}

// SaveResetToken saves the reset token and expiry
func (r *repository) SaveResetToken(ctx context.Context, email string, token string, expiry time.Time) error {
	query := `UPDATE users SET reset_token = $1, reset_token_expires_at = $2 WHERE email = $3`
	_, err := database.DB.Exec(ctx, query, token, expiry, email)
	return err
}

// GetUserByResetToken finds a user by reset token if not expired
func (r *repository) GetUserByResetToken(ctx context.Context, token string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, avatar_url, created_at, updated_at 
		FROM users 
		WHERE reset_token = $1 AND reset_token_expires_at > NOW()
	`
	row := database.DB.QueryRow(ctx, query, token)
	return scanUser(row)
}

// ClearResetToken clears the reset token
func (r *repository) ClearResetToken(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET reset_token = NULL, reset_token_expires_at = NULL WHERE id = $1`
	_, err := database.DB.Exec(ctx, query, id)
	return err
}

// ListUsers returns all users optionally filtered by role
func (r *repository) ListUsers(ctx context.Context, role string) ([]*UserListItem, error) {
	query := `SELECT id, username, email, role, avatar_url, created_at FROM users`
	args := []any{}

	if role != "" {
		query += ` WHERE role = $1`
		args = append(args, role)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*UserListItem
	for rows.Next() {
		u := &UserListItem{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.AvatarURL, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// UpdateUserRole changes a user's role
func (r *repository) UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error {
	query := `UPDATE users SET role = $1 WHERE id = $2`
	tag, err := database.DB.Exec(ctx, query, role, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

func scanUser(row pgx.Row) (*User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}
