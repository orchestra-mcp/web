package models

import "time"

// MagicLinkToken stores a single-use token for passwordless login.
type MagicLinkToken struct {
	ID        uint       `gorm:"primarykey" json:"id"`
	UserID    uint       `gorm:"index" json:"user_id"`
	Email     string     `gorm:"index" json:"email"`
	Token     string     `gorm:"uniqueIndex" json:"token"`
	UsedAt    *time.Time `json:"used_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
}
