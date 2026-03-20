package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SkillHandler handles skill CRUD endpoints.
type SkillHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewSkillHandler creates a new SkillHandler.
func NewSkillHandler(db *gorm.DB, wsHub ...*hub.Hub) *SkillHandler {
	h := &SkillHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// List handles GET /api/skills
func (h *SkillHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get user's team IDs
	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	var skills []models.Skill
	query := h.db.Order("updated_at desc")

	// Apply scope filter if provided
	scopeFilter := c.Query("scope")
	teamFilter := c.Query("team_id")

	if scopeFilter != "" && teamFilter != "" {
		// Both filters active
		switch scopeFilter {
		case "personal":
			query = query.Where("user_id = ? AND team_id IS NULL", user.ID)
		case "team":
			query = query.Where("team_id = ?", teamFilter)
		case "public":
			query = query.Where("scope = 'public'")
		}
	} else if scopeFilter != "" {
		switch scopeFilter {
		case "personal":
			query = query.Where("user_id = ? AND team_id IS NULL", user.ID)
		case "team":
			if len(teamIDs) > 0 {
				query = query.Where("team_id IN ?", teamIDs)
			} else {
				// No teams, return empty
				return c.JSON([]models.Skill{})
			}
		case "public":
			query = query.Where("scope = 'public'")
		}
	} else if teamFilter != "" {
		query = query.Where("team_id = ?", teamFilter)
	} else {
		// Default: personal + team-scoped + public
		if len(teamIDs) > 0 {
			query = query.Where(
				"(user_id = ? AND team_id IS NULL) OR team_id IN ? OR scope = 'public'",
				user.ID, teamIDs,
			)
		} else {
			query = query.Where(
				"user_id = ? OR scope = 'public'",
				user.ID,
			)
		}
	}

	if err := query.Find(&skills).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch skills"})
	}

	return c.JSON(skills)
}

// Create handles POST /api/skills
func (h *SkillHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name        string   `json:"name"`
		Slug        string   `json:"slug"`
		Description string   `json:"description"`
		Content     string   `json:"content"`
		Scope       string   `json:"scope"`
		TeamID      *string  `json:"team_id"`
		Icon        string   `json:"icon"`
		Color       string   `json:"color"`
		Stacks      []string `json:"stacks"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	scope := body.Scope
	if scope == "" {
		scope = "personal"
	}

	var stacksJSON datatypes.JSON
	if body.Stacks != nil {
		raw, _ := json.Marshal(body.Stacks)
		stacksJSON = datatypes.JSON(raw)
	}

	skill := models.Skill{
		UserID:      user.ID,
		TeamID:      body.TeamID,
		Name:        body.Name,
		Slug:        body.Slug,
		Description: body.Description,
		Content:     body.Content,
		Scope:       scope,
		Icon:        body.Icon,
		Color:       body.Color,
		Stacks:      stacksJSON,
		Version:     1,
	}

	// If scope is public, generate public_url from slug
	if scope == "public" && body.Slug != "" {
		skill.PublicURL = body.Slug
	}

	if err := h.db.Create(&skill).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create skill"})
	}

	broadcastSync(h.hub, user.ID, "skill", skill.ID, "upsert")
	return c.Status(fiber.StatusCreated).JSON(skill)
}

// Show handles GET /api/skills/:id
func (h *SkillHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get user's team IDs
	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	var skill models.Skill
	query := h.db.Where("id = ?", c.Params("id"))

	if len(teamIDs) > 0 {
		query = query.Where(
			"user_id = ? OR team_id IN ? OR scope = 'public'",
			user.ID, teamIDs,
		)
	} else {
		query = query.Where("user_id = ? OR scope = 'public'", user.ID)
	}

	if err := query.First(&skill).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}

	return c.JSON(skill)
}

// Update handles PUT /api/skills/:id
func (h *SkillHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get user's team IDs
	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	var skill models.Skill
	query := h.db.Where("id = ?", c.Params("id"))

	// Verify ownership: user owns it directly or is a member of the skill's team
	if len(teamIDs) > 0 {
		query = query.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.First(&skill).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}

	var body struct {
		Name        string   `json:"name"`
		Slug        string   `json:"slug"`
		Description string   `json:"description"`
		Content     string   `json:"content"`
		Scope       string   `json:"scope"`
		TeamID      *string  `json:"team_id"`
		Icon        string   `json:"icon"`
		Color       string   `json:"color"`
		Stacks      []string `json:"stacks"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Slug != "" {
		updates["slug"] = body.Slug
	}
	if body.Description != "" {
		updates["description"] = body.Description
	}
	if body.Content != "" {
		updates["content"] = body.Content
	}
	if body.Scope != "" {
		updates["scope"] = body.Scope
		if body.Scope == "public" {
			slug := body.Slug
			if slug == "" {
				slug = skill.Slug
			}
			updates["public_url"] = slug
		}
	}
	if body.TeamID != nil {
		updates["team_id"] = body.TeamID
	}
	if body.Icon != "" {
		updates["icon"] = body.Icon
	}
	if body.Color != "" {
		updates["color"] = body.Color
	}
	if body.Stacks != nil {
		raw, _ := json.Marshal(body.Stacks)
		updates["stacks"] = datatypes.JSON(raw)
	}

	if err := h.db.Model(&skill).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update skill"})
	}

	broadcastSync(h.hub, user.ID, "skill", skill.ID, "upsert")
	return c.JSON(skill)
}

// Delete handles DELETE /api/skills/:id
func (h *SkillHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get user's team IDs
	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	var skill models.Skill
	query := h.db.Where("id = ?", c.Params("id"))

	// Verify ownership: user owns it directly or is a member of the skill's team
	if len(teamIDs) > 0 {
		query = query.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.First(&skill).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}

	if err := h.db.Delete(&skill).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete skill"})
	}

	broadcastSync(h.hub, user.ID, "skill", skill.ID, "delete")
	return c.JSON(fiber.Map{"ok": true})
}

// PublicShow handles GET /api/skills/public/:slug
func (h *SkillHandler) PublicShow(c fiber.Ctx) error {
	var skill models.Skill
	if err := h.db.Where("slug = ? AND scope = 'public'", c.Params("slug")).
		First(&skill).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "skill not found"})
	}

	return c.JSON(skill)
}
