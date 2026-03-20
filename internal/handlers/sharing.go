package handlers

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SharingHandler handles public content sharing for the community.
type SharingHandler struct {
	db *gorm.DB
}

// NewSharingHandler creates a new SharingHandler.
func NewSharingHandler(db *gorm.DB) *SharingHandler {
	return &SharingHandler{db: db}
}

// CreateShare handles POST /api/community/shares
func (h *SharingHandler) CreateShare(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		EntityType  string `json:"entity_type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Content     string `json:"content"`
		Visibility  string `json:"visibility"`
		Password    string `json:"password"`
		TeamID      *uint  `json:"team_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Title == "" || body.EntityType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and entity_type are required"})
	}

	visibility := body.Visibility
	if visibility == "" {
		visibility = "public"
	}

	slug := slugify(body.Title)

	// Ensure slug uniqueness for this user and entity type.
	var existing models.SharedContent
	base := slug
	counter := 1
	for {
		err := h.db.Where("user_id = ? AND entity_type = ? AND slug = ?", user.ID, body.EntityType, slug).
			First(&existing).Error
		if err != nil {
			break // not found — slug is available
		}
		counter++
		slug = base + "-" + strconv.Itoa(counter)
	}

	share := models.SharedContent{
		UserID:      user.ID,
		EntityType:  body.EntityType,
		Slug:        slug,
		Title:       body.Title,
		Description: body.Description,
		Content:     body.Content,
		Visibility:  visibility,
		TeamID:      body.TeamID,
	}

	// Hash password if provided.
	if body.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
		}
		share.Password = string(hash)
	}

	if err := h.db.Create(&share).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create share"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"share": share})
}

// ListMyShares handles GET /api/community/shares
func (h *SharingHandler) ListMyShares(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	entityType := c.Query("entity_type", "")

	query := h.db.Where("user_id = ?", user.ID)
	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}

	var shares []models.SharedContent
	query.Order("created_at desc").Find(&shares)

	return c.JSON(fiber.Map{"shares": shares})
}

