package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// User represents a registered user of the platform.
type User struct {
	ID                    uint           `gorm:"primarykey" json:"id"`
	Name                  string         `json:"name"`
	Email                 string         `gorm:"uniqueIndex" json:"email"`
	Password              string         `json:"-"`
	Role                  string         `gorm:"default:user" json:"role"` // admin | team_owner | team_manager | user
	AvatarURL             string         `json:"avatar_url"`
	EmailVerifiedAt       *time.Time     `json:"email_verified_at"`
	Status                string         `gorm:"default:active" json:"status"`
	PasswordSet           bool           `gorm:"default:false" json:"password_set"`
	OnboardingCompletedAt *time.Time     `json:"onboarding_completed_at"`
	Settings              datatypes.JSON `json:"settings"`
	RememberToken         string         `json:"-"`
	gorm.Model
}
