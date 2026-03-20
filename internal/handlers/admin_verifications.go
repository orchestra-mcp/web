package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// AdminVerificationHandler handles verification type CRUD.
type AdminVerificationHandler struct {
	db *gorm.DB
}

// NewAdminVerificationHandler creates a new AdminVerificationHandler.
func NewAdminVerificationHandler(db *gorm.DB) *AdminVerificationHandler {
	return &AdminVerificationHandler{db: db}
}

// List returns all verification types.
func (h *AdminVerificationHandler) List(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	var types []models.VerificationType
	h.db.Order("sort_order asc, name asc").Find(&types)
	return c.JSON(fiber.Map{"verifications": types})
}

// Create adds a new verification type.
func (h *AdminVerificationHandler) Create(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	var body models.VerificationType
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Slug == "" || body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slug and name required"})
	}
	if err := h.db.Create(&body).Error; err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "slug already exists"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"verification": body})
}

// Update modifies a verification type.
func (h *AdminVerificationHandler) Update(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	id := c.Params("id")
	var vt models.VerificationType
	if err := h.db.First(&vt, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	h.db.Model(&vt).Updates(body)
	h.db.First(&vt, id)
	return c.JSON(fiber.Map{"verification": vt})
}

// Delete removes a verification type.
func (h *AdminVerificationHandler) Delete(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}
	id := c.Params("id")
	if err := h.db.Delete(&models.VerificationType{}, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}
