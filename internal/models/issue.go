package models

import (
	"time"

	"gorm.io/gorm"
)

// Issue is a user-reported issue or bug report.
type Issue struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	UserID      uint           `json:"user_id"`
	Title       string `json:"title"`
	Description string `gorm:"type:text" json:"description"`
	Status      string `gorm:"default:open" json:"status"`    // open | in-review | closed
	Priority    string `gorm:"default:medium" json:"priority"` // low | medium | high
}
