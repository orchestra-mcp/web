package models

import "gorm.io/datatypes"

// AssignmentRule represents an auto-assignment rule (used by MCP sync).
type AssignmentRule struct {
	Base
	UserID      uint           `gorm:"uniqueIndex:idx_arule_user_rid" json:"user_id"`
	TeamID      *string        `json:"team_id"`
	ProjectSlug string         `gorm:"index" json:"project_slug"`
	RuleID      string         `gorm:"uniqueIndex:idx_arule_user_rid" json:"rule_id"` // e.g. RULE-ABC
	Kind        string         `json:"kind"`
	PersonID    string         `json:"person_id"`
	Body        string         `gorm:"type:text" json:"body"`
	Version     int            `gorm:"default:1" json:"version"`
	Meta        datatypes.JSON `json:"meta"`
}
