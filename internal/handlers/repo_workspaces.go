package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/services"
	"gorm.io/gorm"
)

// RepoWorkspaceHandler handles repository workspace endpoints.
type RepoWorkspaceHandler struct {
	db      *gorm.DB
	manager *services.WorkspaceManager
}

// NewRepoWorkspaceHandler creates a new RepoWorkspaceHandler.
func NewRepoWorkspaceHandler(db *gorm.DB, manager *services.WorkspaceManager) *RepoWorkspaceHandler {
	return &RepoWorkspaceHandler{db: db, manager: manager}
}

// List handles GET /api/repos
func (h *RepoWorkspaceHandler) List(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var repos []models.RepoWorkspace
	if err := h.db.Where("user_id = ?", user.ID).
		Order("created_at desc").
		Find(&repos).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch repos"})
	}

	return c.JSON(repos)
}

// Create handles POST /api/repos
func (h *RepoWorkspaceHandler) Create(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		RepoOwner   string `json:"repo_owner"`
		RepoName    string `json:"repo_name"`
		Branch      string `json:"branch"`
		Description string `json:"description"`
		Language    string `json:"language"`
		Stars       int    `json:"stars"`
		IsPrivate   bool   `json:"is_private"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.RepoOwner == "" || body.RepoName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "repo_owner and repo_name are required"})
	}

	branch := body.Branch
	if branch == "" {
		branch = "main"
	}

	// Check for duplicate.
	var existing models.RepoWorkspace
	if err := h.db.Where("user_id = ? AND repo_owner = ? AND repo_name = ?",
		user.ID, body.RepoOwner, body.RepoName).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "repository already added"})
	}

	// Get GitHub token.
	var oauth models.OAuthAccount
	if err := h.db.Where("user_id = ? AND provider = ?", user.ID, "github").First(&oauth).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "GitHub account not connected"})
	}

	ws := models.RepoWorkspace{
		UserID:      user.ID,
		Name:        fmt.Sprintf("%s/%s", body.RepoOwner, body.RepoName),
		RepoURL:     fmt.Sprintf("https://github.com/%s/%s", body.RepoOwner, body.RepoName),
		RepoOwner:   body.RepoOwner,
		RepoName:    body.RepoName,
		Branch:      branch,
		Status:      "pending",
		Description: body.Description,
		Language:    body.Language,
		Stars:       body.Stars,
		IsPrivate:   body.IsPrivate,
	}

	if err := h.db.Create(&ws).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create workspace"})
	}

	// Clone in background.
	go func() {
		if err := h.manager.CloneRepo(&ws, oauth.AccessToken); err != nil {
			// Status is already set to "error" by CloneRepo
			return
		}
	}()

	return c.Status(fiber.StatusCreated).JSON(ws)
}

// Show handles GET /api/repos/:id
func (h *RepoWorkspaceHandler) Show(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var ws models.RepoWorkspace
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&ws).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found"})
	}

	return c.JSON(ws)
}

// Sync handles POST /api/repos/:id/sync
func (h *RepoWorkspaceHandler) Sync(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var ws models.RepoWorkspace
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&ws).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found"})
	}

	var oauth models.OAuthAccount
	if err := h.db.Where("user_id = ? AND provider = ?", user.ID, "github").First(&oauth).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "GitHub account not connected"})
	}

	// Pull first, then push.
	if err := h.manager.PullFromGitHub(&ws, oauth.AccessToken); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("pull failed: %v", err)})
	}

	if err := h.manager.SyncToGitHub(&ws, oauth.AccessToken); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("push failed: %v", err)})
	}

	// Reload.
	h.db.First(&ws, "id = ?", ws.ID)
	return c.JSON(ws)
}

// Chat handles POST /api/repos/:id/chat
func (h *RepoWorkspaceHandler) Chat(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var ws models.RepoWorkspace
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&ws).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found"})
	}

	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.Prompt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "prompt is required"})
	}

	result, err := h.manager.RunPrompt(&ws, body.Prompt)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"workspace_id": ws.ID,
		"prompt":       body.Prompt,
		"response":     result,
	})
}

// Delete handles DELETE /api/repos/:id
func (h *RepoWorkspaceHandler) Delete(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var ws models.RepoWorkspace
	if err := h.db.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).
		First(&ws).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "workspace not found"})
	}

	// Delete cloned directory.
	h.manager.DeleteWorkspace(&ws)

	if err := h.db.Delete(&ws).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete workspace"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// GitHubRepos handles GET /api/repos/github — lists user's GitHub repos.
func (h *RepoWorkspaceHandler) GitHubRepos(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var oauth models.OAuthAccount
	if err := h.db.Where("user_id = ? AND provider = ?", user.ID, "github").First(&oauth).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "GitHub account not connected"})
	}

	repos, err := fetchGitHubRepos(oauth.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("GitHub API error: %v", err)})
	}

	// Mark already-added repos.
	var existing []models.RepoWorkspace
	h.db.Where("user_id = ?", user.ID).Find(&existing)
	added := make(map[string]bool)
	for _, ws := range existing {
		added[ws.RepoOwner+"/"+ws.RepoName] = true
	}

	type repoResponse struct {
		Owner       string   `json:"owner"`
		Name        string   `json:"name"`
		FullName    string   `json:"full_name"`
		Description string   `json:"description"`
		Language    string   `json:"language"`
		Stars       int      `json:"stars"`
		IsPrivate   bool     `json:"is_private"`
		Branch      string   `json:"default_branch"`
		Topics      []string `json:"topics"`
		Added       bool     `json:"added"`
	}

	var result []repoResponse
	for _, r := range repos {
		result = append(result, repoResponse{
			Owner:       r.Owner.Login,
			Name:        r.Name,
			FullName:    r.FullName,
			Description: r.Description,
			Language:    r.Language,
			Stars:       r.Stars,
			IsPrivate:   r.Private,
			Branch:      r.DefaultBranch,
			Topics:      r.Topics,
			Added:       added[r.FullName],
		})
	}

	return c.JSON(result)
}

// --- GitHub API helpers ---

type ghRepo struct {
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Description   string   `json:"description"`
	Language      string   `json:"language"`
	Stars         int      `json:"stargazers_count"`
	Private       bool     `json:"private"`
	DefaultBranch string   `json:"default_branch"`
	Topics        []string `json:"topics"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func fetchGitHubRepos(token string) ([]ghRepo, error) {
	var allRepos []ghRepo
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/user/repos?per_page=100&page=%d&sort=updated&affiliation=owner,collaborator,organization_member", page)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var repos []ghRepo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		if len(repos) < 100 {
			break
		}
		page++
	}

	return allRepos, nil
}
