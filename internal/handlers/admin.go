package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// AdminHandler handles admin-only endpoints.
type AdminHandler struct {
	db *gorm.DB
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// isAdmin checks if the current user has the admin role.
func isAdmin(c fiber.Ctx) bool {
	user := middleware.CurrentUser(c)
	return user != nil && user.Role == "admin"
}

// ListUsers handles GET /api/admin/users
func (h *AdminHandler) ListUsers(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	var users []models.User
	if err := h.db.Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch users"})
	}

	// Map to the shape expected by the frontend
	type UserRow struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Status   string `json:"status"`
		JoinedAt string `json:"joined_at"`
	}
	rows := make([]UserRow, len(users))
	for i, u := range users {
		rows[i] = UserRow{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			Role:     u.Role,
			Status:   u.Status,
			JoinedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(fiber.Map{"users": rows})
}

// UpdateUserRole handles PATCH /api/admin/users/:id/role
func (h *AdminHandler) UpdateUserRole(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	valid := map[string]bool{"admin": true, "team_owner": true, "team_manager": true, "user": true}
	if !valid[body.Role] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid role"})
	}

	id := c.Params("id")
	if err := h.db.Model(&models.User{}).Where("id = ?", id).Update("role", body.Role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update role"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// SuspendUser handles PATCH /api/admin/users/:id/suspend
func (h *AdminHandler) SuspendUser(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	if err := h.db.Model(&models.User{}).Where("id = ?", id).Update("status", "suspended").Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to suspend user"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// UnsuspendUser handles PATCH /api/admin/users/:id/unsuspend
func (h *AdminHandler) UnsuspendUser(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	if err := h.db.Model(&models.User{}).Where("id = ?", id).Update("status", "active").Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unsuspend user"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
