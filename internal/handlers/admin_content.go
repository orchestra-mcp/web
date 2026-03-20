package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

type AdminContentHandler struct {
	db *gorm.DB
}

func NewAdminContentHandler(db *gorm.DB) *AdminContentHandler {
	return &AdminContentHandler{db: db}
}

// ListContent handles GET /api/admin/content
// Query params: page, per_page, search, entity_type, visibility, sort_by, sort_dir
func (h *AdminContentHandler) ListContent(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	search := c.Query("search")
	entityType := c.Query("entity_type")
	visibility := c.Query("visibility")
	sortBy := c.Query("sort_by", "created_at")
	sortDir := c.Query("sort_dir", "desc")

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Validate sort_by to prevent SQL injection
	allowedSorts := map[string]bool{"created_at": true, "updated_at": true, "title": true, "views_count": true, "likes_count": true, "entity_type": true, "visibility": true}
	if !allowedSorts[sortBy] {
		sortBy = "created_at"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	q := h.db.Model(&models.SharedContent{})
	if search != "" {
		q = q.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if entityType != "" {
		q = q.Where("entity_type = ?", entityType)
	}
	if visibility != "" {
		q = q.Where("visibility = ?", visibility)
	}

	var total int64
	q.Count(&total)

	var items []models.SharedContent
	q.Preload("User").Order(sortBy + " " + sortDir).Offset((page - 1) * perPage).Limit(perPage).Find(&items)

	// Build response with author info
	type contentRow struct {
		ID           uint   `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		EntityType   string `json:"entity_type"`
		EntityID     string `json:"entity_id"`
		Slug         string `json:"slug"`
		Visibility   string `json:"visibility"`
		ViewsCount   int    `json:"views_count"`
		LikesCount   int    `json:"likes_count"`
		UniqueViews  int    `json:"unique_views"`
		UserID       uint   `json:"user_id"`
		AuthorName   string `json:"author_name"`
		AuthorAvatar string `json:"author_avatar"`
		TeamID       *uint  `json:"team_id,omitempty"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}

	rows := make([]contentRow, 0, len(items))
	for _, item := range items {
		var user models.User
		h.db.Select("id, name, avatar_url").First(&user, item.UserID)
		rows = append(rows, contentRow{
			ID: item.ID, Title: item.Title, Description: item.Description,
			EntityType: item.EntityType, EntityID: item.EntityID, Slug: item.Slug,
			Visibility: item.Visibility, ViewsCount: item.ViewsCount,
			LikesCount: item.LikesCount, UniqueViews: item.UniqueViews,
			UserID: item.UserID, AuthorName: user.Name, AuthorAvatar: user.AvatarURL,
			TeamID: item.TeamID,
			CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: item.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return c.JSON(fiber.Map{
		"items":    rows,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// UpdateVisibility handles PATCH /api/admin/content/:id/visibility
func (h *AdminContentHandler) UpdateVisibility(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var body struct {
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Visibility != "public" && body.Visibility != "unlisted" && body.Visibility != "private" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "visibility must be public, unlisted, or private"})
	}

	var item models.SharedContent
	if err := h.db.First(&item, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found"})
	}

	item.Visibility = body.Visibility
	h.db.Save(&item)

	return c.JSON(fiber.Map{"item": item})
}

// DeleteContent handles DELETE /api/admin/content/:id
func (h *AdminContentHandler) DeleteContent(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var item models.SharedContent
	if err := h.db.First(&item, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found"})
	}

	h.db.Delete(&item)
	return c.JSON(fiber.Map{"deleted": true})
}

// BulkAction handles POST /api/admin/content/bulk
// Body: { "ids": [1,2,3], "action": "publish" | "unpublish" | "delete" }
func (h *AdminContentHandler) BulkAction(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	var body struct {
		IDs    []uint `json:"ids"`
		Action string `json:"action"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if len(body.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ids required"})
	}

	switch body.Action {
	case "publish":
		h.db.Model(&models.SharedContent{}).Where("id IN ?", body.IDs).Update("visibility", "public")
	case "unpublish":
		h.db.Model(&models.SharedContent{}).Where("id IN ?", body.IDs).Update("visibility", "private")
	case "delete":
		h.db.Where("id IN ?", body.IDs).Delete(&models.SharedContent{})
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "action must be publish, unpublish, or delete"})
	}

	return c.JSON(fiber.Map{"affected": len(body.IDs), "action": body.Action})
}
