package models

import (
	"time"

	"gorm.io/datatypes"
)

// ConflictLog records sync conflicts for manual resolution.
type ConflictLog struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	UserID       uint           `gorm:"index" json:"user_id"`
	EntityType   string         `json:"entity_type"`
	EntityID     string         `json:"entity_id"`
	LocalPayload datatypes.JSON `json:"local_payload"`
	RemotePayload datatypes.JSON `json:"remote_payload"`
	Resolution   string         `gorm:"default:pending" json:"resolution"` // pending | local_wins | remote_wins | merged
	ResolvedAt   *time.Time     `json:"resolved_at"`
	CreatedAt    time.Time      `json:"created_at"`
}
