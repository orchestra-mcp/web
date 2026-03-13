package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Post is a blog post.
type Post struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Title        string         `json:"title"`
	Slug         string         `gorm:"uniqueIndex" json:"slug"`
	Content      string         `gorm:"type:text" json:"content"`
	Excerpt      string         `json:"excerpt"`
	Status       string         `gorm:"default:draft" json:"status"` // draft | published
	PublishedAt  *time.Time     `json:"published_at"`
	UserID       uint           `json:"user_id"`
	Translations datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"translations"`
}
