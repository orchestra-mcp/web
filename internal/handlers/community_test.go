package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCommunityTestDB(t *testing.T) *gorm.DB {
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
		avatar_url TEXT, settings TEXT, remember_token TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)
	db.Exec(`CREATE TABLE community_posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER, title TEXT, content TEXT,
		status TEXT DEFAULT 'published',
		likes_count INTEGER DEFAULT 0, comments_count INTEGER DEFAULT 0,
		parent_id INTEGER,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`)

	return db
}

func TestListMembers_ReturnsCorrectShape(t *testing.T) {
	db := setupCommunityTestDB(t)
	now := time.Now()

	// Insert a user with public_profile_enabled = true
	db.Exec(`INSERT INTO users (id, name, email, avatar_url, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, "Alice Smith", "alice@example.com", "https://img.example.com/alice.jpg",
		`{"public_profile_enabled": "true", "handle": "alice", "bio": "Go developer", "role": "Core Member"}`,
		now, now,
	)

	// Insert a user with public_profile_enabled = false (should NOT appear)
	db.Exec(`INSERT INTO users (id, name, email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		2, "Bob Jones", "bob@example.com",
		`{"public_profile_enabled": false, "handle": "bob"}`,
		now, now,
	)

	// Insert a user with no public_profile_enabled at all (should NOT appear)
	db.Exec(`INSERT INTO users (id, name, email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		3, "Charlie Brown", "charlie@example.com",
		`{"handle": "charlie"}`,
		now, now,
	)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/members", h.ListMembers)

	req := httptest.NewRequest(http.MethodGet, "/api/public/community/members", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		Members []memberRow `json:"members"`
		Total   int64       `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have exactly 1 member (Alice) — only public_profile_enabled=true
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
	if len(result.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(result.Members))
	}

	m := result.Members[0]
	if m.Name != "Alice Smith" {
		t.Errorf("expected name 'Alice Smith', got '%s'", m.Name)
	}
	if m.Handle != "alice" {
		t.Errorf("expected handle 'alice', got '%s'", m.Handle)
	}
	if m.Bio != "Go developer" {
		t.Errorf("expected bio 'Go developer', got '%s'", m.Bio)
	}
	if m.Role != "Core Member" {
		t.Errorf("expected role 'Core Member', got '%s'", m.Role)
	}
	if !m.IsPublic {
		t.Error("expected is_public=true")
	}
	if m.AvatarURL != "https://img.example.com/alice.jpg" {
		t.Errorf("expected avatar_url, got '%s'", m.AvatarURL)
	}
}

func TestListMembers_DefaultRole(t *testing.T) {
	db := setupCommunityTestDB(t)
	now := time.Now()

	// User without role in settings — should get default "Member"
	db.Exec(`INSERT INTO users (id, name, email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		1, "NoRole User", "norole@example.com",
		`{"public_profile_enabled": "true", "handle": "norole"}`,
		now, now,
	)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/members", h.ListMembers)

	req := httptest.NewRequest(http.MethodGet, "/api/public/community/members", nil)
	resp, _ := app.Test(req)

	var result struct {
		Members []memberRow `json:"members"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(result.Members))
	}
	if result.Members[0].Role != "Member" {
		t.Errorf("expected default role 'Member', got '%s'", result.Members[0].Role)
	}
}

