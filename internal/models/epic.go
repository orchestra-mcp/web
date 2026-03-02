package models

import "gorm.io/datatypes"

// Epic is a high-level grouping of stories within a project.
type Epic struct {
	Base
	ProjectID   string         `gorm:"type:uuid;index" json:"project_id"`
	UserID      uint           `json:"user_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Status      string         `gorm:"default:backlog" json:"status"`
	Priority    string         `gorm:"default:medium" json:"priority"`
	Position    int            `gorm:"default:0" json:"position"`
	Meta        datatypes.JSON `json:"meta"`
	Version     int            `gorm:"default:1" json:"version"`
	Project     Project        `gorm:"foreignKey:ProjectID" json:"-"`
	User        User           `gorm:"foreignKey:UserID" json:"-"`
	Stories     []Story        `gorm:"foreignKey:EpicID" json:"stories,omitempty"`
}
