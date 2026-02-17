package disease

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func RegisterRoutes(app *fiber.App, service Service) {
	h := NewHandler(service)
	group := app.Group("/api/diseases")

	group.Get("/", h.GetDiseases)
	group.Get("/:id", h.GetDiseaseByID)
	group.Post("/", h.CreateDisease)
	group.Put("/:id", h.UpdateDisease)
}

// GetDiseases godoc
// @Summary      List all diseases
// @Description  Get a list of all diseases in the library
// @Tags         diseases
// @Produce      json
// @Success      200  {array}   Disease
// @Failure      500  {object}  map[string]string
// @Router       /diseases [get]
func (h *Handler) GetDiseases(c *fiber.Ctx) error {
	diseases, err := h.service.GetDiseases(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(diseases)
}

// GetDiseaseByID godoc
// @Summary      Get disease details
// @Description  Get full details of a specific disease
// @Tags         diseases
// @Produce      json
// @Param        id   path      string  true  "Disease ID"
// @Success      200  {object}  Disease
// @Failure      404  {object}  map[string]string
// @Router       /diseases/{id} [get]
func (h *Handler) GetDiseaseByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	disease, err := h.service.GetDiseaseByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Disease not found"})
	}
	return c.JSON(disease)
}

// CreateDisease godoc
// @Summary      Create a new disease
// @Description  Create a new disease entry
// @Tags         diseases
// @Accept       json
// @Produce      json
// @Param        disease body Disease true "Disease Data"
// @Success      201  {object}  Disease
// @Failure      400  {object}  map[string]string
// @Router       /diseases [post]
func (h *Handler) CreateDisease(c *fiber.Ctx) error {
	var disease Disease
	if err := c.BodyParser(&disease); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.service.CreateDisease(c.Context(), &disease); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(disease)
}

// UpdateDisease godoc
// @Summary      Update disease details
// @Description  Update details of an existing disease
// @Tags         diseases
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Disease ID"
// @Param        disease body Disease true "Updated Data"
// @Success      200  {object}  Disease
// @Failure      400  {object}  map[string]string
// @Router       /diseases/{id} [put]
func (h *Handler) UpdateDisease(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var disease Disease
	if err := c.BodyParser(&disease); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.service.UpdateDisease(c.Context(), id, &disease); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(disease)
}
