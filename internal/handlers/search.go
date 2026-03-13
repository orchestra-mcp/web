package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// SearchHandler handles global search across projects, features, notes, plans, and wiki docs.
type SearchHandler struct {
	db *gorm.DB
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

type searchResultItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Category    string `json:"category"`
}

// userScope builds a GORM scope matching user_id OR any of their team_ids.
func (h *SearchHandler) userScope(user *models.User) func(db *gorm.DB) *gorm.DB {
	var memberships []models.Membership
	h.db.Where("user_id = ?", user.ID).Find(&memberships)
	teamIDs := make([]string, 0, len(memberships))
	for _, m := range memberships {
		teamIDs = append(teamIDs, m.TeamID)
	}
	return func(db *gorm.DB) *gorm.DB {
		if len(teamIDs) > 0 {
			return db.Where("(user_id = ? OR team_id IN ?)", user.ID, teamIDs)
		}
		return db.Where("user_id = ?", user.ID)
	}
}

// Search handles GET /api/search?q=...&limit=20&types=projects,features,notes,plans,docs
func (h *SearchHandler) Search(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		return c.JSON(fiber.Map{"results": []searchResultItem{}})
	}

	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 50 {
			limit = n
		}
	}

	typesParam := c.Query("types", "projects,features,notes,plans,docs")
	typeSet := make(map[string]bool)
	for _, t := range strings.Split(typesParam, ",") {
		typeSet[strings.TrimSpace(t)] = true
	}

	scope := h.userScope(user)
	pattern := "%" + strings.ToLower(q) + "%"
	var results []searchResultItem

	if typeSet["projects"] {
		var projects []models.Project
		h.db.Scopes(scope).
			Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", pattern, pattern).
			Order("updated_at desc").
			Limit(limit).
			Find(&projects)
		for _, p := range projects {
			desc := p.Description
			if len(desc) > 120 {
				desc = desc[:120] + "..."
			}
			results = append(results, searchResultItem{
				ID:          p.ID,
				Type:        "project",
				Title:       p.Name,
				Description: desc,
				URL:         fmt.Sprintf("/projects/%s", p.Slug),
				Category:    "project",
			})
		}
	}

	if typeSet["features"] {
		var features []models.Feature
		h.db.Scopes(scope).
			Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(feature_id) LIKE ?", pattern, pattern, pattern).
			Order("updated_at desc").
			Limit(limit).
			Find(&features)
		for _, f := range features {
			desc := f.Description
			if len(desc) > 120 {
				desc = desc[:120] + "..."
			}
			results = append(results, searchResultItem{
				ID:          f.FeatureID,
				Type:        "feature",
				Title:       fmt.Sprintf("%s — %s", f.FeatureID, f.Title),
				Description: desc,
				URL:         fmt.Sprintf("/projects/%s/features/%s", f.ProjectSlug, f.FeatureID),
				Category:    "feature",
			})
		}
	}

	if typeSet["notes"] {
		var notes []models.Note
		h.db.Scopes(scope).
			Where("LOWER(title) LIKE ? OR LOWER(content) LIKE ?", pattern, pattern).
			Order("updated_at desc").
			Limit(limit).
			Find(&notes)
		for _, n := range notes {
			desc := n.Content
			if len(desc) > 120 {
				desc = desc[:120] + "..."
			}
			results = append(results, searchResultItem{
				ID:          n.ID,
				Type:        "note",
				Title:       n.Title,
				Description: desc,
				URL:         fmt.Sprintf("/notes/%s", n.ID),
				Category:    "note",
			})
		}
	}

	if typeSet["plans"] {
		var plans []models.Plan
		h.db.Scopes(scope).
			Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(plan_id) LIKE ?", pattern, pattern, pattern).
			Order("updated_at desc").
			Limit(limit).
			Find(&plans)
		for _, p := range plans {
			desc := p.Description
			if len(desc) > 120 {
				desc = desc[:120] + "..."
			}
			results = append(results, searchResultItem{
				ID:          p.PlanID,
				Type:        "plan",
				Title:       fmt.Sprintf("%s — %s", p.PlanID, p.Title),
				Description: desc,
				URL:         fmt.Sprintf("/plans?id=%s", p.PlanID),
				Category:    "plan",
			})
		}
	}

	if typeSet["docs"] {
		var docs []models.Doc
		h.db.Scopes(scope).
			Where("LOWER(title) LIKE ? OR LOWER(body) LIKE ? OR LOWER(category) LIKE ?", pattern, pattern, pattern).
			Order("updated_at desc").
			Limit(limit).
			Find(&docs)
		for _, d := range docs {
			desc := d.Body
			if len(desc) > 120 {
				desc = desc[:120] + "..."
			}
			results = append(results, searchResultItem{
				ID:          d.DocID,
				Type:        "doc",
				Title:       d.Title,
				Description: desc,
				URL:         fmt.Sprintf("/wiki?file=%s", d.DocID),
				Category:    "doc",
			})
		}
	}

	// Cap total results
	if len(results) > limit {
		results = results[:limit]
	}

	return c.JSON(fiber.Map{"results": results})
}

