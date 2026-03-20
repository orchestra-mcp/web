package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// CustomDomainHandler handles custom domain CRUD and DNS verification.
type CustomDomainHandler struct {
	db *gorm.DB
}

// NewCustomDomainHandler creates a new handler.
func NewCustomDomainHandler(db *gorm.DB) *CustomDomainHandler {
	return &CustomDomainHandler{db: db}
}

// List handles GET /api/settings/custom-domains
func (h *CustomDomainHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var domains []models.CustomDomain
	h.db.Where("user_id = ?", user.ID).Order("created_at desc").Find(&domains)

	return c.JSON(fiber.Map{"domains": domains})
}

// Add handles POST /api/settings/custom-domains
func (h *CustomDomainHandler) Add(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Domain string `json:"domain"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	domain := strings.TrimSpace(strings.ToLower(body.Domain))
	if domain == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "domain is required"})
	}

	// Generate verification TXT record value.
	token := make([]byte, 16)
	rand.Read(token)
	txtRecord := "orchestra-verify=" + hex.EncodeToString(token)

	cd := models.CustomDomain{
		UserID:       user.ID,
		Domain:       domain,
		DNSTxtRecord: txtRecord,
	}

	if err := h.db.Create(&cd).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "domain already registered"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add domain"})
	}

	return c.Status(fiber.StatusCreated).JSON(cd)
}

// Verify handles POST /api/settings/custom-domains/:id/verify
func (h *CustomDomainHandler) Verify(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var cd models.CustomDomain
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&cd).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "domain not found"})
	}

	if cd.Verified {
		return c.JSON(fiber.Map{"verified": true, "message": "already verified"})
	}

	// Look up TXT records for the domain.
	records, err := net.LookupTXT(cd.Domain)
	if err != nil {
		return c.JSON(fiber.Map{"verified": false, "message": "DNS lookup failed", "expected": cd.DNSTxtRecord})
	}

	for _, record := range records {
		if strings.TrimSpace(record) == cd.DNSTxtRecord {
			// Mark as verified.
			h.db.Model(&cd).Updates(map[string]any{
				"verified":    true,
				"verified_at": gorm.Expr("NOW()"),
			})
			return c.JSON(fiber.Map{"verified": true, "message": "domain verified"})
		}
	}

	return c.JSON(fiber.Map{
		"verified": false,
		"message":  "TXT record not found",
		"expected": cd.DNSTxtRecord,
	})
}

// Delete handles DELETE /api/settings/custom-domains/:id
func (h *CustomDomainHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	result := h.db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.CustomDomain{})
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "domain not found"})
	}

	return c.JSON(fiber.Map{"deleted": true})
}
