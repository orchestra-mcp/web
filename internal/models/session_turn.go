package models

import "gorm.io/datatypes"

// SessionTurn represents a single turn in an AI chat session (used by MCP sync).
type SessionTurn struct {
	Base
	UserID    uint           `json:"user_id"`
	SessionID string         `gorm:"index" json:"session_id"` // parent AI session UUID
	TurnID    string         `gorm:"uniqueIndex:idx_turn_user_tid" json:"turn_id"` // e.g. turn-001
	Body      string         `gorm:"type:text" json:"body"`
	Version   int            `gorm:"default:1" json:"version"`
	Meta      datatypes.JSON `json:"meta"`
}