func TestListMembers_PostCount(t *testing.T) {
	db := setupCommunityTestDB(t)
	now := time.Now()

	db.Exec(`INSERT INTO users (id, name, email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Alice", "alice@example.com",
		`{"public_profile_enabled": "true", "handle": "alice"}`,
		now, now,
	)

	// Add 2 published posts and 1 draft (should only count published)
	db.Exec(`INSERT INTO community_posts (user_id, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Post 1", "content", "published", now, now)
	db.Exec(`INSERT INTO community_posts (user_id, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Post 2", "content", "published", now, now)
	db.Exec(`INSERT INTO community_posts (user_id, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Draft", "content", "draft", now, now)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/members", h.ListMembers)

	req := httptest.NewRequest(http.MethodGet, "/api/public/community/members", nil)
	resp, _ := app.Test(req)

	var result struct {
		Members []memberRow `json:"members"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(result.Members))
	}
	if result.Members[0].PostCount != 2 {
		t.Errorf("expected post_count=2, got %d", result.Members[0].PostCount)
	}
}

func TestListMembers_EmptyWhenNoPublicProfiles(t *testing.T) {
	db := setupCommunityTestDB(t)
	now := time.Now()

	// Only private users
	db.Exec(`INSERT INTO users (id, name, email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Private User", "priv@example.com",
		`{"public_profile_enabled": false}`,
		now, now,
	)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/members", h.ListMembers)

	req := httptest.NewRequest(http.MethodGet, "/api/public/community/members", nil)
	resp, _ := app.Test(req)

	var result struct {
		Members []memberRow `json:"members"`
		Total   int64       `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
	if len(result.Members) != 0 {
		t.Errorf("expected 0 members, got %d", len(result.Members))
	}
}

func TestListMembers_SearchFilter(t *testing.T) {
	db := setupCommunityTestDB(t)
	now := time.Now()

	db.Exec(`INSERT INTO users (id, name, email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Alice Smith", "alice@example.com",
		`{"public_profile_enabled": "true", "handle": "alice"}`,
		now, now,
	)
	db.Exec(`INSERT INTO users (id, name, email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		2, "Bob Jones", "bob@example.com",
		`{"public_profile_enabled": "true", "handle": "bob"}`,
		now, now,
	)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/members", h.ListMembers)

	// Search for "alice" by name — should find 1 result
	req := httptest.NewRequest(http.MethodGet, "/api/public/community/members?search=Alice", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var result struct {
		Members []memberRow `json:"members"`
		Total   int64       `json:"total"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Total != 1 {
		t.Errorf("expected total=1 for search 'Alice', got %d", result.Total)
	}

	// Search for "bob" by handle — should find 1
	req2 := httptest.NewRequest(http.MethodGet, "/api/public/community/members?search=bob", nil)
	resp2, _ := app.Test(req2)

	var result2 struct {
		Members []memberRow `json:"members"`
		Total   int64       `json:"total"`
	}
	json.NewDecoder(resp2.Body).Decode(&result2)

	if result2.Total != 1 {
		t.Errorf("expected total=1 for search 'bob', got %d", result2.Total)
	}

	// Empty search — should return all 2
	req3 := httptest.NewRequest(http.MethodGet, "/api/public/community/members", nil)
	resp3, _ := app.Test(req3)

	var result3 struct {
		Total int64 `json:"total"`
	}
	json.NewDecoder(resp3.Body).Decode(&result3)

	if result3.Total != 2 {
		t.Errorf("expected total=2 for empty search, got %d", result3.Total)
	}
}

func TestMemberProfile_ReturnsProfileShape(t *testing.T) {
	db := setupCommunityTestDB(t)
	now := time.Now()

	db.Exec(`INSERT INTO users (id, name, email, avatar_url, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, "Alice Smith", "alice@example.com", "https://img.example.com/alice.jpg",
		`{"public_profile_enabled": "true", "handle": "alice", "bio": "Go developer", "role": "Core Member", "cover_url": "https://img.example.com/cover.jpg", "location": "NYC"}`,
		now, now,
	)

	// Add a published post
	db.Exec(`INSERT INTO community_posts (user_id, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Hello World", "My first post", "published", now, now)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/members/:handle", h.MemberProfile)

	req := httptest.NewRequest(http.MethodGet, "/api/public/community/members/alice", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		Profile struct {
			Name        string `json:"name"`
			Handle      string `json:"handle"`
			Bio         string `json:"bio"`
			Role        string `json:"role"`
			AvatarURL   string `json:"avatar_url"`
			CoverURL    string `json:"cover_url"`
			Location    string `json:"location"`
			SocialLinks []struct {
				Platform string `json:"platform"`
				URL      string `json:"url"`
			} `json:"social_links"`
			Stats struct {
				Posts               int `json:"posts"`
				ProfileCompleteness int `json:"profile_completeness"`
			} `json:"stats"`
			RecentPosts []struct {
				ID    uint   `json:"id"`
				Title string `json:"title"`
			} `json:"recent_posts"`
		} `json:"profile"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	p := result.Profile
	if p.Name != "Alice Smith" {
		t.Errorf("expected name 'Alice Smith', got '%s'", p.Name)
	}
	if p.Handle != "alice" {
		t.Errorf("expected handle 'alice', got '%s'", p.Handle)
	}
	if p.Bio != "Go developer" {
		t.Errorf("expected bio 'Go developer', got '%s'", p.Bio)
	}
	if p.Role != "Core Member" {
		t.Errorf("expected role 'Core Member', got '%s'", p.Role)
	}
	if p.Location != "NYC" {
		t.Errorf("expected location 'NYC', got '%s'", p.Location)
	}
	if p.Stats.Posts != 1 {
		t.Errorf("expected stats.posts=1, got %d", p.Stats.Posts)
	}
	if p.Stats.ProfileCompleteness != 100 {
		t.Errorf("expected completeness=100, got %d", p.Stats.ProfileCompleteness)
	}
	if len(p.RecentPosts) != 1 {
		t.Fatalf("expected 1 recent post, got %d", len(p.RecentPosts))
	}
	if p.RecentPosts[0].Title != "Hello World" {
		t.Errorf("expected recent post title 'Hello World', got '%s'", p.RecentPosts[0].Title)
	}
}

func TestMemberProfile_NotFound(t *testing.T) {
	db := setupCommunityTestDB(t)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/members/:handle", h.MemberProfile)

	req := httptest.NewRequest(http.MethodGet, "/api/public/community/members/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestMemberPosts_ReturnsPostsShape(t *testing.T) {
	db := setupCommunityTestDB(t)
	now := time.Now()

	db.Exec(`INSERT INTO users (id, name, email, avatar_url, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, "Alice", "alice@example.com", "https://img.example.com/a.jpg",
		`{"public_profile_enabled": "true", "handle": "alice"}`,
		now, now,
	)
	db.Exec(`INSERT INTO community_posts (user_id, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Post A", "content A", "published", now, now)
	db.Exec(`INSERT INTO community_posts (user_id, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		1, "Draft B", "content B", "draft", now, now)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/members/:handle/posts", h.MemberPosts)

	req := httptest.NewRequest(http.MethodGet, "/api/public/community/members/alice/posts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var result struct {
		Posts []struct {
			ID           uint   `json:"id"`
			Title        string `json:"title"`
			AuthorName   string `json:"author_name"`
			AuthorHandle string `json:"author_handle"`
		} `json:"posts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Should only return published posts
	if len(result.Posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(result.Posts))
	}
	if result.Posts[0].Title != "Post A" {
		t.Errorf("expected title 'Post A', got '%s'", result.Posts[0].Title)
	}
	if result.Posts[0].AuthorName != "Alice" {
		t.Errorf("expected author 'Alice', got '%s'", result.Posts[0].AuthorName)
	}
	if result.Posts[0].AuthorHandle != "alice" {
		t.Errorf("expected handle 'alice', got '%s'", result.Posts[0].AuthorHandle)
	}
}

func TestShowPost_EmbeddedAuthor(t *testing.T) {
	db := setupCommunityTestDB(t)
	now := time.Now()

	db.Exec(`INSERT INTO users (id, name, email, avatar_url, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		1, "Alice", "alice@example.com", "https://img.example.com/a.jpg",
		`{"public_profile_enabled": "true", "handle": "alice"}`,
		now, now,
	)
	db.Exec(`INSERT INTO community_posts (id, user_id, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		10, 1, "My Post", "Hello world", "published", now, now)

	h := NewCommunityHandler(db)
	app := fiber.New()
	app.Get("/api/public/community/posts/:id", h.ShowPost)

	req := httptest.NewRequest(http.MethodGet, "/api/public/community/posts/10", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Post struct {
			ID           uint   `json:"id"`
			Title        string `json:"title"`
			Content      string `json:"content"`
			AuthorName   string `json:"author_name"`
			AuthorHandle string `json:"author_handle"`
			AuthorAvatar string `json:"author_avatar"`
		} `json:"post"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if result.Post.ID != 10 {
		t.Errorf("expected id=10, got %d", result.Post.ID)
	}
	if result.Post.Title != "My Post" {
		t.Errorf("expected title 'My Post', got '%s'", result.Post.Title)
	}
	if result.Post.AuthorName != "Alice" {
		t.Errorf("expected author_name 'Alice', got '%s'", result.Post.AuthorName)
	}
	if result.Post.AuthorHandle != "alice" {
		t.Errorf("expected author_handle 'alice', got '%s'", result.Post.AuthorHandle)
	}
	if result.Post.AuthorAvatar != "https://img.example.com/a.jpg" {
		t.Errorf("expected avatar, got '%s'", result.Post.AuthorAvatar)
	}
}
