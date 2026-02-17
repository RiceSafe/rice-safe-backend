package diagnosis

import (
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func RegisterRoutes(router fiber.Router, service Service) {
	h := NewHandler(service)
	group := router.Group("/diagnosis")

	group.Post("/", h.Diagnose)
	group.Get("/history", h.GetHistory)
}

// Diagnose godoc
// @Summary      Diagnose Disease from Image
// @Description  Upload an image to get a disease prediction and details.
// @Tags         diagnosis
// @Accept       multipart/form-data
// @Produce      json
// @Param        image formData file true "Rice Leaf Image"
// @Param        description formData string false "Symptoms Description"
// @Success      200  {object}  DiagnosisResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     BearerAuth
// @Router       /diagnosis [post]
func (h *Handler) Diagnose(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	// Parse Description
	description := c.FormValue("description")

	// Parse Image File
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Image file is required"})
	}

	// Read File Bytes
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open image"})
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read image"})
	}

	// Call Service
	// Get Latitude/Longitude from request headers or body if sent by mobile
	// For now, defaulting to Bangkok
	req := &DiagnosisRequest{
		Image:       imageBytes,
		Filename:    fileHeader.Filename,
		Description: description,
		Latitude:    13.7563,  // Default Bangkok Lat (Mock)
		Longitude:   100.5018, // Default Bangkok Long (Mock)
	}
	// Try to parse lat/long if provided
	if lat := c.FormValue("latitude"); lat != "" {
		if val, err := fmt.Sscanf(lat, "%f", &req.Latitude); err != nil || val == 0 {
			// Ignore error, keep default
		}
	}
	if long := c.FormValue("longitude"); long != "" {
		if val, err := fmt.Sscanf(long, "%f", &req.Longitude); err != nil || val == 0 {
			// Ignore error, keep default
		}
	}

	resp, err := h.service.Diagnose(c.Context(), userID, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(resp)
}

// GetHistory godoc
// @Summary      Get Diagnosis History
// @Description  Get a list of past diagnoses for the current user
// @Tags         diagnosis
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   HistoryResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /diagnosis/history [get]
func (h *Handler) GetHistory(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	history, err := h.service.GetHistory(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch history"})
	}

	return c.JSON(history)
}