// UpdateShare handles PUT /api/community/shares/:id
func (h *SharingHandler) UpdateShare(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var share models.SharedContent
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&share).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "share not found"})
	}

	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Content     *string `json:"content"`
		Visibility  *string `json:"visibility"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Title != nil {
		share.Title = *body.Title
		share.Slug = slugify(*body.Title)
	}
	if body.Description != nil {
		share.Description = *body.Description
	}
	if body.Content != nil {
		share.Content = *body.Content
	}
	if body.Visibility != nil {
		share.Visibility = *body.Visibility
	}

	if err := h.db.Save(&share).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update share"})
	}

	return c.JSON(fiber.Map{"share": share})
}

// DeleteShare handles DELETE /api/community/shares/:id
func (h *SharingHandler) DeleteShare(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var share models.SharedContent
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&share).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "share not found"})
	}

	if err := h.db.Delete(&share).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete share"})
	}

	return c.JSON(fiber.Map{"deleted": true})
}

// PublicShare handles GET /api/public/community/shares/:handle/:type/:slug
func (h *SharingHandler) PublicShare(c fiber.Ctx) error {
	handle := c.Params("handle")
	entityType := c.Params("type")
	slug := c.Params("slug")

	// Resolve user by handle (stored in settings JSONB).
	var user models.User
	if err := h.db.Where("settings->>'handle' = ?", handle).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	var share models.SharedContent
	if err := h.db.Where("user_id = ? AND entity_type = ? AND slug = ?",
		user.ID, entityType, slug).First(&share).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "share not found"})
	}

	// Access control: check visibility.
	if share.Visibility == "private" {
		viewer := middleware.CurrentUser(c)
		if viewer == nil || viewer.ID != user.ID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "this content is private"})
		}
	}

	// Access control: check team restriction.
	if share.TeamID != nil {
		viewer := middleware.CurrentUser(c)
		if viewer == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "team-only content requires login"})
		}
		// Check if viewer is a member of the team.
		var membership models.Membership
		if err := h.db.Where("user_id = ? AND team_id = ?", viewer.ID, *share.TeamID).
			First(&membership).Error; err != nil {
			// Allow the owner to view their own content.
			if viewer.ID != user.ID {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not a member of this team"})
			}
		}
	}

	// Access control: check password.
	if share.Password != "" {
		password := c.Query("password", "")
		if password == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":             "password required",
				"password_required": true,
			})
		}
		if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(password)); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "incorrect password"})
		}
	}

	// Increment view count.
	h.db.Model(&share).UpdateColumn("views_count", gorm.Expr("views_count + 1"))
	share.ViewsCount++

	return c.JSON(fiber.Map{
		"share": share,
		"author": fiber.Map{
			"id":         user.ID,
			"name":       user.Name,
			"handle":     handle,
			"avatar_url": user.AvatarURL,
		},
	})
}

// CloneShare handles POST /api/community/shares/:id/clone
func (h *SharingHandler) CloneShare(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var original models.SharedContent
	if err := h.db.Where("id = ? AND visibility = 'public'", id).First(&original).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "share not found"})
	}

	slug := slugify(original.Title)
	var existing models.SharedContent
	base := slug
	counter := 1
	for {
		if err := h.db.Where("user_id = ? AND entity_type = ? AND slug = ?", user.ID, original.EntityType, slug).
			First(&existing).Error; err != nil {
			break
		}
		counter++
		slug = base + "-" + strconv.Itoa(counter)
	}

	clone := models.SharedContent{
		UserID:      user.ID,
		EntityType:  original.EntityType,
		Slug:        slug,
		Title:       original.Title,
		Description: original.Description,
		Content:     original.Content,
		Visibility:  "unlisted",
	}

	if err := h.db.Create(&clone).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to clone"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"share": clone, "cloned_from": original.ID})
}

// AddShareComment handles POST /api/community/shares/:id/comments
func (h *SharingHandler) AddShareComment(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")

	var share models.SharedContent
	if err := h.db.Where("id = ?", id).First(&share).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "share not found"})
	}

	var body struct {
		Body string `json:"body"`
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body is required"})
	}

	kind := body.Kind
	if kind == "" {
		kind = "comment"
	}

	comment := models.ShareComment{
		ShareID: share.ID,
		UserID:  user.ID,
		Body:    body.Body,
		Kind:    kind,
	}

	if err := h.db.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add comment"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"comment": comment})
}

// ListShareComments handles GET /api/public/community/shares/:id/comments
func (h *SharingHandler) ListShareComments(c fiber.Ctx) error {
	id := c.Params("id")

	var comments []models.ShareComment
	h.db.Where("share_id = ?", id).
		Order("created_at desc").
		Find(&comments)

	// Enrich with user names.
	type commentRow struct {
		models.ShareComment
		AuthorName string `json:"author_name"`
		AvatarURL  string `json:"avatar_url"`
	}
	rows := make([]commentRow, 0, len(comments))
	for _, cm := range comments {
		var u models.User
		h.db.Select("name", "avatar_url").Where("id = ?", cm.UserID).First(&u)
		rows = append(rows, commentRow{
			ShareComment: cm,
			AuthorName:   u.Name,
			AvatarURL:    u.AvatarURL,
		})
	}

	return c.JSON(fiber.Map{"comments": rows, "count": len(rows)})
}

// ExportShare handles GET /api/community/shares/:id/export?format=md|txt|html
func (h *SharingHandler) ExportShare(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	format := c.Query("format", "md")

	var share models.SharedContent
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&share).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "share not found"})
	}

	switch format {
	case "txt":
		// Strip markdown syntax for plain text.
		plain := share.Content
		// Remove headers
		plain = stripMarkdownHeaders(plain)
		c.Set("Content-Type", "text/plain; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=\""+share.Slug+".txt\"")
		return c.SendString(plain)

	case "html":
		// Simple markdown to HTML conversion.
		html := markdownToHTML(share.Title, share.Content)
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=\""+share.Slug+".html\"")
		return c.SendString(html)

	default: // "md"
		c.Set("Content-Type", "text/markdown; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=\""+share.Slug+".md\"")
		md := "# " + share.Title + "\n\n"
		if share.Description != "" {
			md += "> " + share.Description + "\n\n"
		}
		md += share.Content
		return c.SendString(md)
	}
}

// stripMarkdownHeaders removes # prefixes from lines.
func stripMarkdownHeaders(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, "# ")
		if trimmed != line {
			lines[i] = trimmed
		}
	}
	return strings.Join(lines, "\n")
}

// markdownToHTML does minimal markdown to HTML conversion.
func markdownToHTML(title, content string) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>")
	b.WriteString(title)
	b.WriteString("</title><style>body{font-family:system-ui;max-width:800px;margin:40px auto;padding:0 20px;line-height:1.6;color:#333}h1,h2,h3{color:#111}pre{background:#f5f5f5;padding:16px;border-radius:8px;overflow-x:auto}code{background:#f0f0f0;padding:2px 6px;border-radius:4px}</style></head><body>")
	b.WriteString("<h1>")
	b.WriteString(title)
	b.WriteString("</h1>")

	lines := strings.Split(content, "\n")
	inPre := false
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inPre {
				b.WriteString("</code></pre>")
				inPre = false
			} else {
				b.WriteString("<pre><code>")
				inPre = true
			}
			continue
		}
		if inPre {
			b.WriteString(line)
			b.WriteString("\n")
			continue
		}
		if strings.HasPrefix(line, "### ") {
			b.WriteString("<h3>" + strings.TrimPrefix(line, "### ") + "</h3>")
		} else if strings.HasPrefix(line, "## ") {
			b.WriteString("<h2>" + strings.TrimPrefix(line, "## ") + "</h2>")
		} else if strings.HasPrefix(line, "# ") {
			b.WriteString("<h1>" + strings.TrimPrefix(line, "# ") + "</h1>")
		} else if strings.TrimSpace(line) == "" {
			b.WriteString("<br>")
		} else {
			b.WriteString("<p>" + line + "</p>")
		}
	}

	b.WriteString("</body></html>")
	return b.String()
}

// ListPublicShares handles GET /api/public/community/shares/:handle
func (h *SharingHandler) ListPublicShares(c fiber.Ctx) error {
	handle := c.Params("handle")
	entityType := c.Query("entity_type", "")

	// Resolve user by handle.
	var user models.User
	if err := h.db.Where("settings->>'handle' = ?", handle).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	query := h.db.Where("user_id = ? AND visibility = 'public'", user.ID)
	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}

	var shares []models.SharedContent
	query.Order("created_at desc").Find(&shares)

	return c.JSON(fiber.Map{
		"shares": shares,
		"author": fiber.Map{
			"id":         user.ID,
			"name":       user.Name,
			"handle":     handle,
			"avatar_url": user.AvatarURL,
		},
	})
}
