package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WorkspaceHandler handles workspace CRUD endpoints.
type WorkspaceHandler struct {
	db *gorm.DB
}

// NewWorkspaceHandler creates a new WorkspaceHandler.
func NewWorkspaceHandler(db *gorm.DB) *WorkspaceHandler {
	return &WorkspaceHandler{db: db}
}

// List handles GET /api/workspaces
func (h *WorkspaceHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get user's team IDs
	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	var workspaces []models.Workspace
	query := h.db.Preload("Teams")
	if len(teamIDs) > 0 {
		// Workspaces owned by the user OR linked to any of their teams
		query = query.Where(
			"owner_id = ? OR id IN (?)",
			user.ID,
			h.db.Model(&models.WorkspaceTeam{}).Select("workspace_id").Where("team_id IN ?", teamIDs),
		)
	} else {
		query = query.Where("owner_id = ?", user.ID)
	}

	if err := query.Order("updated_at desc").Find(&workspaces).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch workspaces"})
	}

	return c.JSON(workspaces)
}

// Create handles POST /api/workspaces
func (h *WorkspaceHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name          string   `json:"name"`
		Folders       []string `json:"folders"`
		PrimaryFolder string   `json:"primary_folder"`
		TeamIDs       []string `json:"team_ids"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	foldersJSON, _ := json.Marshal(body.Folders)

	workspace := models.Workspace{
		OwnerID:       user.ID,
		Name:          body.Name,
		Folders:       datatypes.JSON(foldersJSON),
		PrimaryFolder: body.PrimaryFolder,
		Version:       1,
	}

	if err := h.db.Create(&workspace).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create workspace"})
	}

	// Create WorkspaceTeam entries
	for i, teamID := range body.TeamIDs {
		role := "editor"
		if i == 0 {
			role = "owner"
		}
		wt := models.WorkspaceTeam{
			WorkspaceID: workspace.ID,
			TeamID:      teamID,
			Role:        role,
		}
		h.db.Create(&wt)
	}

	// Reload with teams
	h.db.Preload("Teams").First(&workspace, "id = ?", workspace.ID)

	return c.Status(fiber.StatusCreated).JSON(workspace)
}

// Show handles GET /api/workspaces/:id
func (h *WorkspaceHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get user's team IDs
	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	var workspace models.Workspace
	query := h.db.Preload("Teams").Where("id = ?", c.Params("id"))
	if len(teamIDs) > 0 {
		query = query.Where(
			"owner_id = ? OR id IN (?)",
			user.ID,
			h.db.Model(&models.WorkspaceTeam{}).Select("workspace_id").Where("team_id IN ?", teamIDs),
		)
	} else {
		query = query.Where("owner_id = ?", user.ID)
	}

	if err := query.First(&workspace).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found"})
	}

	return c.JSON(workspace)
}

// Update handles PUT /api/workspaces/:id
func (h *WorkspaceHandler) Update(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get user's team IDs
	var teamIDs []string
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Pluck("team_id", &teamIDs)

	var workspace models.Workspace
	query := h.db.Where("id = ?", c.Params("id"))
	if len(teamIDs) > 0 {
		query = query.Where(
			"owner_id = ? OR id IN (?)",
			user.ID,
			h.db.Model(&models.WorkspaceTeam{}).Select("workspace_id").Where("team_id IN ?", teamIDs),
		)
	} else {
		query = query.Where("owner_id = ?", user.ID)
	}

	if err := query.First(&workspace).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found"})
	}

	var body struct {
		Name          string   `json:"name"`
		Folders       []string `json:"folders"`
		PrimaryFolder string   `json:"primary_folder"`
		Status        string   `json:"status"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Folders != nil {
		foldersJSON, _ := json.Marshal(body.Folders)
		updates["folders"] = datatypes.JSON(foldersJSON)
	}
	if body.PrimaryFolder != "" {
		updates["primary_folder"] = body.PrimaryFolder
	}
	if body.Status != "" {
		updates["status"] = body.Status
	}

	if err := h.db.Model(&workspace).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update workspace"})
	}

	return c.JSON(workspace)
}

