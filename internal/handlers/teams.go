package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// TeamHandler handles team management endpoints.
type TeamHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewTeamHandler creates a new TeamHandler.
func NewTeamHandler(db *gorm.DB, cfg ...*config.Config) *TeamHandler {
	h := &TeamHandler{db: db}
	if len(cfg) > 0 && cfg[0] != nil {
		h.cfg = cfg[0]
	}
	return h
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
				"avatar_url":   m.Team.AvatarURL,
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

// MyTeam handles GET /api/team — returns the user's active team.
// Accepts optional ?team_id query param to select a specific team.
// If no team_id is provided, returns the user's first team.
// If the user has no team yet, a default "Personal" team is auto-created.
func (h *TeamHandler) MyTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var membership models.Membership
	q := h.db.Where("user_id = ?", user.ID).Preload("Team")
	if teamID := c.Query("team_id"); teamID != "" {
		q = q.Where("team_id = ?", teamID)
	}
	if err := q.First(&membership).Error; err != nil {
		// Auto-create a default "Personal" team for the user.
		teamName := user.Name + "'s Team"
		if user.Name == "" {
			teamName = "Personal"
		}
		base := slugify(teamName)
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
			Name: teamName,
			Slug: slug,
		}
		if err := h.db.Create(&team).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create default team"})
		}

		membership = models.Membership{
			UserID: user.ID,
			TeamID: team.ID,
			Role:   "owner",
		}
		if err := h.db.Create(&membership).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create membership"})
		}
		membership.Team = team
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
			"avatar_url":   membership.Team.AvatarURL,
			"member_count": memberCount,
			"created_at":   membership.Team.CreatedAt,
			"owner_id":     user.ID,
		},
	})
}

// MyTeamMembers handles GET /api/team/members
// Accepts optional ?team_id query param to select a specific team.
func (h *TeamHandler) MyTeamMembers(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Find user's team
	var myMembership models.Membership
	q := h.db.Where("user_id = ?", user.ID)
	if teamID := c.Query("team_id"); teamID != "" {
		q = q.Where("team_id = ?", teamID)
	}
	if err := q.First(&myMembership).Error; err != nil {
		return c.JSON(fiber.Map{"members": []fiber.Map{}})
	}

	var memberships []models.Membership
	if err := h.db.Where("team_id = ?", myMembership.TeamID).
		Preload("User").
		Find(&memberships).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch members"})
	}

	type MemberRow struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		Status    string `json:"status"`
		AvatarURL string `json:"avatar_url"`
		JoinedAt  string `json:"joined_at"`
	}
	rows := make([]MemberRow, len(memberships))
	for i, m := range memberships {
		rows[i] = MemberRow{
			ID:        m.User.ID,
			Name:      m.User.Name,
			Email:     m.User.Email,
			Role:      m.User.Role,
			Status:    m.User.Status,
			AvatarURL: m.User.AvatarURL,
			JoinedAt:  m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(fiber.Map{"members": rows})
}

// UpdateMyTeam handles PATCH /api/team
// Accepts optional ?team_id query param to select a specific team.
func (h *TeamHandler) UpdateMyTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var myMembership models.Membership
	q := h.db.Where("user_id = ? AND role IN ?", user.ID, []string{"owner", "admin"})
	if teamID := c.Query("team_id"); teamID != "" {
		q = q.Where("team_id = ?", teamID)
	}
	if err := q.First(&myMembership).Error; err != nil {
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

	// Return the updated team
	var updatedTeam models.Team
	h.db.First(&updatedTeam, "id = ?", myMembership.TeamID)

	var memberCount int64
	h.db.Model(&models.Membership{}).Where("team_id = ?", myMembership.TeamID).Count(&memberCount)

	return c.JSON(fiber.Map{
		"ok": true,
		"team": fiber.Map{
			"id":           updatedTeam.ID,
			"name":         updatedTeam.Name,
			"slug":         updatedTeam.Slug,
			"description":  "",
			"plan":         updatedTeam.Plan,
			"avatar_url":   updatedTeam.AvatarURL,
			"member_count": memberCount,
			"created_at":   updatedTeam.CreatedAt,
			"owner_id":     user.ID,
		},
	})
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

// UploadTeamAvatar handles POST /api/team/avatar
// Accepts optional ?team_id query param to select a specific team.
func (h *TeamHandler) UploadTeamAvatar(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Must be owner or admin of the team
	var myMembership models.Membership
	q := h.db.Where("user_id = ? AND role IN ?", user.ID, []string{"owner", "admin"}).Preload("Team")
	if teamID := c.Query("team_id"); teamID != "" {
		q = q.Where("team_id = ?", teamID)
	}
	if err := q.First(&myMembership).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar file is required"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid file type, must be jpg/png/gif/webp"})
	}

	if file.Size > 2*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file too large, max 2MB"})
	}

	baseDir := "uploads"
	if h.cfg != nil && h.cfg.UploadDir != "" {
		baseDir = h.cfg.UploadDir
	}
	avatarDir := filepath.Join(baseDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create upload directory"})
	}

	filename := fmt.Sprintf("team-%s-%d%s", myMembership.TeamID, time.Now().Unix(), ext)
	savePath := filepath.Join(avatarDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}

	// Delete old avatar file if it exists
	if myMembership.Team.AvatarURL != "" && strings.HasPrefix(myMembership.Team.AvatarURL, "/uploads/avatars/") {
		oldFilename := strings.TrimPrefix(myMembership.Team.AvatarURL, "/uploads/avatars/")
		_ = os.Remove(filepath.Join(avatarDir, oldFilename))
	}

	avatarURL := "/uploads/avatars/" + filename
	if err := h.db.Model(&models.Team{}).Where("id = ?", myMembership.TeamID).Update("avatar_url", avatarURL).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update avatar"})
	}

	return c.JSON(fiber.Map{"ok": true, "avatar_url": avatarURL})
}

