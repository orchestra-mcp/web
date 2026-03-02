package models

import "time"

// OtpCode stores a one-time password for login, verification, or password reset.
type OtpCode struct {
	ID        uint       `gorm:"primarykey" json:"id"`
	UserID    uint       `gorm:"index" json:"user_id"`
	Email     string     `gorm:"index" json:"email"`
	Code      string     `json:"code"`
	Type      string     `json:"type"` // login | email_verification | password_reset
	UsedAt    *time.Time `json:"used_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	User      User       `gorm:"foreignKey:UserID" json:"-"`
}
