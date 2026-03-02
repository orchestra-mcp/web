package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// AiSessionHandler handles AI session endpoints.
type AiSessionHandler struct {
	db *gorm.DB
}

// NewAiSessionHandler creates a new AiSessionHandler.
func NewAiSessionHandler(db *gorm.DB) *AiSessionHandler {
	return &AiSessionHandler{db: db}
}

// List handles GET /api/ai/sessions
func (h *AiSessionHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var sessions []models.AiSession
	if err := h.db.Where("user_id = ?", user.ID).
		Order("pinned desc, last_message_at desc").
		Find(&sessions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch sessions"})
	}

	return c.JSON(sessions)
}

// Create handles POST /api/ai/sessions
func (h *AiSessionHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name      string  `json:"name"`
		Model     string  `json:"model"`
		ProjectID *string `json:"project_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	name := body.Name
	if name == "" {
		name = "New Chat"
	}

	model := body.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	session := models.AiSession{
		UserID:    user.ID,
		ProjectID: body.ProjectID,
		Name:      name,
		Model:     model,
		Version:   1,
	}

	if err := h.db.Create(&session).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create session"})
	}

	return c.Status(fiber.StatusCreated).JSON(session)
}

// Rename handles PATCH /api/ai/sessions/:id/rename
func (h *AiSessionHandler) Rename(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var session models.AiSession
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&session).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	if err := h.db.Model(&session).Update("name", body.Name).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rename session"})
	}

	return c.JSON(session)
}

// Delete handles DELETE /api/ai/sessions/:id
func (h *AiSessionHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var session models.AiSession
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&session).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
	}

	if err := h.db.Delete(&session).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete session"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