// ShowMember handles GET /api/team/members/:id — returns a single team member.
func (h *TeamHandler) ShowMember(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	memberUserID := c.Params("id")

	// Find the user's team
	var myMembership models.Membership
	q := h.db.Where("user_id = ?", user.ID)
	if teamID := c.Query("team_id"); teamID != "" {
		q = q.Where("team_id = ?", teamID)
	}
	if err := q.First(&myMembership).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "team not found"})
	}

	// Find the member in the same team
	var membership models.Membership
	if err := h.db.Where("team_id = ? AND user_id = ?", myMembership.TeamID, memberUserID).
		Preload("User").First(&membership).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "member not found"})
	}

	return c.JSON(fiber.Map{
		"id":         membership.User.ID,
		"name":       membership.User.Name,
		"email":      membership.User.Email,
		"role":       membership.Role,
		"status":     membership.User.Status,
		"avatar_url": membership.User.AvatarURL,
		"joined_at":  membership.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UpdateMemberRole handles PATCH /api/team/members/:id/role — team owner/admin updates a member's role.
func (h *TeamHandler) UpdateMemberRole(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	memberUserID := c.Params("id")

	// Find the user's team where they are owner or admin
	var myMembership models.Membership
	q := h.db.Where("user_id = ? AND role IN ?", user.ID, []string{"owner", "admin"})
	if teamID := c.Query("team_id"); teamID != "" {
		q = q.Where("team_id = ?", teamID)
	}
	if err := q.First(&myMembership).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "role is required"})
	}

	valid := map[string]bool{"owner": true, "admin": true, "member": true, "viewer": true}
	if !valid[body.Role] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid role"})
	}

	if err := h.db.Model(&models.Membership{}).
		Where("team_id = ? AND user_id = ?", myMembership.TeamID, memberUserID).
		Update("role", body.Role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update role"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// RemoveMember handles DELETE /api/team/members/:id — team owner/admin removes a member.
func (h *TeamHandler) RemoveMember(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	memberUserID := c.Params("id")

	// Find the user's team where they are owner or admin
	var myMembership models.Membership
	q := h.db.Where("user_id = ? AND role IN ?", user.ID, []string{"owner", "admin"})
	if teamID := c.Query("team_id"); teamID != "" {
		q = q.Where("team_id = ?", teamID)
	}
	if err := q.First(&myMembership).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
	}

	result := h.db.Where("team_id = ? AND user_id = ?", myMembership.TeamID, memberUserID).Delete(&models.Membership{})
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "member not found"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// ListAll handles GET /api/admin/teams — returns ALL teams system-wide (admin only).
func (h *TeamHandler) ListAll(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	var teams []models.Team
	if err := h.db.Order("created_at DESC").Find(&teams).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch teams"})
	}

	result := make([]fiber.Map, 0, len(teams))
	for _, t := range teams {
		var memberCount int64
		h.db.Model(&models.Membership{}).Where("team_id = ?", t.ID).Count(&memberCount)

		// Get owner info
		var owner models.Membership
		var ownerName string
		if err := h.db.Where("team_id = ? AND role = ?", t.ID, "owner").
			Preload("User").First(&owner).Error; err == nil && owner.User.ID != 0 {
			ownerName = owner.User.Name
		}

		result = append(result, fiber.Map{
			"id":           t.ID,
			"name":         t.Name,
			"slug":         t.Slug,
			"plan":         t.Plan,
			"avatar_url":   t.AvatarURL,
			"member_count": memberCount,
			"owner_name":   ownerName,
			"created_at":   t.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"teams": result})
}

// AdminShowTeam handles GET /api/admin/teams/:id — returns a single team with members (admin only).
func (h *TeamHandler) AdminShowTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	teamID := c.Params("id")
	var team models.Team
	if err := h.db.First(&team, "id = ?", teamID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "team not found"})
	}

	var memberCount int64
	h.db.Model(&models.Membership{}).Where("team_id = ?", teamID).Count(&memberCount)

	var memberships []models.Membership
	h.db.Where("team_id = ?", teamID).Preload("User").Find(&memberships)

	members := make([]fiber.Map, 0, len(memberships))
	for _, m := range memberships {
		members = append(members, fiber.Map{
			"id":         m.ID,
			"user_id":    m.UserID,
			"name":       m.User.Name,
			"email":      m.User.Email,
			"avatar_url": m.User.AvatarURL,
			"role":       m.Role,
			"joined_at":  m.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"team":         team,
		"members":      members,
		"member_count": memberCount,
	})
}

