package outbreak

import (
	"github.com/RiceSafe/rice-safe-backend/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func RegisterRoutes(app fiber.Router, service Service) {
	h := NewHandler(service)
	group := app.Group("/outbreaks")

	group.Get("/", h.GetOutbreaks)
	group.Get("/:id", h.GetOutbreakByID)
	group.Post("/:id/verify", auth.RequireRole("EXPERT", "ADMIN"), h.VerifyOutbreak)
}

// GetOutbreakByID godoc
// @Summary      Get outbreak details
// @Description  Get full details of a specific outbreak
// @Tags         Outbreaks
// @Produce      json
// @Security     BearerAuth
// @Param        id         path      string  true  "Outbreak ID"
// @Param        lat        query     number  false "User Latitude for distance"
// @Param        long       query     number  false "User Longitude for distance"
// @Success      200  {object}  OutbreakResponse
// @Failure      404  {object}  map[string]string
// @Router       /outbreaks/{id} [get]
func (h *Handler) GetOutbreakByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("long", 0)
	var userLat, userLon *float64
	if c.Query("lat") != "" && c.Query("long") != "" {
		userLat = &lat
		userLon = &lon
	}

	outbreak, err := h.service.GetOutbreakByID(c.Context(), id, userLat, userLon)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Outbreak not found"})
	}
	return c.JSON(outbreak)
}

// GetOutbreaks godoc
// @Summary      List active outbreaks
// @Description  Get a list of all active disease outbreaks for the map
// @Tags         Outbreaks
// @Produce      json
// @Security     BearerAuth
// @Param        verified   query     boolean false "Filter only verified outbreaks"
// @Param        lat        query     number  false "User Latitude for distance"
// @Param        long       query     number  false "User Longitude for distance"
// @Param        limit      query     int     false "Limit number of results"
// @Success      200  {array}   OutbreakResponse
// @Failure      500  {object}  map[string]string
// @Router       /outbreaks [get]
func (h *Handler) GetOutbreaks(c *fiber.Ctx) error {
	verified := c.QueryBool("verified", false)
	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("long", 0)

	var userLat, userLon *float64
	// Only consider location valid if both are non-zero (simple check, technically 0,0 is valid but rare for users)
	// Better check: check if param exists, but QueryFloat defaults to 0.
	// Let's assume if client sends them, they are valid.
	if c.Query("lat") != "" && c.Query("long") != "" {
		userLat = &lat
		userLon = &lon
	}

	outbreaks, err := h.service.GetActiveOutbreaks(c.Context(), verified, userLat, userLon)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch outbreaks"})
	}

	limit := c.QueryInt("limit", 0)
	if limit > 0 && limit < len(outbreaks) {
		return c.JSON(outbreaks[:limit])
	}

	return c.JSON(outbreaks)
}

// VerifyOutbreak godoc
// @Summary      Verify an outbreak
// @Description  Allows an Expert to verify a reported outbreak
// @Tags         Outbreaks
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Outbreak ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /outbreaks/{id}/verify [post]
func (h *Handler) VerifyOutbreak(c *fiber.Ctx) error {
	idStr := c.Params("id")
	outbreakID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Outbreak ID"})
	}

	userIDStr := c.Locals("user_id").(string)
	expertID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	if err := h.service.VerifyOutbreak(c.Context(), outbreakID, expertID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to verify outbreak"})
	}

	return c.JSON(fiber.Map{"message": "Outbreak successfully verified"})
}
