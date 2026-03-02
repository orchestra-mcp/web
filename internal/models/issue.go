package models

import "gorm.io/gorm"

// Issue is a user-reported issue or bug report.
type Issue struct {
	gorm.Model
	UserID      uint   `json:"user_id"`
	Title       string `json:"title"`
	Description string `gorm:"type:text" json:"description"`
	Status      string `gorm:"default:open" json:"status"`    // open | in-review | closed
	Priority    string `gorm:"default:medium" json:"priority"` // low | medium | high
}
