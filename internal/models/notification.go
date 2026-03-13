package models

import (
	"time"

	"gorm.io/gorm"
)

// Notification is an in-app notification for a user.
type Notification struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UserID    uint           `json:"user_id"`
	Title   string     `json:"title"`
	Message string     `json:"message"`
	Type    string     `gorm:"default:info" json:"type"` // info | warning | success | error
	ReadAt  *time.Time `json:"read_at"`
}
