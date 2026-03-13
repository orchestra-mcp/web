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
	User          User           `gorm:"foreignKey:UserID" json:"-"`
	Title         string         `json:"title"`
	Content       string         `gorm:"type:text" json:"content"`
	Status        string         `gorm:"default:published" json:"status"` // published | draft
	LikesCount    int            `gorm:"default:0" json:"likes_count"`
	CommentsCount int            `gorm:"default:0" json:"comments_count"`
}
