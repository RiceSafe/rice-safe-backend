package diagnosis

import (
	"context"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, d *DiagnosisHistory) error
	GetHistory(ctx context.Context, userID uuid.UUID) ([]*HistoryResponse, error)
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) Create(ctx context.Context, d *DiagnosisHistory) error {
	query := `
		INSERT INTO diagnosis_history (
			user_id, disease_id, prediction, image_url, confidence, latitude, longitude
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	return database.DB.QueryRow(ctx, query,
		d.UserID, d.DiseaseID, d.Prediction, d.ImageURL, d.Confidence, d.Latitude, d.Longitude,
	).Scan(&d.ID, &d.CreatedAt)
}

func (r *repository) GetHistory(ctx context.Context, userID uuid.UUID) ([]*HistoryResponse, error) {
	query := `
		SELECT 
			dh.id, dh.image_url, dh.confidence, dh.created_at,
			COALESCE(dh.prediction, d.alias, 'other_diseases') as prediction,
			COALESCE(d.name, 'ปกติ/โรคอื่นๆ') as disease_name
		FROM diagnosis_history dh
		LEFT JOIN diseases d ON dh.disease_id = d.id
		WHERE dh.user_id = $1
		ORDER BY dh.created_at DESC
	`
	rows, err := database.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*HistoryResponse
	for rows.Next() {
		var h HistoryResponse
		if err := rows.Scan(
			&h.ID, &h.ImageUrl, &h.Confidence, &h.CreatedAt,
			&h.Prediction, &h.DiseaseName,
		); err != nil {
			return nil, err
		}
		history = append(history, &h)
	}
	return history, nil
}