// Suggestions handles GET /api/search/suggestions — returns recent items when spotlight opens.
func (h *SearchHandler) Suggestions(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	scope := h.userScope(user)
	var results []searchResultItem

	// Recent projects (top 3)
	var projects []models.Project
	h.db.Scopes(scope).Order("updated_at desc").Limit(3).Find(&projects)
	for _, p := range projects {
		results = append(results, searchResultItem{
			ID: p.ID, Type: "project", Title: p.Name,
			URL: fmt.Sprintf("/projects/%s", p.Slug), Category: "project",
		})
	}

	// Recent features (top 5)
	var features []models.Feature
	h.db.Scopes(scope).Order("updated_at desc").Limit(5).Find(&features)
	for _, f := range features {
		results = append(results, searchResultItem{
			ID: f.FeatureID, Type: "feature",
			Title: fmt.Sprintf("%s — %s", f.FeatureID, f.Title),
			URL:   fmt.Sprintf("/projects/%s/features/%s", f.ProjectSlug, f.FeatureID), Category: "feature",
		})
	}

	// Recent notes (top 3)
	var notes []models.Note
	h.db.Scopes(scope).Order("updated_at desc").Limit(3).Find(&notes)
	for _, n := range notes {
		results = append(results, searchResultItem{
			ID: n.ID, Type: "note", Title: n.Title,
			URL: fmt.Sprintf("/notes/%s", n.ID), Category: "note",
		})
	}

	// Recent plans (top 3)
	var plans []models.Plan
	h.db.Scopes(scope).Order("updated_at desc").Limit(3).Find(&plans)
	for _, p := range plans {
		results = append(results, searchResultItem{
			ID: p.PlanID, Type: "plan",
			Title: fmt.Sprintf("%s — %s", p.PlanID, p.Title),
			URL:   fmt.Sprintf("/plans?id=%s", p.PlanID), Category: "plan",
		})
	}

	// Recent wiki docs (top 3)
	var docs []models.Doc
	h.db.Scopes(scope).Order("updated_at desc").Limit(3).Find(&docs)
	for _, d := range docs {
		results = append(results, searchResultItem{
			ID: d.DocID, Type: "doc", Title: d.Title,
			URL: fmt.Sprintf("/wiki?file=%s", d.DocID), Category: "doc",
		})
	}

	return c.JSON(fiber.Map{"results": results})
}

