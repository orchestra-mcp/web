package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// NoteHandler handles note CRUD endpoints.
type NoteHandler struct {
	db *gorm.DB
}

// NewNoteHandler creates a new NoteHandler.
func NewNoteHandler(db *gorm.DB) *NoteHandler {
	return &NoteHandler{db: db}
}

// List handles GET /api/notes
func (h *NoteHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var notes []models.Note
	if err := h.db.Where("user_id = ?", user.ID).
		Order("pinned desc, updated_at desc").
		Find(&notes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch notes"})
	}

	return c.JSON(notes)
}

// Create handles POST /api/notes
func (h *NoteHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Title     string  `json:"title"`
		Content   string  `json:"content"`
		ProjectID *string `json:"project_id"`
		TeamID    *string `json:"team_id"`
		Pinned    bool    `json:"pinned"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}

	note := models.Note{
		UserID:    user.ID,
		ProjectID: body.ProjectID,
		TeamID:    body.TeamID,
		Title:     body.Title,
		Content:   body.Content,
		Pinned:    body.Pinned,
		Version:   1,
	}

	if err := h.db.Create(&note).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create note"})
	}

	return c.Status(fiber.StatusCreated).JSON(note)
}

// Show handles GET /api/notes/:id
func (h *NoteHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var note models.Note
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&note).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "note not found"})
	}

	return c.JSON(note)
}

// Update handles PUT /api/notes/:id
func (h *NoteHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var note models.Note
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&note).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "note not found"})
	}

	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Pinned  *bool  `json:"pinned"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if body.Title != "" {
		updates["title"] = body.Title
	}
	if body.Content != "" {
		updates["content"] = body.Content
	}
	if body.Pinned != nil {
		updates["pinned"] = *body.Pinned
	}

	if err := h.db.Model(&note).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update note"})
	}

	return c.JSON(note)
}

// Delete handles DELETE /api/notes/:id
func (h *NoteHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var note models.Note
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&note).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "note not found"})
	}

	if err := h.db.Delete(&note).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete note"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
