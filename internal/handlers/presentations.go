package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/services"
	"gorm.io/gorm"
)

type PresentationHandler struct {
	db      *gorm.DB
	exports *services.ExportService
}

func NewPresentationHandler(db *gorm.DB) *PresentationHandler {
	return &PresentationHandler{db: db, exports: services.NewExportService()}
}

// List handles GET /api/presentations
func (h *PresentationHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var presentations []models.Presentation
	h.db.Where("user_id = ?", user.ID).Order("updated_at desc").Find(&presentations)

	return c.JSON(fiber.Map{"presentations": presentations})
}

// Create handles POST /api/presentations
func (h *PresentationHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
		TeamID      string `json:"team_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}

	slug := slugify(body.Title)
	var existing models.Presentation
	base := slug
	counter := 1
	for {
		if err := h.db.Where("user_id = ? AND slug = ?", user.ID, slug).First(&existing).Error; err != nil {
			break
		}
		counter++
		slug = base + "-" + strconv.Itoa(counter)
	}

	pres := models.Presentation{
		UserID:      user.ID,
		Title:       body.Title,
		Slug:        slug,
		Description: body.Description,
		Visibility:  body.Visibility,
	}
	if pres.Visibility == "" {
		pres.Visibility = "private"
	}
	if body.TeamID != "" {
		pres.TeamID = &body.TeamID
	}

	if err := h.db.Create(&pres).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create presentation"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"presentation": pres})
}

// Show handles GET /api/presentations/:id
func (h *PresentationHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	var pres models.Presentation
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&pres).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "presentation not found"})
	}

	var slides []models.PresentationSlide
	h.db.Where("presentation_id = ?", pres.ID).Order("slide_number asc").Find(&slides)

	return c.JSON(fiber.Map{
		"presentation": pres,
		"slides":       slides,
	})
}

// Update handles PUT /api/presentations/:id
func (h *PresentationHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	var pres models.Presentation
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&pres).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "presentation not found"})
	}

	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Visibility  *string `json:"visibility"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Title != nil {
		pres.Title = *body.Title
		pres.Slug = slugify(*body.Title)
	}
	if body.Description != nil {
		pres.Description = *body.Description
	}
	if body.Visibility != nil {
		pres.Visibility = *body.Visibility
	}

	if err := h.db.Save(&pres).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update presentation"})
	}

	return c.JSON(fiber.Map{"presentation": pres})
}

// Delete handles DELETE /api/presentations/:id
func (h *PresentationHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	var pres models.Presentation
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&pres).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "presentation not found"})
	}

	h.db.Where("presentation_id = ?", pres.ID).Delete(&models.PresentationSlide{})
	h.db.Delete(&pres)

	return c.JSON(fiber.Map{"deleted": true})
}

// ── Slides CRUD ─────────────────────────────────────────────────────────────

// CreateSlide handles POST /api/presentations/:id/slides
func (h *PresentationHandler) CreateSlide(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	presID := c.Params("id")
	var pres models.Presentation
	if err := h.db.Where("id = ? AND user_id = ?", presID, user.ID).First(&pres).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "presentation not found"})
	}

	var body struct {
		Layout  string `json:"layout"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Notes   string `json:"notes"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Auto-assign next slide number.
	var maxNum int
	h.db.Model(&models.PresentationSlide{}).
		Where("presentation_id = ?", pres.ID).
		Select("COALESCE(MAX(slide_number), 0)").Scan(&maxNum)

	slide := models.PresentationSlide{
		PresentationID: pres.ID,
		UserID:         user.ID,
		SlideNumber:    maxNum + 1,
		Layout:         body.Layout,
		Title:          body.Title,
		Content:        body.Content,
		Notes:          body.Notes,
	}
	if slide.Layout == "" {
		slide.Layout = "title-content"
	}

	if err := h.db.Create(&slide).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create slide"})
	}

	// Update slide count.
	h.db.Model(&pres).UpdateColumn("slide_count", gorm.Expr("slide_count + 1"))

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"slide": slide})
}

// UpdateSlide handles PUT /api/presentations/:id/slides/:slideId
func (h *PresentationHandler) UpdateSlide(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slideID := c.Params("slideId")
	var slide models.PresentationSlide
	if err := h.db.Where("id = ? AND user_id = ?", slideID, user.ID).First(&slide).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "slide not found"})
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if v, ok := body["layout"].(string); ok {
		slide.Layout = v
	}
	if v, ok := body["title"].(string); ok {
		slide.Title = v
	}
	if v, ok := body["content"].(string); ok {
		slide.Content = v
	}
	if v, ok := body["notes"].(string); ok {
		slide.Notes = v
	}
	if v, ok := body["slide_number"].(float64); ok {
		slide.SlideNumber = int(v)
	}
	if v, ok := body["properties"]; ok {
		pBytes, _ := json.Marshal(v)
		slide.Properties = pBytes
	}

	if err := h.db.Save(&slide).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update slide"})
	}

	return c.JSON(fiber.Map{"slide": slide})
}

