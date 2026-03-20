package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// AgentHandler handles agent CRUD endpoints.
type AgentHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewAgentHandler creates a new AgentHandler.
func NewAgentHandler(db *gorm.DB, wsHub ...*hub.Hub) *AgentHandler {
	h := &AgentHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// List handles GET /api/agents
func (h *AgentHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	q := h.db.Model(&models.Agent{})

	// Filter by scope if provided
	if scope := c.Query("scope"); scope != "" {
		switch scope {
		case "personal":
			q = q.Where("user_id = ? AND team_id IS NULL", user.ID)
		case "team":
			if teamID := c.Query("team_id"); teamID != "" {
				q = q.Where("team_id = ?", teamID)
			} else if len(teamIDs) > 0 {
				q = q.Where("team_id IN ?", teamIDs)
			} else {
				return c.JSON([]models.Agent{})
			}
		case "public":
			q = q.Where("scope = ?", "public")
		}
	} else {
		// Default: personal + team-scoped + public
		if len(teamIDs) > 0 {
			q = q.Where(
				"(user_id = ? AND team_id IS NULL) OR (team_id IN ?) OR (scope = ?)",
				user.ID, teamIDs, "public",
			)
		} else {
			q = q.Where(
				"(user_id = ? AND team_id IS NULL) OR (scope = ?)",
				user.ID, "public",
			)
		}
	}

	// Additional team_id filter (outside scope filter)
	if teamID := c.Query("team_id"); teamID != "" && c.Query("scope") == "" {
		q = q.Where("team_id = ?", teamID)
	}

	var agents []models.Agent
	if err := q.Order("updated_at desc").Find(&agents).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch agents"})
	}

	return c.JSON(agents)
}

// Create handles POST /api/agents
func (h *AgentHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description string  `json:"description"`
		Content     string  `json:"content"`
		Scope       string  `json:"scope"`
		TeamID      *string `json:"team_id"`
		Icon        string  `json:"icon"`
		Color       string  `json:"color"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	agent := models.Agent{
		UserID:      user.ID,
		TeamID:      body.TeamID,
		Name:        body.Name,
		Slug:        body.Slug,
		Description: body.Description,
		Content:     body.Content,
		Scope:       body.Scope,
		Icon:        body.Icon,
		Color:       body.Color,
		Version:     1,
	}

	if agent.Scope == "" {
		agent.Scope = "personal"
	}

	// Generate public_url for public agents
	if agent.Scope == "public" && agent.Slug != "" {
		agent.PublicURL = agent.Slug
	}

	if err := h.db.Create(&agent).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create agent"})
	}

	broadcastSync(h.hub, user.ID, "agent", agent.ID, "upsert")
	return c.Status(fiber.StatusCreated).JSON(agent)
}

// Show handles GET /api/agents/:id
func (h *AgentHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	var agent models.Agent
	q := h.db.Where("id = ?", c.Params("id"))

	// Verify access: owner, team member, or public
	if len(teamIDs) > 0 {
		q = q.Where(
			"(user_id = ?) OR (team_id IN ?) OR (scope = ?)",
			user.ID, teamIDs, "public",
		)
	} else {
		q = q.Where(
			"(user_id = ?) OR (scope = ?)",
			user.ID, "public",
		)
	}

	if err := q.First(&agent).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "agent not found"})
	}

	return c.JSON(agent)
}

// Update handles PUT /api/agents/:id
func (h *AgentHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var agent models.Agent
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&agent).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "agent not found"})
	}

	var body struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description string  `json:"description"`
		Content     string  `json:"content"`
		Scope       string  `json:"scope"`
		TeamID      *string `json:"team_id"`
		Icon        string  `json:"icon"`
		Color       string  `json:"color"`
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
				slug = agent.Slug
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

	if err := h.db.Model(&agent).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update agent"})
	}

	broadcastSync(h.hub, user.ID, "agent", agent.ID, "upsert")
	return c.JSON(agent)
}

// Delete handles DELETE /api/agents/:id
func (h *AgentHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var agent models.Agent
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&agent).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "agent not found"})
	}

	if err := h.db.Delete(&agent).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete agent"})
	}

	broadcastSync(h.hub, user.ID, "agent", agent.ID, "delete")
	return c.JSON(fiber.Map{"ok": true})
}

// PublicShow handles GET /api/agents/public/:slug
func (h *AgentHandler) PublicShow(c fiber.Ctx) error {
	var agent models.Agent
	if err := h.db.Where("slug = ? AND scope = ?", c.Params("slug"), "public").
		First(&agent).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "agent not found"})
	}

	return c.JSON(agent)
}
