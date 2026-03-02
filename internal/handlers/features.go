package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// FeatureHandler handles feature CRUD endpoints.
type FeatureHandler struct {
	db *gorm.DB
}

// NewFeatureHandler creates a new FeatureHandler.
func NewFeatureHandler(db *gorm.DB) *FeatureHandler {
	return &FeatureHandler{db: db}
}

// List returns features for a project.
// GET /api/projects/:slug/features
func (h *FeatureHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := c.Params("slug")

	var features []models.Feature
	if err := h.db.Where("project_slug = ? AND user_id = ?", slug, user.ID).
		Order("created_at DESC").Find(&features).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list features"})
	}

	return c.JSON(features)
}

// Show returns a single feature.
// GET /api/projects/:slug/features/:id
func (h *FeatureHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := c.Params("slug")
	id := c.Params("id")

	var feature models.Feature
	if err := h.db.Where("project_slug = ? AND feature_id = ? AND user_id = ?", slug, id, user.ID).
		First(&feature).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	return c.JSON(feature)
}

// Update updates a feature.
// PUT /api/projects/:slug/features/:id
func (h *FeatureHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := c.Params("slug")
	id := c.Params("id")

	var feature models.Feature
	if err := h.db.Where("project_slug = ? AND feature_id = ? AND user_id = ?", slug, id, user.ID).
		First(&feature).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if v, ok := body["title"].(string); ok {
		updates["title"] = v
	}
	if v, ok := body["description"].(string); ok {
		updates["description"] = v
	}
	if v, ok := body["status"].(string); ok {
		updates["status"] = v
	}
	if v, ok := body["priority"].(string); ok {
		updates["priority"] = v
	}
	if v, ok := body["assignee"].(string); ok {
		updates["assignee"] = v
	}
	if v, ok := body["estimate"].(string); ok {
		updates["estimate"] = v
	}
	if v, ok := body["body"].(string); ok {
		updates["body"] = v
	}

	if err := h.db.Model(&feature).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update feature"})
	}

	h.db.First(&feature, "id = ?", feature.ID)
	return c.JSON(feature)
}

// Delete deletes a feature.
// DELETE /api/projects/:slug/features/:id
func (h *FeatureHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := c.Params("slug")
	id := c.Params("id")

	result := h.db.Where("project_slug = ? AND feature_id = ? AND user_id = ?", slug, id, user.ID).
		Delete(&models.Feature{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete feature"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	return c.JSON(fiber.Map{"message": "feature deleted"})
}
