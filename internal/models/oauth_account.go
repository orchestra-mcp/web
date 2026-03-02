package models

import "gorm.io/datatypes"

// OAuthAccount links a user to a third-party OAuth provider.
type OAuthAccount struct {
	Base
	UserID         uint           `gorm:"index" json:"user_id"`
	Provider       string         `json:"provider"` // google | github | gitlab
	ProviderUserID string         `gorm:"uniqueIndex:idx_oauth_provider_user" json:"provider_user_id"`
	Email          string         `json:"email"`
	Name           string         `json:"name"`
	Avatar         string         `json:"avatar"`
	AccessToken    string         `json:"-"`
	RefreshToken   string         `json:"-"`
	Meta           datatypes.JSON `json:"meta"`
	User           User           `gorm:"foreignKey:UserID" json:"-"`
}
