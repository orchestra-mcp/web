package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

var (
	uuidRegex       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	errTeamRejected = errors.New("team access rejected")
)

// TeamScopedHandler handles team-scoped API endpoints under /api/teams/:id/*.
type TeamScopedHandler struct {
	db *gorm.DB
}

// NewTeamScopedHandler creates a new TeamScopedHandler.
func NewTeamScopedHandler(db *gorm.DB) *TeamScopedHandler {
	return &TeamScopedHandler{db: db}
}

// requireTeamMember validates the user is a member of the specified team.
// Returns the team ID or an error. On failure the HTTP response is already
// written; callers should return nil when err != nil.
func (h *TeamScopedHandler) requireTeamMember(c fiber.Ctx) (string, *models.User, error) {
	user := middleware.CurrentUser(c)
	if user == nil {
		_ = c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		return "", nil, errTeamRejected
	}

	teamID := c.Params("id")
	if teamID == "" || !uuidRegex.MatchString(teamID) {
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "valid team ID is required"})
		return "", nil, errTeamRejected
	}

	var membership models.Membership
	if err := h.db.Where("team_id = ? AND user_id = ?", teamID, user.ID).
		First(&membership).Error; err != nil {
		_ = c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a member of this team"})
		return "", nil, errTeamRejected
	}

	return teamID, user, nil
}

// ─── Workspaces ──────────────────────────────────────────────────

