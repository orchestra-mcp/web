package handlers

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// HookEventHandler bridges MCP hook events to WebSocket clients.
type HookEventHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewHookEventHandler creates a new HookEventHandler.
func NewHookEventHandler(db *gorm.DB, h *hub.Hub) *HookEventHandler {
	return &HookEventHandler{db: db, hub: h}
}

// Receive handles POST /api/hooks/events
// Accepts MCP hook events from the desktop app or CLI bridge,
// persists them to the mcp_event_logs table, and broadcasts
// to the user's connected WebSocket clients.
func (h *HookEventHandler) Receive(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		EventType string          `json:"event_type"`
		SessionID string          `json:"session_id"`
		ToolName  string          `json:"tool_name"`
		AgentType string          `json:"agent_type"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.EventType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "event_type is required"})
	}

	// Persist to mcp_event_logs
	log := models.MCPEventLog{
		UserID:    user.ID,
		EventType: body.EventType,
		SessionID: body.SessionID,
		ToolName:  body.ToolName,
		AgentType: body.AgentType,
		Data:      datatypes.JSON(body.Data),
	}
	h.db.Create(&log)

	// Map hook event types to WS event actions
	action := mapEventAction(body.EventType)

	// Broadcast to the user's connected clients
	if h.hub != nil {
		h.hub.BroadcastToUser(user.ID, hub.Event{
			Type:       "mcp",
			EntityType: entityTypeFromEvent(body.EventType),
			EntityID:   body.SessionID,
			Action:     action,
			UserID:     user.ID,
			Timestamp:  time.Now().Unix(),
			ToolName:   body.ToolName,
			SessionID:  body.SessionID,
			AgentType:  body.AgentType,
		})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// List handles GET /api/hooks/events — returns recent MCP event logs.
func (h *HookEventHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var logs []models.MCPEventLog
	h.db.Where("user_id = ?", user.ID).
		Order("created_at desc").
		Limit(100).
		Find(&logs)

	return c.JSON(fiber.Map{"events": logs, "count": len(logs)})
}

// mapEventAction converts Claude Code hook event names to WS action strings.
func mapEventAction(eventType string) string {
	switch eventType {
	case "tool_use_start", "tool_use_end":
		return "tool_called"
	case "agent_tool_use_start", "agent_tool_use_end":
		return "tool_called"
	case "notification":
		return "notification"
	case "subagent_start", "subagent_end":
		return "agent_spawned"
	default:
		return eventType
	}
}

// entityTypeFromEvent derives an entity type from the hook event name.
func entityTypeFromEvent(eventType string) string {
	switch eventType {
	case "tool_use_start", "tool_use_end", "agent_tool_use_start", "agent_tool_use_end":
		return "tool"
	case "notification":
		return "notification"
	case "subagent_start", "subagent_end":
		return "agent"
	default:
		return "mcp_event"
	}
}
