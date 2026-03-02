package models

import (
	"time"

	"gorm.io/datatypes"
)

// Task is an atomic unit of work within a story.
type Task struct {
	Base
	StoryID     string         `gorm:"type:uuid;index" json:"story_id"`
	ProjectID   string         `gorm:"type:uuid;index" json:"project_id"`
	UserID      uint           `json:"user_id"`
	AssigneeID  *uint          `json:"assignee_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `gorm:"default:backlog" json:"status"`
	Priority    string         `gorm:"default:medium" json:"priority"`
	Labels      datatypes.JSON `json:"labels"`
	DueAt       *time.Time     `json:"due_at"`
	Position    int            `gorm:"default:0" json:"position"`
	Meta        datatypes.JSON `json:"meta"`
	Version     int            `gorm:"default:1" json:"version"`
	Story       Story          `gorm:"foreignKey:StoryID" json:"-"`
	Project     Project        `gorm:"foreignKey:ProjectID" json:"-"`
	User        User           `gorm:"foreignKey:UserID" json:"-"`
}
