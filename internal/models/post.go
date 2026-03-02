package models

import (
	"time"

	"gorm.io/gorm"
)

// Post is a blog post.
type Post struct {
	gorm.Model
	Title       string     `json:"title"`
	Slug        string     `gorm:"uniqueIndex" json:"slug"`
	Content     string     `gorm:"type:text" json:"content"`
	Excerpt     string     `json:"excerpt"`
	Status      string     `gorm:"default:draft" json:"status"` // draft | published
	PublishedAt *time.Time `json:"published_at"`
	UserID      uint       `json:"user_id"`
}
