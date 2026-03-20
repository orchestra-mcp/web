package models

import "gorm.io/datatypes"

// Workspace represents a collection of project folders on a machine.
// Workspaces can be shared across multiple teams via WorkspaceTeam pivot.
type Workspace struct {
	Base
	OwnerID      uint           `json:"owner_id"`
	Name         string         `json:"name"`
	Folders      datatypes.JSON `json:"folders"`
	PrimaryFolder string        `json:"primary_folder"`
	Status       string         `gorm:"default:active" json:"status"` // active | archived
	Metadata     datatypes.JSON `json:"metadata"`
	Version      int            `gorm:"default:1" json:"version"`

	// Relations
	Teams []WorkspaceTeam `gorm:"foreignKey:WorkspaceID" json:"teams,omitempty"`
}

// WorkspaceTeam is the pivot table linking workspaces to teams (many-to-many).
type WorkspaceTeam struct {
	WorkspaceID string `gorm:"type:uuid;primaryKey" json:"workspace_id"`
	TeamID      string `gorm:"type:uuid;primaryKey" json:"team_id"`
	Role        string `gorm:"default:editor" json:"role"` // owner | editor | viewer

	Workspace Workspace `gorm:"foreignKey:WorkspaceID" json:"-"`
}
