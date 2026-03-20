package models

// BadgeDefinition defines a badge type that can be awarded to users.
type BadgeDefinition struct {
	Base
	Slug           string `gorm:"uniqueIndex" json:"slug"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Icon           string `json:"icon"`            // boxicons class e.g. "bxs-medal"
	Color          string `json:"color"`           // hex color e.g. "#00e5ff"
	Category       string `json:"category"`        // e.g. "achievement", "streak", "contribution"
	PointsRequired int    `json:"points_required"` // 0 = manual award only
	AutoAward      bool   `gorm:"default:false" json:"auto_award"`
	SortOrder      int    `gorm:"default:0" json:"sort_order"`
}

// UserBadge links a user to an awarded badge.
type UserBadge struct {
	Base
	UserID            uint   `gorm:"uniqueIndex:idx_user_badge" json:"user_id"`
	BadgeDefinitionID string `gorm:"type:uuid;uniqueIndex:idx_user_badge" json:"badge_definition_id"`
	AwardedBy         string `json:"awarded_by,omitempty"` // admin user ID
	Note              string `json:"note"`
}
