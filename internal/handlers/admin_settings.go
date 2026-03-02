package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm/clause"
	"gorm.io/gorm"
)

// AdminSettingsHandler handles system settings key-value endpoints.
type AdminSettingsHandler struct {
	db *gorm.DB
}

// NewAdminSettingsHandler creates a new AdminSettingsHandler.
func NewAdminSettingsHandler(db *gorm.DB) *AdminSettingsHandler {
	return &AdminSettingsHandler{db: db}
}

// validKeys lists the allowed settings keys.
var validKeys = map[string]bool{
	"general":      true,
	"features":     true,
	"homepage":     true,
	"agents":       true,
	"contact":      true,
	"pricing":      true,
	"download":     true,
	"integrations": true,
	"smtp":         true,
	"seo":          true,
	"coming_soon":  true,
}

// publicKeys lists settings keys readable without authentication.
var publicKeys = map[string]bool{
	"coming_soon": true,
}

// defaultSettings returns default values for each key.
func defaultSettings(key string) interface{} {
	defaults := map[string]interface{}{
		"general": map[string]interface{}{
			"site_name":        "Orchestra",
			"tagline":          "The AI-Agentic IDE",
			"url":              "https://orchestra.dev",
			"support_email":    "support@orchestra.dev",
			"maintenance_mode": false,
		},
		"features": []map[string]interface{}{
			{"key": "blog", "label": "Blog", "enabled": true, "admin_only": false},
			{"key": "marketplace", "label": "Marketplace", "enabled": true, "admin_only": false},
			{"key": "docs", "label": "Docs", "enabled": true, "admin_only": false},
			{"key": "download", "label": "Download", "enabled": true, "admin_only": false},
			{"key": "solutions", "label": "Solutions", "enabled": true, "admin_only": false},
			{"key": "pricing", "label": "Pricing", "enabled": true, "admin_only": false},
		},
		"homepage": map[string]interface{}{
			"hero_headline": "The AI-native workspace for developers",
			"hero_subtext":  "Orchestra connects your IDE to 131 AI tools across 5 platforms.",
			"hero_cta":      "Get started free",
		},
		"agents": map[string]interface{}{
			"headline": "Multi-agent orchestration built in",
			"subtext":  "Define agents, connect models, and run workflows.",
		},
		"contact": map[string]interface{}{
			"headline": "Get in touch",
			"email":    "hello@orchestra.dev",
			"hours":    "Mon–Fri, 9am–5pm PST",
		},
		"pricing": map[string]interface{}{
			"plans": []map[string]interface{}{
				{"name": "Free", "price": "0", "period": "forever", "highlighted": false},
				{"name": "Pro", "price": "19", "period": "month", "highlighted": true},
				{"name": "Enterprise", "price": "custom", "period": "", "highlighted": false},
			},
		},
		"download": map[string]interface{}{
			"macos":   map[string]string{"url": "", "version": "0.0.2", "release_date": "2026-02-27"},
			"windows": map[string]string{"url": "", "version": "0.0.2", "release_date": "2026-02-27"},
			"linux":   map[string]string{"url": "", "version": "0.0.2", "release_date": "2026-02-27"},
		},
		"integrations": map[string]interface{}{
			"google": map[string]string{"client_id": "", "client_secret": ""},
			"github": map[string]string{"client_id": "", "client_secret": ""},
		},
		"smtp": map[string]interface{}{
			"host":      "smtp.gmail.com",
			"port":      587,
			"username":  "",
			"password":  "",
			"from_name": "Orchestra",
			"from_email": "noreply@orchestra.dev",
		},
		"seo": map[string]interface{}{
			"title_template":   "%s | Orchestra",
			"meta_description": "Orchestra MCP — the AI-native developer workspace with 131 tools across 5 platforms.",
			"og_image_url":     "/og-image.png",
			"robots_txt":       "User-agent: *\nAllow: /",
			"sitemap_url":      "/sitemap.xml",
		},
		"coming_soon": map[string]interface{}{
			"enabled": false,
			"message": "We're putting the finishing touches on something amazing. Stay tuned!",
			"title":   "Coming Soon",
		},
	}
	if v, ok := defaults[key]; ok {
		return v
	}
	return map[string]interface{}{}
}

// GetPublicSetting handles GET /api/settings/:key — no auth required, only for public keys.
func (h *AdminSettingsHandler) GetPublicSetting(c fiber.Ctx) error {
	key := c.Params("key")
	if !publicKeys[key] {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	var setting models.SystemSetting
	if err := h.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return c.JSON(fiber.Map{"key": key, "value": defaultSettings(key)})
	}

	var value interface{}
	_ = json.Unmarshal(setting.Value, &value)
	return c.JSON(fiber.Map{"key": key, "value": value})
}

// GetSetting handles GET /api/admin/settings/:key
func (h *AdminSettingsHandler) GetSetting(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	key := c.Params("key")
	if !validKeys[key] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid settings key"})
	}

	var setting models.SystemSetting
	if err := h.db.Where("key = ?", key).First(&setting).Error; err != nil {
		// Return defaults if not yet set
		return c.JSON(fiber.Map{"key": key, "value": defaultSettings(key)})
	}

	var value interface{}
	_ = json.Unmarshal(setting.Value, &value)
	return c.JSON(fiber.Map{"key": key, "value": value, "updated_at": setting.UpdatedAt})
}

// UpdateSetting handles PATCH /api/admin/settings/:key
func (h *AdminSettingsHandler) UpdateSetting(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	key := c.Params("key")
	if !validKeys[key] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid settings key"})
	}

	// Accept the raw JSON body as the value
	rawValue := c.Body()
	if len(rawValue) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body is required"})
	}

	setting := models.SystemSetting{
		Key:   key,
		Value: rawValue,
	}

	if err := h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&setting).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save setting"})
	}

	return c.JSON(fiber.Map{"ok": true, "key": key})
}
