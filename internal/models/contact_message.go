package models

import "gorm.io/gorm"

// ContactMessage stores a contact form submission.
type ContactMessage struct {
	gorm.Model
	Name    string `json:"name"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Message string `gorm:"type:text" json:"message"`
	Status  string `gorm:"default:new" json:"status"` // new | read | replied
}
