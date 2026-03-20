package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// PersonHandler handles person profile endpoints.
type PersonHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewPersonHandler creates a new PersonHandler.
func NewPersonHandler(db *gorm.DB, wsHub ...*hub.Hub) *PersonHandler {
	h := &PersonHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// List handles GET /api/persons
func (h *PersonHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	query := teamScopedPersons(h.db, user.ID).
		Order("created_at DESC")

	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var persons []models.Person
	if err := query.Find(&persons).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list persons"})
	}

	return c.JSON(fiber.Map{
		"items": persons,
		"meta": fiber.Map{
			"total":  len(persons),
			"limit":  100,
			"offset": 0,
		},
	})
}

// Show handles GET /api/persons/:id
func (h *PersonHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var person models.Person
	if err := teamScopedPersons(h.db, user.ID).
		Where("person_id = ?", id).
		First(&person).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "person not found"})
	}

	return c.JSON(fiber.Map{"person": person})
}
