package models

import (
	"time"

	"gorm.io/datatypes"
)

// SyncLog is an append-only log of all entity changes used for sync.
type SyncLog struct {
	ID             string         `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID         uint           `json:"user_id"`
	DeviceID       string         `json:"device_id"`
	EntityType     string         `json:"entity_type"`
	EntityID       string         `json:"entity_id"`
	Action         string         `json:"action"` // upsert | delete
	Payload        datatypes.JSON `json:"payload"`
	Version        int64          `json:"version"`
	IdempotencyKey *string        `gorm:"uniqueIndex:idx_sync_idempotency,where:idempotency_key IS NOT NULL" json:"idempotency_key"`
	TeamID         *string        `gorm:"type:uuid" json:"team_id"`
	CreatedAt      time.Time      `json:"created_at"`
}
