package models

import (
	"time"

	"gorm.io/gorm"
)

// GitHubIssue stores a synced GitHub issue or pull request.
type GitHubIssue struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	GitHubID     int            `json:"github_id"`
	Repo         string         `json:"repo"`
	Title        string         `json:"title"`
	Body         string         `gorm:"type:text" json:"body"`
	State        string         `gorm:"default:open" json:"state"`  // open | closed | merged | draft
	Type         string         `gorm:"default:issue" json:"type"`  // issue | pr
	Author       string         `json:"author"`
	AuthorAvatar string         `json:"author_avatar"`
	Labels       string         `gorm:"type:text" json:"-"`         // comma-separated, serialized in handler
}

// GitHubRepo stores a tracked GitHub repository.
type GitHubRepo struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Owner     string         `json:"owner"`
	Name      string         `json:"name"`
	FullName  string         `gorm:"uniqueIndex" json:"full_name"`
}
