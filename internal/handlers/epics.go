package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// EpicHandler handles epic CRUD endpoints.
type EpicHandler struct {
	db *gorm.DB
}

// NewEpicHandler creates a new EpicHandler.
func NewEpicHandler(db *gorm.DB) *EpicHandler {
	return &EpicHandler{db: db}
}

// projectBySlug is a helper to load a project owned by the current user.
func (h *EpicHandler) projectBySlug(slug string, userID uint) (*models.Project, error) {
	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", slug, userID).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// List handles GET /api/projects/:slug/epics
func (h *EpicHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var epics []models.Epic
	if err := h.db.Where("project_id = ?", project.ID).
		Order("position asc, created_at asc").
		Find(&epics).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch epics"})
	}

	return c.JSON(epics)
}

// Create handles POST /api/projects/:slug/epics
func (h *EpicHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}

	priority := body.Priority
	if priority == "" {
		priority = "medium"
	}

	epic := models.Epic{
		ProjectID:   project.ID,
		UserID:      user.ID,
		Title:       body.Title,
		Description: body.Description,
		Status:      "backlog",
		Priority:    priority,
		Version:     1,
	}

	if err := h.db.Create(&epic).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create epic"})
	}

	return c.Status(fiber.StatusCreated).JSON(epic)
}

// Show handles GET /api/projects/:slug/epics/:epicId
func (h *EpicHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var epic models.Epic
	if err := h.db.Where("id = ? AND project_id = ?", c.Params("epicId"), project.ID).
		First(&epic).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "epic not found"})
	}

	return c.JSON(epic)
}

// Update handles PUT /api/projects/:slug/epics/:epicId
func (h *EpicHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var epic models.Epic
	if err := h.db.Where("id = ? AND project_id = ?", c.Params("epicId"), project.ID).
		First(&epic).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "epic not found"})
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		Position    *int   `json:"position"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if body.Title != "" {
		updates["title"] = body.Title
	}
	if body.Description != "" {
		updates["description"] = body.Description
	}
	if body.Status != "" {
		updates["status"] = body.Status
	}
	if body.Priority != "" {
		updates["priority"] = body.Priority
	}
	if body.Position != nil {
		updates["position"] = *body.Position
	}

	if err := h.db.Model(&epic).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update epic"})
	}

	return c.JSON(epic)
}

// Delete handles DELETE /api/projects/:slug/epics/:epicId
func (h *EpicHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var epic models.Epic
	if err := h.db.Where("id = ? AND project_id = ?", c.Params("epicId"), project.ID).
		First(&epic).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "epic not found"})
	}

	if err := h.db.Delete(&epic).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete epic"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