// DeleteSlide handles DELETE /api/presentations/:id/slides/:slideId
func (h *PresentationHandler) DeleteSlide(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	slideID := c.Params("slideId")
	var slide models.PresentationSlide
	if err := h.db.Where("id = ? AND user_id = ?", slideID, user.ID).First(&slide).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "slide not found"})
	}

	presID := slide.PresentationID
	h.db.Delete(&slide)

	// Update slide count.
	h.db.Model(&models.Presentation{}).Where("id = ?", presID).
		UpdateColumn("slide_count", gorm.Expr("GREATEST(slide_count - 1, 0)"))

	return c.JSON(fiber.Map{"deleted": true})
}

// ReorderSlides handles PUT /api/presentations/:id/slides/reorder
func (h *PresentationHandler) ReorderSlides(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	presID := c.Params("id")
	var pres models.Presentation
	if err := h.db.Where("id = ? AND user_id = ?", presID, user.ID).First(&pres).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "presentation not found"})
	}

	var body struct {
		Order []string `json:"order"` // slide IDs in desired order
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	for i, slideID := range body.Order {
		h.db.Model(&models.PresentationSlide{}).
			Where("id = ? AND presentation_id = ?", slideID, pres.ID).
			UpdateColumn("slide_number", i+1)
	}

	return c.JSON(fiber.Map{"reordered": true})
}

// ── Public endpoints ────────────────────────────────────────────────────────

// PublicList handles GET /api/public/presentations/:handle
func (h *PresentationHandler) PublicList(c fiber.Ctx) error {
	handle := c.Params("handle")

	var user models.User
	if err := h.db.Where("settings->>'handle' = ?", handle).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	var presentations []models.Presentation
	h.db.Where("user_id = ? AND visibility = 'public'", user.ID).
		Order("updated_at desc").Find(&presentations)

	return c.JSON(fiber.Map{
		"presentations": presentations,
		"author": fiber.Map{
			"id":         user.ID,
			"name":       user.Name,
			"handle":     handle,
			"avatar_url": user.AvatarURL,
		},
	})
}

// PublicShow handles GET /api/public/presentations/:handle/:slug
func (h *PresentationHandler) PublicShow(c fiber.Ctx) error {
	handle := c.Params("handle")
	slug := c.Params("slug")

	var user models.User
	if err := h.db.Where("settings->>'handle' = ?", handle).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	var pres models.Presentation
	if err := h.db.Where("user_id = ? AND slug = ? AND visibility = 'public'", user.ID, slug).First(&pres).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "presentation not found"})
	}

	var slides []models.PresentationSlide
	h.db.Where("presentation_id = ?", pres.ID).Order("slide_number asc").Find(&slides)

	return c.JSON(fiber.Map{
		"presentation": pres,
		"slides":       slides,
		"author": fiber.Map{
			"id":         user.ID,
			"name":       user.Name,
			"handle":     handle,
			"avatar_url": user.AvatarURL,
		},
	})
}

// Export handles GET /api/presentations/:id/export?format=pdf|pptx|gslides
func (h *PresentationHandler) Export(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	format := c.Query("format", "pdf")

	var pres models.Presentation
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&pres).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "presentation not found"})
	}

	var slides []models.PresentationSlide
	h.db.Where("presentation_id = ?", pres.ID).Order("slide_number asc").Find(&slides)

	switch format {
	case "gslides":
		url := h.exports.GoogleSlidesURL(pres)
		return c.JSON(fiber.Map{
			"format": "gslides",
			"url":    url,
			"title":  pres.Title,
			"slides": len(slides),
		})

	case "pptx":
		md := h.exports.GeneratePPTXMarkdown(pres, slides)
		c.Set("Content-Type", "text/markdown; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=\""+pres.Slug+".md\"")
		return c.SendString(md)

	default: // "pdf" — returns HTML for headless browser capture
		htmlContent := h.exports.RenderPresentationHTML(pres, slides)
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=\""+pres.Slug+".html\"")
		return c.SendString(htmlContent)
	}
}
