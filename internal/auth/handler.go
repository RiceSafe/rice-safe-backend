package auth

import (
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func RegisterRoutes(app *fiber.App, service Service) {
	h := NewHandler(service)
	group := app.Group("/api/auth")

	group.Post("/register", h.Register)
	group.Post("/login", h.Login)
}

// Register godoc
// @Summary Register a new user
// @Description Register with username, email, password, and optional role
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Register Payload"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} fiber.Map
// @Failure 500 {object} fiber.Map
// @Router /auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Basic Validation (Simple check, can be enhanced with validator lib later)
	if req.Email == "" || req.Password == "" || req.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing required fields"})
	}

	res, err := h.service.Register(c.Context(), req)
	if err != nil {
		if err.Error() == "email already exists" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

// Login godoc
// @Summary Login user
// @Description Login with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login Payload"
// @Success 200 {object} AuthResponse
// @Failure 401 {object} fiber.Map
// @Failure 500 {object} fiber.Map
// @Router /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing email or password"})
	}

	res, err := h.service.Login(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	return c.Status(fiber.StatusOK).JSON(res)
}
