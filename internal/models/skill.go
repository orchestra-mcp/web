package models

import "gorm.io/datatypes"

// Skill represents an AI skill (slash command) that can be included in projects.
// Skills can be personal, team-scoped, or public (read-only + clone via public URL).
type Skill struct {
	Base
	UserID      uint           `json:"user_id"`
	TeamID      *string        `gorm:"type:uuid" json:"team_id"`
	Name        string         `json:"name"`
	Slug        string         `gorm:"index" json:"slug"`
	Description string         `json:"description"`
	Content     string         `gorm:"type:text" json:"content"` // SKILL.md body (markdown)
	Scope       string         `gorm:"default:personal" json:"scope"` // personal | team | public
	PublicURL   string         `json:"public_url"`
	Icon        string         `json:"icon"`
	Color       string         `json:"color"`
	Stacks      datatypes.JSON `json:"stacks"` // e.g. ["go", "react", "rust"]
	Version     int            `gorm:"default:1" json:"version"`
	Meta        datatypes.JSON `json:"meta"`
}

// ProjectSkill is the pivot table linking skills to projects.
type ProjectSkill struct {
	ProjectID string `gorm:"type:uuid;primaryKey" json:"project_id"`
	SkillID   string `gorm:"type:uuid;primaryKey" json:"skill_id"`
	Enabled   bool   `gorm:"default:true" json:"enabled"`
	Skill     Skill  `gorm:"foreignKey:SkillID;references:ID" json:"-"`
}
