package models

import "gorm.io/datatypes"

// Feature represents a project feature (used by MCP sync).
type Feature struct {
	Base
	UserID      uint           `gorm:"uniqueIndex:idx_feature_user_fid" json:"user_id"`
	TeamID      *string        `json:"team_id"`
	ProjectSlug string         `gorm:"index" json:"project_slug"`
	FeatureID   string         `gorm:"uniqueIndex:idx_feature_user_fid" json:"feature_id"` // e.g. FEAT-ABC
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `gorm:"default:backlog" json:"status"`
	Priority    string         `gorm:"default:P2" json:"priority"`
	Assignee    string         `json:"assignee"`
	Labels      datatypes.JSON `json:"labels"`
	DependsOn   datatypes.JSON `json:"depends_on"`
	Blocks      datatypes.JSON `json:"blocks"`
	Estimate    string         `json:"estimate"`
	Body        string         `gorm:"type:text" json:"body"`
	Version     int            `gorm:"default:1" json:"version"`
	Meta        datatypes.JSON `json:"meta"`
}
