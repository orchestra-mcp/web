package models

import (
	"time"

	"gorm.io/gorm"
)

// CommunityPost is a user-generated community post.
type CommunityPost struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UserID        uint           `json:"user_id"`
	Title         string         `json:"title"`
	Content       string         `gorm:"type:text" json:"content"`
	Icon          string         `gorm:"default:''" json:"icon"`           // boxicons class e.g. "bx-notepad"
	Color         string         `gorm:"default:''" json:"color"`          // hex color e.g. "#a900ff"
	Media         string         `gorm:"type:text;default:''" json:"media"` // JSON array of media items
	Tags          string         `gorm:"type:text;default:'[]'" json:"tags"` // JSON array of tag strings
	Status        string         `gorm:"default:published" json:"status"`  // published | draft
	LikesCount    int            `gorm:"default:0" json:"likes_count"`
	CommentsCount int            `gorm:"default:0" json:"comments_count"`
	ParentID      *uint          `gorm:"index" json:"parent_id,omitempty"`

	User User `gorm:"-:migration" json:"-"`
}
