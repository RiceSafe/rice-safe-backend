package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type GCSService struct {
	client          *storage.Client
	bucketName      string
	credentialsFile string
}

func NewGCSService(bucketName, credentialsFile string) (*GCSService, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %v", err)
	}

	return &GCSService{
		client:          client,
		bucketName:      bucketName,
		credentialsFile: credentialsFile,
	}, nil
}

func (s *GCSService) UploadFile(file *multipart.FileHeader, folder string) (string, error) {
	ctx := context.Background()

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Generate unique filename: folder/timestamp-filename
	filename := fmt.Sprintf("%s/%d-%s", folder, time.Now().Unix(), file.Filename)

	wc := s.client.Bucket(s.bucketName).Object(filename).NewWriter(ctx)
	if _, err = io.Copy(wc, src); err != nil {
		return "", fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %v", err)
	}

	return filename, nil
}

func (s *GCSService) GetFileUrl(filename string) (string, error) {
	if filename == "" {
		return "", nil
	}
	// Generate Signed URL (valid for 7 days)
	url, err := s.client.Bucket(s.bucketName).SignedURL(filename, &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(7 * 24 * time.Hour), // 7 days
	})
	if err != nil {
		return "", fmt.Errorf("SignedURL: %v", err)
	}

	return url, nil
}
