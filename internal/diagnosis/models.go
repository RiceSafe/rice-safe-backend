package diagnosis

import (
	"time"

	"github.com/RiceSafe/rice-safe-backend/internal/disease"
	"github.com/google/uuid"
)

type DiagnosisRequest struct {
	Image       []byte
	Filename    string
	Description string
	Latitude    float64
	Longitude   float64
}

type DiagnosisResponse struct {
	DiagnosisID   uuid.UUID        `json:"diagnosis_id"`
	DiseaseResult *disease.Disease `json:"disease_result"`
	Prediction    string           `json:"prediction"`
	InfoMessage   string           `json:"info_message"`
	Confidence    float64          `json:"confidence"`
	ImageUrl      string           `json:"image_url"`
	CreatedAt     time.Time        `json:"created_at"`
}

type DiagnosisHistory struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	DiseaseID  *uuid.UUID `json:"disease_id"`
	Prediction string     `json:"prediction"`
	ImageURL   string     `json:"image_url"`
	Confidence float64    `json:"confidence"`
	Location   string     `json:"location"`
	Latitude   float64    `json:"latitude"`
	Longitude  float64    `json:"longitude"`
	CreatedAt  time.Time  `json:"created_at"`
}

type HistoryResponse struct {
	ID          uuid.UUID `json:"id"`
	ImageUrl    string    `json:"image_url"`
	Prediction  string    `json:"prediction"`
	DiseaseName string    `json:"disease_name"`
	Confidence  float64   `json:"confidence"`
	CreatedAt   time.Time `json:"created_at"`
}
