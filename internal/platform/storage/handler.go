package storage

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// UploadFile godoc
// @Summary      Upload a file
// @Description  Upload an image file to Google Cloud Storage.
// @Tags         platform
// @Accept       multipart/form-data
// @Produce      json
// @Param        image formData file true "Image file to upload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /upload [post]
func (h *Handler) UploadFile(c *fiber.Ctx) error {
	if h.service == nil {
		log.Println("Storage service is nil")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Storage service not initialized"})
	}

	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Image is required"})
	}

	// Upload to "dev" folder for development
	filename, err := h.service.UploadFile(file, "dev")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to upload image", "details": err.Error()})
	}

	// Generate Signed URL
	url, err := h.service.GetFileUrl(filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate signed URL", "details": err.Error()})
	}

	return c.JSON(fiber.Map{
		"url":      url,
		"filename": filename,
		"message":  "Upload successful",
	})
}
