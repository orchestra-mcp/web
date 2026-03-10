package models

import "gorm.io/datatypes"

// Plan represents a project plan (used by MCP sync).
type Plan struct {
	Base
	UserID      uint           `gorm:"uniqueIndex:idx_plan_user_pid" json:"user_id"`
	TeamID      *string        `json:"team_id"`
	ProjectSlug string         `gorm:"index" json:"project_slug"`
	PlanID      string         `gorm:"uniqueIndex:idx_plan_user_pid" json:"plan_id"` // e.g. PLAN-ABC
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `gorm:"default:draft" json:"status"`
	Body        string         `gorm:"type:text" json:"body"`
	Version     int            `gorm:"default:1" json:"version"`
	Meta        datatypes.JSON `json:"meta"`
}
