package disease

import (
	"context"
	"errors"
	"time"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository interface {
	GetAll(ctx context.Context) ([]Disease, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Disease, error)
	GetByAlias(ctx context.Context, alias string) (*Disease, error)
	Create(ctx context.Context, disease *Disease) error
	Update(ctx context.Context, disease *Disease) error
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) GetAll(ctx context.Context) ([]Disease, error) {
	query := `
		SELECT id, alias, name, category, image_url, description, spread_details, 
		       match_weather, symptoms, prevention, treatment, created_at, updated_at
		FROM diseases
		ORDER BY name ASC
	`
	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diseases []Disease
	for rows.Next() {
		var d Disease
		if err := scanRow(rows, &d); err != nil {
			return nil, err
		}
		diseases = append(diseases, d)
	}
	return diseases, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Disease, error) {
	query := `
		SELECT id, alias, name, category, image_url, description, spread_details, 
		       match_weather, symptoms, prevention, treatment, created_at, updated_at
		FROM diseases WHERE id = $1
	`
	row := database.DB.QueryRow(ctx, query, id)
	var d Disease
	if err := scanRow(row, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *repository) GetByAlias(ctx context.Context, alias string) (*Disease, error) {
	query := `
		SELECT id, alias, name, category, image_url, description, spread_details, 
		       match_weather, symptoms, prevention, treatment, created_at, updated_at
		FROM diseases WHERE alias = $1
	`
	row := database.DB.QueryRow(ctx, query, alias)
	var d Disease
	if err := scanRow(row, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *repository) Create(ctx context.Context, d *Disease) error {
	query := `
		INSERT INTO diseases (alias, name, category, image_url, description, spread_details, 
		                      match_weather, symptoms, prevention, treatment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`
	return database.DB.QueryRow(ctx, query,
		d.Alias, d.Name, d.Category, d.ImageURL, d.Description, d.SpreadDetails,
		d.MatchWeather, d.Symptoms, d.Prevention, d.Treatment,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *repository) Update(ctx context.Context, d *Disease) error {
	query := `
		UPDATE diseases
		SET alias = $1, name = $2, category = $3, image_url = $4, description = $5, spread_details = $6,
			match_weather = $7, symptoms = $8, prevention = $9, treatment = $10, updated_at = $11
		WHERE id = $12
	`
	_, err := database.DB.Exec(ctx, query,
		d.Alias, d.Name, d.Category, d.ImageURL, d.Description, d.SpreadDetails,
		d.MatchWeather, d.Symptoms, d.Prevention, d.Treatment, time.Now(), d.ID,
	)
	return err
}

// Scannable interface to handle both Row and Rows
type Scannable interface {
	Scan(dest ...interface{}) error
}

func scanRow(row Scannable, d *Disease) error {
	err := row.Scan(
		&d.ID, &d.Alias, &d.Name, &d.Category, &d.ImageURL, &d.Description, &d.SpreadDetails,
		&d.MatchWeather, &d.Symptoms, &d.Prevention, &d.Treatment, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("disease not found")
		}
		return err
	}
	return nil
}
