package models

import (
	"time"

	"gorm.io/gorm"
)

// ContactMessage stores a contact form submission.
type ContactMessage struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Subject   string         `json:"subject"`
	Message   string         `gorm:"type:text" json:"message"`
	Status    string         `gorm:"default:new" json:"status"` // new | read | replied
}
