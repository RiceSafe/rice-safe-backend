package auth

import (
	"context"
	"errors"
	"time"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
}

type repository struct{}

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
func (r *repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, username, email, password_hash, role, avatar_url, created_at, updated_at FROM users WHERE id = $1`

	row := database.DB.QueryRow(ctx, query, id)
	return scanUser(row)
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
