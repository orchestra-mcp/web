package models

import (
	"time"

	"gorm.io/gorm"
)

// SharedContent represents a piece of user content shared publicly on the community platform.
type SharedContent struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UserID       uint           `gorm:"index" json:"user_id"`
	EntityType   string         `gorm:"index" json:"entity_type"` // note, skill, agent, workflow, prompt, api_collection, presentation, doc
	EntityID     string         `json:"entity_id,omitempty"`      // UUID of the linked entity (api_collection, presentation, doc)
	Slug         string         `gorm:"index" json:"slug"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Content      string         `gorm:"type:text" json:"content"` // markdown body
	Visibility   string         `gorm:"default:public" json:"visibility"` // public, unlisted, private
	ViewsCount   int            `gorm:"default:0" json:"views_count"`
	LikesCount   int            `gorm:"default:0" json:"likes_count"`
	UniqueViews  int            `gorm:"default:0" json:"unique_views"`
	Password     string         `json:"-"`                              // bcrypt hash, empty = no password
	TeamID       *uint          `gorm:"index" json:"team_id,omitempty"` // nil = no team restriction
	CustomDomain string         `json:"custom_domain,omitempty"`

	User User `gorm:"-:migration" json:"-"`
}

// ShareComment is a comment on a shared content item.
type ShareComment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	ShareID   uint           `gorm:"index" json:"share_id"`
	UserID    uint           `gorm:"index" json:"user_id"`
	Body      string         `gorm:"type:text" json:"body"`
	Kind      string         `gorm:"default:comment" json:"kind"` // comment, change_request

	User User `gorm:"-:migration" json:"-"`
}
