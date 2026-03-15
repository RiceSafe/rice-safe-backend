package notification

import (
	"time"

	"github.com/google/uuid"
)

// NotificationSettings represents a user's notification preferences
type NotificationSettings struct {
	UserID             uuid.UUID `json:"user_id"`
	Enabled            bool      `json:"enabled"`
	RadiusKm           float64   `json:"radius_km"`
	NotifyNearby       bool      `json:"notify_nearby"`
	Latitude           *float64  `json:"latitude,omitempty"`
	Longitude          *float64  `json:"longitude,omitempty"`
}

// Notification represents a single alert sent to a user
type Notification struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Type        string     `json:"type"` // e.g., "OUTBREAK_NEARBY"
	ReferenceID *uuid.UUID `json:"reference_id,omitempty"`
	IsRead      bool       `json:"is_read"`
	CreatedAt   time.Time  `json:"created_at"`
}

// UpdateSettingsRequest represents the JSON payload to update settings
type UpdateSettingsRequest struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	RadiusKm           *float64 `json:"radius_km,omitempty"`
	NotifyNearby       *bool    `json:"notify_nearby,omitempty"`
	Latitude           *float64 `json:"latitude,omitempty"`
	Longitude          *float64 `json:"longitude,omitempty"`
}
