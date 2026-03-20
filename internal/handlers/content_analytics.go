package handlers

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

type ContentAnalyticsHandler struct {
	db *gorm.DB
}

func NewContentAnalyticsHandler(db *gorm.DB) *ContentAnalyticsHandler {
	return &ContentAnalyticsHandler{db: db}
}

// RecordView handles POST /api/public/community/shares/:id/view
// Records a page view. Body is empty, uses request headers for metadata.
func (h *ContentAnalyticsHandler) RecordView(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid content id"})
	}

	// Verify content exists
	var content models.SharedContent
	if err := h.db.First(&content, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found"})
	}

	// Hash IP for privacy
	ip := c.IP()
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(ip)))

	view := models.ContentView{
		ContentID:  uint(id),
		ViewerHash: hash,
		UserAgent:  string(c.Request().Header.UserAgent()),
		Referer:    c.Get("Referer"),
	}
	h.db.Create(&view)

	// Update aggregate counters on shared_content
	h.db.Model(&content).UpdateColumn("views_count", gorm.Expr("views_count + 1"))

	// Check if this is a unique viewer (first time this hash viewed this content)
	var existingCount int64
	h.db.Model(&models.ContentView{}).Where("content_id = ? AND viewer_hash = ?", id, hash).Count(&existingCount)
	if existingCount <= 1 { // This is their first view (the one we just created)
		h.db.Model(&content).UpdateColumn("unique_views", gorm.Expr("unique_views + 1"))
	}

	return c.JSON(fiber.Map{"recorded": true})
}

// GetAnalytics handles GET /api/community/shares/:id/analytics
// Returns view analytics for the content owner. Requires auth.
// Query params: period=7d|30d|90d (default 30d)
func (h *ContentAnalyticsHandler) GetAnalytics(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid content id"})
	}

	// Verify ownership
	var content models.SharedContent
	if err := h.db.First(&content, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found"})
	}
	if content.UserID != user.ID && !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not your content"})
	}

	period := c.Query("period", "30d")
	days := 30
	switch period {
	case "7d":
		days = 7
	case "90d":
		days = 90
	}

	since := time.Now().AddDate(0, 0, -days)

	// Total views in period
	var totalViews int64
	h.db.Model(&models.ContentView{}).Where("content_id = ? AND created_at >= ?", id, since).Count(&totalViews)

	// Unique visitors in period
	var uniqueVisitors int64
	h.db.Model(&models.ContentView{}).Where("content_id = ? AND created_at >= ?", id, since).
		Distinct("viewer_hash").Count(&uniqueVisitors)

	// Views by day (for time-series chart)
	type dayRow struct {
		Date  string `json:"date"`
		Views int64  `json:"views"`
	}
	var dailyViews []dayRow
	h.db.Raw(`
		SELECT DATE(created_at) as date, COUNT(*) as views
		FROM content_views
		WHERE content_id = ? AND created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date
	`, id, since).Scan(&dailyViews)

	// Top referers
	type refRow struct {
		Referer string `json:"referer"`
		Count   int64  `json:"count"`
	}
	var topReferers []refRow
	h.db.Raw(`
		SELECT COALESCE(NULLIF(referer, ''), '(direct)') as referer, COUNT(*) as count
		FROM content_views
		WHERE content_id = ? AND created_at >= ?
		GROUP BY referer
		ORDER BY count DESC
		LIMIT 10
	`, id, since).Scan(&topReferers)

	return c.JSON(fiber.Map{
		"content_id":      id,
		"period":          period,
		"total_views":     totalViews,
		"unique_visitors": uniqueVisitors,
		"all_time_views":  content.ViewsCount,
		"all_time_unique": content.UniqueViews,
		"daily_views":     dailyViews,
		"top_referers":    topReferers,
	})
}
