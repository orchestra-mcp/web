package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// AdminUserGamificationHandler handles badge/points operations on users.
type AdminUserGamificationHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewAdminUserGamificationHandler creates a new handler.
func NewAdminUserGamificationHandler(db *gorm.DB, wsHub ...*hub.Hub) *AdminUserGamificationHandler {
	h := &AdminUserGamificationHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// ListUserBadges returns badges awarded to a user.
// GET /api/admin/users/:id/badges
func (h *AdminUserGamificationHandler) ListUserBadges(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	userID := c.Params("id")

	var userBadges []models.UserBadge
	h.db.Where("user_id = ?", userID).Find(&userBadges)

	type row struct {
		ID                string `json:"id"`
		BadgeDefinitionID string `json:"badge_definition_id"`
		BadgeName         string `json:"badge_name"`
		BadgeIcon         string `json:"badge_icon"`
		BadgeColor        string `json:"badge_color"`
		BadgeCategory     string `json:"badge_category"`
		AwardedAt         string `json:"awarded_at"`
	}
	rows := make([]row, 0, len(userBadges))
	for _, ub := range userBadges {
		var bd models.BadgeDefinition
		h.db.Where("id = ?", ub.BadgeDefinitionID).First(&bd)
		rows = append(rows, row{
			ID: ub.ID, BadgeDefinitionID: ub.BadgeDefinitionID,
			BadgeName: bd.Name, BadgeIcon: bd.Icon, BadgeColor: bd.Color,
			BadgeCategory: bd.Category, AwardedAt: ub.CreatedAt.Format("2006-01-02"),
		})
	}
	return c.JSON(fiber.Map{"badges": rows})
}

// AwardBadge awards a badge to a user.
// POST /api/admin/users/:id/badges
func (h *AdminUserGamificationHandler) AwardBadge(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	userID := c.Params("id")
	admin := middleware.CurrentUser(c)

	var body struct {
		BadgeDefinitionID string `json:"badge_definition_id"`
		Note              string `json:"note"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.BadgeDefinitionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "badge_definition_id required"})
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	adminID := ""
	if admin != nil {
		adminID = strconv.FormatUint(uint64(admin.ID), 10)
	}

	ub := models.UserBadge{
		UserID:            user.ID,
		BadgeDefinitionID: body.BadgeDefinitionID,
		AwardedBy:         adminID,
		Note:              body.Note,
	}
	if err := h.db.Create(&ub).Error; err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "badge already awarded"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user_badge": ub})
}

// RevokeBadge removes a badge from a user.
// DELETE /api/admin/users/:id/badges/:badge_id
func (h *AdminUserGamificationHandler) RevokeBadge(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	badgeID := c.Params("badge_id")
	if err := h.db.Delete(&models.UserBadge{}, badgeID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(fiber.Map{"message": "revoked"})
}

// AddPoints adds points to a user's settings.
// POST /api/admin/users/:id/points
func (h *AdminUserGamificationHandler) AddPoints(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	userID := c.Params("id")

	var body struct {
		Amount int    `json:"amount"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.Amount == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount required (positive to add, negative to deduct)"})
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	// Read current points from settings
	var settings map[string]interface{}
	if err := json.Unmarshal(user.Settings, &settings); err != nil {
		settings = map[string]interface{}{}
	}

	currentPoints := 0
	if p, ok := settings["points"]; ok {
		if pn, ok := p.(float64); ok {
			currentPoints = int(pn)
		}
	}

	newPoints := currentPoints + body.Amount
	if newPoints < 0 {
		newPoints = 0
	}
	settings["points"] = newPoints

	settingsJSON, _ := json.Marshal(settings)
	h.db.Model(&user).Update("settings", settingsJSON)

	// Auto-award badges based on new points threshold
	awarded := CheckAndAwardBadges(h.db, user.ID, newPoints, h.hub)
	awardedNames := make([]string, 0, len(awarded))
	for _, b := range awarded {
		awardedNames = append(awardedNames, b.Name)
	}

	return c.JSON(fiber.Map{
		"points":         newPoints,
		"previous":       currentPoints,
		"change":         body.Amount,
		"reason":         body.Reason,
		"badges_awarded": awardedNames,
	})
}

// GetPoints returns the user's current point balance.
// GET /api/admin/users/:id/points
func (h *AdminUserGamificationHandler) GetPoints(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	userID := c.Params("id")

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(user.Settings, &settings); err != nil {
		settings = map[string]interface{}{}
	}

	points := 0
	if p, ok := settings["points"]; ok {
		if pn, ok := p.(float64); ok {
			points = int(pn)
		}
	}

	return c.JSON(fiber.Map{"points": points})
}
