package auth

import (
	"mime/multipart"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service  Service
	validate *validator.Validate
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
	}
}

func RegisterRoutes(app *fiber.App, service Service, jwtSecret string) {
	h := NewHandler(service)
	group := app.Group("/api/auth")

	// Public routes
	group.Post("/register", h.Register)
	group.Post("/login", h.Login)
	group.Post("/forgot-password", h.ForgotPassword)
	group.Post("/reset-password", h.ResetPassword)

	// Protected routes
	group.Get("/me", Protected(jwtSecret), h.GetProfile)
	group.Post("/change-password", Protected(jwtSecret), h.ChangePassword)
	group.Put("/me", Protected(jwtSecret), h.UpdateProfile)

	// Admin User Management Routes (/api/users)
	usersGroup := app.Group("/api/users", Protected(jwtSecret), RequireRole("ADMIN"))
	usersGroup.Get("/", h.ListUsers)
	usersGroup.Put("/:id/role", h.UpdateUserRole)
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Update username and/or avatar.
// @Tags         Auth
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        username formData string false "New Username"
// @Param        avatar formData file false "New Avatar Image"
// @Success      200  {object}  User
// @Failure      400  {object}  map[string]string
// @Router       /auth/me [put]
func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	id, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	username := c.FormValue("username")

	var avatar *multipart.FileHeader
	file, err := c.FormFile("avatar")
	if err == nil {
		avatar = file
	}

	user, err := h.service.UpdateProfile(c.Context(), id, username, avatar)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(user)
}

// Register godoc
// @Summary Register a new user
// @Description Register with username, email, password, and optional role
// @Tags         Auth
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

	// Validate Struct
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed", "details": formatValidationErrors(err)})
	}

	res, err := h.service.Register(c.Context(), &req)
	if err != nil {
		if err.Error() == "email already exists" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

// GetProfile returns the current user's profile
// @Summary Get user profile
// @Description Get the profile of the currently logged-in user
// @Tags         Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} User
// @Failure 401 {object} fiber.Map
// @Router /auth/me [get]
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	id, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	user, err := h.service.GetProfile(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

// ChangePassword handles password change
// @Summary Change user password
// @Description Change the password for the currently logged-in user
// @Tags         Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Change Password Payload"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} fiber.Map
// @Failure 401 {object} fiber.Map
// @Router /auth/change-password [post]
func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed", "details": formatValidationErrors(err)})
	}

	userID := c.Locals("user_id").(string)
	id, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	if err := h.service.ChangePassword(c.Context(), id, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password updated successfully"})
}

// ForgotPassword handles password reset request
// @Summary Request password reset
// @Description Request a password reset code via email
// @Tags         Auth
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Forgot Password Payload"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} fiber.Map
// @Router /auth/forgot-password [post]
func (h *Handler) ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed", "details": formatValidationErrors(err)})
	}

	if err := h.service.ForgotPassword(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process request"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "If email exists, a reset code has been sent"})
}

// ResetPassword handles password reset using token
// @Summary Reset password
// @Description Reset password using the code received via email
// @Tags         Auth
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Reset Password Payload"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} fiber.Map
// @Router /auth/reset-password [post]
func (h *Handler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed", "details": formatValidationErrors(err)})
	}

	if err := h.service.ResetPassword(c.Context(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password reset successfully"})
}

// Login godoc
// @Summary Login user
// @Description Login with email and password
// @Tags         Auth
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
	// Validate Request
	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Validation failed",
			"details": formatValidationErrors(err),
		})
	}

	res, err := h.service.Login(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

// ListUsers godoc
// @Summary      List all users
// @Description  Get a list of all registered users, optionally filtered by role (ADMIN only)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        role  query  string  false  "Filter by role (FARMER, EXPERT, ADMIN)"
// @Success      200   {array}   UserListItem
// @Failure      403   {object}  map[string]string
// @Router       /users [get]
func (h *Handler) ListUsers(c *fiber.Ctx) error {
	role := c.Query("role")
	users, err := h.service.ListUsers(c.Context(), role)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if users == nil {
		users = []*UserListItem{}
	}
	return c.JSON(users)
}

// UpdateUserRole godoc
// @Summary      Update user role
// @Description  Change a user's role (e.g. promote to EXPERT or ADMIN) (ADMIN only)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string           true  "User ID"
// @Param        body  body  UpdateRoleRequest true  "New role"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Router       /users/{id}/role [put]
func (h *Handler) UpdateUserRole(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var req UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed", "details": formatValidationErrors(err)})
	}

	if err := h.service.UpdateUserRole(c.Context(), id, req.Role); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "User role updated successfully"})
}

func formatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, err := range validationErrors {
			errors[err.Field()] = "Failed validation on tag: " + err.Tag()
		}
	} else {
		errors["internal"] = err.Error()
	}
	return errors
}
