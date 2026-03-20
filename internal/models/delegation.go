package models

import (
	"time"

	"gorm.io/datatypes"
)

// Delegation represents a task delegation between team members (used by MCP sync).
type Delegation struct {
	Base
	UserID       uint           `gorm:"uniqueIndex:idx_delegation_user_did" json:"user_id"`
	TeamID       *string        `json:"team_id"`
	ProjectSlug  string         `gorm:"index" json:"project_slug"`
	DelegationID string         `gorm:"uniqueIndex:idx_delegation_user_did" json:"delegation_id"` // e.g. DEL-ABC
	FeatureID    string         `json:"feature_id"`
	FromPerson   string         `json:"from_person"`
	ToPerson     string         `gorm:"index" json:"to_person"`
	Question     string         `gorm:"type:text" json:"question"`
	Context      string         `gorm:"type:text" json:"context"`
	Response     string         `gorm:"type:text" json:"response"`
	Status       string         `gorm:"default:pending" json:"status"`
	RespondedAt  *time.Time     `json:"responded_at"`
	Body         string         `gorm:"type:text" json:"body"`
	Version      int            `gorm:"default:1" json:"version"`
	Meta         datatypes.JSON `json:"meta"`
}
