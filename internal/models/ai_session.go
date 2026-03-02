package models

import (
	"time"

	"gorm.io/datatypes"
)

// AiSession represents a conversation session with an AI model.
type AiSession struct {
	Base
	UserID        uint           `json:"user_id"`
	ProjectID     *string        `gorm:"type:uuid;index" json:"project_id"`
	Name          string         `gorm:"default:'New Chat'" json:"name"`
	Model         string         `gorm:"default:'claude-sonnet-4-6'" json:"model"`
	Pinned        bool           `gorm:"default:false" json:"pinned"`
	LastMessageAt *time.Time     `json:"last_message_at"`
	MessageCount  int            `gorm:"default:0" json:"message_count"`
	Meta          datatypes.JSON `json:"meta"`
	Version       int            `gorm:"default:1" json:"version"`
	User          User           `gorm:"foreignKey:UserID" json:"-"`
}
