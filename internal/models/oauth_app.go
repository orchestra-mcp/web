package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OAuthApp represents a third-party application registered as an OAuth2 client.
type OAuthApp struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Name         string         `json:"name"`
	ClientID     string         `gorm:"uniqueIndex" json:"client_id"`
	ClientSecret string         `json:"-"`
	RedirectURIs datatypes.JSON `json:"redirect_uris"` // JSON array of strings
	Scopes       datatypes.JSON `json:"scopes"`        // JSON array of strings
	OwnerID      uint           `json:"owner_id"`
	IconURL      string         `json:"icon_url"`
	Owner        User           `gorm:"foreignKey:OwnerID" json:"-"`
}

// OAuthAuthorization represents a pending authorization code grant.
type OAuthAuthorization struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	AppID       uint           `json:"app_id"`
	UserID      uint           `json:"user_id"`
	Code        string         `gorm:"uniqueIndex" json:"code"`
	Scopes      datatypes.JSON `json:"scopes"`
	RedirectURI string         `json:"redirect_uri"`
	ExpiresAt   time.Time      `json:"expires_at"`
	Used        bool           `gorm:"default:false" json:"used"`
	App         OAuthApp       `gorm:"foreignKey:AppID" json:"-"`
	User        User           `gorm:"foreignKey:UserID" json:"-"`
}

// OAuthAccessToken represents an issued access/refresh token pair.
type OAuthAccessToken struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	AppID        uint           `json:"app_id"`
	UserID       uint           `json:"user_id"`
	AccessToken  string         `gorm:"uniqueIndex" json:"-"`
	RefreshToken string         `gorm:"uniqueIndex" json:"-"`
	Scopes       datatypes.JSON `json:"scopes"`
	ExpiresAt    time.Time      `json:"expires_at"`
	App          OAuthApp       `gorm:"foreignKey:AppID" json:"-"`
	User         User           `gorm:"foreignKey:UserID" json:"-"`
}
