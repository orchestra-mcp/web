package models

import (
	"time"

	"gorm.io/gorm"
)

// PushSubscription stores a browser/device web push subscription for a user.
type PushSubscription struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UserID    uint           `gorm:"index" json:"user_id"`
	Endpoint  string         `gorm:"uniqueIndex" json:"endpoint"`
	P256dh    string         `json:"p256dh"`
	Auth      string         `json:"auth"`
	Platform  string         `json:"platform"` // web, android, ios
	UserAgent string         `json:"user_agent"`
}
