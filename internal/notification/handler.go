package notification

import (
	"strconv"

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

func (h *Handler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/notifications")
	group.Use(auth.Protected())

	group.Get("/", h.GetNotifications)
	group.Get("/unread-count", h.GetUnreadCount)
	group.Put("/:id/read", h.MarkAsRead)
	group.Put("/read-all", h.MarkAllAsRead)

	settings := router.Group("/settings/notifications")
	settings.Use(auth.Protected())
	settings.Get("/", h.GetSettings)
	settings.Put("/", h.UpdateSettings)
}

// GetNotifications godoc
// @Summary      Get user notifications
// @Description  Retrieves a paginated list of notifications for the authenticated user.
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Param        limit    query     int  false  "Limit"  default(20)
// @Param        offset   query     int  false  "Offset" default(0)
// @Success      200      {array}   Notification
// @Router       /notifications [get]
// @Security     BearerAuth
func (h *Handler) GetNotifications(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	notifications, err := h.service.GetNotifications(c.Context(), userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if notifications == nil {
		notifications = []*Notification{}
	}

	return c.JSON(notifications)
}

// GetUnreadCount godoc
// @Summary      Get unread notification count
// @Description  Retrieves the total number of unread notifications.
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Success      200      {object}  map[string]int
// @Router       /notifications/unread-count [get]
// @Security     BearerAuth
func (h *Handler) GetUnreadCount(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	count, err := h.service.GetUnreadCount(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"unread_count": count})
}

// MarkAsRead godoc
// @Summary      Mark notification as read
// @Description  Marks a specific notification as read.
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "Notification ID"
// @Success      200      {object}  map[string]string
// @Router       /notifications/{id}/read [put]
// @Security     BearerAuth
func (h *Handler) MarkAsRead(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	notifID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid notification ID"})
	}

	if err := h.service.MarkAsRead(c.Context(), notifID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Notification marked as read"})
}

// MarkAllAsRead godoc
// @Summary      Mark all notifications as read
// @Description  Marks all unread notifications for the user as read.
// @Tags         Notifications
// @Accept       json
// @Produce      json
// @Success      200      {object}  map[string]string
// @Router       /notifications/read-all [put]
// @Security     BearerAuth
func (h *Handler) MarkAllAsRead(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	if err := h.service.MarkAllAsRead(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "All notifications marked as read"})
}

// GetSettings godoc
// @Summary      Get notification settings
// @Description  Retrieves the user's notification preferences.
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Success      200      {object}  NotificationSettings
// @Router       /settings/notifications [get]
// @Security     BearerAuth
func (h *Handler) GetSettings(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	settings, err := h.service.GetSettings(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(settings)
}

// UpdateSettings godoc
// @Summary      Update notification settings
// @Description  Updates the user's notification preferences.
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        request  body      UpdateSettingsRequest  true  "Settings Data"
// @Success      200      {object}  NotificationSettings
// @Router       /settings/notifications [put]
// @Security     BearerAuth
func (h *Handler) UpdateSettings(c *fiber.Ctx) error {
	userIDStr := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	var req UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
	}

	settings, err := h.service.UpsertSettings(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(settings)
}
