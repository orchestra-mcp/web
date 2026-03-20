package models

// UserSetting stores key-value preferences synced across devices via PowerSync.
type UserSetting struct {
	Base
	UserID uint   `gorm:"uniqueIndex:idx_user_setting_key" json:"user_id"`
	Key    string `gorm:"uniqueIndex:idx_user_setting_key" json:"key"`
	Value  string `gorm:"type:text" json:"value"`
}
