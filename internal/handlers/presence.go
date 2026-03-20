package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// PresenceHandler provides team presence endpoints.
type PresenceHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewPresenceHandler creates a new PresenceHandler.
func NewPresenceHandler(db *gorm.DB, h *hub.Hub) *PresenceHandler {
	return &PresenceHandler{db: db, hub: h}
}

// TeamPresence handles GET /api/teams/:id/presence.
// Returns which team members are currently online (connected via WebSocket).
func (h *PresenceHandler) TeamPresence(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	teamID := c.Params("id")

	// Verify user is a member of the team.
	var membership models.Membership
	if err := h.db.Where("team_id = ? AND user_id = ?", teamID, user.ID).
		First(&membership).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a team member"})
	}

	// Get all team members.
	var members []models.Membership
	h.db.Where("team_id = ?", teamID).Preload("User").Find(&members)

	onlineIDs := h.hub.OnlineUserIDs()
	onlineSet := make(map[uint]bool, len(onlineIDs))
	for _, id := range onlineIDs {
		onlineSet[id] = true
	}

	type memberPresence struct {
		UserID uint   `json:"user_id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Avatar string `json:"avatar_url"`
		Online bool   `json:"online"`
	}

	var result []memberPresence
	for _, m := range members {
		result = append(result, memberPresence{
			UserID: m.UserID,
			Name:   m.User.Name,
			Email:  m.User.Email,
			Avatar: m.User.AvatarURL,
			Online: onlineSet[m.UserID],
		})
	}

	return c.JSON(fiber.Map{
		"team_id": teamID,
		"members": result,
	})
}
