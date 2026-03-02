package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// StoryHandler handles story CRUD endpoints.
type StoryHandler struct {
	db *gorm.DB
}

// NewStoryHandler creates a new StoryHandler.
func NewStoryHandler(db *gorm.DB) *StoryHandler {
	return &StoryHandler{db: db}
}

// projectBySlugForStory loads a project owned by the current user.
func (h *StoryHandler) projectBySlug(slug string, userID uint) (*models.Project, error) {
	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", slug, userID).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// epicByID loads an epic belonging to a project.
func (h *StoryHandler) epicByID(epicID, projectID string) (*models.Epic, error) {
	var epic models.Epic
	if err := h.db.Where("id = ? AND project_id = ?", epicID, projectID).First(&epic).Error; err != nil {
		return nil, err
	}
	return &epic, nil
}

// List handles GET /api/projects/:slug/epics/:epicId/stories
func (h *StoryHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	epic, err := h.epicByID(c.Params("epicId"), project.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "epic not found"})
	}

	var stories []models.Story
	if err := h.db.Where("epic_id = ?", epic.ID).
		Order("position asc, created_at asc").
		Find(&stories).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch stories"})
	}

	return c.JSON(stories)
}

// Create handles POST /api/projects/:slug/epics/:epicId/stories
func (h *StoryHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	epic, err := h.epicByID(c.Params("epicId"), project.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "epic not found"})
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		StoryPoints *int   `json:"story_points"`
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

	story := models.Story{
		EpicID:      epic.ID,
		ProjectID:   project.ID,
		UserID:      user.ID,
		Title:       body.Title,
		Description: body.Description,
		Status:      "backlog",
		Priority:    priority,
		StoryPoints: body.StoryPoints,
		Version:     1,
	}

	if err := h.db.Create(&story).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create story"})
	}

	return c.Status(fiber.StatusCreated).JSON(story)
}

// Show handles GET /api/projects/:slug/epics/:epicId/stories/:storyId
func (h *StoryHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	epic, err := h.epicByID(c.Params("epicId"), project.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "epic not found"})
	}

	var story models.Story
	if err := h.db.Where("id = ? AND epic_id = ?", c.Params("storyId"), epic.ID).
		First(&story).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "story not found"})
	}

	return c.JSON(story)
}

// Update handles PUT /api/projects/:slug/epics/:epicId/stories/:storyId
func (h *StoryHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	epic, err := h.epicByID(c.Params("epicId"), project.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "epic not found"})
	}

	var story models.Story
	if err := h.db.Where("id = ? AND epic_id = ?", c.Params("storyId"), epic.ID).
		First(&story).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "story not found"})
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		StoryPoints *int   `json:"story_points"`
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
	if body.StoryPoints != nil {
		updates["story_points"] = *body.StoryPoints
	}
	if body.Position != nil {
		updates["position"] = *body.Position
	}

	if err := h.db.Model(&story).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update story"})
	}

	return c.JSON(story)
}

// Delete handles DELETE /api/projects/:slug/epics/:epicId/stories/:storyId
func (h *StoryHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.projectBySlug(c.Params("slug"), user.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	epic, err := h.epicByID(c.Params("epicId"), project.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "epic not found"})
	}

	var story models.Story
	if err := h.db.Where("id = ? AND epic_id = ?", c.Params("storyId"), epic.ID).
		First(&story).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "story not found"})
	}

	if err := h.db.Delete(&story).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete story"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
