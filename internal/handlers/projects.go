package handlers

import (
	"fmt"
	"regexp"
	"strings"

	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// ProjectHandler handles project CRUD endpoints.
type ProjectHandler struct {
	db *gorm.DB
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

// slugify converts a name into a URL-safe slug.
func slugify(name string) string {
	lower := strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

// uniqueSlug ensures the slug is unique per user, appending -2, -3, etc. if needed.
func (h *ProjectHandler) uniqueSlug(base string, userID uint, excludeID string) string {
	slug := base
	for i := 2; ; i++ {
		var count int64
		q := h.db.Model(&models.Project{}).Where("slug = ? AND user_id = ?", slug, userID)
		if excludeID != "" {
			q = q.Where("id != ?", excludeID)
		}
		q.Count(&count)
		if count == 0 {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

// List handles GET /api/projects
func (h *ProjectHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var projects []models.Project
	if err := h.db.Where("user_id = ?", user.ID).
		Order("created_at desc").
		Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch projects"})
	}

	return c.JSON(projects)
}

// Create handles POST /api/projects
func (h *ProjectHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		TeamID      *string `json:"team_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	slug := h.uniqueSlug(slugify(body.Name), user.ID, "")

	project := models.Project{
		UserID:      user.ID,
		TeamID:      body.TeamID,
		Name:        body.Name,
		Slug:        slug,
		Description: body.Description,
		SyncStatus:  "not_synced",
		Version:     1,
	}

	if err := h.db.Create(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create project"})
	}

	return c.Status(fiber.StatusCreated).JSON(project)
}

// Show handles GET /api/projects/:slug
func (h *ProjectHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", c.Params("slug"), user.ID).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var features []models.Feature
	h.db.Where("project_slug = ? AND user_id = ?", project.Slug, user.ID).
		Order("created_at desc").
		Find(&features)

	return c.JSON(fiber.Map{
		"id":          project.ID,
		"name":        project.Name,
		"slug":        project.Slug,
		"description": project.Description,
		"created_at":  project.CreatedAt,
		"updated_at":  project.UpdatedAt,
		"features":    features,
	})
}

// Update handles PUT /api/projects/:slug
func (h *ProjectHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", c.Params("slug"), user.ID).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
		updates["slug"] = h.uniqueSlug(slugify(body.Name), user.ID, project.ID)
	}
	if body.Description != "" {
		updates["description"] = body.Description
	}

	if err := h.db.Model(&project).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update project"})
	}

	return c.JSON(project)
}

// Delete handles DELETE /api/projects/:slug
func (h *ProjectHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", c.Params("slug"), user.ID).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	if err := h.db.Delete(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete project"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// Tree handles GET /api/projects/:slug/tree
func (h *ProjectHandler) Tree(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", c.Params("slug"), user.ID).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	// Load epics with stories and tasks
	var epics []models.Epic
	if err := h.db.Where("project_id = ?", project.ID).
		Order("position asc, created_at asc").
		Find(&epics).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load epics"})
	}

	for i := range epics {
		var stories []models.Story
		if err := h.db.Where("epic_id = ?", epics[i].ID).
			Order("position asc, created_at asc").
			Find(&stories).Error; err != nil {
			continue
		}
		for j := range stories {
			var tasks []models.Task
			h.db.Where("story_id = ?", stories[j].ID).
				Order("position asc, created_at asc").
				Find(&tasks)
			stories[j].Tasks = tasks
		}
		epics[i].Stories = stories
	}

	var features []models.Feature
	h.db.Where("project_slug = ? AND user_id = ?", project.Slug, user.ID).
		Order("created_at desc").
		Find(&features)

	return c.JSON(fiber.Map{
		"project": fiber.Map{
			"id":          project.ID,
			"name":        project.Name,
			"slug":        project.Slug,
			"description": project.Description,
			"created_at":  project.CreatedAt,
			"updated_at":  project.UpdatedAt,
			"features":    features,
		},
		"epics": epics,
	})
}
