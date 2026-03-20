package models

// VerificationType defines a verification badge type (verified, contributor, sponsor, enterprise).
type VerificationType struct {
	Base
	Slug      string `gorm:"uniqueIndex" json:"slug"`
	Name      string `json:"name"`
	Color     string `json:"color"`      // hex color e.g. "#00e5ff"
	BadgeText string `json:"badge_text"` // tooltip text e.g. "Verified Member"
	Icon      string `json:"icon"`       // boxicons class e.g. "bxs-badge-check"
	SortOrder int    `gorm:"default:0" json:"sort_order"`
}

// UserVerification links a user to a verification type.
type UserVerification struct {
	Base
	UserID             uint   `gorm:"uniqueIndex:idx_user_verification" json:"user_id"`
	VerificationTypeID string `gorm:"type:uuid;uniqueIndex:idx_user_verification" json:"verification_type_id"`
	VerifiedBy         string `json:"verified_by,omitempty"` // admin who verified
	Note               string `json:"note"`
}
