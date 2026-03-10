package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// DocHandler handles wiki/documentation endpoints.
type DocHandler struct {
	db *gorm.DB
}

// NewDocHandler creates a new DocHandler.
func NewDocHandler(db *gorm.DB) *DocHandler {
	return &DocHandler{db: db}
}

// List returns docs for a project.
// Includes both user-synced docs and system docs (from repo docs/ folder).
// GET /api/projects/:slug/docs
func (h *DocHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := c.Params("slug")

	q := h.db.Where(
		"(project_slug = ? AND user_id = ?) OR (project_slug = ? AND user_id = 0)",
		slug, user.ID, "_system",
	)

	if category := c.Query("category"); category != "" {
		q = q.Where("category = ?", category)
	}

	var docs []models.Doc
	if err := q.Order("category ASC, title ASC").Find(&docs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list docs"})
	}

	return c.JSON(docs)
}

// Show returns a single doc.
// Checks both user-scoped and system docs.
// GET /api/projects/:slug/docs/:id
func (h *DocHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := c.Params("slug")
	id := c.Params("id")

	var doc models.Doc
	if err := h.db.Where(
		"(project_slug = ? AND doc_id = ? AND user_id = ?) OR (project_slug = ? AND doc_id = ? AND user_id = 0)",
		slug, id, user.ID, "_system", id,
	).First(&doc).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "doc not found"})
	}

	return c.JSON(doc)
}

// SystemList returns all system-level docs (from the repo docs/ folder).
// No auth required — used by the wiki when no project is selected.
// GET /api/docs
func (h *DocHandler) SystemList(c fiber.Ctx) error {
	var docs []models.Doc

	q := h.db.Where("project_slug = ? AND user_id = 0", "_system")

	if category := c.Query("category"); category != "" {
		q = q.Where("category = ?", category)
	}

	if err := q.Order("category ASC, title ASC").Find(&docs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list docs"})
	}

	return c.JSON(docs)
}

// SystemShow returns a single system doc by ID.
// No auth required.
// GET /api/docs/:id
func (h *DocHandler) SystemShow(c fiber.Ctx) error {
	id := c.Params("id")

	var doc models.Doc
	if err := h.db.Where("project_slug = ? AND doc_id = ? AND user_id = 0", "_system", id).
		First(&doc).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "doc not found"})
	}

	return c.JSON(doc)
}

// SystemUpdate updates fields of a system doc (title, body, icon, color).
// Auth required.
// PUT /api/docs/:id
func (h *DocHandler) SystemUpdate(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var doc models.Doc
	if err := h.db.Where("project_slug = ? AND doc_id = ? AND user_id = 0", "_system", id).
		First(&doc).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "doc not found"})
	}

	var body struct {
		Title *string `json:"title"`
		Body  *string `json:"body"`
		Icon  *string `json:"icon"`
		Color *string `json:"color"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Body != nil {
		doc.Body = *body.Body
	}
	if body.Title != nil && *body.Title != "" {
		doc.Title = *body.Title
	}
	if body.Icon != nil {
		doc.Icon = *body.Icon
	}
	if body.Color != nil {
		doc.Color = *body.Color
	}
	doc.Version++

	if err := h.db.Save(&doc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update doc"})
	}

	return c.JSON(doc)
}

// SystemPin toggles the pinned state of a system doc.
// Auth required.
// PATCH /api/docs/:id/pin
func (h *DocHandler) SystemPin(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var doc models.Doc
	if err := h.db.Where("project_slug = ? AND doc_id = ? AND user_id = 0", "_system", id).
		First(&doc).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "doc not found"})
	}

	doc.Pinned = !doc.Pinned
	if err := h.db.Save(&doc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update doc"})
	}

	return c.JSON(doc)
}

// SystemDelete deletes a system doc.
// Auth required.
// DELETE /api/docs/:id
func (h *DocHandler) SystemDelete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	result := h.db.Where("project_slug = ? AND doc_id = ? AND user_id = 0", "_system", id).
		Delete(&models.Doc{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete doc"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "doc not found"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
