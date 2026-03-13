package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// CommentHandler handles blog post comments.
type CommentHandler struct {
	db *gorm.DB
}

// NewCommentHandler creates a new CommentHandler.
func NewCommentHandler(db *gorm.DB) *CommentHandler {
	return &CommentHandler{db: db}
}

// List returns comments for a blog post slug (public, no auth required).
// GET /api/blog/:slug/comments
func (h *CommentHandler) List(c fiber.Ctx) error {
	slug := c.Params("slug")
	var comments []models.Comment
	h.db.Where("post_slug = ?", slug).Order("created_at desc").Limit(100).Find(&comments)
	return c.JSON(comments)
}

// Create adds a comment to a blog post (auth required).
// POST /api/blog/:slug/comments
func (h *CommentHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := c.Params("slug")
	var body struct {
		Body string `json:"body"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if body.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body is required"})
	}
	if len(body.Body) > 2000 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "comment too long (max 2000 chars)"})
	}

	comment := models.Comment{
		PostSlug:  slug,
		UserID:    user.ID,
		UserName:  user.Name,
		AvatarURL: user.AvatarURL,
		Body:      body.Body,
	}
	if err := h.db.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create comment"})
	}

	return c.Status(fiber.StatusCreated).JSON(comment)
}
