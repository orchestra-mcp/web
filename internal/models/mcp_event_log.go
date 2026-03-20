package models

import (
	"time"

	"gorm.io/datatypes"
)

// MCPEventLog stores MCP hook events for audit and replay.
type MCPEventLog struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UserID    uint           `gorm:"index" json:"user_id"`
	EventType string         `gorm:"index" json:"event_type"`
	SessionID string         `json:"session_id"`
	ToolName  string         `json:"tool_name"`
	AgentType string         `json:"agent_type"`
	Data      datatypes.JSON `gorm:"type:json" json:"data"`
}
