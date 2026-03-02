package models

import "gorm.io/datatypes"

// Story represents a user story within an epic.
type Story struct {
	Base
	EpicID      string         `gorm:"type:uuid;index" json:"epic_id"`
	ProjectID   string         `gorm:"type:uuid;index" json:"project_id"`
	UserID      uint           `json:"user_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `gorm:"default:backlog" json:"status"`
	Priority    string         `gorm:"default:medium" json:"priority"`
	StoryPoints *int           `json:"story_points"`
	Position    int            `gorm:"default:0" json:"position"`
	Meta        datatypes.JSON `json:"meta"`
	Version     int            `gorm:"default:1" json:"version"`
	Epic        Epic           `gorm:"foreignKey:EpicID" json:"-"`
	Project     Project        `gorm:"foreignKey:ProjectID" json:"-"`
	User        User           `gorm:"foreignKey:UserID" json:"-"`
	Tasks       []Task         `gorm:"foreignKey:StoryID" json:"tasks,omitempty"`
}
