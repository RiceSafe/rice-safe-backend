package outbreak

import (
	"context"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, o *Outbreak) error
	GetActiveOutbreaks(ctx context.Context, verifiedOnly bool) ([]*OutbreakResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*OutbreakResponse, error)
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, o *Outbreak) error {
	query := `
		INSERT INTO outbreaks (
			disease_id, diagnosis_id, reported_by_user_id, latitude, longitude, is_active
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	// Active=true
	o.IsActive = true

	return database.DB.QueryRow(ctx, query,
		o.DiseaseID, o.DiagnosisID, o.ReportedByUserID, o.Latitude, o.Longitude, o.IsActive,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

func (r *repository) GetActiveOutbreaks(ctx context.Context, verifiedOnly bool) ([]*OutbreakResponse, error) {
	query := `
		SELECT 
			o.id, o.disease_id, d.name, 
			COALESCE(dh.image_url, d.image_url) as image_url,
			o.latitude, o.longitude, o.is_active, o.is_verified, 
			o.created_at, o.updated_at
		FROM outbreaks o
		JOIN diseases d ON o.disease_id = d.id
		LEFT JOIN diagnosis_history dh ON o.diagnosis_id = dh.id
		WHERE o.is_active = TRUE
	`
	if verifiedOnly {
		query += " AND o.is_verified = TRUE"
	}
	query += " ORDER BY o.created_at DESC"

	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outbreaks []*OutbreakResponse
	for rows.Next() {
		var o OutbreakResponse
		if err := rows.Scan(
			&o.ID, &o.DiseaseID, &o.DiseaseName, &o.ImageURL,
			&o.Latitude, &o.Longitude, &o.IsActive, &o.IsVerified,
			&o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		outbreaks = append(outbreaks, &o)
	}
	return outbreaks, nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*OutbreakResponse, error) {
	query := `
		SELECT 
			o.id, o.disease_id, d.name, 
			COALESCE(dh.image_url, d.image_url) as image_url,
			o.latitude, o.longitude, o.is_active, o.is_verified, 
			o.created_at, o.updated_at
		FROM outbreaks o
		JOIN diseases d ON o.disease_id = d.id
		LEFT JOIN diagnosis_history dh ON o.diagnosis_id = dh.id
		WHERE o.id = $1
	`
	row := database.DB.QueryRow(ctx, query, id)

	var o OutbreakResponse
	if err := row.Scan(
		&o.ID, &o.DiseaseID, &o.DiseaseName, &o.ImageURL,
		&o.Latitude, &o.Longitude, &o.IsActive, &o.IsVerified,
		&o.CreatedAt, &o.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &o, nil
}
