package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UserIntegration stores per-user Discord/Slack workspace configuration.
// Separate from OAuthAccount which handles authentication tokens.
type UserIntegration struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UserID     uint           `gorm:"uniqueIndex:idx_user_provider" json:"user_id"`
	Provider   string         `gorm:"uniqueIndex:idx_user_provider" json:"provider"` // discord | slack
	GuildID    string         `json:"guild_id"`                                      // Discord server ID
	ChannelID  string         `json:"channel_id"`                                    // Default channel
	TeamID     string         `json:"team_id"`                                       // Slack workspace ID
	WebhookURL string         `json:"webhook_url"`                                   // User's webhook
	Enabled    bool           `gorm:"default:true" json:"enabled"`
	Meta       datatypes.JSON `json:"meta"` // Extra provider-specific data
}
