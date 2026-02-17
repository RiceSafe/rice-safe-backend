package storage

import "mime/multipart"

// Service defines the interface for file storage operations
type Service interface {
	// UploadFile uploads a file and returns the stored filename (object key)
	UploadFile(file *multipart.FileHeader, folder string) (string, error)
	// GetFileUrl returns a signed URL for the given filename
	GetFileUrl(filename string) (string, error)
}
