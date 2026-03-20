package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// CustomDomainMiddleware checks the Host header for custom domains.
// If the request comes from a verified custom domain, it sets the owner's
// user ID in the context for downstream handlers to use.
//
// This is a preparation stub — full routing integration will be added
// when custom domain support is enabled in production.
func CustomDomainMiddleware(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		host := c.Hostname()

		// Skip standard application hosts.
		if isStandardHost(host) {
			return c.Next()
		}

		// Look up the domain.
		var cd models.CustomDomain
		if err := db.Where("domain = ? AND verified = ?", host, true).First(&cd).Error; err != nil {
			// Not a known custom domain — continue normally.
			return c.Next()
		}

		// Set the domain owner's user ID in locals for downstream handlers.
		c.Locals("custom_domain_user_id", cd.UserID)
		c.Locals("custom_domain", cd.Domain)

		return c.Next()
	}
}

// isStandardHost returns true for the app's own domains (not custom).
func isStandardHost(host string) bool {
	// Standard hosts that should not trigger custom domain lookup.
	standardSuffixes := []string{
		"localhost",
		"orchestra.dev",
		"orchestra.local",
		"127.0.0.1",
	}
	for _, suffix := range standardSuffixes {
		if host == suffix || len(host) > len(suffix) && host[len(host)-len(suffix)-1] == '.' && host[len(host)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}
