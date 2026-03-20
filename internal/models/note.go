package models

import "gorm.io/datatypes"

// Note is a freeform markdown document owned by a user.
type Note struct {
	Base
	UserID    uint           `json:"user_id"`
	ProjectID *string        `gorm:"type:uuid;index" json:"project_id"`
	TeamID    *string        `gorm:"type:uuid" json:"team_id"`
	Title     string         `json:"title"`
	Content   string         `gorm:"type:text" json:"content"`
	Pinned    bool           `gorm:"default:false" json:"pinned"`
	Tags      datatypes.JSON `json:"tags"`
	Scope     string         `gorm:"default:personal" json:"scope"` // personal | team | public
	PublicURL string         `json:"public_url"`
	Slug      string         `gorm:"index" json:"slug"`
	Meta      datatypes.JSON `json:"meta"`
	Version   int            `gorm:"default:1" json:"version"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
}
