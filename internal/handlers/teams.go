package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// TeamHandler handles team management endpoints.
type TeamHandler struct {
	db *gorm.DB
}

// NewTeamHandler creates a new TeamHandler.
func NewTeamHandler(db *gorm.DB) *TeamHandler {
	return &TeamHandler{db: db}
}

// List handles GET /api/teams
func (h *TeamHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var memberships []models.Membership
	if err := h.db.Where("user_id = ?", user.ID).
		Preload("Team").
		Find(&memberships).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch teams"})
	}

	teams := make([]fiber.Map, 0, len(memberships))
	for _, m := range memberships {
		var memberCount int64
		h.db.Model(&models.Membership{}).Where("team_id = ?", m.TeamID).Count(&memberCount)
		teams = append(teams, fiber.Map{
			"team": fiber.Map{
				"id":           m.Team.ID,
				"name":         m.Team.Name,
				"slug":         m.Team.Slug,
				"plan":         m.Team.Plan,
				"member_count": memberCount,
				"created_at":   m.Team.CreatedAt,
			},
			"role": m.Role,
		})
	}

	return c.JSON(teams)
}

// Create handles POST /api/teams
func (h *TeamHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	base := slugify(body.Name)
	slug := base
	for i := 2; ; i++ {
		var count int64
		h.db.Model(&models.Team{}).Where("slug = ?", slug).Count(&count)
		if count == 0 {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}

	team := models.Team{
		Name: body.Name,
		Slug: slug,
	}

	if err := h.db.Create(&team).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create team"})
	}

	// Add creator as owner
	membership := models.Membership{
		UserID: user.ID,
		TeamID: team.ID,
		Role:   "owner",
	}
	h.db.Create(&membership)

	return c.Status(fiber.StatusCreated).JSON(team)
}

// Show handles GET /api/teams/:id
func (h *TeamHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var membership models.Membership
	if err := h.db.Where("team_id = ? AND user_id = ?", c.Params("id"), user.ID).
		Preload("Team").
		First(&membership).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "team not found"})
	}

	return c.JSON(membership.Team)
}

// MyTeam handles GET /api/team — returns the user's first team
func (h *TeamHandler) MyTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var membership models.Membership
	if err := h.db.Where("user_id = ?", user.ID).
		Preload("Team").
		First(&membership).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no team found"})
	}

	// Count members
	var memberCount int64
	h.db.Model(&models.Membership{}).Where("team_id = ?", membership.TeamID).Count(&memberCount)

	return c.JSON(fiber.Map{
		"team": fiber.Map{
			"id":           membership.Team.ID,
			"name":         membership.Team.Name,
			"slug":         membership.Team.Slug,
			"description":  "",
			"plan":         membership.Team.Plan,
			"member_count": memberCount,
			"created_at":   membership.Team.CreatedAt,
			"owner_id":     user.ID,
		},
	})
}

// MyTeamMembers handles GET /api/team/members
func (h *TeamHandler) MyTeamMembers(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Find user's team
	var myMembership models.Membership
	if err := h.db.Where("user_id = ?", user.ID).First(&myMembership).Error; err != nil {
		return c.JSON(fiber.Map{"members": []fiber.Map{}})
	}

	var memberships []models.Membership
	if err := h.db.Where("team_id = ?", myMembership.TeamID).
		Preload("User").
		Find(&memberships).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch members"})
	}

	type MemberRow struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Status   string `json:"status"`
		JoinedAt string `json:"joined_at"`
	}
	rows := make([]MemberRow, len(memberships))
	for i, m := range memberships {
		rows[i] = MemberRow{
			ID:       m.User.ID,
			Name:     m.User.Name,
			Email:    m.User.Email,
			Role:     m.User.Role,
			Status:   m.User.Status,
			JoinedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(fiber.Map{"members": rows})
}

// UpdateMyTeam handles PATCH /api/team
func (h *TeamHandler) UpdateMyTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var myMembership models.Membership
	if err := h.db.Where("user_id = ? AND role IN ?", user.ID, []string{"owner", "admin"}).
		First(&myMembership).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}

	if len(updates) > 0 {
		h.db.Model(&models.Team{}).Where("id = ?", myMembership.TeamID).Updates(updates)
	}

	return c.JSON(fiber.Map{"ok": true})
}

// Delete handles DELETE /api/teams/:id
func (h *TeamHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	teamID := c.Params("id")

	// Only the owner can delete a team
	var membership models.Membership
	if err := h.db.Where("team_id = ? AND user_id = ? AND role = ?", teamID, user.ID, "owner").
		First(&membership).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only the team owner can delete the team"})
	}

	// Delete all memberships first
	h.db.Where("team_id = ?", teamID).Delete(&models.Membership{})

	// Delete the team
	if err := h.db.Where("id = ?", teamID).Delete(&models.Team{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete team"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// Invite handles POST /api/teams/:id/invite
func (h *TeamHandler) Invite(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Ensure requesting user is owner or admin
	var membership models.Membership
	if err := h.db.Where("team_id = ? AND user_id = ? AND role IN ?",
		c.Params("id"), user.ID, []string{"owner", "admin"}).
		First(&membership).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
	}

	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	role := body.Role
	if role == "" {
		role = "member"
	}

	var invitee models.User
	if err := h.db.Where("email = ?", body.Email).First(&invitee).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	// Check not already a member
	var existing models.Membership
	if err := h.db.Where("team_id = ? AND user_id = ?", c.Params("id"), invitee.ID).
		First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "user is already a member"})
	}

	newMembership := models.Membership{
		UserID: invitee.ID,
		TeamID: c.Params("id"),
		Role:   role,
	}

	if err := h.db.Create(&newMembership).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to invite user"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
