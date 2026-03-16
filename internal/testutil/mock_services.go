package testutil

import (
	"fmt"
	"io"
	"mime/multipart"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/ai_client"
)

// MockAIService implements ai_client.Client for testing
type MockAIService struct {
	// PredictFunc allows overriding the behavior per test
	PredictFunc func(image []byte, filename, description string) (*ai_client.PredictionResponse, error)
}

func (m *MockAIService) Predict(image []byte, filename, description string) (*ai_client.PredictionResponse, error) {
	if m.PredictFunc != nil {
		return m.PredictFunc(image, filename, description)
	}
	// Default behavior if not overridden: return disease
	return &ai_client.PredictionResponse{
		Prediction:  "rice_blast",
		Confidence:  "95.00%",
	}, nil
}

// MockStorageService implements storage.Service for testing
type MockStorageService struct {
	UploadBytesFunc func(data []byte, filename, folder string) (string, error)
	UploadFunc      func(file io.Reader, filename, folder string) (string, error)
	GetFileUrlFunc  func(objectName string) (string, error)
	UploadFileFunc  func(file *multipart.FileHeader, folder string) (string, error)
}

func (m *MockStorageService) UploadBytes(data []byte, filename, folder string) (string, error) {
	if m.UploadBytesFunc != nil {
		return m.UploadBytesFunc(data, filename, folder)
	}
	// Default: pretend we uploaded the file just fine
	return fmt.Sprintf("mock-storage/%s/%s", folder, filename), nil
}

func (m *MockStorageService) Upload(file io.Reader, filename, folder string) (string, error) {
	if m.UploadFunc != nil {
		return m.UploadFunc(file, filename, folder)
	}
	return fmt.Sprintf("mock-storage/%s/%s", folder, filename), nil
}

func (m *MockStorageService) GetFileUrl(objectName string) (string, error) {
	if m.GetFileUrlFunc != nil {
		return m.GetFileUrlFunc(objectName)
	}
	return "https://mock-signed-url.com/" + objectName, nil
}

func (m *MockStorageService) UploadFile(file *multipart.FileHeader, folder string) (string, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(file, folder)
	}
	return fmt.Sprintf("mock-storage/%s/file", folder), nil
}
