package handlers

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserIntegrationsHandler handles per-user Discord/Slack integration settings.
type UserIntegrationsHandler struct {
	db *gorm.DB
}

// NewUserIntegrationsHandler creates a new handler.
func NewUserIntegrationsHandler(db *gorm.DB) *UserIntegrationsHandler {
	return &UserIntegrationsHandler{db: db}
}

var validProviders = map[string]bool{"discord": true, "slack": true}

// List returns all integration configs for the current user.
func (h *UserIntegrationsHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var integrations []models.UserIntegration
	h.db.Where("user_id = ?", user.ID).Find(&integrations)
	return c.JSON(fiber.Map{"integrations": integrations})
}

// Upsert creates or updates a user integration for a provider.
func (h *UserIntegrationsHandler) Upsert(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	provider := c.Params("provider")
	if !validProviders[provider] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid provider, must be discord or slack"})
	}

	var body struct {
		GuildID    string `json:"guild_id"`
		ChannelID  string `json:"channel_id"`
		TeamID     string `json:"team_id"`
		WebhookURL string `json:"webhook_url"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	integration := models.UserIntegration{
		UserID:     user.ID,
		Provider:   provider,
		GuildID:    body.GuildID,
		ChannelID:  body.ChannelID,
		TeamID:     body.TeamID,
		WebhookURL: body.WebhookURL,
		Enabled:    true,
	}
	if body.Enabled != nil {
		integration.Enabled = *body.Enabled
	}

	if err := h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"guild_id", "channel_id", "team_id", "webhook_url", "enabled", "updated_at"}),
	}).Create(&integration).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save integration"})
	}

	// Re-fetch to get the full row with ID
	h.db.Where("user_id = ? AND provider = ?", user.ID, provider).First(&integration)
	return c.JSON(fiber.Map{"ok": true, "integration": integration})
}

// Delete removes a user integration for a provider.
func (h *UserIntegrationsHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	provider := c.Params("provider")
	if !validProviders[provider] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid provider"})
	}

	h.db.Where("user_id = ? AND provider = ?", user.ID, provider).Delete(&models.UserIntegration{})
	return c.JSON(fiber.Map{"ok": true})
}

// AppInstallURLs returns OAuth install URLs for adding bots to user's servers/workspaces.
// Built from admin-configured credentials. Only returns URLs for enabled providers.
func (h *UserIntegrationsHandler) AppInstallURLs(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	apps := fiber.Map{}

	// Load integrations admin setting
	var integrationsSetting models.SystemSetting
	integrations := map[string]interface{}{}
	if err := h.db.Where("key = ?", "integrations").First(&integrationsSetting).Error; err == nil {
		_ = json.Unmarshal(integrationsSetting.Value, &integrations)
	}

	// Load discord admin setting for application_id
	var discordSetting models.SystemSetting
	discordCfg := map[string]interface{}{}
	if err := h.db.Where("key = ?", "discord").First(&discordSetting).Error; err == nil {
		_ = json.Unmarshal(discordSetting.Value, &discordCfg)
	}

	// Load slack admin setting for client_id
	var slackSetting models.SystemSetting
	slackCfg := map[string]interface{}{}
	if err := h.db.Where("key = ?", "slack").First(&slackSetting).Error; err == nil {
		_ = json.Unmarshal(slackSetting.Value, &slackCfg)
	}

	// Discord bot install URL
	if enabled, _ := integrations["discord_enabled"].(bool); enabled {
		clientID, _ := discordCfg["client_id"].(string)
		if clientID == "" {
			clientID, _ = integrations["discord_client_id"].(string)
		}
		if clientID != "" {
			apps["discord"] = fiber.Map{
				"install_url": fmt.Sprintf(
					"https://discord.com/api/oauth2/authorize?client_id=%s&permissions=274878024704&scope=bot%%20applications.commands",
					url.QueryEscape(clientID),
				),
				"name": "Discord",
			}
		}
	}

	// Slack app install URL
	if enabled, _ := integrations["slack_enabled"].(bool); enabled {
		clientID, _ := slackCfg["client_id"].(string)
		if clientID == "" {
			clientID, _ = integrations["slack_client_id"].(string)
		}
		if clientID != "" {
			apps["slack"] = fiber.Map{
				"install_url": fmt.Sprintf(
					"https://slack.com/oauth/v2/authorize?client_id=%s&scope=chat:write,commands,channels:read&user_scope=",
					url.QueryEscape(clientID),
				),
				"name": "Slack",
			}
		}
	}

	return c.JSON(fiber.Map{"apps": apps})
}
