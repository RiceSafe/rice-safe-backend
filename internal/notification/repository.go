package notification

import (
	"context"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/google/uuid"
)

type Repository interface {
	// Settings
	GetSettings(ctx context.Context, userID uuid.UUID) (*NotificationSettings, error)
	UpsertSettings(ctx context.Context, settings *NotificationSettings) error

	// Notifications
	GetNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	CreateNotification(ctx context.Context, notif *Notification) error

	// Discovery
	FindUsersInRadius(ctx context.Context, lat, lon float64) ([]uuid.UUID, error) // Returns users whose radius includes the lat/lon (simplified)
}

type repository struct{}

func NewRepository() Repository {
	return &repository{}
}

func (r *repository) GetSettings(ctx context.Context, userID uuid.UUID) (*NotificationSettings, error) {
	query := `
		SELECT user_id, enabled, radius_km, notify_nearby, latitude, longitude
		FROM notification_settings
		WHERE user_id = $1
	`
	var s NotificationSettings
	err := database.DB.QueryRow(ctx, query, userID).Scan(
		&s.UserID, &s.Enabled, &s.RadiusKm, &s.NotifyNearby, &s.Latitude, &s.Longitude,
	)

	if err != nil {
		// If no settings exist yet, return defaults
		if err.Error() == "no rows in result set" {
			return &NotificationSettings{
				UserID:             userID,
				Enabled:            true,
				RadiusKm:           5.0,
				NotifyNearby:       true,
			}, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *repository) UpsertSettings(ctx context.Context, settings *NotificationSettings) error {
	query := `
		INSERT INTO notification_settings (user_id, enabled, radius_km, notify_nearby, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			enabled = EXCLUDED.enabled,
			radius_km = EXCLUDED.radius_km,
			notify_nearby = EXCLUDED.notify_nearby,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude
	`
	_, err := database.DB.Exec(ctx, query,
		settings.UserID, settings.Enabled, settings.RadiusKm,
		settings.NotifyNearby,
		settings.Latitude, settings.Longitude,
	)
	return err
}

func (r *repository) GetNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error) {
	query := `
		SELECT id, user_id, title, body, type, reference_id, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := database.DB.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var n Notification
		err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.Type, &n.ReferenceID, &n.IsRead, &n.CreatedAt)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, &n)
	}
	return notifications, nil
}

func (r *repository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`
	var count int
	err := database.DB.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *repository) MarkAsRead(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2`
	_, err := database.DB.Exec(ctx, query, notificationID, userID)
	return err
}

func (r *repository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false`
	_, err := database.DB.Exec(ctx, query, userID)
	return err
}

func (r *repository) CreateNotification(ctx context.Context, notif *Notification) error {
	query := `
		INSERT INTO notifications (user_id, title, body, type, reference_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return database.DB.QueryRow(ctx, query,
		notif.UserID, notif.Title, notif.Body, notif.Type, notif.ReferenceID,
	).Scan(&notif.ID, &notif.CreatedAt)
}

// FindUsersInRadius returns user IDs who have notifications enabled and whose radius includes the given point.
// For now, this is a simplified version finding users who have recently submitted a diagnosis near the outbreak
// (since we don't store user's "home" location in the users table currently, we can infer location from their recent diagnosis history).
func (r *repository) FindUsersInRadius(ctx context.Context, lat, lon float64) ([]uuid.UUID, error) {
	// Formula: Haversine distance. Earth radius in km = 6371
	// We check against the user's notification_settings radius_km. They must have notifications enabled, and must have set their farm location.

	query := `
		SELECT user_id
		FROM notification_settings
		WHERE 
			enabled = true AND notify_nearby = true AND
			latitude IS NOT NULL AND longitude IS NOT NULL AND
			(
				6371 * acos(
					cos(radians($1)) * cos(radians(latitude)) * 
					cos(radians(longitude) - radians($2)) + 
					sin(radians($1)) * sin(radians(latitude))
				)
			) <= radius_km
	`
	rows, err := database.DB.Query(ctx, query, lat, lon)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, uid)
	}
	return userIDs, nil
}
