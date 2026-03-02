package models

import "gorm.io/gorm"

// Page is a CMS static page.
type Page struct {
	gorm.Model
	Title   string `json:"title"`
	Slug    string `gorm:"uniqueIndex" json:"slug"`
	Content string `gorm:"type:text" json:"content"`
	Status  string `gorm:"default:draft" json:"status"` // draft | published
	UserID  uint   `json:"user_id"`
}
