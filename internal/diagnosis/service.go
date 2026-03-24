package diagnosis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/RiceSafe/rice-safe-backend/internal/disease"
	"github.com/RiceSafe/rice-safe-backend/internal/notification"
	"github.com/RiceSafe/rice-safe-backend/internal/outbreak"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/ai_client"
	"github.com/RiceSafe/rice-safe-backend/internal/platform/storage"
	"github.com/google/uuid"
)

type Service interface {
	Diagnose(ctx context.Context, userID uuid.UUID, req *DiagnosisRequest) (*DiagnosisResponse, error)
	GetHistory(ctx context.Context, userID uuid.UUID) ([]*HistoryResponse, error)
}

type service struct {
	repo         Repository
	diseaseRepo  disease.Repository
	outbreakRepo outbreak.Repository
	storage      storage.Service
	aiClient     ai_client.Client
	notifService notification.Service
}

func NewService(repo Repository, diseaseRepo disease.Repository, outbreakRepo outbreak.Repository, storage storage.Service, aiClient ai_client.Client, notifService notification.Service) Service {
	return &service{
		repo:         repo,
		diseaseRepo:  diseaseRepo,
		outbreakRepo: outbreakRepo,
		storage:      storage,
		aiClient:     aiClient,
		notifService: notifService,
	}
}

func (s *service) Diagnose(ctx context.Context, userID uuid.UUID, req *DiagnosisRequest) (*DiagnosisResponse, error) {
	// Upload Image to Storage (GCS)
	imageURL, err := s.storage.UploadBytes(req.Image, req.Filename, "diagnosis")
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	// Call AI Service
	prediction, err := s.aiClient.Predict(req.Image, req.Filename, req.Description)
	if err != nil {
		return nil, fmt.Errorf("ai prediction failed: %w", err)
	}

	// Parse Confidence
	confStr := strings.TrimRight(prediction.Confidence, "%")
	confidence, _ := strconv.ParseFloat(confStr, 64)

	var diseaseResult *disease.Disease
	var diseaseID *uuid.UUID

	// Handle Logic based on Prediction Result
	var infoMessage string

	switch prediction.Prediction {
	case "not_rice":
		infoMessage = "ไม่พบใบข้าวในรูปภาพ กรุณาถ่ายรูปใหม่อีกครั้ง"
	case "not_clear":
		infoMessage = "รูปภาพไม่ชัดเจน กรุณาถ่ายรูปใหม่อีกครั้ง"
	case "other_diseases":
		infoMessage = "ปกติ/โรคอื่นๆ"
	default:
		d, err := s.diseaseRepo.GetByAlias(ctx, prediction.Prediction)
		if err == nil {
			diseaseResult = d
			diseaseID = &d.ID
			infoMessage = "ตรวจพบโรค: " + d.Name
		} else {
			fmt.Printf("Disease alias not found: %s\n", prediction.Prediction)
			infoMessage = "ตรวจพบโรคที่ไม่รู้จัก"
		}
	}

	// Sign Image URL if disease found
	if diseaseResult != nil && diseaseResult.ImageURL != nil && *diseaseResult.ImageURL != "" {
		signedURL, _ := s.storage.GetFileUrl(*diseaseResult.ImageURL)
		if signedURL != "" {
			diseaseResult.ImageURL = &signedURL
		}
	}

	// Save History
	history := &DiagnosisHistory{
		UserID:     userID,
		DiseaseID:  diseaseID,
		Prediction: prediction.Prediction,
		ImageURL:   imageURL,
		Confidence: confidence,
		Location:   "Unknown",
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
	}
	if err := s.repo.Create(ctx, history); err != nil {
		return nil, fmt.Errorf("failed to save history: %w", err)
	}

	// Generate Signed URL for User's Uploaded Image
	signedUserImage, _ := s.storage.GetFileUrl(imageURL)

	// Auto-Outbreak Logic
	isDisease := prediction.Prediction != "not_rice" && prediction.Prediction != "not_clear" && prediction.Prediction != "other_diseases"

	if isDisease && diseaseID != nil && req.Latitude != nil && req.Longitude != nil {
		ob := &outbreak.Outbreak{
			DiseaseID:        *diseaseID,
			DiagnosisID:      &history.ID,
			ReportedByUserID: &userID,
			Latitude:         *req.Latitude,
			Longitude:        *req.Longitude,
		}
		if err := s.outbreakRepo.Create(ctx, ob); err != nil {
			fmt.Printf("Failed to auto-create outbreak: %v\n", err)
		} else {
			fmt.Println("Auto-created outbreak successfully")
			// Trigger notification to nearby users
			if err := s.notifService.NotifyNearbyFarmers(ctx, ob, diseaseResult.Name); err != nil {
				fmt.Printf("Failed to notify nearby farmers: %v\n", err)
			}
		}
	}

	return &DiagnosisResponse{
		DiagnosisID:   history.ID,
		DiseaseResult: diseaseResult,
		Prediction:    prediction.Prediction,
		InfoMessage:   infoMessage,
		Confidence:    confidence,
		ImageUrl:      signedUserImage,
		CreatedAt:     history.CreatedAt,
	}, nil
}

func (s *service) GetHistory(ctx context.Context, userID uuid.UUID) ([]*HistoryResponse, error) {
	history, err := s.repo.GetHistory(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Sign Image URLs
	for _, h := range history {
		if h.ImageUrl != "" {
			signedURL, _ := s.storage.GetFileUrl(h.ImageUrl)
			if signedURL != "" {
				h.ImageUrl = signedURL
			}
		}
	}

	return history, nil
}
