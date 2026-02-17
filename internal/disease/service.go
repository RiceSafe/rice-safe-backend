package disease

import (
	"context"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/google/uuid"
)

type Service interface {
	GetDiseases(ctx context.Context, category string) ([]Disease, error)
	GetCategories(ctx context.Context) ([]string, error)
	GetDiseaseByID(ctx context.Context, id uuid.UUID) (*Disease, error)
	GetByAlias(ctx context.Context, alias string) (*Disease, error)
	CreateDisease(ctx context.Context, disease *Disease) error
	UpdateDisease(ctx context.Context, id uuid.UUID, disease *Disease) error
}

type service struct {
	repo    Repository
	storage storage.Service
}

func NewService(repo Repository, storage storage.Service) Service {
	return &service{repo: repo, storage: storage}
}

func (s *service) GetDiseases(ctx context.Context, category string) ([]Disease, error) {
	diseases, err := s.repo.GetAll(ctx, category)
	if err != nil {
		return nil, err
	}
	// Sign URLs
	for i := range diseases {
		if diseases[i].ImageURL != "" {
			url, err := s.storage.GetFileUrl(diseases[i].ImageURL)
			if err == nil {
				diseases[i].ImageURL = url
			}
		}
	}
	return diseases, nil
}

func (s *service) GetCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetCategories(ctx)
}

func (s *service) GetByAlias(ctx context.Context, alias string) (*Disease, error) {
	d, err := s.repo.GetByAlias(ctx, alias)
	if err != nil {
		return nil, err
	}
	// Sign URL
	if d.ImageURL != "" {
		url, err := s.storage.GetFileUrl(d.ImageURL)
		if err == nil {
			d.ImageURL = url
		}
	}
	return d, nil
}

func (s *service) GetDiseaseByID(ctx context.Context, id uuid.UUID) (*Disease, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Sign URL
	if d.ImageURL != "" {
		url, err := s.storage.GetFileUrl(d.ImageURL)
		if err == nil {
			d.ImageURL = url
		}
	}
	return d, nil
}

func (s *service) CreateDisease(ctx context.Context, d *Disease) error {
	return s.repo.Create(ctx, d)
}

func (s *service) UpdateDisease(ctx context.Context, id uuid.UUID, d *Disease) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	d.ID = existing.ID
	return s.repo.Update(ctx, d)
}
