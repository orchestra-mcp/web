package handlers

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// FileEventHandler broadcasts file change events from tunnels to team members.
type FileEventHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewFileEventHandler creates a new FileEventHandler.
func NewFileEventHandler(db *gorm.DB, h *hub.Hub) *FileEventHandler {
	return &FileEventHandler{db: db, hub: h}
}

// fileChangeEntry represents a single file change event from the CLI.
type fileChangeEntry struct {
	Path   string `json:"path"`   // relative file path
	Action string `json:"action"` // created, modified, deleted
}

// Report handles POST /api/tunnels/:id/file-events — receives file change
// events from a local CLI (via tunnel heartbeat or direct push) and broadcasts
// them to all online team members.
func (h *FileEventHandler) Report(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	tunnelID := c.Params("id")

	// Verify tunnel ownership.
	var tunnel models.Tunnel
	if err := h.db.Where("id = ? AND user_id = ?", tunnelID, user.ID).
		First(&tunnel).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tunnel not found"})
	}

	var body struct {
		Events []fileChangeEntry `json:"events"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if len(body.Events) == 0 {
		return c.JSON(fiber.Map{"ok": true, "broadcasted": 0})
	}

	// Get team member IDs for broadcast.
	var memberIDs []uint
	if tunnel.TeamID != nil && *tunnel.TeamID != "" {
		h.db.Model(&models.Membership{}).
			Where("team_id = ?", *tunnel.TeamID).
			Pluck("user_id", &memberIDs)
	}
	if len(memberIDs) == 0 {
		memberIDs = []uint{user.ID}
	}

	now := time.Now().Unix()
	for _, entry := range body.Events {
		event := hub.Event{
			Type:       "file_change",
			EntityType: "file",
			EntityID:   entry.Path,
			Action:     entry.Action,
			UserID:     user.ID,
			Timestamp:  now,
			FilePath:   entry.Path,
			TunnelID:   tunnelID,
			Workspace:  tunnel.Workspace,
		}
		h.hub.BroadcastToUsers(memberIDs, event)
	}

	return c.JSON(fiber.Map{"ok": true, "broadcasted": len(body.Events)})
}
