package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// TaskHandler handles task CRUD endpoints.
type TaskHandler struct {
	db *gorm.DB
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(db *gorm.DB) *TaskHandler {
	return &TaskHandler{db: db}
}

func (h *TaskHandler) projectBySlug(slug string, userID uint) (*models.Project, error) {
	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", slug, userID).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (h *TaskHandler) epicByID(epicID, projectID string) (*models.Epic, error) {
	var epic models.Epic
	if err := h.db.Where("id = ? AND project_id = ?", epicID, projectID).First(&epic).Error; err != nil {
		return nil, err
	}
	return &epic, nil
}

func (h *TaskHandler) storyByID(storyID, epicID string) (*models.Story, error) {
	var story models.Story
	if err := h.db.Where("id = ? AND epic_id = ?", storyID, epicID).First(&story).Error; err != nil {
		return nil, err
	}
	return &story, nil
}

// List handles GET /api/projects/:slug/epics/:epicId/stories/:storyId/tasks
func (h *TaskHandler) List(c fiber.Ctx) error {
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

	story, err := h.storyByID(c.Params("storyId"), epic.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "story not found"})
	}

	var tasks []models.Task
	if err := h.db.Where("story_id = ?", story.ID).
		Order("position asc, created_at asc").
		Find(&tasks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch tasks"})
	}

	return c.JSON(tasks)
}

// Create handles POST /api/projects/:slug/epics/:epicId/stories/:storyId/tasks
func (h *TaskHandler) Create(c fiber.Ctx) error {
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

	story, err := h.storyByID(c.Params("storyId"), epic.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "story not found"})
	}

	var body struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Priority    string  `json:"priority"`
		AssigneeID  *uint   `json:"assignee_id"`
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

	task := models.Task{
		StoryID:     story.ID,
		ProjectID:   project.ID,
		UserID:      user.ID,
		AssigneeID:  body.AssigneeID,
		Title:       body.Title,
		Description: body.Description,
		Status:      "backlog",
		Priority:    priority,
		Version:     1,
	}

	if err := h.db.Create(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create task"})
	}

	return c.Status(fiber.StatusCreated).JSON(task)
}

// Show handles GET /api/projects/:slug/epics/:epicId/stories/:storyId/tasks/:taskId
func (h *TaskHandler) Show(c fiber.Ctx) error {
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

	story, err := h.storyByID(c.Params("storyId"), epic.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "story not found"})
	}

	var task models.Task
	if err := h.db.Where("id = ? AND story_id = ?", c.Params("taskId"), story.ID).
		First(&task).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
	}

	return c.JSON(task)
}

// Update handles PUT /api/projects/:slug/epics/:epicId/stories/:storyId/tasks/:taskId
func (h *TaskHandler) Update(c fiber.Ctx) error {
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

	story, err := h.storyByID(c.Params("storyId"), epic.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "story not found"})
	}

	var task models.Task
	if err := h.db.Where("id = ? AND story_id = ?", c.Params("taskId"), story.ID).
		First(&task).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		Position    *int   `json:"position"`
		AssigneeID  *uint  `json:"assignee_id"`
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
	if body.AssigneeID != nil {
		updates["assignee_id"] = *body.AssigneeID
	}

	if err := h.db.Model(&task).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update task"})
	}

	return c.JSON(task)
}

// Delete handles DELETE /api/projects/:slug/epics/:epicId/stories/:storyId/tasks/:taskId
func (h *TaskHandler) Delete(c fiber.Ctx) error {
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

	story, err := h.storyByID(c.Params("storyId"), epic.ID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "story not found"})
	}

	var task models.Task
	if err := h.db.Where("id = ? AND story_id = ?", c.Params("taskId"), story.ID).
		First(&task).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "task not found"})
	}

	if err := h.db.Delete(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete task"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
