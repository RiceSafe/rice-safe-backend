package community

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
	group := app.Group("/community")

	group.Post("/posts", h.CreatePost)
	group.Get("/posts", h.GetPosts)
	group.Get("/posts/:id", h.GetPostByID)
	group.Post("/posts/:id/comments", h.CreateComment)
	group.Post("/posts/:id/like", h.ToggleLike)

	// Admin operations
	group.Delete("/posts/:id", auth.RequireRole("ADMIN"), h.DeletePost)
}

// CreatePost godoc
// @Summary      Create a new post
// @Description  Create a community post with optional image
// @Tags         Community
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        content formData string true "Post Content"
// @Param        image formData file false "Post Image"
// @Success      201  {object}  Post
// @Failure      400  {object}  map[string]string
// @Router       /community/posts [post]
// CreatePost godoc
func (h *Handler) CreatePost(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	content := c.FormValue("content")
	if content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Content is required"})
	}

	file, _ := c.FormFile("image") // Optional

	post, err := h.service.CreatePost(c.Context(), userID, content, file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(post)
}

// GetPosts godoc
// @Summary      Get community feed
// @Description  Get list of posts with pagination
// @Tags         Community
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Limit"
// @Param        offset query int false "Offset"
// @Success      200  {array}   PostResponse
// @Router       /community/posts [get]
func (h *Handler) GetPosts(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	posts, err := h.service.GetPosts(c.Context(), userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(posts)
}

// GetPostByID godoc
// @Summary      Get post details
// @Description  Get a single post details
// @Tags         Community
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Success      200  {object}  PostResponse
// @Router       /community/posts/{id} [get]
func (h *Handler) GetPostByID(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	idStr := c.Params("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID"})
	}

	post, err := h.service.GetPostByID(c.Context(), postID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Post not found"})
	}

	comments, err := h.service.GetComments(c.Context(), postID)
	if err == nil {
		return c.JSON(fiber.Map{
			"post":     post,
			"comments": comments,
		})
	}

	return c.JSON(fiber.Map{"post": post, "comments": []interface{}{}})
}

// CreateComment godoc
// @Summary      Add a comment
// @Description  Add a comment to a post
// @Tags         Community
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Param        request body CreateCommentRequest true "Content"
// @Success      201  {object}  Comment
// @Router       /community/posts/{id}/comments [post]
func (h *Handler) CreateComment(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	idStr := c.Params("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Post ID"})
	}

	var req CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
	}
	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Content is required"})
	}

	comment, err := h.service.CreateComment(c.Context(), userID, postID, req.Content)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(comment)
}

// ToggleLike godoc
// @Summary      Like/Unlike a post
// @Description  Toggle like status for a post
// @Tags         Community
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Post ID"
// @Success      200  {object}  map[string]bool
// @Router       /community/posts/{id}/like [post]
func (h *Handler) ToggleLike(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	idStr := c.Params("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Post ID"})
	}

	liked, err := h.service.ToggleLike(c.Context(), userID, postID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"liked": liked})
}

// DeletePost godoc
// @Summary      Delete a community post
// @Description  Remove a community post for moderation purposes (ADMIN only)
// @Tags         Community
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Post ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Router       /community/posts/{id} [delete]
func (h *Handler) DeletePost(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid post ID"})
	}

	if err := h.service.DeletePost(c.Context(), id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Post deleted successfully"})
}
