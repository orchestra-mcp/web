package models

import "gorm.io/datatypes"

// Request represents a queued user request (used by MCP sync).
type Request struct {
	Base
	UserID      uint           `gorm:"uniqueIndex:idx_request_user_rid" json:"user_id"`
	TeamID      *string        `json:"team_id"`
	ProjectSlug string         `gorm:"index" json:"project_slug"`
	RequestID   string         `gorm:"uniqueIndex:idx_request_user_rid" json:"request_id"` // e.g. REQ-ABC
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Kind        string         `gorm:"default:feature" json:"kind"`
	Priority    string         `gorm:"default:P2" json:"priority"`
	Status      string         `gorm:"default:pending" json:"status"`
	Body        string         `gorm:"type:text" json:"body"`
	Version     int            `gorm:"default:1" json:"version"`
	Meta        datatypes.JSON `json:"meta"`
}
