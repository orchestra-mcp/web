package models

import (
	"time"

	"gorm.io/gorm"
)

// Base provides UUID primary key and standard timestamps for models.
type Base struct {
	ID        string         `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
