package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// DashboardHandler handles the dashboard batch endpoint.
type DashboardHandler struct {
	db *gorm.DB
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// Dashboard handles GET /api/dashboard
// Returns projects, notes, and stats in a single call.
func (h *DashboardHandler) Dashboard(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Get team IDs for team-scoped access
	teamIDs := userTeamIDs(h.db, user.ID)

	// 1. Projects (personal + team, limit 20)
	projectQ := h.db.Order("updated_at desc").Limit(20)
	if len(teamIDs) > 0 {
		projectQ = projectQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		projectQ = projectQ.Where("user_id = ?", user.ID)
	}

	var projects []models.Project
	projectQ.Find(&projects)

	type projectItem struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description string  `json:"description"`
		WorkspaceID *string `json:"workspace_id"`
		TeamID      *string `json:"team_id"`
	}
	projectRows := make([]projectItem, 0, len(projects))
	for _, p := range projects {
		projectRows = append(projectRows, projectItem{
			ID:          p.ID,
			Name:        p.Name,
			Slug:        p.Slug,
			Description: p.Description,
			WorkspaceID: p.WorkspaceID,
			TeamID:      p.TeamID,
		})
	}

	// 2. Notes (personal + team, limit 5)
	noteQ := h.db.Order("pinned desc, updated_at desc").Limit(5)
	if len(teamIDs) > 0 {
		noteQ = noteQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		noteQ = noteQ.Where("user_id = ?", user.ID)
	}

	var notes []models.Note
	noteQ.Find(&notes)

	// 3. Stats
	var projectCount, featureCount, noteCount, tunnelCount int64

	// Count all accessible projects
	pCountQ := h.db.Model(&models.Project{})
	if len(teamIDs) > 0 {
		pCountQ = pCountQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		pCountQ = pCountQ.Where("user_id = ?", user.ID)
	}
	pCountQ.Count(&projectCount)

	// Count features across user's projects
	if len(projects) > 0 {
		slugs := make([]string, 0, len(projects))
		for _, p := range projects {
			slugs = append(slugs, p.Slug)
		}
		h.db.Model(&models.Feature{}).Where("project_slug IN ?", slugs).Count(&featureCount)
	}

	// Count notes
	nCountQ := h.db.Model(&models.Note{})
	if len(teamIDs) > 0 {
		nCountQ = nCountQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		nCountQ = nCountQ.Where("user_id = ?", user.ID)
	}
	nCountQ.Count(&noteCount)

	// Count active tunnels
	h.db.Model(&models.Tunnel{}).Where("user_id = ? AND status = 'active'", user.ID).Count(&tunnelCount)

	// 4. API Collections (personal + team, limit 5)
	colQ := h.db.Order("updated_at desc").Limit(5)
	if len(teamIDs) > 0 {
		colQ = colQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		colQ = colQ.Where("user_id = ?", user.ID)
	}
	var collections []models.ApiCollection
	colQ.Find(&collections)

	// Build collection items with endpoint counts
	type collectionItem struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		Description   string `json:"description"`
		BaseURL       string `json:"base_url"`
		AuthType      string `json:"auth_type"`
		EndpointCount int64  `json:"endpoint_count"`
		UpdatedAt     string `json:"updated_at"`
	}
	collectionRows := make([]collectionItem, 0, len(collections))
	for _, col := range collections {
		var epCount int64
		h.db.Model(&models.ApiEndpoint{}).Where("collection_id = ?", col.ID).Count(&epCount)
		collectionRows = append(collectionRows, collectionItem{
			ID:            col.ID,
			Name:          col.Name,
			Slug:          col.Slug,
			Description:   col.Description,
			BaseURL:       col.BaseURL,
			AuthType:      col.AuthType,
			EndpointCount: epCount,
			UpdatedAt:     col.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	// 5. Presentations (personal + team, limit 5)
	presQ := h.db.Order("updated_at desc").Limit(5)
	if len(teamIDs) > 0 {
		presQ = presQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		presQ = presQ.Where("user_id = ?", user.ID)
	}
	var presentations []models.Presentation
	presQ.Find(&presentations)

	type presentationItem struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		SlideCount  int    `json:"slide_count"`
		Visibility  string `json:"visibility"`
		UpdatedAt   string `json:"updated_at"`
	}
	presentationRows := make([]presentationItem, 0, len(presentations))
	for _, p := range presentations {
		presentationRows = append(presentationRows, presentationItem{
			ID:          p.ID,
			Title:       p.Title,
			Slug:        p.Slug,
			Description: p.Description,
			SlideCount:  p.SlideCount,
			Visibility:  p.Visibility,
			UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	// 6. Docs (personal + team, limit 5)
	docQ := h.db.Order("updated_at desc").Limit(5)
	if len(teamIDs) > 0 {
		docQ = docQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		docQ = docQ.Where("user_id = ?", user.ID)
	}
	var docs []models.Doc
	docQ.Find(&docs)

	type docItem struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		DocID     string `json:"doc_id"`
		Category  string `json:"category"`
		Published bool   `json:"published"`
		UpdatedAt string `json:"updated_at"`
		WordCount int    `json:"word_count"`
	}
	docRows := make([]docItem, 0, len(docs))
	for _, d := range docs {
		wc := len(strings.Fields(d.Body))
		docRows = append(docRows, docItem{
			ID:        d.ID,
			Title:     d.Title,
			DocID:     d.DocID,
			Category:  d.Category,
			Published: d.Published,
			UpdatedAt: d.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			WordCount: wc,
		})
	}

	// Count collections
	var collectionCount, presentationCount, docCount int64

	cCountQ := h.db.Model(&models.ApiCollection{})
	if len(teamIDs) > 0 {
		cCountQ = cCountQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		cCountQ = cCountQ.Where("user_id = ?", user.ID)
	}
	cCountQ.Count(&collectionCount)

	// Count presentations
	prCountQ := h.db.Model(&models.Presentation{})
	if len(teamIDs) > 0 {
		prCountQ = prCountQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		prCountQ = prCountQ.Where("user_id = ?", user.ID)
	}
	prCountQ.Count(&presentationCount)

	// Count docs
	dCountQ := h.db.Model(&models.Doc{})
	if len(teamIDs) > 0 {
		dCountQ = dCountQ.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		dCountQ = dCountQ.Where("user_id = ?", user.ID)
	}
	dCountQ.Count(&docCount)

	return c.JSON(fiber.Map{
		"projects":      projectRows,
		"notes":         notes,
		"collections":   collectionRows,
		"presentations": presentationRows,
		"docs":          docRows,
		"stats": fiber.Map{
			"project_count":      projectCount,
			"feature_count":      featureCount,
			"note_count":         noteCount,
			"tunnel_count":       tunnelCount,
			"collection_count":   collectionCount,
			"presentation_count": presentationCount,
			"doc_count":          docCount,
		},
	})
}
