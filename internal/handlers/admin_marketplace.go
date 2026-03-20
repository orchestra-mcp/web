package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// AdminMarketplaceHandler handles marketplace admin endpoints.
type AdminMarketplaceHandler struct {
	db *gorm.DB
}

// NewAdminMarketplaceHandler creates a new AdminMarketplaceHandler.
func NewAdminMarketplaceHandler(db *gorm.DB) *AdminMarketplaceHandler {
	return &AdminMarketplaceHandler{db: db}
}

// ListPending returns community posts tagged "marketplace" with status "pending_review" or published.
// GET /api/admin/marketplace/pending
func (h *AdminMarketplaceHandler) ListPending(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	var posts []models.CommunityPost
	h.db.Where("tags LIKE '%marketplace%' AND parent_id IS NULL").
		Order("created_at desc").
		Find(&posts)

	type row struct {
		ID         uint   `json:"id"`
		Title      string `json:"title"`
		Content    string `json:"content"`
		Tags       string `json:"tags"`
		Status     string `json:"status"`
		UserID     uint   `json:"user_id"`
		UserName   string `json:"user_name"`
		CreatedAt  string `json:"created_at"`
		Approved   bool   `json:"approved"`
	}

	rows := make([]row, 0, len(posts))
	for _, p := range posts {
		var user models.User
		h.db.Select("id, name").First(&user, p.UserID)

		// Check if tags contain "marketplace_approved"
		approved := false
		var tags []string
		if err := json.Unmarshal([]byte(p.Tags), &tags); err == nil {
			for _, t := range tags {
				if t == "marketplace_approved" {
					approved = true
					break
				}
			}
		}

		rows = append(rows, row{
			ID:        p.ID,
			Title:     p.Title,
			Content:   p.Content,
			Tags:      p.Tags,
			Status:    p.Status,
			UserID:    p.UserID,
			UserName:  user.Name,
			CreatedAt: p.CreatedAt.Format("2006-01-02 15:04"),
			Approved:  approved,
		})
	}

	return c.JSON(fiber.Map{"submissions": rows})
}

// Approve adds the "marketplace_approved" tag to a post.
// POST /api/admin/marketplace/:id/approve
func (h *AdminMarketplaceHandler) Approve(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
	var post models.CommunityPost
	if err := h.db.First(&post, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	// Parse existing tags and add marketplace_approved
	var tags []string
	if err := json.Unmarshal([]byte(post.Tags), &tags); err != nil {
		tags = []string{}
	}

	// Remove marketplace_rejected if present, add marketplace_approved
	filtered := make([]string, 0, len(tags)+1)
	for _, t := range tags {
		if t != "marketplace_rejected" && t != "marketplace_approved" {
			filtered = append(filtered, t)
		}
	}
	filtered = append(filtered, "marketplace_approved")

	tagsJSON, _ := json.Marshal(filtered)
	h.db.Model(&post).Updates(map[string]interface{}{
		"tags":   string(tagsJSON),
		"status": "published",
	})

	reviewer := middleware.CurrentUser(c)
	reviewerName := ""
	if reviewer != nil {
		reviewerName = reviewer.Name
	}

	return c.JSON(fiber.Map{"message": "approved", "id": post.ID, "reviewer": reviewerName})
}

// Reject adds the "marketplace_rejected" tag to a post.
// POST /api/admin/marketplace/:id/reject
func (h *AdminMarketplaceHandler) Reject(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
	var post models.CommunityPost
	if err := h.db.First(&post, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	var body struct {
		Reason string `json:"reason"`
	}
	json.Unmarshal(c.Body(), &body)

	var tags []string
	if err := json.Unmarshal([]byte(post.Tags), &tags); err != nil {
		tags = []string{}
	}

	filtered := make([]string, 0, len(tags)+1)
	for _, t := range tags {
		if t != "marketplace_approved" && t != "marketplace_rejected" {
			filtered = append(filtered, t)
		}
	}
	filtered = append(filtered, "marketplace_rejected")

	tagsJSON, _ := json.Marshal(filtered)
	h.db.Model(&post).Update("tags", string(tagsJSON))

	return c.JSON(fiber.Map{"message": "rejected", "id": post.ID, "reason": body.Reason})
}
