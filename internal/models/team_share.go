package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TeamShare tracks entity sharing with teams.
type TeamShare struct {
	ID           string         `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID       uint           `json:"user_id"`
	TeamID       string         `gorm:"type:uuid;index" json:"team_id"`
	EntityType   string         `gorm:"index" json:"entity_type"`
	EntityID     string         `gorm:"index" json:"entity_id"`
	ShareWithAll bool           `gorm:"default:true" json:"share_with_all"`
	MemberIDs    datatypes.JSON `json:"member_ids"`
	Permission   string         `gorm:"default:read" json:"permission"`
	ContentHash  string         `json:"content_hash"`
	EntityData   datatypes.JSON `json:"entity_data"`
	Version      int            `gorm:"default:1" json:"version"`
	SharedAt     time.Time      `json:"shared_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
