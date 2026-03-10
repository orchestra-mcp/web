package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT, email TEXT, password TEXT, role TEXT DEFAULT 'user',
		email_verified_at DATETIME, status TEXT DEFAULT 'active',
		password_set BOOLEAN DEFAULT 0, onboarding_completed_at DATETIME,
		settings TEXT, remember_token TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE repo_workspaces (
		id TEXT PRIMARY KEY, user_id INTEGER, team_id TEXT,
		name TEXT, repo_url TEXT, repo_owner TEXT, repo_name TEXT,
		branch TEXT DEFAULT 'main', clone_path TEXT,
		status TEXT DEFAULT 'pending', last_sync_at DATETIME,
		commit_sha TEXT, error TEXT, description TEXT,
		language TEXT, stars INTEGER DEFAULT 0, topics TEXT,
		is_private BOOLEAN DEFAULT 0, meta TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE o_auth_accounts (
		id TEXT PRIMARY KEY, user_id INTEGER, provider TEXT,
		provider_id TEXT, provider_user_id TEXT,
		email TEXT, name TEXT, avatar TEXT,
		access_token TEXT, refresh_token TEXT, expires_at DATETIME,
		meta TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)

	db.Exec(`INSERT INTO users (id, name, email) VALUES (1, 'Test User', 'test@test.com')`)
	return db
}

func fakeRepoAuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		var user models.User
		db.First(&user)
		c.Locals("user", &user)
		return c.Next()
	}
}

func repoApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db := setupRepoTestDB(t)
	wsMgr := services.NewWorkspaceManager(db, t.TempDir())
	app := fiber.New()
	app.Use(fakeRepoAuthMiddleware(db))
	h := NewRepoWorkspaceHandler(db, wsMgr)
	app.Get("/api/repos", h.List)
	app.Post("/api/repos", h.Create)
	app.Get("/api/repos/:id", h.Show)
	app.Delete("/api/repos/:id", h.Delete)
	app.Post("/api/repos/:id/chat", h.Chat)
	return app, db
}

func TestRepoWorkspace_ListEmpty(t *testing.T) {
	app, _ := repoApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []any
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty list, got %d", len(results))
	}
}

func TestRepoWorkspace_ListWithData(t *testing.T) {
	app, db := repoApp(t)

	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-1', 1, 'owner/repo1', 'https://github.com/owner/repo1', 'owner', 'repo1', 'main', 'ready')`)
	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-2', 1, 'owner/repo2', 'https://github.com/owner/repo2', 'owner', 'repo2', 'main', 'pending')`)

	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []any
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestRepoWorkspace_Create(t *testing.T) {
	app, db := repoApp(t)

	// Create requires a GitHub OAuth account for the user.
	db.Exec(`INSERT INTO o_auth_accounts (id, user_id, provider, provider_user_id, access_token)
		VALUES ('oa-1', 1, 'github', 'gh-123', 'fake-token')`)

	body := `{"repo_owner":"octocat","repo_name":"hello-world","branch":"main"}`
	req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if result["repo_owner"] != "octocat" {
		t.Errorf("expected repo_owner=octocat, got %s", result["repo_owner"])
	}
	if result["repo_name"] != "hello-world" {
		t.Errorf("expected repo_name=hello-world, got %s", result["repo_name"])
	}
	if result["status"] != "pending" {
		t.Errorf("expected status=pending, got %s", result["status"])
	}

	// Verify the record exists in the database.
	var count int64
	db.Model(&models.RepoWorkspace{}).Where("repo_owner = ? AND repo_name = ?", "octocat", "hello-world").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 DB row, got %d", count)
	}
}

func TestRepoWorkspace_CreateMissingFields(t *testing.T) {
	app, _ := repoApp(t)

	body := `{"repo_owner":"","repo_name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing fields, got %d", resp.StatusCode)
	}
}

func TestRepoWorkspace_CreateNoGitHub(t *testing.T) {
	app, _ := repoApp(t)

	// No OAuth account inserted, so GitHub is not connected.
	body := `{"repo_owner":"octocat","repo_name":"hello-world"}`
	req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 when GitHub not connected, got %d", resp.StatusCode)
	}
}

func TestRepoWorkspace_Show(t *testing.T) {
	app, db := repoApp(t)

	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-show', 1, 'owner/myrepo', 'https://github.com/owner/myrepo', 'owner', 'myrepo', 'main', 'ready')`)

	req := httptest.NewRequest(http.MethodGet, "/api/repos/ws-show", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["id"] != "ws-show" {
		t.Errorf("expected id=ws-show, got %s", result["id"])
	}
	if result["repo_name"] != "myrepo" {
		t.Errorf("expected repo_name=myrepo, got %s", result["repo_name"])
	}
}

