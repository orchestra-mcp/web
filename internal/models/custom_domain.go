package models

import "time"

// CustomDomain maps a user-owned domain to their public content profile.
type CustomDomain struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	UserID       uint       `gorm:"index" json:"user_id"`
	Domain       string     `gorm:"uniqueIndex" json:"domain"`
	Verified     bool       `gorm:"default:false" json:"verified"`
	DNSTxtRecord string     `json:"dns_txt_record"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