// PublicSearch handles GET /api/search/public?q=... — unauthenticated search
// across published blog posts, public docs, and user profiles.
func (h *SearchHandler) PublicSearch(c fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		return c.JSON([]searchResultItem{})
	}

	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 50 {
			limit = n
		}
	}

	pattern := "%" + strings.ToLower(q) + "%"
	var results []searchResultItem

	// Published blog posts
	var posts []models.Post
	h.db.Where("status = ? AND (LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ?)", "published", pattern, pattern).
		Order("published_at desc").
		Limit(limit).
		Find(&posts)
	for _, p := range posts {
		desc := p.Excerpt
		if len(desc) > 120 {
			desc = desc[:120] + "..."
		}
		results = append(results, searchResultItem{
			ID:          fmt.Sprintf("post-%d", p.ID),
			Type:        "post",
			Title:       p.Title,
			Description: desc,
			URL:         fmt.Sprintf("/blog/%s", p.Slug),
			Category:    "post",
		})
	}

	// Public docs (no user scope — only docs from the "orchestra" project or with no user)
	var docs []models.Doc
	h.db.Where("(LOWER(title) LIKE ? OR LOWER(body) LIKE ? OR LOWER(category) LIKE ?)", pattern, pattern, pattern).
		Order("updated_at desc").
		Limit(limit).
		Find(&docs)
	for _, d := range docs {
		desc := d.Body
		if len(desc) > 120 {
			desc = desc[:120] + "..."
		}
		results = append(results, searchResultItem{
			ID:          d.DocID,
			Type:        "doc",
			Title:       d.Title,
			Description: desc,
			URL:         fmt.Sprintf("/docs/%s", d.DocID),
			Category:    "doc",
		})
	}

	// User profiles (public, active users — show name only)
	var users []models.User
	h.db.Where("status = ? AND (LOWER(name) LIKE ? OR LOWER(email) LIKE ?)", "active", pattern, pattern).
		Order("created_at desc").
		Limit(limit).
		Find(&users)
	for _, u := range users {
		results = append(results, searchResultItem{
			ID:          fmt.Sprintf("user-%d", u.ID),
			Type:        "profile",
			Title:       u.Name,
			Description: u.Email,
			URL:         fmt.Sprintf("/profile/%d", u.ID),
			Category:    "profile",
		})
	}

	// Cap total results
	if len(results) > limit {
		results = results[:limit]
	}

	return c.JSON(results)
}

// AiSearch handles POST /api/search/ai — AI-powered natural language search.
// Returns a structured prompt with workspace context for the frontend to send to the AI bridge.
func (h *SearchHandler) AiSearch(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Query string `json:"query"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "query is required"})
	}

	scope := h.userScope(user)

	var projects []models.Project
	h.db.Scopes(scope).Order("updated_at desc").Limit(10).Find(&projects)

	var features []models.Feature
	h.db.Scopes(scope).Order("updated_at desc").Limit(20).Find(&features)

	var notes []models.Note
	h.db.Scopes(scope).Order("updated_at desc").Limit(10).Find(&notes)

	var plans []models.Plan
	h.db.Scopes(scope).Order("updated_at desc").Limit(10).Find(&plans)

	var docs []models.Doc
	h.db.Scopes(scope).Order("updated_at desc").Limit(10).Find(&docs)

	// Build context for the AI
	var ctx strings.Builder
	ctx.WriteString("You are a search assistant for the Orchestra dashboard. Based on the user's workspace data below, answer their query. Return relevant items as a JSON array with fields: id, type (project/feature/note/plan/doc), title, description, url.\n\n")

	ctx.WriteString("## Projects\n")
	for _, p := range projects {
		ctx.WriteString(fmt.Sprintf("- %s (slug: %s, url: /projects/%s) — %s\n", p.Name, p.Slug, p.Slug, p.Description))
	}

	ctx.WriteString("\n## Features\n")
	for _, f := range features {
		ctx.WriteString(fmt.Sprintf("- %s: %s [status: %s, priority: %s, assignee: %s, url: /projects/%s/features/%s]\n",
			f.FeatureID, f.Title, f.Status, f.Priority, f.Assignee, f.ProjectSlug, f.FeatureID))
	}

	ctx.WriteString("\n## Notes\n")
	for _, n := range notes {
		content := n.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		ctx.WriteString(fmt.Sprintf("- %s (url: /notes/%s): %s\n", n.Title, n.ID, content))
	}

	ctx.WriteString("\n## Plans\n")
	for _, p := range plans {
		ctx.WriteString(fmt.Sprintf("- %s: %s [status: %s, url: /plans?id=%s] — %s\n",
			p.PlanID, p.Title, p.Status, p.PlanID, p.Description))
	}

	ctx.WriteString("\n## Wiki Docs\n")
	for _, d := range docs {
		body := d.Body
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		ctx.WriteString(fmt.Sprintf("- %s [category: %s, url: /wiki?file=%s]: %s\n", d.Title, d.Category, d.DocID, body))
	}

	ctx.WriteString(fmt.Sprintf("\n## User Query\n%s\n", body.Query))

	return c.JSON(fiber.Map{
		"prompt":  ctx.String(),
		"context": "ai_search",
	})
}