// Delete handles DELETE /api/workspaces/:id
func (h *WorkspaceHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var workspace models.Workspace
	if err := h.db.Where("id = ? AND owner_id = ?", c.Params("id"), user.ID).
		First(&workspace).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found or not owner"})
	}

	// Remove workspace-team associations
	h.db.Where("workspace_id = ?", workspace.ID).Delete(&models.WorkspaceTeam{})

	if err := h.db.Delete(&workspace).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete workspace"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// AddTeam handles POST /api/workspaces/:id/teams
func (h *WorkspaceHandler) AddTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Verify workspace exists and user has access
	var workspace models.Workspace
	if err := h.db.Where("id = ? AND owner_id = ?", c.Params("id"), user.ID).
		First(&workspace).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found or not owner"})
	}

	var body struct {
		TeamID string `json:"team_id"`
		Role   string `json:"role"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.TeamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "team_id is required"})
	}

	role := body.Role
	if role == "" {
		role = "editor"
	}

	// Check not already linked
	var existing models.WorkspaceTeam
	if err := h.db.Where("workspace_id = ? AND team_id = ?", workspace.ID, body.TeamID).
		First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "team already linked to workspace"})
	}

	wt := models.WorkspaceTeam{
		WorkspaceID: workspace.ID,
		TeamID:      body.TeamID,
		Role:        role,
	}

	if err := h.db.Create(&wt).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add team"})
	}

	return c.Status(fiber.StatusCreated).JSON(wt)
}

// RemoveTeam handles DELETE /api/workspaces/:id/teams/:teamId
func (h *WorkspaceHandler) RemoveTeam(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Verify workspace exists and user is owner
	var workspace models.Workspace
	if err := h.db.Where("id = ? AND owner_id = ?", c.Params("id"), user.ID).
		First(&workspace).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found or not owner"})
	}

	result := h.db.Where("workspace_id = ? AND team_id = ?", workspace.ID, c.Params("teamId")).
		Delete(&models.WorkspaceTeam{})
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "team not linked to workspace"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// Sync handles POST /api/workspaces/sync
// Upserts a workspace from a desktop client, idempotent by local_id.
func (h *WorkspaceHandler) Sync(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name          string   `json:"name"`
		Folders       []string `json:"folders"`
		PrimaryFolder string   `json:"primary_folder"`
		Source        string   `json:"source"`
		LocalID       string   `json:"local_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if body.Name == "" || body.LocalID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and local_id required"})
	}

	foldersJSON, _ := json.Marshal(body.Folders)

	// Upsert: find existing workspace by metadata->local_id for this owner
	var existing models.Workspace
	err := h.db.Where(
		"owner_id = ? AND metadata->>'local_id' = ?", user.ID, body.LocalID,
	).First(&existing).Error

	if err == nil {
		// Update existing
		updates := map[string]interface{}{
			"name":           body.Name,
			"folders":        datatypes.JSON(foldersJSON),
			"primary_folder": body.PrimaryFolder,
		}
		h.db.Model(&existing).Updates(updates)
		return c.JSON(existing)
	}

	// Create new
	metaJSON, _ := json.Marshal(map[string]string{
		"source":   body.Source,
		"local_id": body.LocalID,
	})

	workspace := models.Workspace{
		OwnerID:       user.ID,
		Name:          body.Name,
		Folders:       datatypes.JSON(foldersJSON),
		PrimaryFolder: body.PrimaryFolder,
		Metadata:      datatypes.JSON(metaJSON),
		Version:       1,
	}

	if err := h.db.Create(&workspace).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create failed"})
	}

	return c.Status(fiber.StatusCreated).JSON(workspace)
}
