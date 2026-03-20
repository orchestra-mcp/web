package models

import (
	"time"

	"gorm.io/datatypes"
)

// Project represents a user's or team's project, scoped to a workspace.
type Project struct {
	Base
	UserID       uint           `gorm:"uniqueIndex:idx_project_slug_user" json:"user_id"`
	TeamID       *string        `gorm:"type:uuid" json:"team_id"`
	WorkspaceID  *string        `gorm:"type:uuid" json:"workspace_id"`
	Name         string         `json:"name"`
	Slug         string         `gorm:"uniqueIndex:idx_project_slug_user" json:"slug"`
	Description  string         `json:"description"`
	IsPublic     bool           `gorm:"default:false" json:"is_public"`
	SyncStatus   string         `gorm:"default:not_synced" json:"sync_status"`
	LastSyncedAt *time.Time     `json:"last_synced_at"`
	Stats        datatypes.JSON `json:"stats"`
	Meta         datatypes.JSON `json:"meta"`
	Version      int            `gorm:"default:1" json:"version"`

	User      User       `gorm:"foreignKey:UserID" json:"-"`
	Team      *Team      `gorm:"foreignKey:TeamID" json:"-"`
	Workspace *Workspace `gorm:"foreignKey:WorkspaceID" json:"-"`
}
