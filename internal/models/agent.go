package models

import "gorm.io/datatypes"

// Agent represents an AI agent definition that can be included in projects.
// Agents can be personal, team-scoped, or public (read-only + clone via public URL).
type Agent struct {
	Base
	UserID      uint           `json:"user_id"`
	TeamID      *string        `gorm:"type:uuid" json:"team_id"`
	Name        string         `json:"name"`
	Slug        string         `gorm:"index" json:"slug"`
	Description string         `json:"description"`
	Content     string         `gorm:"type:text" json:"content"` // agent .md body (markdown)
	Scope       string         `gorm:"default:personal" json:"scope"` // personal | team | public
	PublicURL   string         `json:"public_url"`
	Icon        string         `json:"icon"`
	Color       string         `json:"color"`
	Version     int            `gorm:"default:1" json:"version"`
	Meta        datatypes.JSON `json:"meta"`
}

// ProjectAgent is the pivot table linking agents to projects.
type ProjectAgent struct {
	ProjectID string `gorm:"type:uuid;primaryKey" json:"project_id"`
	AgentID   string `gorm:"type:uuid;primaryKey" json:"agent_id"`
	Enabled   bool   `gorm:"default:true" json:"enabled"`
	Agent     Agent  `gorm:"foreignKey:AgentID;references:ID" json:"-"`
}
