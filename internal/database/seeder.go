package database

import (
	"encoding/json"
	"log"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// SeedDefaults inserts default system settings if they don't already exist.
// Safe to call on every boot — uses FirstOrCreate for idempotency.
func SeedDefaults(db *gorm.DB) {
	settings := map[string]interface{}{
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
			"host":       "smtp.gmail.com",
			"port":       587,
			"username":   "",
			"password":   "",
			"from_name":  "Orchestra",
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

	for key, val := range settings {
		raw, err := json.Marshal(val)
		if err != nil {
			log.Printf("seeder: failed to marshal %s: %v", key, err)
			continue
		}

		s := models.SystemSetting{Key: key}
		result := db.Where("key = ?", key).FirstOrCreate(&s, models.SystemSetting{
			Key:   key,
			Value: raw,
		})
		if result.Error != nil {
			log.Printf("seeder: failed to seed %s: %v", key, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("seeder: seeded default %s", key)
		}
	}
}
