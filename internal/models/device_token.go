package models

import "time"

// DeviceToken registers a client device for sync tracking.
type DeviceToken struct {
	Base
	UserID     uint       `json:"user_id"`
	DeviceID   string     `gorm:"uniqueIndex:idx_device_user" json:"device_id"`
	Name       string     `json:"name"`
	Platform   string     `json:"platform"` // ios | android | web | desktop | extension
	LastSeenAt *time.Time `json:"last_seen_at"`
	User       User       `gorm:"foreignKey:UserID" json:"-"`
}
