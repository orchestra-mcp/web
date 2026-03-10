package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSearchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create tables manually without gen_random_uuid() default
	db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT, email TEXT, password TEXT, role TEXT DEFAULT 'user',
		email_verified_at DATETIME, status TEXT DEFAULT 'active',
		password_set BOOLEAN DEFAULT 0, onboarding_completed_at DATETIME,
		settings TEXT, remember_token TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE projects (
		id TEXT PRIMARY KEY, user_id INTEGER, team_id TEXT,
		name TEXT, slug TEXT, description TEXT,
		sync_status TEXT DEFAULT 'not_synced', last_synced_at DATETIME,
		stats TEXT, meta TEXT, version INTEGER DEFAULT 1,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE features (
		id TEXT PRIMARY KEY, user_id INTEGER, team_id TEXT,
		project_slug TEXT, feature_id TEXT, title TEXT, description TEXT,
		status TEXT DEFAULT 'backlog', priority TEXT DEFAULT 'P2',
		assignee TEXT, labels TEXT, depends_on TEXT, blocks TEXT,
		estimate TEXT, body TEXT, version INTEGER DEFAULT 1, meta TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE notes (
		id TEXT PRIMARY KEY, user_id INTEGER, project_id TEXT, team_id TEXT,
		title TEXT, content TEXT, pinned BOOLEAN DEFAULT 0,
		tags TEXT, meta TEXT, version INTEGER DEFAULT 1,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE plans (
		id TEXT PRIMARY KEY, user_id INTEGER, team_id TEXT,
		project_slug TEXT, plan_id TEXT, title TEXT, description TEXT,
		status TEXT DEFAULT 'draft', body TEXT, version INTEGER DEFAULT 1, meta TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE docs (
		id TEXT PRIMARY KEY, user_id INTEGER, team_id TEXT,
		project_slug TEXT, doc_id TEXT, title TEXT, category TEXT,
		tags TEXT, parent_id TEXT, body TEXT, version INTEGER DEFAULT 1, meta TEXT,
		pinned BOOLEAN DEFAULT 0, icon TEXT, color TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE memberships (
		id TEXT PRIMARY KEY, user_id INTEGER, team_id TEXT,
		role TEXT DEFAULT 'member',
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)

	// Seed a user
	db.Exec(`INSERT INTO users (id, name, email) VALUES (1, 'Test User', 'test@test.com')`)

	// Seed projects
	db.Exec(`INSERT INTO projects (id, user_id, name, slug, description, updated_at) VALUES ('p1', 1, 'Orchestra MCP', 'orchestra-mcp', 'AI IDE platform', '2026-03-09 01:00:00')`)
	db.Exec(`INSERT INTO projects (id, user_id, name, slug, description, updated_at) VALUES ('p2', 1, 'Dashboard App', 'dashboard', 'Web dashboard for monitoring', '2026-03-09 02:00:00')`)

	// Seed features
	db.Exec(`INSERT INTO features (id, user_id, project_slug, feature_id, title, description, status, updated_at) VALUES ('f1', 1, 'orchestra-mcp', 'FEAT-ABC', 'Add dark mode', 'Implement dark theme', 'todo', '2026-03-09 01:00:00')`)
	db.Exec(`INSERT INTO features (id, user_id, project_slug, feature_id, title, description, status, updated_at) VALUES ('f2', 1, 'orchestra-mcp', 'FEAT-DEF', 'Search spotlight', 'Global CMD+K search', 'in-progress', '2026-03-09 02:00:00')`)

	// Seed notes
	db.Exec(`INSERT INTO notes (id, user_id, title, content, updated_at) VALUES ('n1', 1, 'Setup Guide', 'How to setup Orchestra locally', '2026-03-09 01:00:00')`)
	db.Exec(`INSERT INTO notes (id, user_id, title, content, updated_at) VALUES ('n2', 1, 'Meeting Notes', 'Discussed dark mode design', '2026-03-09 02:00:00')`)

	// Seed plans
	db.Exec(`INSERT INTO plans (id, user_id, project_slug, plan_id, title, description, status, updated_at) VALUES ('pl1', 1, 'orchestra-mcp', 'PLAN-AAA', 'V2 Roadmap', 'Roadmap for version 2 release', 'approved', '2026-03-09 01:00:00')`)
	db.Exec(`INSERT INTO plans (id, user_id, project_slug, plan_id, title, description, status, updated_at) VALUES ('pl2', 1, 'dashboard', 'PLAN-BBB', 'Search Overhaul', 'Revamp global search with plans and wiki', 'draft', '2026-03-09 02:00:00')`)

	// Seed wiki docs
	db.Exec(`INSERT INTO docs (id, user_id, project_slug, doc_id, title, category, body, updated_at) VALUES ('d1', 1, '_system', 'getting-started', 'Getting Started', 'guides', 'Welcome to Orchestra. This guide helps you get started.', '2026-03-09 01:00:00')`)
	db.Exec(`INSERT INTO docs (id, user_id, project_slug, doc_id, title, category, body, updated_at) VALUES ('d2', 1, '_system', 'dark-mode-spec', 'Dark Mode Specification', 'design', 'Technical spec for dark mode implementation', '2026-03-09 02:00:00')`)

	return db
}

// fakeAuthMiddleware injects a test user into the context.
func fakeAuthMiddleware(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		var user models.User
		db.First(&user)
		c.Locals("user", &user)
		return c.Next()
	}
}

func searchApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db := setupSearchTestDB(t)
	app := fiber.New()
	app.Use(fakeAuthMiddleware(db))
	h := NewSearchHandler(db)
	app.Get("/api/search", h.Search)
	app.Get("/api/search/suggestions", h.Suggestions)
	app.Post("/api/search/ai", h.AiSearch)
	return app, db
}

func doSearch(t *testing.T, app *fiber.App, query string) []any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/search?"+query, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	results, _ := body["results"].([]any)
	return results
}

func TestSearch_EmptyQuery(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=")
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestSearch_FindsProjects(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=orchestra&types=projects")
	if len(results) != 1 {
		t.Fatalf("expected 1 project result, got %d", len(results))
	}
	first := results[0].(map[string]any)
	if first["type"] != "project" {
		t.Errorf("expected type=project, got %s", first["type"])
	}
	if first["url"] != "/projects/orchestra-mcp" {
		t.Errorf("expected url=/projects/orchestra-mcp, got %s", first["url"])
	}
}

func TestSearch_FindsFeatures(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=dark&types=features")
	if len(results) != 1 {
		t.Fatalf("expected 1 feature result, got %d", len(results))
	}
	if results[0].(map[string]any)["type"] != "feature" {
		t.Errorf("expected type=feature")
	}
}

func TestSearch_FindsNotes(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=setup&types=notes")
	if len(results) != 1 {
		t.Fatalf("expected 1 note result, got %d", len(results))
	}
	if results[0].(map[string]any)["type"] != "note" {
		t.Errorf("expected type=note")
	}
}

func TestSearch_CrossTypeSearch(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=dark")
	// "dark" matches: feature (dark mode), note (dark mode design), doc (dark mode spec)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	types := map[string]bool{}
	for _, r := range results {
		types[r.(map[string]any)["type"].(string)] = true
	}
	if !types["feature"] || !types["note"] {
		t.Errorf("expected both feature and note types, got %v", types)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=ORCHESTRA&types=projects")
	if len(results) != 1 {
		t.Errorf("expected 1 result for case-insensitive search, got %d", len(results))
	}
}

func TestSearch_LimitParam(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=dark&limit=1")
	if len(results) > 1 {
		t.Errorf("expected at most 1 result with limit=1, got %d", len(results))
	}
}

func TestSearch_FeatureIDSearch(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=FEAT-ABC&types=features")
	if len(results) != 1 {
		t.Fatalf("expected 1 result searching by feature ID, got %d", len(results))
	}
}

func TestSearch_Suggestions(t *testing.T) {
	app, _ := searchApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/search/suggestions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	results := body["results"].([]any)
	// Should return recent projects (2) + features (2) + notes (2) + plans (2) + docs (2) = 10
	if len(results) < 6 {
		t.Errorf("expected at least 6 suggestions, got %d", len(results))
	}
	// Verify mixed types
	types := map[string]bool{}
	for _, r := range results {
		types[r.(map[string]any)["type"].(string)] = true
	}
	if !types["project"] || !types["feature"] || !types["note"] || !types["plan"] || !types["doc"] {
		t.Errorf("expected project, feature, note, plan, and doc types in suggestions, got %v", types)
	}
}

func TestSearch_AiSearch(t *testing.T) {
	app, _ := searchApp(t)
	body := `{"query":"what features are in progress?"}`
	req := httptest.NewRequest(http.MethodPost, "/api/search/ai", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	prompt, ok := result["prompt"].(string)
	if !ok || prompt == "" {
		t.Fatal("expected non-empty prompt in response")
	}
	if !strings.Contains(prompt, "FEAT-ABC") || !strings.Contains(prompt, "FEAT-DEF") {
		t.Error("expected prompt to contain feature IDs")
	}
	if !strings.Contains(prompt, "what features are in progress?") {
		t.Error("expected prompt to contain user query")
	}
	if result["context"] != "ai_search" {
		t.Errorf("expected context=ai_search, got %s", result["context"])
	}
}

func TestSearch_AiSearch_EmptyQuery(t *testing.T) {
	app, _ := searchApp(t)
	body := `{"query":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/search/ai", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for empty query, got %d", resp.StatusCode)
	}
}

func TestSearch_FindsPlans(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=roadmap&types=plans")
	if len(results) != 1 {
		t.Fatalf("expected 1 plan result, got %d", len(results))
	}
	first := results[0].(map[string]any)
	if first["type"] != "plan" {
		t.Errorf("expected type=plan, got %s", first["type"])
	}
	if first["url"] != "/plans?id=PLAN-AAA" {
		t.Errorf("expected url=/plans?id=PLAN-AAA, got %s", first["url"])
	}
}

func TestSearch_FindsPlansByID(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=PLAN-BBB&types=plans")
	if len(results) != 1 {
		t.Fatalf("expected 1 plan result searching by ID, got %d", len(results))
	}
}

func TestSearch_FindsDocs(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=getting+started&types=docs")
	if len(results) != 1 {
		t.Fatalf("expected 1 doc result, got %d", len(results))
	}
	first := results[0].(map[string]any)
	if first["type"] != "doc" {
		t.Errorf("expected type=doc, got %s", first["type"])
	}
	if first["url"] != "/wiki?file=getting-started" {
		t.Errorf("expected url=/wiki?file=getting-started, got %s", first["url"])
	}
}

func TestSearch_FindsDocsByCategory(t *testing.T) {
	app, _ := searchApp(t)
	results := doSearch(t, app, "q=design&types=docs")
	if len(results) != 1 {
		t.Fatalf("expected 1 doc result searching by category, got %d", len(results))
	}
}

func TestSearch_CrossTypeWithPlansAndDocs(t *testing.T) {
	app, _ := searchApp(t)
	// "dark" matches: feature (dark mode), note (dark mode design), doc (dark mode spec)
	results := doSearch(t, app, "q=dark")
	if len(results) < 3 {
		t.Fatalf("expected at least 3 results across types, got %d", len(results))
	}
	types := map[string]bool{}
	for _, r := range results {
		types[r.(map[string]any)["type"].(string)] = true
	}
	if !types["feature"] || !types["note"] || !types["doc"] {
		t.Errorf("expected feature, note, and doc types, got %v", types)
	}
}

func TestSearch_SuggestionsIncludePlansAndDocs(t *testing.T) {
	app, _ := searchApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/search/suggestions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	results := body["results"].([]any)
	types := map[string]bool{}
	for _, r := range results {
		types[r.(map[string]any)["type"].(string)] = true
	}
	if !types["plan"] {
		t.Errorf("expected plan type in suggestions, got %v", types)
	}
	if !types["doc"] {
		t.Errorf("expected doc type in suggestions, got %v", types)
	}
}

func TestSearch_AiSearchIncludesPlansAndDocs(t *testing.T) {
	app, _ := searchApp(t)
	body := `{"query":"what plans exist?"}`
	req := httptest.NewRequest(http.MethodPost, "/api/search/ai", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	prompt := result["prompt"].(string)
	if !strings.Contains(prompt, "PLAN-AAA") {
		t.Error("expected AI prompt to contain plan ID PLAN-AAA")
	}
	if !strings.Contains(prompt, "## Plans") {
		t.Error("expected AI prompt to contain ## Plans section")
	}
	if !strings.Contains(prompt, "## Wiki Docs") {
		t.Error("expected AI prompt to contain ## Wiki Docs section")
	}
	if !strings.Contains(prompt, "Getting Started") {
		t.Error("expected AI prompt to contain wiki doc title")
	}
}

func TestSearch_TeamScope(t *testing.T) {
	db := setupSearchTestDB(t)
	// Add a team project not owned by user 1
	db.Exec(`INSERT INTO projects (id, user_id, team_id, name, slug, description, updated_at) VALUES ('p3', 999, 'team-1', 'Team Project', 'team-proj', 'Shared team project', '2026-03-09 03:00:00')`)
	// Add user 1 as team member
	db.Exec(`INSERT INTO memberships (id, user_id, team_id) VALUES ('m1', 1, 'team-1')`)

	app := fiber.New()
	app.Use(fakeAuthMiddleware(db))
	h := NewSearchHandler(db)
	app.Get("/api/search", h.Search)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=team&types=projects", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	results := body["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("expected 1 team project result, got %d", len(results))
	}
	if results[0].(map[string]any)["title"] != "Team Project" {
		t.Errorf("expected Team Project, got %s", results[0].(map[string]any)["title"])
	}
}
