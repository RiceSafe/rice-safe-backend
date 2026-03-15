package notification

import (
	"context"
	"fmt"
	"log"

	"github.com/RiceSafe/rice-safe-backend/internal/outbreak"
	"github.com/google/uuid"
)

type Service interface {
	GetSettings(ctx context.Context, userID uuid.UUID) (*NotificationSettings, error)
	UpsertSettings(ctx context.Context, userID uuid.UUID, req *UpdateSettingsRequest) (*NotificationSettings, error)
	GetNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	NotifyNearbyFarmers(ctx context.Context, ob *outbreak.Outbreak, diseaseName string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetSettings(ctx context.Context, userID uuid.UUID) (*NotificationSettings, error) {
	return s.repo.GetSettings(ctx, userID)
}

func (s *service) UpsertSettings(ctx context.Context, userID uuid.UUID, req *UpdateSettingsRequest) (*NotificationSettings, error) {
	// Get current
	current, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.RadiusKm != nil {
		current.RadiusKm = *req.RadiusKm
	}
	if req.NotifyNearby != nil {
		current.NotifyNearby = *req.NotifyNearby
	}
	if req.Latitude != nil {
		current.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		current.Longitude = req.Longitude
	}

	if err := s.repo.UpsertSettings(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

func (s *service) GetNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error) {
	return s.repo.GetNotifications(ctx, userID, limit, offset)
}

func (s *service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.GetUnreadCount(ctx, userID)
}

func (s *service) MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error {
	return s.repo.MarkAsRead(ctx, notificationID, userID)
}

func (s *service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

func (s *service) NotifyNearbyFarmers(ctx context.Context, ob *outbreak.Outbreak, diseaseName string) error {
	// Find users nearby based on the outbreak location
	userIDs, err := s.repo.FindUsersInRadius(ctx, ob.Latitude, ob.Longitude)
	if err != nil {
		return fmt.Errorf("failed to find nearby users: %w", err)
	}

	if len(userIDs) == 0 {
		return nil // No one to notify
	}

	title := fmt.Sprintf("Disease Alert: %s", diseaseName)
	body := fmt.Sprintf("A new case of %s has been diagnosed near your location. Please check your crops.", diseaseName)

	var errs int
	for _, uid := range userIDs {
		// Do not notify the user who reported it
		if ob.ReportedByUserID != nil && uid == *ob.ReportedByUserID {
			continue
		}

		notif := &Notification{
			UserID:      uid,
			Title:       title,
			Body:        body,
			Type:        "OUTBREAK_NEARBY",
			ReferenceID: &ob.ID,
		}

		if err := s.repo.CreateNotification(ctx, notif); err != nil {
			log.Printf("Failed to create notification for user %s: %v", uid, err)
			errs++
		}
	}

	if errs > 0 {
		return fmt.Errorf("%d notifications failed to send", errs)
	}
	return nil
}
