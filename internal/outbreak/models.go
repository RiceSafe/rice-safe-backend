package outbreak

import (
	"time"

	"github.com/google/uuid"
)

type Outbreak struct {
	ID               uuid.UUID  `json:"id"`
	DiseaseID        uuid.UUID  `json:"disease_id"`
	DiagnosisID      *uuid.UUID `json:"diagnosis_id"`
	ReportedByUserID *uuid.UUID `json:"reported_by_user_id"`
	Latitude         float64    `json:"latitude"`
	Longitude        float64    `json:"longitude"`
	IsActive         bool       `json:"is_active"`
	IsVerified       bool       `json:"is_verified"`
	VerifiedBy       *uuid.UUID `json:"verified_by"`
	VerifiedAt       *time.Time `json:"verified_at"`
	CreatedAt        time.Time  `json:"created_at"`
	ExpiresAt        *time.Time `json:"expires_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type OutbreakResponse struct {
	ID          uuid.UUID `json:"id"`
	DiseaseID   uuid.UUID `json:"disease_id"`
	DiseaseName string    `json:"disease_name"`
	ImageURL    string    `json:"image_url"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Distance    *float64  `json:"distance"`
	IsActive    bool      `json:"is_active"`
	IsVerified  bool      `json:"is_verified"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