// AdminUpdateTeam handles PATCH /api/admin/teams/:id — update team details (admin only).
func (h *TeamHandler) AdminUpdateTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	teamID := c.Params("id")
	var team models.Team
	if err := h.db.First(&team, "id = ?", teamID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "team not found"})
	}

	var body struct {
		Name string `json:"name"`
		Plan string `json:"plan"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Plan != "" {
		updates["plan"] = body.Plan
	}
	if len(updates) > 0 {
		h.db.Model(&team).Updates(updates)
	}

	return c.JSON(fiber.Map{"ok": true})
}

// AdminRemoveMember handles DELETE /api/admin/teams/:id/members/:user_id (admin only).
func (h *TeamHandler) AdminRemoveMember(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	teamID := c.Params("id")
	userID := c.Params("user_id")

	if err := h.db.Where("team_id = ? AND user_id = ?", teamID, userID).Delete(&models.Membership{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to remove member"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// AdminUpdateMemberRole handles PATCH /api/admin/teams/:id/members/:user_id (admin only).
func (h *TeamHandler) AdminUpdateMemberRole(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	teamID := c.Params("id")
	userID := c.Params("user_id")

	var body struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "role is required"})
	}

	valid := map[string]bool{"owner": true, "admin": true, "member": true, "viewer": true}
	if !valid[body.Role] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid role"})
	}

	if err := h.db.Model(&models.Membership{}).Where("team_id = ? AND user_id = ?", teamID, userID).Update("role", body.Role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update role"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// AdminDeleteTeam handles DELETE /api/admin/teams/:id (admin only).
func (h *TeamHandler) AdminDeleteTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	teamID := c.Params("id")
	h.db.Where("team_id = ?", teamID).Delete(&models.Membership{})
	if err := h.db.Where("id = ?", teamID).Delete(&models.Team{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete team"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// AdminAddMember handles POST /api/admin/teams/:id/members — admin adds a user to a team.
func (h *TeamHandler) AdminAddMember(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	teamID := c.Params("id")
	var body struct {
		UserID uint   `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.UserID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id is required"})
	}
	if body.Role == "" {
		body.Role = "member"
	}

	// Check team exists
	var team models.Team
	if err := h.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "team not found"})
	}

	// Check user exists
	var target models.User
	if err := h.db.First(&target, body.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	// Check not already a member
	var existing models.Membership
	if err := h.db.Where("team_id = ? AND user_id = ?", teamID, body.UserID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "user is already a member of this team"})
	}

	m := models.Membership{
		UserID: body.UserID,
		TeamID: teamID,
		Role:   body.Role,
	}
	if err := h.db.Create(&m).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add member"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"ok": true})
}

// AdminUserTeams handles GET /api/admin/users/:id/memberships — list teams for a user with membership details.
func (h *TeamHandler) AdminUserTeams(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	userID := c.Params("id")
	var memberships []models.Membership
	h.db.Where("user_id = ?", userID).Preload("Team").Find(&memberships)

	type row struct {
		MembershipID string `json:"membership_id"`
		TeamID       string `json:"team_id"`
		TeamName     string `json:"team_name"`
		TeamPlan     string `json:"team_plan"`
		Role         string `json:"role"`
	}
	rows := make([]row, 0, len(memberships))
	for _, m := range memberships {
		rows = append(rows, row{
			MembershipID: m.ID,
			TeamID:       m.TeamID,
			TeamName:     m.Team.Name,
			TeamPlan:     m.Team.Plan,
			Role:         m.Role,
		})
	}

	return c.JSON(fiber.Map{"memberships": rows})
}

// AdminRemoveUserFromTeam handles DELETE /api/admin/users/:id/memberships/:team_id
func (h *TeamHandler) AdminRemoveUserFromTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil || user.Role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin only"})
	}

	userID := c.Params("id")
	teamID := c.Params("team_id")

	result := h.db.Where("user_id = ? AND team_id = ?", userID, teamID).Delete(&models.Membership{})
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "membership not found"})
	}

	return c.JSON(fiber.Map{"ok": true})
}