func TestRepoWorkspace_ShowNotFound(t *testing.T) {
	app, _ := repoApp(t)

	req := httptest.NewRequest(http.MethodGet, "/api/repos/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] != "workspace not found" {
		t.Errorf("expected error='workspace not found', got %s", result["error"])
	}
}

func TestRepoWorkspace_Delete(t *testing.T) {
	app, db := repoApp(t)

	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-del', 1, 'owner/delme', 'https://github.com/owner/delme', 'owner', 'delme', 'main', 'ready')`)

	req := httptest.NewRequest(http.MethodDelete, "/api/repos/ws-del", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["ok"] != true {
		t.Errorf("expected ok=true, got %v", result["ok"])
	}

	// Verify record is soft-deleted (GORM soft delete means it should not appear in normal queries).
	var count int64
	db.Model(&models.RepoWorkspace{}).Where("id = ?", "ws-del").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 rows after delete, got %d", count)
	}
}

func TestRepoWorkspace_DeleteNotFound(t *testing.T) {
	app, _ := repoApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/repos/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRepoWorkspace_ChatNotReady(t *testing.T) {
	app, db := repoApp(t)

	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-chat', 1, 'owner/chatrepo', 'https://github.com/owner/chatrepo', 'owner', 'chatrepo', 'main', 'pending')`)

	body := `{"prompt":"explain this codebase"}`
	req := httptest.NewRequest(http.MethodPost, "/api/repos/ws-chat/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	// RunPrompt returns an error when status is not "ready", handler returns 500.
	if resp.StatusCode != 500 {
		t.Fatalf("expected 500 for workspace not ready, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	errMsg, _ := result["error"].(string)
	if !strings.Contains(errMsg, "not ready") {
		t.Errorf("expected error to contain 'not ready', got %s", errMsg)
	}
}

func TestRepoWorkspace_ChatMissingPrompt(t *testing.T) {
	app, db := repoApp(t)

	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-chat2', 1, 'owner/chatrepo2', 'https://github.com/owner/chatrepo2', 'owner', 'chatrepo2', 'main', 'ready')`)

	body := `{"prompt":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/repos/ws-chat2/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for empty prompt, got %d", resp.StatusCode)
	}
}

func TestRepoWorkspace_ChatNotFound(t *testing.T) {
	app, _ := repoApp(t)

	body := `{"prompt":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/repos/nonexistent/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRepoWorkspace_UserScopedList(t *testing.T) {
	app, db := repoApp(t)

	// Insert a second user.
	db.Exec(`INSERT INTO users (id, name, email) VALUES (2, 'Other User', 'other@test.com')`)

	// Workspace for user 1.
	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-u1', 1, 'user1/repo', 'https://github.com/user1/repo', 'user1', 'repo', 'main', 'ready')`)

	// Workspace for user 2 (should not appear in user 1's list).
	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-u2', 2, 'user2/repo', 'https://github.com/user2/repo', 'user2', 'repo', 'main', 'ready')`)

	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var results []map[string]any
	json.NewDecoder(resp.Body).Decode(&results)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for user-scoped list, got %d", len(results))
	}
	if results[0]["id"] != "ws-u1" {
		t.Errorf("expected ws-u1, got %s", results[0]["id"])
	}
}

func TestRepoWorkspace_CreateDuplicate(t *testing.T) {
	app, db := repoApp(t)

	db.Exec(`INSERT INTO o_auth_accounts (id, user_id, provider, provider_user_id, access_token)
		VALUES ('oa-dup', 1, 'github', 'gh-456', 'fake-token')`)
	db.Exec(`INSERT INTO repo_workspaces (id, user_id, name, repo_url, repo_owner, repo_name, branch, status)
		VALUES ('ws-dup', 1, 'octocat/hello-world', 'https://github.com/octocat/hello-world', 'octocat', 'hello-world', 'main', 'ready')`)

	body := `{"repo_owner":"octocat","repo_name":"hello-world"}`
	req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409 for duplicate repo, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] != "repository already added" {
		t.Errorf("expected error='repository already added', got %s", result["error"])
	}
}
