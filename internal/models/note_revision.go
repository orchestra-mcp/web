package models

import "time"

// NoteRevision stores previous versions of a note for conflict resolution.
type NoteRevision struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	NoteID    string    `gorm:"type:uuid;index" json:"note_id"`
	UserID    uint      `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	Content   string    `gorm:"type:text" json:"content"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}
