package models

import (
	"time"

	"gorm.io/datatypes"
)

// SystemSetting stores key-value admin configuration as JSONB.
type SystemSetting struct {
	Key       string         `gorm:"primaryKey" json:"key"`
	Value     datatypes.JSON `gorm:"type:jsonb" json:"value"`
	UpdatedAt time.Time      `json:"updated_at"`
}
