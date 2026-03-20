package models

import "time"

// ContentView records a single page view on a shared content item.
type ContentView struct {
	ID         int64     `gorm:"primarykey" json:"id"`
	ContentID  uint      `gorm:"index" json:"content_id"`
	ViewerHash string    `json:"viewer_hash"` // SHA-256(IP) for unique counting
	UserAgent  string    `json:"user_agent"`
	Referer    string    `json:"referer"`
	CreatedAt  time.Time `json:"created_at"`
}
