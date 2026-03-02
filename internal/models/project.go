package models

import (
	"time"

	"gorm.io/datatypes"
)

// Project represents a user's or team's project.
type Project struct {
	Base
	UserID       uint           `gorm:"uniqueIndex:idx_project_slug_user" json:"user_id"`
	TeamID       *string        `json:"team_id"`
	Name         string         `json:"name"`
	Slug         string         `gorm:"uniqueIndex:idx_project_slug_user" json:"slug"`
	Description  string         `json:"description"`
	SyncStatus   string         `gorm:"default:not_synced" json:"sync_status"`
	LastSyncedAt *time.Time     `json:"last_synced_at"`
	Stats        datatypes.JSON `json:"stats"`
	Meta         datatypes.JSON `json:"meta"`
	Version      int            `gorm:"default:1" json:"version"`
	User         User           `gorm:"foreignKey:UserID" json:"-"`
}