// TeamWorkspaces handles GET /api/teams/:id/workspaces
func (h *TeamScopedHandler) TeamWorkspaces(c fiber.Ctx) error {
	teamID, _, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	// Get workspaces linked to this team via WorkspaceTeam pivot
	var wts []models.WorkspaceTeam
	if err := h.db.Where("team_id = ?", teamID).Find(&wts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch workspaces"})
	}

	wsIDs := make([]string, 0, len(wts))
	for _, wt := range wts {
		wsIDs = append(wsIDs, wt.WorkspaceID)
	}

	var workspaces []models.Workspace
	if len(wsIDs) > 0 {
		h.db.Where("id IN ?", wsIDs).Order("updated_at desc").Find(&workspaces)
	}

	// Compute project_count and member_count for each workspace
	type wsRow struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		Slug         string    `json:"slug"`
		TeamID       string    `json:"team_id"`
		Description  string    `json:"description"`
		ProjectCount int64     `json:"project_count"`
		MemberCount  int64     `json:"member_count"`
		Version      int       `json:"version"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
	}

	rows := make([]wsRow, 0, len(workspaces))
	for _, ws := range workspaces {
		var projCount int64
		h.db.Model(&models.Project{}).Where("workspace_id = ?", ws.ID).Count(&projCount)

		// member_count = team members (since workspace is team-scoped)
		var memberCount int64
		h.db.Model(&models.Membership{}).Where("team_id = ?", teamID).Count(&memberCount)

		slug := slugify(ws.Name)

		rows = append(rows, wsRow{
			ID:           ws.ID,
			Name:         ws.Name,
			Slug:         slug,
			TeamID:       teamID,
			Description:  ws.PrimaryFolder,
			ProjectCount: projCount,
			MemberCount:  memberCount,
			Version:      ws.Version,
			CreatedAt:    ws.CreatedAt,
			UpdatedAt:    ws.UpdatedAt,
		})
	}

	return c.JSON(fiber.Map{"workspaces": rows})
}

// CreateTeamWorkspace handles POST /api/teams/:id/workspaces
func (h *TeamScopedHandler) CreateTeamWorkspace(c fiber.Ctx) error {
	teamID, user, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	foldersJSON, _ := json.Marshal([]string{})

	workspace := models.Workspace{
		OwnerID:       user.ID,
		Name:          body.Name,
		PrimaryFolder: body.Description,
		Folders:       foldersJSON,
		Version:       1,
	}

	if err := h.db.Create(&workspace).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create workspace"})
	}

	// Link to the team
	wt := models.WorkspaceTeam{
		WorkspaceID: workspace.ID,
		TeamID:      teamID,
		Role:        "owner",
	}
	h.db.Create(&wt)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"workspace": fiber.Map{
			"id":            workspace.ID,
			"name":          workspace.Name,
			"slug":          slugify(workspace.Name),
			"team_id":       teamID,
			"description":   body.Description,
			"project_count": 0,
			"member_count":  0,
			"version":       workspace.Version,
			"created_at":    workspace.CreatedAt,
			"updated_at":    workspace.UpdatedAt,
		},
	})
}

// ─── Projects ────────────────────────────────────────────────────

// TeamProjects handles GET /api/teams/:id/projects
func (h *TeamScopedHandler) TeamProjects(c fiber.Ctx) error {
	teamID, _, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	var projects []models.Project
	query := h.db.Where("team_id = ?", teamID).Order("updated_at desc")

	// Optional workspace filter
	if wsID := c.Query("workspace_id"); wsID != "" {
		query = query.Where("workspace_id = ?", wsID)
	}

	if err := query.Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch projects"})
	}

	// Map to frontend shape
	type projRow struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Slug        string    `json:"slug"`
		Description string    `json:"description"`
		WorkspaceID *string   `json:"workspace_id"`
		TeamID      *string   `json:"team_id"`
		Version     int       `json:"version"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
	rows := make([]projRow, 0, len(projects))
	for _, p := range projects {
		rows = append(rows, projRow{
			ID:          p.ID,
			Name:        p.Name,
			Slug:        p.Slug,
			Description: p.Description,
			WorkspaceID: p.WorkspaceID,
			TeamID:      p.TeamID,
			Version:     p.Version,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	return c.JSON(fiber.Map{"projects": rows})
}

// WorkspaceProjects handles GET /api/workspaces/:id/projects
func (h *TeamScopedHandler) WorkspaceProjects(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	wsID := c.Params("id")

	var projects []models.Project
	if err := h.db.Where("workspace_id = ?", wsID).Order("updated_at desc").
		Find(&projects).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch projects"})
	}

	type projRow struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Slug        string    `json:"slug"`
		Description string    `json:"description"`
		WorkspaceID *string   `json:"workspace_id"`
		TeamID      *string   `json:"team_id"`
		Version     int       `json:"version"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
	rows := make([]projRow, 0, len(projects))
	for _, p := range projects {
		rows = append(rows, projRow{
			ID:          p.ID,
			Name:        p.Name,
			Slug:        p.Slug,
			Description: p.Description,
			WorkspaceID: p.WorkspaceID,
			TeamID:      p.TeamID,
			Version:     p.Version,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	return c.JSON(fiber.Map{"projects": rows})
}

// ─── Features ────────────────────────────────────────────────────

// TeamFeatures handles GET /api/teams/:id/features
// Used by My Tasks page — returns all features across team projects.
func (h *TeamScopedHandler) TeamFeatures(c fiber.Ctx) error {
	teamID, _, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	query := h.db.Where("team_id = ?", teamID).Order("updated_at desc")

	// Optional filters
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if assigneeID := c.Query("assignee_id"); assigneeID != "" {
		query = query.Where("assignee = ?", assigneeID)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if projectSlug := c.Query("project_slug"); projectSlug != "" {
		query = query.Where("project_slug = ?", projectSlug)
	}

	var features []models.Feature
	if err := query.Find(&features).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch features"})
	}

	// Map to frontend shape
	type featRow struct {
		ID         string    `json:"id"`
		ProjectID  string    `json:"project_id"`
		Title      string    `json:"title"`
		Body       string    `json:"body"`
		Status     string    `json:"status"`
		Priority   string    `json:"priority"`
		Kind       string    `json:"kind"`
		AssigneeID *uint     `json:"assignee_id"`
		Labels     []string  `json:"labels"`
		Estimate   string    `json:"estimate"`
		Version    int       `json:"version"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
	}

	rows := make([]featRow, 0, len(features))
	for _, f := range features {
		var labels []string
		_ = json.Unmarshal(f.Labels, &labels)
		if labels == nil {
			labels = []string{}
		}

		row := featRow{
			ID:        f.ID,
			ProjectID: f.ProjectSlug,
			Title:     f.Title,
			Body:      f.Body,
			Status:    f.Status,
			Priority:  f.Priority,
			Kind:      "feature",
			Labels:    labels,
			Estimate:  f.Estimate,
			Version:   f.Version,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		}

		// Parse assignee as a number if possible
		if f.Assignee != "" {
			var uid uint
			if _, err := fmt.Sscanf(f.Assignee, "%d", &uid); err == nil {
				row.AssigneeID = &uid
			}
		}

		rows = append(rows, row)
	}

	return c.JSON(fiber.Map{"features": rows})
}

// ProjectFeatures handles GET /api/projects/:id/features (by project UUID)
// Wrapper that returns features in the { features: [...] } envelope.
func (h *TeamScopedHandler) ProjectFeatures(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	projectID := c.Params("id")

	// Find the project (by ID or slug)
	var project models.Project
	if err := h.db.Where("id = ? OR slug = ?", projectID, projectID).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	query := h.db.Where("project_slug = ?", project.Slug).Order("updated_at desc")

	// Optional filters
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if assigneeID := c.Query("assignee_id"); assigneeID != "" {
		query = query.Where("assignee = ?", assigneeID)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if kind := c.Query("kind"); kind != "" {
		// filter on Meta JSONB for kind
		query = query.Where("meta->>'kind' = ?", kind)
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}

	var features []models.Feature
	if err := query.Find(&features).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch features"})
	}

	type featRow struct {
		ID         string    `json:"id"`
		ProjectID  string    `json:"project_id"`
		Title      string    `json:"title"`
		Body       string    `json:"body"`
		Status     string    `json:"status"`
		Priority   string    `json:"priority"`
		Kind       string    `json:"kind"`
		AssigneeID *uint     `json:"assignee_id"`
		Labels     []string  `json:"labels"`
		Estimate   string    `json:"estimate"`
		Version    int       `json:"version"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
	}

	rows := make([]featRow, 0, len(features))
	for _, f := range features {
		var labels []string
		_ = json.Unmarshal(f.Labels, &labels)
		if labels == nil {
			labels = []string{}
		}

		row := featRow{
			ID:        f.ID,
			ProjectID: f.ProjectSlug,
			Title:     f.Title,
			Body:      f.Body,
			Status:    f.Status,
			Priority:  f.Priority,
			Kind:      "feature",
			Labels:    labels,
			Estimate:  f.Estimate,
			Version:   f.Version,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		}

		if f.Assignee != "" {
			var uid uint
			if _, err := fmt.Sscanf(f.Assignee, "%d", &uid); err == nil {
				row.AssigneeID = &uid
			}
		}

		rows = append(rows, row)
	}

	return c.JSON(fiber.Map{"features": rows})
}

// UpdateFeature handles PATCH /api/features/:id
func (h *TeamScopedHandler) UpdateFeature(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	featureID := c.Params("id")

	var feature models.Feature
	if err := h.db.Where("id = ?", featureID).First(&feature).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if v, ok := body["title"].(string); ok {
		updates["title"] = v
	}
	if v, ok := body["description"].(string); ok {
		updates["description"] = v
	}
	if v, ok := body["status"].(string); ok {
		updates["status"] = v
	}
	if v, ok := body["priority"].(string); ok {
		updates["priority"] = v
	}
	if v, ok := body["body"].(string); ok {
		updates["body"] = v
	}
	if v, ok := body["estimate"].(string); ok {
		updates["estimate"] = v
	}
	// assignee_id (number) → stored as string in the assignee field
	if v, ok := body["assignee_id"]; ok {
		switch val := v.(type) {
		case float64:
			updates["assignee"] = fmt.Sprintf("%d", int(val))
		case string:
			updates["assignee"] = val
		}
	}
	if v, ok := body["assignee"].(string); ok {
		updates["assignee"] = v
	}

	if len(updates) > 0 {
		if err := h.db.Model(&feature).Updates(updates).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update feature"})
		}
	}

	// Reload
	h.db.First(&feature, "id = ?", feature.ID)

	// Return in frontend shape
	var labels []string
	_ = json.Unmarshal(feature.Labels, &labels)
	if labels == nil {
		labels = []string{}
	}

	result := fiber.Map{
		"id":         feature.ID,
		"project_id": feature.ProjectSlug,
		"title":      feature.Title,
		"body":       feature.Body,
		"status":     feature.Status,
		"priority":   feature.Priority,
		"kind":       "feature",
		"labels":     labels,
		"estimate":   feature.Estimate,
		"version":    feature.Version,
		"created_at": feature.CreatedAt,
		"updated_at": feature.UpdatedAt,
	}

	if feature.Assignee != "" {
		var uid uint
		if _, err := fmt.Sscanf(feature.Assignee, "%d", &uid); err == nil {
			result["assignee_id"] = uid
		}
	}

	return c.JSON(fiber.Map{"feature": result})
}

// ShowFeature handles GET /api/features/:id (flat alias)
func (h *TeamScopedHandler) ShowFeature(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	featureID := c.Params("id")
	var feature models.Feature
	if err := h.db.Where("id = ?", featureID).First(&feature).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	var labels []string
	_ = json.Unmarshal(feature.Labels, &labels)
	if labels == nil {
		labels = []string{}
	}

	result := fiber.Map{
		"id":         feature.ID,
		"project_id": feature.ProjectSlug,
		"title":      feature.Title,
		"body":       feature.Body,
		"status":     feature.Status,
		"priority":   feature.Priority,
		"kind":       "feature",
		"labels":     labels,
		"estimate":   feature.Estimate,
		"version":    feature.Version,
		"created_at": feature.CreatedAt,
		"updated_at": feature.UpdatedAt,
	}

	if feature.Assignee != "" {
		var uid uint
		if _, err := fmt.Sscanf(feature.Assignee, "%d", &uid); err == nil {
			result["assignee_id"] = uid
		}
	}

	return c.JSON(fiber.Map{"feature": result})
}

// DeleteFeature handles DELETE /api/features/:id (flat alias)
func (h *TeamScopedHandler) DeleteFeature(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	featureID := c.Params("id")
	if err := h.db.Where("id = ?", featureID).Delete(&models.Feature{}).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "feature not found"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// ─── Analytics ───────────────────────────────────────────────────

// TeamAnalytics handles GET /api/teams/:id/analytics
func (h *TeamScopedHandler) TeamAnalytics(c fiber.Ctx) error {
	teamID, _, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	// Optional workspace filter
	wsFilter := c.Query("workspace_id")

	// 1. Get team projects (team-scoped OR personal)
	user := middleware.CurrentUser(c)
	projectQ := h.db.Where("team_id = ? OR (team_id IS NULL AND user_id = ?)", teamID, user.ID)
	if wsFilter != "" {
		projectQ = projectQ.Where("workspace_id = ?", wsFilter)
	}

	var projects []models.Project
	projectQ.Order("name asc").Find(&projects)

	// 2. Build project health for each project
	type projectHealth struct {
		ID               string `json:"id"`
		Name             string `json:"name"`
		Health           string `json:"health"`
		CompletionPct    int    `json:"completionPct"`
		TotalFeatures    int    `json:"totalFeatures"`
		DoneFeatures     int    `json:"doneFeatures"`
		InProgressCount  int    `json:"inProgressCount"`
		BlockedCount     int    `json:"blockedCount"`
		ReviewQueueCount int    `json:"reviewQueueCount"`
	}

	projectHealths := make([]projectHealth, 0, len(projects))
	burndownData := map[string][]fiber.Map{}
	totalVelocity := 0

	for _, p := range projects {
		var features []models.Feature
		h.db.Where("project_slug = ?", p.Slug).Find(&features)

		total := len(features)
		done := 0
		inProgress := 0
		blocked := 0
		review := 0

		for _, f := range features {
			switch f.Status {
			case "done":
				done++
			case "in-progress":
				inProgress++
			case "needs-edits":
				blocked++
			case "in-review":
				review++
			}
		}

		pct := 0
		if total > 0 {
			pct = int(math.Round(float64(done) / float64(total) * 100))
		}

		health := "on-track"
		if blocked > 0 {
			health = "at-risk"
		}
		if total > 0 && float64(done)/float64(total) < 0.2 && inProgress == 0 {
			health = "behind"
		}

		projectHealths = append(projectHealths, projectHealth{
			ID:               p.ID,
			Name:             p.Name,
			Health:           health,
			CompletionPct:    pct,
			TotalFeatures:    total,
			DoneFeatures:     done,
			InProgressCount:  inProgress,
			BlockedCount:     blocked,
			ReviewQueueCount: review,
		})

		totalVelocity += done

		// Simple burndown for each project (remaining over last 7 days)
		now := time.Now()
		points := make([]fiber.Map, 7)
		remaining := total - done
		for i := 6; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			ideal := int(math.Round(float64(total) * (1 - float64(7-i)/7)))
			points[6-i] = fiber.Map{
				"date":      day.Format("Jan 02"),
				"remaining": remaining + i,
				"ideal":     ideal,
			}
		}
		burndownData[p.ID] = points
	}

	// 3. Velocity (completed per week, last 4 weeks)
	velocityData := make([]fiber.Map, 4)
	now := time.Now()
	for i := 3; i >= 0; i-- {
		weekStart := now.AddDate(0, 0, -7*(i+1))
		weekEnd := now.AddDate(0, 0, -7*i)

		var count int64
		q := h.db.Model(&models.Feature{}).Where("status = 'done'").
			Where("updated_at BETWEEN ? AND ?", weekStart, weekEnd)

		// Scope to team projects
		var projectSlugs []string
		for _, p := range projects {
			projectSlugs = append(projectSlugs, p.Slug)
		}
		if len(projectSlugs) > 0 {
			q = q.Where("project_slug IN ?", projectSlugs)
		}

		q.Count(&count)

		velocityData[3-i] = fiber.Map{
			"week":      weekStart.Format("Jan 02"),
			"completed": count,
		}
	}

	avgVelocity := 0.0
	for _, v := range velocityData {
		avgVelocity += float64(v["completed"].(int64))
	}
	if len(velocityData) > 0 {
		avgVelocity /= float64(len(velocityData))
	}

	// 4. Recent activity from sync_log
	var syncLogs []models.SyncLog
	logQ := h.db.Where("team_id = ?", teamID).Order("created_at desc").Limit(20)
	logQ.Find(&syncLogs)

	activities := make([]fiber.Map, 0, len(syncLogs))
	for _, sl := range syncLogs {
		action := "created"
		switch sl.Action {
		case "upsert":
			action = "created"
		case "delete":
			action = "completed"
		}
		activities = append(activities, fiber.Map{
			"id":           sl.ID,
			"timestamp":    sl.CreatedAt.Format(time.RFC3339),
			"featureId":    sl.EntityID,
			"featureTitle": sl.EntityType + " " + sl.EntityID,
			"projectId":    "",
			"projectName":  "",
			"action":       action,
		})
	}

	// 5. Workload (features per team member)
	var memberships []models.Membership
	h.db.Where("team_id = ?", teamID).Preload("User").Find(&memberships)

	workload := make([]fiber.Map, 0, len(memberships))
	for _, m := range memberships {
		assigneeStr := fmt.Sprintf("%d", m.UserID)
		statusCounts := map[string]int{}
		total := 0

		var memberFeatures []models.Feature
		mq := h.db.Where("assignee = ?", assigneeStr)
		if len(projects) > 0 {
			var slugs []string
			for _, p := range projects {
				slugs = append(slugs, p.Slug)
			}
			mq = mq.Where("project_slug IN ?", slugs)
		}
		mq.Find(&memberFeatures)

		for _, f := range memberFeatures {
			statusCounts[f.Status]++
			total++
		}

		workload = append(workload, fiber.Map{
			"person":       m.User.Name,
			"statusCounts": statusCounts,
			"total":        total,
		})
	}

	return c.JSON(fiber.Map{
		"projects":        projectHealths,
		"burndownData":    burndownData,
		"velocityData":    velocityData,
		"averageVelocity": avgVelocity,
		"activities":      activities,
		"workload":        workload,
	})
}

// ─── Activity ────────────────────────────────────────────────────

// TeamActivity handles GET /api/teams/:id/activity
func (h *TeamScopedHandler) TeamActivity(c fiber.Ctx) error {
	teamID, _, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	limit := 50
	if lStr := c.Query("limit"); lStr != "" {
		var l int
		if _, err := fmt.Sscanf(lStr, "%d", &l); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	var syncLogs []models.SyncLog
	query := h.db.Where("team_id = ?", teamID).Order("created_at desc").Limit(limit)

	if entityType := c.Query("entity_type"); entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}

	query.Find(&syncLogs)

	activities := make([]fiber.Map, 0, len(syncLogs))
	for _, sl := range syncLogs {
		action := "created"
		switch sl.Action {
		case "upsert":
			action = "created"
		case "delete":
			action = "completed"
		}
		activities = append(activities, fiber.Map{
			"id":           sl.ID,
			"timestamp":    sl.CreatedAt.Format(time.RFC3339),
			"featureId":    sl.EntityID,
			"featureTitle": sl.EntityType + " " + sl.EntityID,
			"projectId":    "",
			"projectName":  "",
			"action":       action,
			"entity_type":  sl.EntityType,
			"user_id":      sl.UserID,
		})
	}

	return c.JSON(fiber.Map{"activities": activities})
}

// ─── Skills (team-scoped) ────────────────────────────────────────

// TeamSkills handles GET /api/teams/:id/skills
func (h *TeamScopedHandler) TeamSkills(c fiber.Ctx) error {
	teamID, _, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	var skills []models.Skill
	h.db.Where("team_id = ? OR scope = 'public'", teamID).
		Order("updated_at desc").Find(&skills)

	return c.JSON(skills)
}

// ─── Agents (team-scoped) ────────────────────────────────────────

// TeamAgents handles GET /api/teams/:id/agents
func (h *TeamScopedHandler) TeamAgents(c fiber.Ctx) error {
	teamID, _, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	var agents []models.Agent
	h.db.Where("team_id = ? OR scope = 'public'", teamID).
		Order("updated_at desc").Find(&agents)

	return c.JSON(agents)
}

// ─── Notes (team-scoped) ─────────────────────────────────────────

// TeamNotes handles GET /api/teams/:id/notes
func (h *TeamScopedHandler) TeamNotes(c fiber.Ctx) error {
	teamID, user, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	var notes []models.Note
	h.db.Where("team_id = ? OR (user_id = ? AND (team_id IS NULL))", teamID, user.ID).
		Order("pinned desc, updated_at desc").Find(&notes)

	return c.JSON(notes)
}

// ─── Members (team-scoped) ───────────────────────────────────────

// TeamMembers handles GET /api/teams/:id/members
func (h *TeamScopedHandler) TeamMembers(c fiber.Ctx) error {
	teamID, _, err := h.requireTeamMember(c)
	if err != nil {
		return nil
	}

	var memberships []models.Membership
	h.db.Where("team_id = ?", teamID).Preload("User").Find(&memberships)

	type memberRow struct {
		ID        uint      `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		AvatarURL string    `json:"avatar_url"`
		Role      string    `json:"role"`
		JoinedAt  time.Time `json:"joined_at"`
	}

	rows := make([]memberRow, 0, len(memberships))
	for _, m := range memberships {
		rows = append(rows, memberRow{
			ID:        m.User.ID,
			Name:      m.User.Name,
			Email:     m.User.Email,
			AvatarURL: m.User.AvatarURL,
			Role:      m.Role,
			JoinedAt:  m.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"members": rows})
}

// ─── My Tasks ────────────────────────────────────────────────────

// MyTasks handles GET /api/my-tasks
// Returns features assigned to the current user across all projects.
func (h *TeamScopedHandler) MyTasks(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	assigneeStr := fmt.Sprintf("%d", user.ID)

	query := h.db.Where("assignee = ? OR assignee = ?", assigneeStr, user.Name).
		Order("updated_at desc")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}

	var features []models.Feature
	if err := query.Find(&features).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch tasks"})
	}

	// Resolve project names
	slugSet := map[string]bool{}
	for _, f := range features {
		slugSet[f.ProjectSlug] = true
	}
	slugs := make([]string, 0, len(slugSet))
	for s := range slugSet {
		slugs = append(slugs, s)
	}

	projectNames := map[string]string{}
	if len(slugs) > 0 {
		var projects []models.Project
		h.db.Select("slug, name").Where("slug IN ?", slugs).Find(&projects)
		for _, p := range projects {
			projectNames[p.Slug] = p.Name
		}
	}

	type taskRow struct {
		ID          string    `json:"id"`
		ProjectID   string    `json:"project_id"`
		ProjectName string    `json:"project_name"`
		Title       string    `json:"title"`
		Body        string    `json:"body"`
		Status      string    `json:"status"`
		Priority    string    `json:"priority"`
		Kind        string    `json:"kind"`
		AssigneeID  *uint     `json:"assignee_id"`
		Labels      []string  `json:"labels"`
		Estimate    string    `json:"estimate"`
		Version     int       `json:"version"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	rows := make([]taskRow, 0, len(features))
	for _, f := range features {
		var labels []string
		_ = json.Unmarshal(f.Labels, &labels)
		if labels == nil {
			labels = []string{}
		}

		row := taskRow{
			ID:          f.FeatureID,
			ProjectID:   f.ProjectSlug,
			ProjectName: projectNames[f.ProjectSlug],
			Title:       f.Title,
			Body:        f.Body,
			Status:      f.Status,
			Priority:    f.Priority,
			Kind:        "feature",
			Labels:      labels,
			Estimate:    f.Estimate,
			Version:     f.Version,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		}

		if f.Assignee != "" {
			var uid uint
			if _, err := fmt.Sscanf(f.Assignee, "%d", &uid); err == nil {
				row.AssigneeID = &uid
			}
		}

		rows = append(rows, row)
	}

	return c.JSON(fiber.Map{
		"items": rows,
		"meta": fiber.Map{
			"total":  len(rows),
			"limit":  100,
			"offset": 0,
		},
	})
}

// ─── Workspace Project Creation ──────────────────────────────────

// CreateWorkspaceProject handles POST /api/workspaces/:id/projects
func (h *TeamScopedHandler) CreateWorkspaceProject(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	wsID := c.Params("id")

	var workspace models.Workspace
	if err := h.db.Where("id = ?", wsID).First(&workspace).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found"})
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	slug := slugify(body.Name)

	project := models.Project{
		UserID:      user.ID,
		Name:        body.Name,
		Slug:        slug,
		Description: body.Description,
		WorkspaceID: &wsID,
		Version:     1,
	}

	var wt models.WorkspaceTeam
	if err := h.db.Where("workspace_id = ?", wsID).First(&wt).Error; err == nil {
		project.TeamID = &wt.TeamID
	}

	if err := h.db.Create(&project).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create project"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":           project.ID,
		"name":         project.Name,
		"slug":         project.Slug,
		"description":  project.Description,
		"workspace_id": project.WorkspaceID,
		"team_id":      project.TeamID,
		"version":      project.Version,
		"created_at":   project.CreatedAt,
		"updated_at":   project.UpdatedAt,
	})
}

// ─── Workspace CRUD (by ID) ─────────────────────────────────────

// UpdateWorkspace handles PATCH /api/workspaces/:id (new PATCH endpoint)
func (h *TeamScopedHandler) UpdateWorkspace(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	wsID := c.Params("id")

	var workspace models.Workspace
	if err := h.db.Where("id = ?", wsID).First(&workspace).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found"})
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
	if body.Description != "" {
		updates["primary_folder"] = body.Description
	}

	if len(updates) > 0 {
		h.db.Model(&workspace).Updates(updates)
	}

	// Reload
	h.db.First(&workspace, "id = ?", workspace.ID)

	return c.JSON(fiber.Map{
		"workspace": fiber.Map{
			"id":          workspace.ID,
			"name":        workspace.Name,
			"slug":        slugify(workspace.Name),
			"description": workspace.PrimaryFolder,
			"version":     workspace.Version,
			"created_at":  workspace.CreatedAt,
			"updated_at":  workspace.UpdatedAt,
		},
	})
}
