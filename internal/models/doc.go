package models

import "gorm.io/datatypes"

// Doc represents a wiki/documentation page (synced from MCP docs plugin).
type Doc struct {
	Base
	UserID      uint           `gorm:"uniqueIndex:idx_doc_user_did" json:"user_id"`
	TeamID      *string        `json:"team_id"`
	ProjectSlug string         `gorm:"index" json:"project_slug"`
	DocID       string         `gorm:"uniqueIndex:idx_doc_user_did" json:"doc_id"` // slug from MCP docs plugin
	Title       string         `json:"title"`
	Category    string         `gorm:"index" json:"category"`
	Tags        datatypes.JSON `json:"tags"`
	ParentID    string         `json:"parent_id"`
	Body        string         `gorm:"type:text" json:"body"`
	Version     int            `gorm:"default:1" json:"version"`
	Meta        datatypes.JSON `json:"meta"`
	Pinned      bool           `gorm:"default:false" json:"pinned"`
	Icon        string         `json:"icon"`
	Color       string         `json:"color"`
}
