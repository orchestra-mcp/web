package models

import (
	"time"

	"gorm.io/gorm"
)

// Comment is a user comment on a blog post.
type Comment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	PostSlug  string         `gorm:"index" json:"post_slug"`
	UserID    uint           `json:"user_id"`
	UserName  string         `json:"user_name"`
	AvatarURL string         `json:"avatar_url"`
	Body      string         `gorm:"type:text" json:"body"`
}
