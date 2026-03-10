package models

import "gorm.io/datatypes"

// Team represents an organization or workspace that groups users.
type Team struct {
	Base
	Name      string         `json:"name"`
	Slug      string         `gorm:"uniqueIndex" json:"slug"`
	Plan      string         `gorm:"default:free" json:"plan"`
	AvatarURL string         `json:"avatar_url"`
	Settings  datatypes.JSON `json:"settings"`
	Meta      datatypes.JSON `json:"meta"`
}
