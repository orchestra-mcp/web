package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// RequestHandler handles request queue endpoints.
type RequestHandler struct {
	db *gorm.DB
}

// NewRequestHandler creates a new RequestHandler.
func NewRequestHandler(db *gorm.DB) *RequestHandler {
	return &RequestHandler{db: db}
}

// List handles GET /api/projects/:slug/requests
func (h *RequestHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := resolveProjectSlug(h.db, c.Params("slug"))

	query := teamScopedRequests(h.db, user.ID).
		Where("project_slug = ?", slug).
		Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if kind := c.Query("kind"); kind != "" {
		query = query.Where("kind = ?", kind)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}

	var requests []models.Request
	if err := query.Find(&requests).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list requests"})
	}

	return c.JSON(fiber.Map{
		"items": requests,
		"meta": fiber.Map{
			"total":  len(requests),
			"limit":  100,
			"offset": 0,
		},
	})
}

// Show handles GET /api/requests/:id
func (h *RequestHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var request models.Request
	if err := teamScopedRequests(h.db, user.ID).
		Where("request_id = ?", id).
		First(&request).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "request not found"})
	}

	return c.JSON(fiber.Map{"request": request})
}
