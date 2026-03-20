package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// AdminBadgeHandler handles badge definition CRUD.
type AdminBadgeHandler struct {
	db *gorm.DB
}

// NewAdminBadgeHandler creates a new AdminBadgeHandler.
func NewAdminBadgeHandler(db *gorm.DB) *AdminBadgeHandler {
	return &AdminBadgeHandler{db: db}
}

// List returns all badge definitions.
// GET /api/admin/badges
func (h *AdminBadgeHandler) List(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	var badges []models.BadgeDefinition
	h.db.Order("sort_order asc, name asc").Find(&badges)
	return c.JSON(fiber.Map{"badges": badges})
}

// Create adds a new badge definition.
// POST /api/admin/badges
func (h *AdminBadgeHandler) Create(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	var body models.BadgeDefinition
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Slug == "" || body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slug and name required"})
	}
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "badge slug already exists"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"badge": body})
}

// Update modifies a badge definition.
// PUT /api/admin/badges/:id
func (h *AdminBadgeHandler) Update(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	id := c.Params("id")
	var badge models.BadgeDefinition
	if err := h.db.First(&badge, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "badge not found"})
	}
	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	h.db.Model(&badge).Updates(body)
	h.db.First(&badge, id)
	return c.JSON(fiber.Map{"badge": badge})
}

// Delete removes a badge definition.
// DELETE /api/admin/badges/:id
func (h *AdminBadgeHandler) Delete(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	id := c.Params("id")
	if err := h.db.Delete(&models.BadgeDefinition{}, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "badge not found"})
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}
