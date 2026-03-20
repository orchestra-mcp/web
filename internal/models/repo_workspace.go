package models

import (
	"time"

	"gorm.io/datatypes"
)

// RepoWorkspace represents a cloned GitHub repository managed by the server.
type RepoWorkspace struct {
	Base
	UserID      uint           `gorm:"index" json:"user_id"`
	TeamID      *string        `json:"team_id"`
	Name        string         `json:"name"`
	RepoURL     string         `json:"repo_url"`
	RepoOwner   string         `json:"repo_owner"`
	RepoName    string         `json:"repo_name"`
	Branch      string         `gorm:"default:main" json:"branch"`
	ClonePath   string         `json:"-"`
	Status      string         `gorm:"default:pending" json:"status"` // pending|cloning|ready|error|syncing
	LastSyncAt  *time.Time     `json:"last_synced_at"`
	CommitSHA   string         `json:"commit_sha"`
	Error       string         `json:"error"`
	Description string         `json:"description"`
	Language    string         `json:"language"`
	Stars       int            `json:"stars"`
	Topics      datatypes.JSON `json:"topics"`
	IsPrivate   bool           `json:"is_private"`
	Meta        datatypes.JSON `json:"meta"`
}
