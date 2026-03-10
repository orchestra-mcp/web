package models

import "gorm.io/datatypes"

// Person represents a team member profile (used by MCP sync).
type Person struct {
	Base
	UserID      uint           `gorm:"uniqueIndex:idx_person_user_pid" json:"user_id"`
	TeamID      *string        `json:"team_id"`
	ProjectSlug string         `gorm:"index" json:"project_slug"`
	PersonID    string         `gorm:"uniqueIndex:idx_person_user_pid" json:"person_id"` // e.g. PERS-ABC
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	Role        string         `gorm:"default:developer" json:"role"`
	Status      string         `gorm:"default:active" json:"status"`
	Bio         string         `json:"bio"`
	Body        string         `gorm:"type:text" json:"body"`
	Version     int            `gorm:"default:1" json:"version"`
	Meta        datatypes.JSON `json:"meta"`
}
