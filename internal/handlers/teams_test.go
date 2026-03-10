package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTeamTestDB(t *testing.T) *gorm.DB {
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
		settings TEXT, remember_token TEXT, avatar_url TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE teams (
		id TEXT PRIMARY KEY, name TEXT, slug TEXT, plan TEXT DEFAULT 'free',
		avatar_url TEXT, settings TEXT, meta TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE memberships (
		id TEXT PRIMARY KEY, user_id INTEGER, team_id TEXT,
		role TEXT DEFAULT 'member',
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)

	db.Exec(`INSERT INTO users (id, name, email) VALUES (1, 'Test User', 'test@test.com')`)
	db.Exec(`INSERT INTO teams (id, name, slug, plan) VALUES ('team-1', 'Test Team', 'test-team', 'free')`)
	db.Exec(`INSERT INTO memberships (id, user_id, team_id, role) VALUES ('m-1', 1, 'team-1', 'owner')`)

	return db
}

func teamApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db := setupTeamTestDB(t)
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		var user models.User
		db.First(&user)
		c.Locals("user", &user)
		return c.Next()
	})
	h := NewTeamHandler(db)
	app.Get("/api/team", h.MyTeam)
	app.Patch("/api/team", h.UpdateMyTeam)
	app.Post("/api/team/avatar", h.UploadTeamAvatar)
	return app, db
}

func TestUploadTeamAvatar_Success(t *testing.T) {
	app, _ := teamApp(t)

	// Clean up uploads after test
	defer os.RemoveAll("uploads")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("avatar", "logo.png")
	// Write a minimal valid PNG (1x1 pixel)
	part.Write([]byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG header
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
	})
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/team/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

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
		t.Fatalf("expected ok=true, got %v", result["ok"])
	}
	avatarURL, _ := result["avatar_url"].(string)
	if avatarURL == "" {
		t.Fatal("expected avatar_url in response")
	}
	if len(avatarURL) < 10 {
		t.Fatalf("avatar_url too short: %s", avatarURL)
	}
}

func TestUploadTeamAvatar_InvalidType(t *testing.T) {
	app, _ := teamApp(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("avatar", "file.txt")
	part.Write([]byte("not an image"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/team/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUploadTeamAvatar_NoFile(t *testing.T) {
	app, _ := teamApp(t)

	req := httptest.NewRequest(http.MethodPost, "/api/team/avatar", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUploadTeamAvatar_NonOwnerForbidden(t *testing.T) {
	db := setupTeamTestDB(t)
	// Change role to regular member
	db.Exec(`UPDATE memberships SET role = 'member' WHERE id = 'm-1'`)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		var user models.User
		db.First(&user)
		c.Locals("user", &user)
		return c.Next()
	})
	h := NewTeamHandler(db)
	app.Post("/api/team/avatar", h.UploadTeamAvatar)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("avatar", "logo.png")
	part.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/team/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestMyTeam_IncludesAvatarURL(t *testing.T) {
	db := setupTeamTestDB(t)
	db.Exec(`UPDATE teams SET avatar_url = '/uploads/avatars/team-1.png' WHERE id = 'team-1'`)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		var user models.User
		db.First(&user)
		c.Locals("user", &user)
		return c.Next()
	})
	h := NewTeamHandler(db)
	app.Get("/api/team", h.MyTeam)

	req := httptest.NewRequest(http.MethodGet, "/api/team", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	team, _ := result["team"].(map[string]any)
	if team == nil {
		t.Fatal("expected team in response")
	}
	avatarURL, _ := team["avatar_url"].(string)
	if avatarURL != "/uploads/avatars/team-1.png" {
		t.Fatalf("expected avatar_url=/uploads/avatars/team-1.png, got %q", avatarURL)
	}
}

func TestMyTeamMembers_IncludesAvatarURL(t *testing.T) {
	db := setupTeamTestDB(t)
	db.Exec(`UPDATE users SET avatar_url = '/uploads/avatars/user-1.png' WHERE id = 1`)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		var user models.User
		db.First(&user)
		c.Locals("user", &user)
		return c.Next()
	})
	h := NewTeamHandler(db)
	app.Get("/api/team/members", h.MyTeamMembers)

	req := httptest.NewRequest(http.MethodGet, "/api/team/members", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	members, _ := result["members"].([]any)
	if len(members) == 0 {
		t.Fatal("expected at least 1 member")
	}
	first, _ := members[0].(map[string]any)
	memberAvatar, _ := first["avatar_url"].(string)
	if memberAvatar != "/uploads/avatars/user-1.png" {
		t.Fatalf("expected member avatar_url=/uploads/avatars/user-1.png, got %q", memberAvatar)
	}
}
