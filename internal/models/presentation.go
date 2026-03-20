package models

import "gorm.io/datatypes"

// Presentation represents a slide deck.
type Presentation struct {
	Base
	UserID      uint           `gorm:"index" json:"user_id"`
	TeamID      *string        `gorm:"type:uuid;index" json:"team_id,omitempty"`
	Title       string         `json:"title"`
	Slug        string         `gorm:"index" json:"slug"`
	Description string         `json:"description"`
	Theme       datatypes.JSON `json:"theme"`      // colors, fonts, spacing
	Visibility  string         `gorm:"default:private" json:"visibility"` // private, team, public
	SlideCount  int            `gorm:"default:0" json:"slide_count"`
	Version     int            `gorm:"default:1" json:"version"`
}

// PresentationSlide is a single slide within a presentation.
type PresentationSlide struct {
	Base
	PresentationID string         `gorm:"type:uuid;index" json:"presentation_id"`
	UserID         uint           `gorm:"index" json:"user_id"`
	SlideNumber    int            `gorm:"index" json:"slide_number"`
	Layout         string         `gorm:"default:title-content" json:"layout"` // title, title-content, two-column, image-full, quote, blank
	Title          string         `json:"title"`
	Content        string         `gorm:"type:text" json:"content"` // markdown body
	Notes          string         `gorm:"type:text" json:"notes"`   // presenter notes
	Properties     datatypes.JSON `json:"properties"`                // per-slide overrides (bg color, image, transition)
}
