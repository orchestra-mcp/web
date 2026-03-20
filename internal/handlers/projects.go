package handlers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// uuidRE matches a UUID string (v4 or similar).
var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// projectByParam scopes a query to find a project by slug, or by UUID id if the param looks like one.
func projectByParam(q *gorm.DB, param string) *gorm.DB {
	if uuidRE.MatchString(param) {
		return q.Where("slug = ? OR id = ?", param, param)
	}
	return q.Where("slug = ?", param)
}

// resolveProjectSlug resolves a slug-or-UUID param to the actual project slug.
// Returns the param unchanged if it's already a slug or if no project is found.
func resolveProjectSlug(db *gorm.DB, param string) string {
	if !uuidRE.MatchString(param) {
		return param
	}
	var slug string
	db.Model(&models.Project{}).Where("id = ?", param).Pluck("slug", &slug)
	if slug != "" {
		return slug
	}
	return param
}

// ProjectHandler handles project CRUD endpoints.
type ProjectHandler struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(db *gorm.DB, wsHub ...*hub.Hub) *ProjectHandler {
	h := &ProjectHandler{db: db}
	if len(wsHub) > 0 {
		h.hub = wsHub[0]
	}
	return h
}

// slugify converts a name into a URL-safe slug.
func slugify(name string) string {
	lower := strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := re.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

// uniqueSlug ensures the slug is unique per user, appending -2, -3, etc. if needed.
func (h *ProjectHandler) uniqueSlug(base string, userID uint, excludeID string) string {
	slug := base
	for i := 2; ; i++ {
		var count int64
		q := h.db.Model(&models.Project{}).Where("slug = ? AND user_id = ?", slug, userID)
		if excludeID != "" {
			q = q.Where("id != ?", excludeID)
		}
		q.Count(&count)
		if count == 0 {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

// List handles GET /api/projects
func (h *ProjectHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var projects []models.Project
	if err := teamScopedProjects(h.db, user.ID).
		Order("created_at desc").
		Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch projects"})
	}

	// Also include any projects whose slug appears in the user's features/plans
	// (handles the case where projects were seeded under a different user_id).
	seen := make(map[string]bool, len(projects))
	for _, p := range projects {
		seen[p.Slug] = true
	}
	var featureSlugs []string
	h.db.Model(&models.Feature{}).
		Where("user_id = ?", user.ID).
		Distinct("project_slug").
		Pluck("project_slug", &featureSlugs)
	var missingSlug []string
	for _, s := range featureSlugs {
		if s != "" && !seen[s] {
			missingSlug = append(missingSlug, s)
		}
	}
	if len(missingSlug) > 0 {
		var extra []models.Project
		h.db.Where("slug IN ?", missingSlug).Find(&extra)
		projects = append(projects, extra...)
	}

	// Return slug as id so MCP tools (which use slug as project_id) work seamlessly.
	type projectRow struct {
		ID          string  `json:"id"`
		UUID        string  `json:"uuid"`
		Slug        string  `json:"slug"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		TeamID      *string `json:"team_id"`
		WorkspaceID *string `json:"workspace_id"`
		IsPublic    bool    `json:"is_public"`
		SyncStatus  string  `json:"sync_status"`
		Version     int     `json:"version"`
	}
	rows := make([]projectRow, 0, len(projects))
	for _, p := range projects {
		rows = append(rows, projectRow{
			ID:          p.Slug,
			UUID:        p.ID,
			Slug:        p.Slug,
			Name:        p.Name,
			Description: p.Description,
			TeamID:      p.TeamID,
			WorkspaceID: p.WorkspaceID,
			IsPublic:    p.IsPublic,
			SyncStatus:  p.SyncStatus,
			Version:     p.Version,
		})
	}
	return c.JSON(rows)
}

// Create handles POST /api/projects
func (h *ProjectHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		TeamID      *string `json:"team_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	slug := h.uniqueSlug(slugify(body.Name), user.ID, "")

	project := models.Project{
		UserID:      user.ID,
		TeamID:      body.TeamID,
		Name:        body.Name,
		Slug:        slug,
		Description: body.Description,
		SyncStatus:  "not_synced",
		Version:     1,
	}

	if err := h.db.Create(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create project"})
	}

	broadcastSync(h.hub, user.ID, "project", project.Slug, "upsert")
	return c.Status(fiber.StatusCreated).JSON(project)
}

// Show handles GET /api/projects/:slug
func (h *ProjectHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	param := c.Params("slug")
	var project models.Project
	if err := projectByParam(teamScopedProjects(h.db, user.ID), param).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var features []models.Feature
	teamScopedFeatures(h.db, user.ID).
		Where("project_slug = ?", project.Slug).
		Order("created_at desc").
		Find(&features)

	return c.JSON(fiber.Map{
		"id":          project.ID,
		"name":        project.Name,
		"slug":        project.Slug,
		"description": project.Description,
		"created_at":  project.CreatedAt,
		"updated_at":  project.UpdatedAt,
		"features":    features,
	})
}

// Stats handles GET /api/projects/:slug/stats — returns feature status counts.
func (h *ProjectHandler) Stats(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	param := c.Params("slug")
	var project models.Project
	if err := projectByParam(teamScopedProjects(h.db, user.ID), param).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	type statusCount struct {
		Status string
		Count  int64
	}
	var counts []statusCount
	teamScopedFeatures(h.db, user.ID).
		Model(&models.Feature{}).
		Select("status, count(*) as count").
		Where("project_slug = ?", project.Slug).
		Group("status").
		Scan(&counts)

	result := fiber.Map{
		"total":       int64(0),
		"done":        int64(0),
		"in_progress": int64(0),
		"backlog":     int64(0),
		"todo":        int64(0),
	}
	var total int64
	for _, sc := range counts {
		total += sc.Count
		switch sc.Status {
		case "done":
			result["done"] = sc.Count
		case "in-progress":
			result["in_progress"] = sc.Count
		case "backlog":
			result["backlog"] = sc.Count
		case "todo":
			result["todo"] = sc.Count
		}
	}
	result["total"] = total

	return c.JSON(result)
}

// Update handles PUT/PATCH /api/projects/:slug
func (h *ProjectHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	param := c.Params("slug")
	var project models.Project
	if err := projectByParam(teamScopedProjects(h.db, user.ID), param).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
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
		updates["slug"] = h.uniqueSlug(slugify(body.Name), user.ID, project.ID)
	}
	if body.Description != "" {
		updates["description"] = body.Description
	}

	if err := h.db.Model(&project).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update project"})
	}

	broadcastSync(h.hub, user.ID, "project", project.Slug, "upsert")
	return c.JSON(project)
}

// Delete handles DELETE /api/projects/:slug
func (h *ProjectHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	param := c.Params("slug")
	var project models.Project
	if err := projectByParam(teamScopedProjects(h.db, user.ID), param).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	if err := h.db.Delete(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete project"})
	}

	broadcastSync(h.hub, user.ID, "project", project.Slug, "delete")
	return c.JSON(fiber.Map{"ok": true})
}

// Share handles POST /api/projects/:slug/share — shares a project with a team
// and optionally makes it public.
func (h *ProjectHandler) Share(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", c.Params("slug"), user.ID).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var body struct {
		TeamID   string `json:"team_id"`
		IsPublic *bool  `json:"is_public"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}

	if body.TeamID != "" {
		var membership models.Membership
		if err := h.db.Where("team_id = ? AND user_id = ?", body.TeamID, user.ID).
			First(&membership).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "you are not a member of this team"})
		}
		updates["team_id"] = body.TeamID
	}

	if body.IsPublic != nil {
		updates["is_public"] = *body.IsPublic
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team_id or is_public is required"})
	}

	if err := h.db.Model(&project).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to share project"})
	}

	h.db.First(&project, "id = ?", project.ID)
	return c.JSON(project)
}

// Unshare handles DELETE /api/projects/:slug/share — removes team sharing
// and makes the project private.
func (h *ProjectHandler) Unshare(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ?", c.Params("slug"), user.ID).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	if err := h.db.Model(&project).Updates(map[string]interface{}{
		"team_id":   nil,
		"is_public": false,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unshare project"})
	}

	h.db.First(&project, "id = ?", project.ID)
	return c.JSON(project)
}

// PublicShow handles GET /api/public/projects/:user/:slug — public read-only
// project view with features, health, and activity. No auth required.
func (h *ProjectHandler) PublicShow(c fiber.Ctx) error {
	userSlug := c.Params("user")
	projectSlug := c.Params("slug")

	// Find the project owner by username or ID.
	var owner models.User
	if err := h.db.Where("username = ?", userSlug).First(&owner).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	var project models.Project
	if err := h.db.Where("slug = ? AND user_id = ? AND is_public = ?", projectSlug, owner.ID, true).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	// Load features for this project.
	var features []models.Feature
	h.db.Where("project_slug = ?", project.Slug).
		Order("created_at desc").
		Find(&features)

	// Compute health stats.
	total := len(features)
	done := 0
	inProgress := 0
	inReview := 0
	todo := 0
	for _, f := range features {
		switch f.Status {
		case "done":
			done++
		case "in-progress", "in-testing", "in-docs":
			inProgress++
		case "in-review":
			inReview++
		case "todo":
			todo++
		}
	}

	completionPct := 0
	if total > 0 {
		completionPct = int(float64(done) / float64(total) * 100)
	}

	// Build kanban columns.
	kanban := fiber.Map{
		"todo":        []fiber.Map{},
		"in-progress": []fiber.Map{},
		"in-review":   []fiber.Map{},
		"done":        []fiber.Map{},
	}
	for _, f := range features {
		col := f.Status
		switch col {
		case "in-testing", "in-docs":
			col = "in-progress"
		case "needs-edits":
			col = "in-progress"
		}
		entry := fiber.Map{
			"id":       f.ID,
			"title":    f.Title,
			"status":   f.Status,
			"priority": f.Priority,
		}
		if existing, ok := kanban[col]; ok {
			kanban[col] = append(existing.([]fiber.Map), entry)
		}
	}

	// Load team members if project has a team.
	var members []fiber.Map
	if project.TeamID != nil {
		var memberships []models.Membership
		h.db.Where("team_id = ?", *project.TeamID).Preload("User").Find(&memberships)
		for _, m := range memberships {
			members = append(members, fiber.Map{
				"name":   m.User.Name,
				"avatar": m.User.AvatarURL,
				"role":   m.Role,
			})
		}
	}

	return c.JSON(fiber.Map{
		"project": fiber.Map{
			"name":        project.Name,
			"slug":        project.Slug,
			"description": project.Description,
			"created_at":  project.CreatedAt,
		},
		"health": fiber.Map{
			"completion_pct": completionPct,
			"total":          total,
			"done":           done,
			"in_progress":    inProgress,
			"in_review":      inReview,
			"todo":           todo,
		},
		"kanban":  kanban,
		"members": members,
	})
}

// Tree handles GET /api/projects/:slug/tree
func (h *ProjectHandler) Tree(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var project models.Project
	if err := teamScopedProjects(h.db, user.ID).
		Where("slug = ?", c.Params("slug")).
		First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	// Load epics with stories and tasks
	var epics []models.Epic
	if err := h.db.Where("project_id = ?", project.ID).
		Order("position asc, created_at asc").
		Find(&epics).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load epics"})
	}

	for i := range epics {
		var stories []models.Story
		if err := h.db.Where("epic_id = ?", epics[i].ID).
			Order("position asc, created_at asc").
			Find(&stories).Error; err != nil {
			continue
		}
		for j := range stories {
			var tasks []models.Task
			h.db.Where("story_id = ?", stories[j].ID).
				Order("position asc, created_at asc").
				Find(&tasks)
			stories[j].Tasks = tasks
		}
		epics[i].Stories = stories
	}

	var features []models.Feature
	teamScopedFeatures(h.db, user.ID).
		Where("project_slug = ?", project.Slug).
		Order("created_at desc").
		Find(&features)

	return c.JSON(fiber.Map{
		"project": fiber.Map{
			"id":          project.ID,
			"name":        project.Name,
			"slug":        project.Slug,
			"description": project.Description,
			"created_at":  project.CreatedAt,
			"updated_at":  project.UpdatedAt,
			"features":    features,
		},
		"epics": epics,
	})
}
