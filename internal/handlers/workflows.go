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

// WorkflowHandler handles workflow CRUD endpoints.
type WorkflowHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewWorkflowHandler creates a new WorkflowHandler.
func NewWorkflowHandler(db *gorm.DB, wsHub ...*hub.Hub) *WorkflowHandler {
	h := &WorkflowHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// teamScopedWorkflows returns a query scoped to user's own + team workflows.
func teamScopedWorkflows(db *gorm.DB, userID uint) *gorm.DB {
	teamIDs := userTeamIDs(db, userID)
	if len(teamIDs) == 0 {
		return db.Where("user_id = ?", userID)
	}
	return db.Where("user_id = ? OR team_id IN ?", userID, teamIDs)
}

// List handles GET /api/workflows
func (h *WorkflowHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	query := teamScopedWorkflows(h.db, user.ID).Order("updated_at DESC")

	if projectSlug := c.Query("project"); projectSlug != "" {
		query = query.Where("project_slug = ?", projectSlug)
	}

	var workflows []models.Workflow
	if err := query.Find(&workflows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list workflows"})
	}

	return c.JSON(fiber.Map{
		"items": workflows,
		"meta": fiber.Map{
			"total":  len(workflows),
			"limit":  100,
			"offset": 0,
		},
	})
}

// Show handles GET /api/workflows/:id
func (h *WorkflowHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var workflow models.Workflow
	// Support lookup by workflow_id (WFL-XXX) or UUID
	q := teamScopedWorkflows(h.db, user.ID)
	if len(id) > 4 && id[:4] == "WFL-" {
		q = q.Where("workflow_id = ?", id)
	} else {
		q = q.Where("id = ?", id)
	}
	if err := q.First(&workflow).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workflow not found"})
	}

	return c.JSON(workflow)
}

// Create handles POST /api/workflows
func (h *WorkflowHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		WorkflowID   string          `json:"workflow_id"`
		ProjectSlug  string          `json:"project_slug"`
		Name         string          `json:"name"`
		Description  string          `json:"description"`
		InitialState string          `json:"initial_state"`
		IsDefault    bool            `json:"is_default"`
		States       json.RawMessage `json:"states"`
		Transitions  json.RawMessage `json:"transitions"`
		Gates        json.RawMessage `json:"gates"`
		TeamID       *string         `json:"team_id"`
		Meta         json.RawMessage `json:"meta"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	if body.InitialState == "" {
		body.InitialState = "todo"
	}

	states := body.States
	if states == nil {
		states = json.RawMessage(`{}`)
	}
	transitions := body.Transitions
	if transitions == nil {
		transitions = json.RawMessage(`[]`)
	}
	gates := body.Gates
	if gates == nil {
		gates = json.RawMessage(`{}`)
	}
	meta := body.Meta
	if meta == nil {
		meta = json.RawMessage(`{}`)
	}

	workflow := models.Workflow{
		UserID:       user.ID,
		TeamID:       body.TeamID,
		WorkflowID:   body.WorkflowID,
		ProjectSlug:  body.ProjectSlug,
		Name:         body.Name,
		Description:  body.Description,
		InitialState: body.InitialState,
		IsDefault:    body.IsDefault,
		States:       datatypes.JSON(states),
		Transitions:  datatypes.JSON(transitions),
		Gates:        datatypes.JSON(gates),
		Meta:         datatypes.JSON(meta),
		Version:      1,
	}

	if err := h.db.Create(&workflow).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create workflow"})
	}

	broadcastSync(h.hub, user.ID, "workflow", workflow.ID, "upsert")
	return c.Status(fiber.StatusCreated).JSON(workflow)
}

// Update handles PATCH /api/workflows/:id
func (h *WorkflowHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var workflow models.Workflow
	q := teamScopedWorkflows(h.db, user.ID)
	if len(id) > 4 && id[:4] == "WFL-" {
		q = q.Where("workflow_id = ?", id)
	} else {
		q = q.Where("id = ?", id)
	}
	if err := q.First(&workflow).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workflow not found"})
	}

	var body struct {
		Name         string          `json:"name"`
		Description  string          `json:"description"`
		InitialState string          `json:"initial_state"`
		IsDefault    *bool           `json:"is_default"`
		States       json.RawMessage `json:"states"`
		Transitions  json.RawMessage `json:"transitions"`
		Gates        json.RawMessage `json:"gates"`
		Meta         json.RawMessage `json:"meta"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Description != "" {
		updates["description"] = body.Description
	}
	if body.InitialState != "" {
		updates["initial_state"] = body.InitialState
	}
	if body.IsDefault != nil {
		updates["is_default"] = *body.IsDefault
	}
	if body.States != nil {
		updates["states"] = datatypes.JSON(body.States)
	}
	if body.Transitions != nil {
		updates["transitions"] = datatypes.JSON(body.Transitions)
	}
	if body.Gates != nil {
		updates["gates"] = datatypes.JSON(body.Gates)
	}
	if body.Meta != nil {
		updates["meta"] = datatypes.JSON(body.Meta)
	}

	if len(updates) > 0 {
		if err := h.db.Model(&workflow).Updates(updates).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update workflow"})
		}
	}

	// Reload to return fresh data
	h.db.First(&workflow, "id = ?", workflow.ID)

	broadcastSync(h.hub, user.ID, "workflow", workflow.ID, "upsert")
	return c.JSON(workflow)
}

// Delete handles DELETE /api/workflows/:id
func (h *WorkflowHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var workflow models.Workflow
	q := teamScopedWorkflows(h.db, user.ID)
	if len(id) > 4 && id[:4] == "WFL-" {
		q = q.Where("workflow_id = ?", id)
	} else {
		q = q.Where("id = ?", id)
	}
	if err := q.First(&workflow).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workflow not found"})
	}

	if err := h.db.Delete(&workflow).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete workflow"})
	}

	broadcastSync(h.hub, user.ID, "workflow", workflow.ID, "delete")
	return c.JSON(fiber.Map{"ok": true})
}
