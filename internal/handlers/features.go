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

// FeatureHandler handles feature CRUD endpoints.
type FeatureHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewFeatureHandler creates a new FeatureHandler.
func NewFeatureHandler(db *gorm.DB, wsHub ...*hub.Hub) *FeatureHandler {
	h := &FeatureHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// featureRow is the frontend-compatible shape for a feature.
type featureRow struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Status     string    `json:"status"`
	Priority   string    `json:"priority"`
	Kind       string    `json:"kind"`
	AssigneeID *uint     `json:"assignee_id"`
	Labels     []string  `json:"labels"`
	Estimate   string    `json:"estimate"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func toFeatureRow(f models.Feature) featureRow {
	var labels []string
	_ = json.Unmarshal(f.Labels, &labels)
	if labels == nil {
		labels = []string{}
	}

	// Use feature_id (e.g. FEAT-XXX) as the canonical id so MCP tools work.
	// Fall back to GORM UUID only if feature_id is empty.
	id := f.FeatureID
	if id == "" {
		id = f.ID
	}
	row := featureRow{
		ID:        id,
		ProjectID: f.ProjectSlug,
		Title:     f.Title,
		Body:      f.Body,
		Status:    f.Status,
		Priority:  f.Priority,
		Kind:      "feature",
		Labels:    labels,
		Estimate:  f.Estimate,
		Version:   f.Version,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}

	if f.Assignee != "" {
		var uid uint
		if _, err := fmt.Sscanf(f.Assignee, "%d", &uid); err == nil {
			row.AssigneeID = &uid
		}
	}

	return row
}

// List returns features for a project.
// GET /api/projects/:slug/features
func (h *FeatureHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := resolveProjectSlug(h.db, c.Params("slug"))

	query := teamScopedFeatures(h.db, user.ID).
		Where("project_slug = ?", slug).
		Order("created_at DESC")

	// Optional filters
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if assigneeID := c.Query("assignee_id"); assigneeID != "" {
		query = query.Where("assignee = ?", assigneeID)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}

	var features []models.Feature
	if err := query.Find(&features).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list features"})
	}

	rows := make([]featureRow, 0, len(features))
	for _, f := range features {
		rows = append(rows, toFeatureRow(f))
	}

	return c.JSON(fiber.Map{"features": rows})
}

// Show returns a single feature.
// GET /api/projects/:slug/features/:id
func (h *FeatureHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := resolveProjectSlug(h.db, c.Params("slug"))
	id := c.Params("id")

	var feature models.Feature
	if err := teamScopedFeatures(h.db, user.ID).
		Where("project_slug = ? AND feature_id = ?", slug, id).
		First(&feature).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	return c.JSON(fiber.Map{"feature": toFeatureRow(feature)})
}

// Update updates a feature.
// PUT /api/projects/:slug/features/:id
func (h *FeatureHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := resolveProjectSlug(h.db, c.Params("slug"))
	id := c.Params("id")

	var feature models.Feature
	if err := teamScopedFeatures(h.db, user.ID).
		Where("project_slug = ? AND feature_id = ?", slug, id).
		First(&feature).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if v, ok := body["title"].(string); ok {
		updates["title"] = v
	}
	if v, ok := body["description"].(string); ok {
		updates["description"] = v
	}
	if v, ok := body["status"].(string); ok {
		updates["status"] = v
	}
	if v, ok := body["priority"].(string); ok {
		updates["priority"] = v
	}
	if v, ok := body["assignee"].(string); ok {
		updates["assignee"] = v
	}
	// assignee_id (number) → stored as string in the assignee field
	if v, ok := body["assignee_id"]; ok {
		switch val := v.(type) {
		case float64:
			updates["assignee"] = fmt.Sprintf("%d", int(val))
		case string:
			updates["assignee"] = val
		}
	}
	if v, ok := body["estimate"].(string); ok {
		updates["estimate"] = v
	}
	if v, ok := body["body"].(string); ok {
		updates["body"] = v
	}

	if err := h.db.Model(&feature).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update feature"})
	}

	h.db.First(&feature, "id = ?", feature.ID)
	broadcastSync(h.hub, user.ID, "feature", feature.FeatureID, "upsert")
	return c.JSON(fiber.Map{"feature": toFeatureRow(feature)})
}

// Delete deletes a feature.
// DELETE /api/projects/:slug/features/:id
func (h *FeatureHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := resolveProjectSlug(h.db, c.Params("slug"))
	id := c.Params("id")

	result := teamScopedFeatures(h.db, user.ID).
		Where("project_slug = ? AND feature_id = ?", slug, id).
		Delete(&models.Feature{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete feature"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	broadcastSync(h.hub, user.ID, "feature", id, "delete")
	return c.JSON(fiber.Map{"message": "feature deleted"})
}
