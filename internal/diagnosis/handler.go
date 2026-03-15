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
// @Tags         Diagnosis
// @Accept       multipart/form-data
// @Produce      json
// @Param        image formData file true "Rice Leaf Image"
// @Param        description formData string false "Symptoms Description"
// @Param        latitude formData number false "Latitude (Optional, prevents outbreak if missing)"
// @Param        longitude formData number false "Longitude (Optional, prevents outbreak if missing)"
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
	latStr := c.FormValue("latitude")
	lonStr := c.FormValue("longitude")

	var parsedLat, parsedLon *float64

	if latStr != "" && lonStr != "" {
		var l, lo float64
		if val, err := fmt.Sscanf(latStr, "%f", &l); err == nil && val > 0 {
			parsedLat = &l
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid latitude format"})
		}
		
		if val, err := fmt.Sscanf(lonStr, "%f", &lo); err == nil && val > 0 {
			parsedLon = &lo
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid longitude format"})
		}
	}

	req := &DiagnosisRequest{
		Image:       imageBytes,
		Filename:    fileHeader.Filename,
		Description: description,
		Latitude:    parsedLat,
		Longitude:   parsedLon,
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
// @Tags         Diagnosis
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
