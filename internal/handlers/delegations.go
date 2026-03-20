package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// DelegationHandler handles delegation CRUD endpoints.
type DelegationHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewDelegationHandler creates a new DelegationHandler.
func NewDelegationHandler(db *gorm.DB, h *hub.Hub) *DelegationHandler {
	return &DelegationHandler{db: db, hub: h}
}

// MyPending returns all delegations visible to the current user (team-scoped).
// GET /api/delegations
func (h *DelegationHandler) MyPending(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var delegations []models.Delegation
	if err := teamScopedDelegations(h.db, user.ID).
		Order("created_at DESC").
		Find(&delegations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list delegations"})
	}

	return c.JSON(fiber.Map{"delegations": delegations})
}

// Show returns a single delegation by UUID primary key.
// GET /api/delegations/:id
func (h *DelegationHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var delegation models.Delegation
	if err := teamScopedDelegations(h.db, user.ID).
		Where("id = ?", id).
		First(&delegation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "delegation not found"})
	}

	return c.JSON(fiber.Map{"delegation": delegation})
}

// Respond sets the response on a delegation and marks it as answered.
// POST /api/delegations/:id/respond
func (h *DelegationHandler) Respond(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var delegation models.Delegation
	if err := teamScopedDelegations(h.db, user.ID).
		Where("id = ?", id).
		First(&delegation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "delegation not found"})
	}

	var body struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	now := time.Now()
	updates := map[string]interface{}{
		"response":     body.Response,
		"status":       "answered",
		"responded_at": now,
		"version":      delegation.Version + 1,
	}

	if err := h.db.Model(&delegation).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to respond to delegation"})
	}

	h.db.First(&delegation, "id = ?", delegation.ID)

	// Write sync_log so MCP clients can pull the response.
	payload, _ := json.Marshal(map[string]interface{}{
		"id":            delegation.DelegationID,
		"feature_id":    delegation.FeatureID,
		"from_person":   delegation.FromPerson,
		"to_person":     delegation.ToPerson,
		"question":      delegation.Question,
		"context":       delegation.Context,
		"response":      delegation.Response,
		"status":        delegation.Status,
		"responded_at":  now.Format(time.RFC3339),
		"project_slug":  delegation.ProjectSlug,
		"body":          delegation.Body,
	})
	idemKey := fmt.Sprintf("delegation:%s:%d", delegation.DelegationID, delegation.Version)
	syncLog := models.SyncLog{
		UserID:         user.ID,
		DeviceID:       "web",
		EntityType:     "delegation",
		EntityID:       delegation.DelegationID,
		Action:         "upsert",
		Payload:        payload,
		Version:        int64(delegation.Version),
		IdempotencyKey: &idemKey,
		TeamID:         delegation.TeamID,
	}
	h.db.Create(&syncLog)

	// Notify the delegator (from_person) that the delegation was answered.
	if h.hub != nil && delegation.FromPerson != "" {
		var fromEmail string
		h.db.Model(&models.Person{}).Where("person_id = ?", delegation.FromPerson).Pluck("email", &fromEmail)
		if fromEmail != "" {
			var fromUser models.User
			if h.db.Where("email = ?", fromEmail).First(&fromUser).Error == nil {
				h.hub.BroadcastToUser(fromUser.ID, hub.Event{
					Type:       "notification",
					EntityType: "delegation",
					EntityID:   delegation.DelegationID,
					Action:     "answered",
					UserID:     user.ID,
					Timestamp:  now.Unix(),
					Title:      "Delegation Answered",
					Message:    fmt.Sprintf("%s responded to %s: %s", user.Name, delegation.DelegationID, truncate(body.Response, 80)),
					NType:      "success",
				})
			}
		}
	}

	return c.JSON(fiber.Map{"delegation": delegation})
}

// ProjectList returns all delegations for a project slug scoped to the user.
// GET /api/projects/:slug/delegations
func (h *DelegationHandler) ProjectList(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := resolveProjectSlug(h.db, c.Params("slug"))

	var delegations []models.Delegation
	if err := teamScopedDelegations(h.db, user.ID).
		Where("project_slug = ?", slug).
		Order("created_at DESC").
		Find(&delegations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list delegations"})
	}

	return c.JSON(fiber.Map{"delegations": delegations})
}

// teamScopedDelegations scopes delegation queries to the user and their teams.
func teamScopedDelegations(db *gorm.DB, userID uint) *gorm.DB {
	teamIDs := userTeamIDs(db, userID)
	if len(teamIDs) == 0 {
		return db.Where("user_id = ?", userID)
	}
	return db.Where("user_id = ? OR team_id IN ?", userID, teamIDs)
}
