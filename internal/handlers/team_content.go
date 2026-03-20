package handlers

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

type TeamContentHandler struct {
	db *gorm.DB
}

func NewTeamContentHandler(db *gorm.DB) *TeamContentHandler {
	return &TeamContentHandler{db: db}
}

// ListTeamContent handles GET /api/teams/:teamId/content
// Returns all shared content scoped to a team.
// Query params: entity_type, page, per_page
func (h *TeamContentHandler) ListTeamContent(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	teamID := c.Params("teamId")

	// Verify membership
	teamIDs := userTeamIDs(h.db, user.ID)
	isMember := false
	for _, id := range teamIDs {
		if id == teamID {
			isMember = true
			break
		}
	}
	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a team member"})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	entityType := c.Query("entity_type")

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	q := h.db.Model(&models.SharedContent{}).Where("team_id = ?", teamID)
	if entityType != "" {
		q = q.Where("entity_type = ?", entityType)
	}

	var total int64
	q.Count(&total)

	var items []models.SharedContent
	q.Order("updated_at desc").Offset((page - 1) * perPage).Limit(perPage).Find(&items)

	// Enrich with author info
	type contentRow struct {
		ID           uint   `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		EntityType   string `json:"entity_type"`
		Slug         string `json:"slug"`
		Visibility   string `json:"visibility"`
		ViewsCount   int    `json:"views_count"`
		AuthorName   string `json:"author_name"`
		AuthorAvatar string `json:"author_avatar"`
		UserID       uint   `json:"user_id"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	}

	rows := make([]contentRow, 0, len(items))
	for _, item := range items {
		var u models.User
		h.db.Select("id, name, avatar_url").First(&u, item.UserID)
		rows = append(rows, contentRow{
			ID: item.ID, Title: item.Title, Description: item.Description,
			EntityType: item.EntityType, Slug: item.Slug, Visibility: item.Visibility,
			ViewsCount: item.ViewsCount, UserID: item.UserID,
			AuthorName: u.Name, AuthorAvatar: u.AvatarURL,
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

// ShareWithTeam handles POST /api/teams/:teamId/content/:id/share
// Shares an existing content item with a team by setting its team_id.
func (h *TeamContentHandler) ShareWithTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	teamID := c.Params("teamId")
	contentID := c.Params("id")

	// Verify team membership
	teamIDs := userTeamIDs(h.db, user.ID)
	isMember := false
	for _, id := range teamIDs {
		if id == teamID {
			isMember = true
			break
		}
	}
	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a team member"})
	}

	// Find the content (must own it)
	var content models.SharedContent
	if err := h.db.Where("id = ? AND user_id = ?", contentID, user.ID).First(&content).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found or not yours"})
	}

	// Parse team_id as uint for the model
	tid, err := strconv.ParseUint(teamID, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid team id"})
	}
	teamUint := uint(tid)
	content.TeamID = &teamUint
	h.db.Save(&content)

	return c.JSON(fiber.Map{"shared": true, "team_id": teamID})
}

// UnshareFromTeam handles DELETE /api/teams/:teamId/content/:id/share
// Removes a content item from a team by clearing its team_id.
func (h *TeamContentHandler) UnshareFromTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	contentID := c.Params("id")

	var content models.SharedContent
	if err := h.db.Where("id = ? AND user_id = ?", contentID, user.ID).First(&content).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "content not found or not yours"})
	}

	content.TeamID = nil
	h.db.Save(&content)

	return c.JSON(fiber.Map{"unshared": true})
}

// TeamActivity handles GET /api/teams/:teamId/activity
// Returns recent content changes by team members (for activity feed widget).
func (h *TeamContentHandler) TeamActivity(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	teamID := c.Params("teamId")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit < 1 || limit > 50 {
		limit = 20
	}

	// Verify membership
	teamIDs := userTeamIDs(h.db, user.ID)
	isMember := false
	for _, id := range teamIDs {
		if id == teamID {
			isMember = true
			break
		}
	}
	if !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a team member"})
	}

	// Get recent team content, ordered by most recently updated
	var items []models.SharedContent
	h.db.Where("team_id = ?", teamID).
		Order("updated_at desc").
		Limit(limit).
		Find(&items)

	type activityRow struct {
		ID           uint   `json:"id"`
		Title        string `json:"title"`
		EntityType   string `json:"entity_type"`
		Slug         string `json:"slug"`
		AuthorName   string `json:"author_name"`
		AuthorAvatar string `json:"author_avatar"`
		UserID       uint   `json:"user_id"`
		UpdatedAt    string `json:"updated_at"`
	}

	rows := make([]activityRow, 0, len(items))
	for _, item := range items {
		var u models.User
		h.db.Select("id, name, avatar_url").First(&u, item.UserID)
		rows = append(rows, activityRow{
			ID: item.ID, Title: item.Title, EntityType: item.EntityType,
			Slug: item.Slug, UserID: item.UserID,
			AuthorName: u.Name, AuthorAvatar: u.AvatarURL,
			UpdatedAt: item.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return c.JSON(fiber.Map{"activities": rows})
}

// InlineComments handles GET /api/community/shares/:id/inline-comments
// Returns inline comments (change requests) for a shared content item.
func (h *TeamContentHandler) InlineComments(c fiber.Ctx) error {
	id := c.Params("id")

	var comments []models.ShareComment
	h.db.Where("share_id = ?", id).Order("created_at asc").Find(&comments)

	type commentRow struct {
		ID         uint   `json:"id"`
		Body       string `json:"body"`
		Kind       string `json:"kind"`
		AuthorName string `json:"author_name"`
		AvatarURL  string `json:"avatar_url"`
		UserID     uint   `json:"user_id"`
		CreatedAt  string `json:"created_at"`
	}

	rows := make([]commentRow, 0, len(comments))
	for _, cm := range comments {
		var u models.User
		h.db.Select("id, name, avatar_url").Where("id = ?", cm.UserID).First(&u)
		rows = append(rows, commentRow{
			ID: cm.ID, Body: cm.Body, Kind: cm.Kind,
			AuthorName: u.Name, AvatarURL: u.AvatarURL,
			UserID: cm.UserID,
			CreatedAt: cm.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return c.JSON(fiber.Map{"comments": rows, "count": len(rows)})
}

// AddInlineComment handles POST /api/community/shares/:id/inline-comments
func (h *TeamContentHandler) AddInlineComment(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid content id"})
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
		ShareID: uint(id),
		UserID:  user.ID,
		Body:    body.Body,
		Kind:    kind,
	}

	if err := h.db.Create(&comment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add comment"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"comment": comment})
}
