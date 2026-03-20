package handlers

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// PlanHandler handles project-scoped plan endpoints.
type PlanHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewPlanHandler creates a new PlanHandler.
func NewPlanHandler(db *gorm.DB, wsHub ...*hub.Hub) *PlanHandler {
	h := &PlanHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// planRow is the frontend-compatible shape for a plan.
type planRow struct {
	ID           string   `json:"id"`
	ProjectID    string   `json:"project_id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Body         string   `json:"body"`
	FeatureCount int      `json:"featureCount"`
	FeatureIDs   []string `json:"featureIds"`
	Version      int      `json:"version"`
}

func (h *PlanHandler) toPlanRow(p models.Plan) planRow {
	// Count features that have a label matching plan:{plan_id}
	planLabel := "plan:" + p.PlanID
	var featureCount int64
	h.db.Model(&models.Feature{}).
		Where("project_slug = ? AND labels::text LIKE ?", p.ProjectSlug, "%"+planLabel+"%").
		Count(&featureCount)

	// Get feature IDs
	var featureIDs []string
	if featureCount > 0 {
		var features []models.Feature
		h.db.Select("feature_id").
			Where("project_slug = ? AND labels::text LIKE ?", p.ProjectSlug, "%"+planLabel+"%").
			Find(&features)
		for _, f := range features {
			featureIDs = append(featureIDs, f.FeatureID)
		}
	}
	if featureIDs == nil {
		featureIDs = []string{}
	}

	return planRow{
		ID:           p.PlanID,
		ProjectID:    p.ProjectSlug,
		Title:        p.Title,
		Description:  p.Description,
		Status:       p.Status,
		Body:         p.Body,
		FeatureCount: int(featureCount),
		FeatureIDs:   featureIDs,
		Version:      p.Version,
	}
}

// List handles GET /api/projects/:slug/plans
func (h *PlanHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := resolveProjectSlug(h.db, c.Params("slug"))

	query := teamScopedPlans(h.db, user.ID).
		Where("project_slug = ?", slug).
		Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var plans []models.Plan
	if err := query.Find(&plans).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list plans"})
	}

	rows := make([]planRow, 0, len(plans))
	for _, p := range plans {
		rows = append(rows, h.toPlanRow(p))
	}

	return c.JSON(fiber.Map{
		"items": rows,
		"meta": fiber.Map{
			"total":  len(rows),
			"limit":  100,
			"offset": 0,
		},
	})
}

// Show handles GET /api/projects/:slug/plans/:planId
func (h *PlanHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slug := resolveProjectSlug(h.db, c.Params("slug"))
	planID := c.Params("planId")

	var plan models.Plan
	if err := teamScopedPlans(h.db, user.ID).
		Where("project_slug = ? AND plan_id = ?", slug, planID).
		First(&plan).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "plan not found"})
	}

	return c.JSON(h.toPlanRow(plan))
}

// teamScopedPlans returns a query scoped to user's own + team plans.
func teamScopedPlans(db *gorm.DB, userID uint) *gorm.DB {
	teamIDs := userTeamIDs(db, userID)
	if len(teamIDs) == 0 {
		return db.Where("user_id = ?", userID)
	}
	return db.Where("user_id = ? OR team_id IN ?", userID, teamIDs)
}

// listUserSettings extracts a string field from the user's Settings JSONB.
func listUserSettings(settings json.RawMessage, key string) string {
	var s map[string]interface{}
	if err := json.Unmarshal(settings, &s); err != nil {
		return ""
	}
	v, _ := s[key].(string)
	return v
}

// settingsBool extracts a bool field from the user's Settings JSONB.
func settingsBool(settings json.RawMessage, key string) bool {
	var s map[string]interface{}
	if err := json.Unmarshal(settings, &s); err != nil {
		return false
	}
	v, _ := s[key].(bool)
	return v
}

// settingsMap extracts a map field from the user's Settings JSONB.
func settingsMap(settings json.RawMessage, key string) map[string]interface{} {
	var s map[string]interface{}
	if err := json.Unmarshal(settings, &s); err != nil {
		return nil
	}
	v, _ := s[key].(map[string]interface{})
	return v
}

// settingsRaw extracts a raw interface{} field from user Settings JSONB (for arrays, nested objects, etc).
func settingsRaw(settings json.RawMessage, key string) interface{} {
	var s map[string]interface{}
	if err := json.Unmarshal(settings, &s); err != nil {
		return nil
	}
	return s[key]
}

// settingsBoolDefault extracts a bool field from user Settings JSONB with a default value.
// Handles both JSON boolean (true/false) and string ("true"/"false") representations.
func settingsBoolDefault(settings json.RawMessage, key string, defaultVal bool) bool {
	var s map[string]interface{}
	if err := json.Unmarshal(settings, &s); err != nil {
		return defaultVal
	}
	v, ok := s[key]
	if !ok {
		return defaultVal
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true"
	default:
		return defaultVal
	}
}

// settingsHandle extracts the "handle" from settings, falling back to a slug of the user's name.
func settingsHandle(u models.User) string {
	h := listUserSettings(json.RawMessage(u.Settings), "handle")
	if h != "" {
		return h
	}
	// Fallback: slugified name
	lower := strings.ToLower(u.Name)
	lower = strings.ReplaceAll(lower, " ", "-")
	return lower
}
