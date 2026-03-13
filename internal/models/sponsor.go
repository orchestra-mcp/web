package models

import (
	"time"

	"gorm.io/gorm"
)

// Sponsor represents a sponsor displayed on the site.
type Sponsor struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Name        string         `json:"name"`
	Slug        string         `gorm:"uniqueIndex" json:"slug"`
	LogoURL     string         `json:"logo_url"`
	WebsiteURL  string         `json:"website_url"`
	Tier        string         `gorm:"default:silver" json:"tier"`         // platinum | gold | silver | bronze
	Description string         `gorm:"type:text" json:"description"`
	Order       int            `gorm:"default:0" json:"order"`
	Status      string         `gorm:"default:active" json:"status"`       // active | inactive
}
