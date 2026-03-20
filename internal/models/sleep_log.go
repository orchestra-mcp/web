package models

// SleepLog stores individual sleep entries (separate from SleepConfig which is for shutdown).
type SleepLog struct {
	Base
	UserID        uint    `gorm:"index" json:"user_id"`
	BedTime       string  `json:"bed_time"`
	WakeTime      string  `json:"wake_time"`
	QualityRating int     `gorm:"default:3" json:"quality_rating"`
	DurationHours float64 `json:"duration_hours"`
	LoggedAt      string  `json:"logged_at"`
}
