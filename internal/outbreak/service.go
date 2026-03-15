package outbreak

import (
	"context"
	"math"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/google/uuid"
)

type Service interface {
	GetActiveOutbreaks(ctx context.Context, verifiedOnly bool, userLat, userLon *float64) ([]*OutbreakResponse, error)
	GetOutbreakByID(ctx context.Context, id uuid.UUID, userLat, userLon *float64) (*OutbreakResponse, error)
	VerifyOutbreak(ctx context.Context, outbreakID uuid.UUID, expertID uuid.UUID) error
}

type service struct {
	repo    Repository
	storage storage.Service
}

func NewService(repo Repository, storage storage.Service) Service {
	return &service{
		repo:    repo,
		storage: storage,
	}
}

func (s *service) GetActiveOutbreaks(ctx context.Context, verifiedOnly bool, userLat, userLon *float64) ([]*OutbreakResponse, error) {
	outbreaks, err := s.repo.GetActiveOutbreaks(ctx, verifiedOnly)
	if err != nil {
		return nil, err
	}

	for _, o := range outbreaks {
		// Sign Image URLs
		if o.ImageURL != "" {
			signedURL, _ := s.storage.GetFileUrl(o.ImageURL)
			if signedURL != "" {
				o.ImageURL = signedURL
			}
		}

		// Calculate Distance if user location provided
		if userLat != nil && userLon != nil {
			distKm := haversine(*userLat, *userLon, o.Latitude, o.Longitude)
			o.Distance = &distKm
		}
	}

	return outbreaks, nil
}

func (s *service) GetOutbreakByID(ctx context.Context, id uuid.UUID, userLat, userLon *float64) (*OutbreakResponse, error) {
	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Sign Image URL
	if o.ImageURL != "" {
		signedURL, _ := s.storage.GetFileUrl(o.ImageURL)
		if signedURL != "" {
			o.ImageURL = signedURL
		}
	}

	// Calculate Distance if user location provided
	if userLat != nil && userLon != nil {
		distKm := haversine(*userLat, *userLon, o.Latitude, o.Longitude)
		o.Distance = &distKm
	}

	return o, nil
}

// haversine calculates distance in kilometers between two coordinates
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in kilometers
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func (s *service) VerifyOutbreak(ctx context.Context, outbreakID uuid.UUID, expertID uuid.UUID) error {
	return s.repo.VerifyOutbreak(ctx, outbreakID, expertID)
}
