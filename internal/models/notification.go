package models

import (
	"time"

	"gorm.io/gorm"
)

// Notification is an in-app notification for a user.
type Notification struct {
	gorm.Model
	UserID  uint       `json:"user_id"`
	Title   string     `json:"title"`
	Message string     `json:"message"`
	Type    string     `gorm:"default:info" json:"type"` // info | warning | success | error
	ReadAt  *time.Time `json:"read_at"`
}
