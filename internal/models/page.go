package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Page is a CMS static page.
type Page struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Title        string         `json:"title"`
	Slug         string         `gorm:"uniqueIndex" json:"slug"`
	Content      string         `gorm:"type:text" json:"content"`
	Status       string         `gorm:"default:draft" json:"status"` // draft | published
	UserID       uint           `json:"user_id"`
	Translations datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"translations"`
}
