package models

import (
	"time"

	"gorm.io/datatypes"
)

// ActionHistory records a smart action dispatched through a tunnel.
type ActionHistory struct {
	ID         string         `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID     uint           `gorm:"index;not null" json:"user_id"`
	TunnelID   string         `gorm:"index;not null" json:"tunnel_id"`
	ActionType string         `gorm:"not null" json:"action_type"` // run_prompt, run_command, sync_workspace, run_tests, get_status
	ToolName   string         `json:"tool_name"`
	Params     datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"params"`
	Success    bool           `json:"success"`
	DurationMs int64          `json:"duration_ms"`
	Error      string         `json:"error,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}
