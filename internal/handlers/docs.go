package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// DocHandler handles wiki/documentation endpoints.
type DocHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewDocHandler creates a new DocHandler.
func NewDocHandler(db *gorm.DB, wsHub ...*hub.Hub) *DocHandler {
	h := &DocHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
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

// SystemCreate creates a new system doc.
// Auth required (admin).
// POST /api/docs
func (h *DocHandler) SystemCreate(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Title     string  `json:"title"`
		Body      string  `json:"body"`
		Category  string  `json:"category"`
		Icon      string  `json:"icon"`
		Color     string  `json:"color"`
		Pinned    bool    `json:"pinned"`
		Published bool    `json:"published"`
		DocID     *string `json:"doc_id"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}

	// Auto-generate doc_id from title if not provided.
	docID := ""
	if body.DocID != nil && *body.DocID != "" {
		docID = *body.DocID
	} else {
		// Simple slug: lowercase, replace spaces with dashes, strip non-alphanum.
		slug := body.Title
		result := make([]byte, 0, len(slug))
		for _, ch := range slug {
			if ch >= 'a' && ch <= 'z' {
				result = append(result, byte(ch))
			} else if ch >= 'A' && ch <= 'Z' {
				result = append(result, byte(ch+32))
			} else if ch == ' ' || ch == '-' || ch == '_' {
				result = append(result, '-')
			}
		}
		docID = string(result)
	}

	doc := models.Doc{
		UserID:      0, // system doc
		ProjectSlug: "_system",
		DocID:       docID,
		Title:       body.Title,
		Body:        body.Body,
		Category:    body.Category,
		Icon:        body.Icon,
		Color:       body.Color,
		Pinned:      body.Pinned,
		Published:   body.Published,
		Version:     1,
	}

	if err := h.db.Create(&doc).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create doc"})
	}

	if h.hub != nil {
		broadcastSync(h.hub, user.ID, "doc", doc.DocID, "upsert")
	}
	return c.Status(fiber.StatusCreated).JSON(doc)
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

	broadcastSync(h.hub, user.ID, "doc", doc.DocID, "upsert")
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

	broadcastSync(h.hub, user.ID, "doc", doc.DocID, "upsert")
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

	broadcastSync(h.hub, user.ID, "doc", id, "delete")
	return c.JSON(fiber.Map{"ok": true})
}

// PublicDocList returns published docs for a team's project (no auth required).
// GET /api/public/docs/:team/:project
func (h *DocHandler) PublicDocList(c fiber.Ctx) error {
	teamSlug := c.Params("team")
	projectSlug := c.Params("project")

	// Find the team by slug.
	var team models.Team
	if err := h.db.Where("slug = ?", teamSlug).First(&team).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "team not found"})
	}

	var docs []models.Doc
	if err := h.db.Where("team_id = ? AND project_slug = ? AND published = ?", team.ID, projectSlug, true).
		Order("title ASC").Find(&docs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list docs"})
	}

	return c.JSON(docs)
}

// PublicDocShow returns a single published doc (no auth required).
// GET /api/public/docs/:team/:project/:slug
func (h *DocHandler) PublicDocShow(c fiber.Ctx) error {
	teamSlug := c.Params("team")
	projectSlug := c.Params("project")
	docSlug := c.Params("slug")

	// Find the team by slug.
	var team models.Team
	if err := h.db.Where("slug = ?", teamSlug).First(&team).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "team not found"})
	}

	var doc models.Doc
	if err := h.db.Where("team_id = ? AND project_slug = ? AND doc_id = ? AND published = ?",
		team.ID, projectSlug, docSlug, true).First(&doc).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "doc not found"})
	}

	return c.JSON(doc)
}
