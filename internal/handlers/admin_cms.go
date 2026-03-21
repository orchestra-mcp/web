package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/helpers"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AdminCmsHandler handles admin CMS and extended user management endpoints.
type AdminCmsHandler struct {
	db          *gorm.DB
	authService *services.AuthService
	hub         *hub.Hub
}

// NewAdminCmsHandlerWithAuth creates a new AdminCmsHandler with auth service.
func NewAdminCmsHandlerWithAuth(db *gorm.DB, authSvc *services.AuthService, wsHub *hub.Hub) *AdminCmsHandler {
	return &AdminCmsHandler{db: db, authService: authSvc, hub: wsHub}
}

// GetUser handles GET /api/admin/users/:id
func (h *AdminCmsHandler) GetUser(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	var projectCount, noteCount, sessionCount int64
	h.db.Model(&models.Project{}).Where("user_id = ?", user.ID).Count(&projectCount)
	h.db.Model(&models.Note{}).Where("user_id = ?", user.ID).Count(&noteCount)
	h.db.Model(&models.AiSession{}).Where("user_id = ?", user.ID).Count(&sessionCount)

	var memberCount int64
	h.db.Model(&models.Membership{}).Where("user_id = ?", user.ID).Count(&memberCount)

	return c.JSON(fiber.Map{
		"user":          user,
		"project_count": projectCount,
		"note_count":    noteCount,
		"session_count": sessionCount,
		"team_count":    memberCount,
	})
}

// UserProjects handles GET /api/admin/users/:id/projects
func (h *AdminCmsHandler) UserProjects(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var projects []models.Project
	h.db.Where("user_id = ?", id).Order("created_at DESC").Find(&projects)
	return c.JSON(fiber.Map{"projects": projects})
}

// UserNotes handles GET /api/admin/users/:id/notes
func (h *AdminCmsHandler) UserNotes(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var notes []models.Note
	h.db.Where("user_id = ?", id).Order("created_at DESC").Find(&notes)
	return c.JSON(fiber.Map{"notes": notes})
}

// UserSessions handles GET /api/admin/users/:id/sessions
func (h *AdminCmsHandler) UserSessions(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var sessions []models.AiSession
	h.db.Where("user_id = ?", id).Order("created_at DESC").Find(&sessions)
	return c.JSON(fiber.Map{"sessions": sessions})
}

// UserTeams handles GET /api/admin/users/:id/teams
func (h *AdminCmsHandler) UserTeams(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var memberships []models.Membership
	h.db.Where("user_id = ?", id).Preload("Team").Find(&memberships)

	type teamRow struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Plan        string `json:"plan"`
		MemberCount int64  `json:"member_count"`
		Role        string `json:"role"`
	}
	rows := make([]teamRow, 0, len(memberships))
	for _, m := range memberships {
		var cnt int64
		h.db.Model(&models.Membership{}).Where("team_id = ?", m.TeamID).Count(&cnt)
		rows = append(rows, teamRow{
			ID:          m.TeamID,
			Name:        m.Team.Name,
			Plan:        m.Team.Plan,
			MemberCount: cnt,
			Role:        m.Role,
		})
	}
	return c.JSON(fiber.Map{"teams": rows})
}

// UserIssues handles GET /api/admin/users/:id/issues
func (h *AdminCmsHandler) UserIssues(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var issues []models.Issue
	h.db.Where("user_id = ?", id).Order("created_at DESC").Find(&issues)
	return c.JSON(fiber.Map{"issues": issues})
}

// GetLastOTP handles GET /api/admin/users/:id/otp
func (h *AdminCmsHandler) GetLastOTP(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var otp models.OtpCode
	if err := h.db.Where("user_id = ?", id).Order("created_at DESC").First(&otp).Error; err != nil {
		return c.JSON(fiber.Map{"code": nil, "message": "no OTP found"})
	}

	return c.JSON(fiber.Map{
		"code":       otp.Code,
		"type":       otp.Type,
		"expires_at": otp.ExpiresAt,
		"used":       otp.UsedAt != nil,
		"expired":    otp.ExpiresAt.Before(time.Now()),
	})
}

// ForceResetPassword handles POST /api/admin/users/:id/password
func (h *AdminCmsHandler) ForceResetPassword(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "password is required"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	id := c.Params("id")
	if err := h.db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"password":     string(hashed),
		"password_set": true,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update password"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// Impersonate handles POST /api/admin/users/:id/impersonate
func (h *AdminCmsHandler) Impersonate(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	id := c.Params("id")
	var target models.User
	if err := h.db.First(&target, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	token, err := h.authService.GenerateJWT(&target)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"ok":    true,
		"token": token,
		"user":  target,
	})
}

// NotifyUser handles POST /api/admin/users/:id/notify
func (h *AdminCmsHandler) NotifyUser(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	var body struct {
		Title   string `json:"title"`
		Message string `json:"message"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	id := c.Params("id")
	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	notifType := body.Type
	if notifType == "" {
		notifType = "info"
	}

	notif := models.Notification{
		UserID:  user.ID,
		Title:   body.Title,
		Message: body.Message,
		Type:    notifType,
	}
	h.db.Create(&notif)

	// Push realtime notification via WebSocket
	if h.hub != nil {
		h.hub.BroadcastToUser(user.ID, hub.Event{
			Type:       "notification",
			EntityType: "notification",
			EntityID:   fmt.Sprintf("%d", notif.ID),
			Action:     "upsert",
			UserID:     user.ID,
			Timestamp:  notif.CreatedAt.UnixMilli(),
			Title:      notif.Title,
			Message:    notif.Message,
			NType:      notif.Type,
		})
	}

	// Send Web Push notification
	services.SendPush(h.db, user.ID, body.Title, body.Message, notifType)

	return c.JSON(fiber.Map{"ok": true})
}

// UpsertSubscription handles POST /api/admin/users/:id/subscription
func (h *AdminCmsHandler) UpsertSubscription(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	var body struct {
		Plan  string `json:"plan"`
		Notes string `json:"notes"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	return c.JSON(fiber.Map{"ok": true, "plan": body.Plan})
}

// Pages CRUD

func (h *AdminCmsHandler) ListPages(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var pages []models.Page
	h.db.Order("created_at DESC").Find(&pages)

	locale := helpers.ContentLocale(c)
	if locale != helpers.DefaultLocale {
		for i := range pages {
			if tr := helpers.ResolveTranslation(pages[i].Translations, locale); tr != nil {
				if v, ok := tr["title"].(string); ok && v != "" {
					pages[i].Title = v
				}
				if v, ok := tr["content"].(string); ok && v != "" {
					pages[i].Content = v
				}
			}
		}
	}
	return c.JSON(fiber.Map{"pages": pages})
}

func (h *AdminCmsHandler) CreatePage(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	user := middleware.CurrentUser(c)

	var body struct {
		Title        string                            `json:"title"`
		Slug         string                            `json:"slug"`
		Content      string                            `json:"content"`
		Status       string                            `json:"status"`
		Translations map[string]map[string]interface{} `json:"translations"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}
	status := body.Status
	if status == "" {
		status = "draft"
	}
	var translationsJSON []byte
	if body.Translations != nil {
		translationsJSON, _ = json.Marshal(body.Translations)
	} else {
		translationsJSON = []byte("{}")
	}
	page := models.Page{Title: body.Title, Slug: body.Slug, Content: body.Content, Status: status, UserID: user.ID, Translations: translationsJSON}
	if err := h.db.Create(&page).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create page"})
	}
	return c.Status(fiber.StatusCreated).JSON(page)
}

func (h *AdminCmsHandler) UpdatePage(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var body struct {
		Title   string `json:"title"`
		Slug    string `json:"slug"`
		Content string `json:"content"`
		Status  string `json:"status"`
		Locale  string `json:"locale"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	locale := body.Locale
	if locale == "" {
		locale = helpers.DefaultLocale
	}

	if locale == helpers.DefaultLocale {
		updates := map[string]interface{}{}
		if body.Title != "" {
			updates["title"] = body.Title
		}
		if body.Slug != "" {
			updates["slug"] = body.Slug
		}
		if body.Content != "" {
			updates["content"] = body.Content
		}
		if body.Status != "" {
			updates["status"] = body.Status
		}
		h.db.Model(&models.Page{}).Where("id = ?", c.Params("id")).Updates(updates)
	} else {
		var page models.Page
		if err := h.db.First(&page, c.Params("id")).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "page not found"})
		}
		fields := map[string]interface{}{}
		if body.Title != "" {
			fields["title"] = body.Title
		}
		if body.Content != "" {
			fields["content"] = body.Content
		}
		merged := helpers.MergeTranslation(page.Translations, locale, fields)
		h.db.Model(&page).Update("translations", merged)
	}

	var updated models.Page
	h.db.First(&updated, c.Params("id"))
	return c.JSON(fiber.Map{"ok": true, "page": updated})
}

func (h *AdminCmsHandler) DeletePage(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	h.db.Delete(&models.Page{}, c.Params("id"))
	return c.JSON(fiber.Map{"ok": true})
}

// Posts CRUD

func (h *AdminCmsHandler) ListPosts(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var posts []models.Post
	h.db.Order("created_at DESC").Find(&posts)

	locale := helpers.ContentLocale(c)
	if locale != helpers.DefaultLocale {
		for i := range posts {
			if tr := helpers.ResolveTranslation(posts[i].Translations, locale); tr != nil {
				if v, ok := tr["title"].(string); ok && v != "" {
					posts[i].Title = v
				}
				if v, ok := tr["content"].(string); ok && v != "" {
					posts[i].Content = v
				}
				if v, ok := tr["excerpt"].(string); ok && v != "" {
					posts[i].Excerpt = v
				}
			}
		}
	}
	return c.JSON(fiber.Map{"posts": posts})
}

func (h *AdminCmsHandler) CreatePost(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	user := middleware.CurrentUser(c)

	var body struct {
		Title        string                            `json:"title"`
		Slug         string                            `json:"slug"`
		Excerpt      string                            `json:"excerpt"`
		Content      string                            `json:"content"`
		Status       string                            `json:"status"`
		Translations map[string]map[string]interface{} `json:"translations"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}
	status := body.Status
	if status == "" {
		status = "draft"
	}
	var translationsJSON []byte
	if body.Translations != nil {
		translationsJSON, _ = json.Marshal(body.Translations)
	} else {
		translationsJSON = []byte("{}")
	}
	post := models.Post{Title: body.Title, Slug: body.Slug, Excerpt: body.Excerpt, Content: body.Content, Status: status, UserID: user.ID, Translations: translationsJSON}
	if err := h.db.Create(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create post"})
	}
	return c.Status(fiber.StatusCreated).JSON(post)
}

func (h *AdminCmsHandler) UpdatePost(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var body struct {
		Title   string `json:"title"`
		Slug    string `json:"slug"`
		Excerpt string `json:"excerpt"`
		Content string `json:"content"`
		Status  string `json:"status"`
		Locale  string `json:"locale"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	locale := body.Locale
	if locale == "" {
		locale = helpers.DefaultLocale
	}

	if locale == helpers.DefaultLocale {
		updates := map[string]interface{}{}
		if body.Title != "" {
			updates["title"] = body.Title
		}
		if body.Status != "" {
			updates["status"] = body.Status
		}
		if body.Content != "" {
			updates["content"] = body.Content
		}
		if body.Excerpt != "" {
			updates["excerpt"] = body.Excerpt
		}
		if body.Status == "published" {
			now := time.Now()
			updates["published_at"] = &now
		}
		h.db.Model(&models.Post{}).Where("id = ?", c.Params("id")).Updates(updates)
	} else {
		var post models.Post
		if err := h.db.First(&post, c.Params("id")).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
		}
		fields := map[string]interface{}{}
		if body.Title != "" {
			fields["title"] = body.Title
		}
		if body.Content != "" {
			fields["content"] = body.Content
		}
		if body.Excerpt != "" {
			fields["excerpt"] = body.Excerpt
		}
		merged := helpers.MergeTranslation(post.Translations, locale, fields)
		h.db.Model(&post).Update("translations", merged)
	}

	var updated models.Post
	h.db.First(&updated, c.Params("id"))
	return c.JSON(fiber.Map{"ok": true, "post": updated})
}

func (h *AdminCmsHandler) DeletePost(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	h.db.Delete(&models.Post{}, c.Params("id"))
	return c.JSON(fiber.Map{"ok": true})
}

// Categories

func (h *AdminCmsHandler) ListCategories(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	// Return static for now; a Category model can be added later
	return c.JSON(fiber.Map{"categories": []fiber.Map{
		{"id": 1, "name": "General", "slug": "general", "count": 0},
		{"id": 2, "name": "Tutorials", "slug": "tutorials", "count": 0},
		{"id": 3, "name": "Updates", "slug": "updates", "count": 0},
	}})
}

func (h *AdminCmsHandler) CreateCategory(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"ok": true, "name": body.Name, "slug": body.Slug})
}

func (h *AdminCmsHandler) DeleteCategory(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// Marketplace

func (h *AdminCmsHandler) ListMarketplace(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	return c.JSON(fiber.Map{"packs": []fiber.Map{
		{"name": "orchestra-go-backend", "version": "1.0.0", "downloads": 1200, "status": "active", "author": "orchestra"},
		{"name": "orchestra-rust-engine", "version": "0.9.0", "downloads": 842, "status": "active", "author": "orchestra"},
		{"name": "orchestra-react", "version": "2.1.0", "downloads": 3400, "status": "active", "author": "orchestra"},
	}})
}

// Contact messages

func (h *AdminCmsHandler) ListContact(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var messages []models.ContactMessage
	h.db.Order("created_at DESC").Find(&messages)
	return c.JSON(fiber.Map{"messages": messages})
}

func (h *AdminCmsHandler) DeleteContact(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	h.db.Delete(&models.ContactMessage{}, c.Params("id"))
	return c.JSON(fiber.Map{"ok": true})
}

// Issues

func (h *AdminCmsHandler) ListIssues(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var issues []models.Issue
	h.db.Order("created_at DESC").Find(&issues)
	return c.JSON(fiber.Map{"issues": issues})
}

func (h *AdminCmsHandler) UpdateIssue(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var body struct {
		Status   string `json:"status"`
		Priority string `json:"priority"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	updates := map[string]interface{}{}
	if body.Status != "" {
		updates["status"] = body.Status
	}
	if body.Priority != "" {
		updates["priority"] = body.Priority
	}
	h.db.Model(&models.Issue{}).Where("id = ?", c.Params("id")).Updates(updates)
	return c.JSON(fiber.Map{"ok": true})
}

// SendNotification handles POST /api/admin/notifications/send
func (h *AdminCmsHandler) SendNotification(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var body struct {
		Recipients string  `json:"recipients"` // "all" | "role:admin" | "user:{id}"
		UserIDs    []uint  `json:"user_ids"`
		Title      string  `json:"title"`
		Message    string  `json:"message"`
		Type       string  `json:"type"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	notifType := body.Type
	if notifType == "" {
		notifType = "info"
	}

	var targetIDs []uint
	if body.Recipients == "all" || body.Recipients == "" {
		var users []models.User
		h.db.Select("id").Find(&users)
		for _, u := range users {
			targetIDs = append(targetIDs, u.ID)
		}
	} else {
		targetIDs = body.UserIDs
	}

	for _, uid := range targetIDs {
		notif := models.Notification{
			UserID:  uid,
			Title:   body.Title,
			Message: body.Message,
			Type:    notifType,
		}
		h.db.Create(&notif)

		// Push realtime notification via WebSocket
		if h.hub != nil {
			h.hub.BroadcastToUser(uid, hub.Event{
				Type:       "notification",
				EntityType: "notification",
				EntityID:   fmt.Sprintf("%d", notif.ID),
				Action:     "upsert",
				UserID:     uid,
				Timestamp:  notif.CreatedAt.UnixMilli(),
				Title:      notif.Title,
				Message:    notif.Message,
				NType:      notif.Type,
			})
		}

		// Send Web Push notification
		services.SendPush(h.db, uid, body.Title, body.Message, notifType)
	}

	return c.JSON(fiber.Map{"ok": true, "sent_to": len(targetIDs)})
}

// ListNotificationsSent handles GET /api/admin/notifications
func (h *AdminCmsHandler) ListNotificationsSent(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var notifications []models.Notification
	h.db.Order("created_at DESC").Limit(100).Find(&notifications)
	return c.JSON(fiber.Map{"notifications": notifications})
}

// SeedNotifications handles POST /api/admin/notifications/seed
// Creates sample notifications for the current user (for testing realtime flow).
func (h *AdminCmsHandler) SeedNotifications(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	seeds := []models.Notification{
		{UserID: user.ID, Title: "Welcome to Orchestra", Message: "Your workspace is ready. Start by creating a project.", Type: "info"},
		{UserID: user.ID, Title: "Feature completed", Message: "FEAT-ABC has been moved to done by the AI agent.", Type: "success"},
		{UserID: user.ID, Title: "Build failed", Message: "CI pipeline for branch feature/auth failed at test step.", Type: "error"},
		{UserID: user.ID, Title: "Review requested", Message: "FEAT-XYZ is waiting for your review approval.", Type: "warning"},
		{UserID: user.ID, Title: "New team member", Message: "Alice joined your team as a developer.", Type: "info"},
	}

	for i := range seeds {
		h.db.Create(&seeds[i])

		if h.hub != nil {
			h.hub.BroadcastToUser(user.ID, hub.Event{
				Type:       "notification",
				EntityType: "notification",
				EntityID:   fmt.Sprintf("%d", seeds[i].ID),
				Action:     "upsert",
				UserID:     user.ID,
				Timestamp:  seeds[i].CreatedAt.UnixMilli(),
				Title:      seeds[i].Title,
				Message:    seeds[i].Message,
				NType:      seeds[i].Type,
			})
		}
	}

	return c.JSON(fiber.Map{"ok": true, "seeded": len(seeds)})
}

// ── Sponsors ───────────────────────────────────────────────────────────────

// ListSponsors handles GET /api/admin/sponsors
func (h *AdminCmsHandler) ListSponsors(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var sponsors []models.Sponsor
	h.db.Order(`"order" ASC, created_at DESC`).Find(&sponsors)
	return c.JSON(fiber.Map{"sponsors": sponsors})
}

// CreateSponsor handles POST /api/admin/sponsors
func (h *AdminCmsHandler) CreateSponsor(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var input struct {
		Name        string `json:"name"`
		LogoURL     string `json:"logo_url"`
		WebsiteURL  string `json:"website_url"`
		Tier        string `json:"tier"`
		Description string `json:"description"`
		Order       int    `json:"order"`
		Status      string `json:"status"`
	}
	if err := c.Bind().JSON(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if input.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	if input.Tier == "" {
		input.Tier = "silver"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	sponsor := models.Sponsor{
		Name:        input.Name,
		Slug:        adminSlugify(input.Name),
		LogoURL:     input.LogoURL,
		WebsiteURL:  input.WebsiteURL,
		Tier:        input.Tier,
		Description: input.Description,
		Order:       input.Order,
		Status:      input.Status,
	}
	if err := h.db.Create(&sponsor).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create sponsor"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"sponsor": sponsor})
}

// UpdateSponsor handles PUT /api/admin/sponsors/:id
func (h *AdminCmsHandler) UpdateSponsor(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	id := c.Params("id")
	var sponsor models.Sponsor
	if err := h.db.First(&sponsor, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sponsor not found"})
	}
	var input map[string]interface{}
	if err := c.Bind().JSON(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	h.db.Model(&sponsor).Updates(input)
	return c.JSON(fiber.Map{"sponsor": sponsor})
}

// DeleteSponsor handles DELETE /api/admin/sponsors/:id
func (h *AdminCmsHandler) DeleteSponsor(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	id := c.Params("id")
	if err := h.db.Delete(&models.Sponsor{}, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sponsor not found"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// ── Community Posts ────────────────────────────────────────────────────────

// ListCommunityPosts handles GET /api/admin/community/posts
func (h *AdminCmsHandler) ListCommunityPosts(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var posts []models.CommunityPost
	h.db.Preload("User").Order("created_at DESC").Find(&posts)

	type row struct {
		ID            uint   `json:"id"`
		UserID        uint   `json:"user_id"`
		AuthorName    string `json:"author_name"`
		AuthorHandle  string `json:"author_handle"`
		AuthorAvatar  string `json:"author_avatar"`
		Title         string `json:"title"`
		Content       string `json:"content"`
		Status        string `json:"status"`
		LikesCount    int    `json:"likes_count"`
		CommentsCount int    `json:"comments_count"`
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
	}
	rows := make([]row, 0, len(posts))
	for _, p := range posts {
		name := p.User.Name
		if name == "" {
			name = "Unknown"
		}
		handle := p.User.Email
		if handle == "" {
			handle = "user"
		}
		rows = append(rows, row{
			ID: p.ID, UserID: p.UserID,
			AuthorName: name, AuthorHandle: handle,
			AuthorAvatar: p.User.AvatarURL,
			Title: p.Title, Content: p.Content, Status: p.Status,
			LikesCount: p.LikesCount, CommentsCount: p.CommentsCount,
			CreatedAt: p.CreatedAt.Format(time.RFC3339),
			UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
		})
	}
	return c.JSON(fiber.Map{"posts": rows})
}

// UpdateCommunityPost handles PATCH /api/admin/community/posts/:id
func (h *AdminCmsHandler) UpdateCommunityPost(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	id := c.Params("id")
	var post models.CommunityPost
	if err := h.db.First(&post, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}
	var input map[string]interface{}
	if err := c.Bind().JSON(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	h.db.Model(&post).Updates(input)
	return c.JSON(fiber.Map{"post": post})
}

// DeleteCommunityPost handles DELETE /api/admin/community/posts/:id
func (h *AdminCmsHandler) DeleteCommunityPost(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	id := c.Params("id")
	if err := h.db.Delete(&models.CommunityPost{}, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// ── GitHub Issues ──────────────────────────────────────────────────────────

// ListGitHubRepos handles GET /api/admin/github/repos
func (h *AdminCmsHandler) ListGitHubRepos(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	var repos []models.GitHubRepo
	h.db.Order("full_name ASC").Find(&repos)
	return c.JSON(fiber.Map{"repos": repos})
}

// ListGitHubIssues handles GET /api/admin/github/issues
func (h *AdminCmsHandler) ListGitHubIssues(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	repo := c.Query("repo")
	q := h.db.Order("created_at DESC")
	if repo != "" {
		q = q.Where("repo = ?", repo)
	}
	var issues []models.GitHubIssue
	q.Find(&issues)

	type row struct {
		ID           uint     `json:"id"`
		GitHubID     int      `json:"github_id"`
		Repo         string   `json:"repo"`
		Title        string   `json:"title"`
		Body         string   `json:"body"`
		State        string   `json:"state"`
		Type         string   `json:"type"`
		Author       string   `json:"author"`
		AuthorAvatar string   `json:"author_avatar"`
		Labels       []string `json:"labels"`
		CreatedAt    string   `json:"created_at"`
		UpdatedAt    string   `json:"updated_at"`
	}
	rows := make([]row, 0, len(issues))
	for _, gi := range issues {
		labels := splitLabels(gi.Labels)
		rows = append(rows, row{
			ID: gi.ID, GitHubID: gi.GitHubID, Repo: gi.Repo,
			Title: gi.Title, Body: gi.Body, State: gi.State, Type: gi.Type,
			Author: gi.Author, AuthorAvatar: gi.AuthorAvatar, Labels: labels,
			CreatedAt: gi.CreatedAt.Format(time.RFC3339),
			UpdatedAt: gi.UpdatedAt.Format(time.RFC3339),
		})
	}
	return c.JSON(fiber.Map{"issues": rows})
}

// SyncGitHubIssues handles POST /api/admin/github/sync
func (h *AdminCmsHandler) SyncGitHubIssues(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}
	// Placeholder — actual GitHub API sync would go here.
	// For now return success so the frontend doesn't error.
	return c.JSON(fiber.Map{"ok": true, "message": "GitHub sync not yet configured. Add a GitHub token in admin settings."})
}

// ── Public endpoints (no admin check) ──────────────────────────────────────

// PublicSponsors handles GET /api/public/sponsors
func (h *AdminCmsHandler) PublicSponsors(c fiber.Ctx) error {
	var sponsors []models.Sponsor
	h.db.Where("status = ?", "active").Order("`order` ASC, created_at DESC").Find(&sponsors)
	return c.JSON(fiber.Map{"sponsors": sponsors})
}

// PublicPosts handles GET /api/public/blog — returns published blog posts from DB.
func (h *AdminCmsHandler) PublicPosts(c fiber.Ctx) error {
	var posts []models.Post
	h.db.Where("status = ?", "published").Order("published_at DESC, created_at DESC").Find(&posts)

	locale := helpers.ContentLocale(c)
	type postDTO struct {
		ID        uint   `json:"id"`
		Slug      string `json:"slug"`
		Title     string `json:"title"`
		Excerpt   string `json:"excerpt"`
		Tag       string `json:"tag"`
		Date      string `json:"date"`
		ReadTime  string `json:"read_time"`
	}
	result := make([]postDTO, 0, len(posts))
	for _, p := range posts {
		title := p.Title
		excerpt := p.Excerpt
		if locale != helpers.DefaultLocale {
			if tr := helpers.ResolveTranslation(p.Translations, locale); tr != nil {
				if v, ok := tr["title"].(string); ok && v != "" {
					title = v
				}
				if v, ok := tr["excerpt"].(string); ok && v != "" {
					excerpt = v
				}
			}
		}
		date := ""
		if p.PublishedAt != nil {
			date = p.PublishedAt.Format("Jan 02, 2006")
		} else {
			date = p.CreatedAt.Format("Jan 02, 2006")
		}
		result = append(result, postDTO{
			ID:      p.ID,
			Slug:    p.Slug,
			Title:   title,
			Excerpt: excerpt,
			Date:    date,
		})
	}
	return c.JSON(fiber.Map{"posts": result})
}

// PublicIssues handles GET /api/public/issues
func (h *AdminCmsHandler) PublicIssues(c fiber.Ctx) error {
	var issues []models.GitHubIssue
	h.db.Order("created_at DESC").Limit(100).Find(&issues)
	return c.JSON(fiber.Map{"issues": issues})
}

// ── helpers ────────────────────────────────────────────────────────────────

func adminSlugify(s string) string {
	out := make([]byte, 0, len(s))
	for _, ch := range []byte(s) {
		if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			out = append(out, ch)
		} else if ch >= 'A' && ch <= 'Z' {
			out = append(out, ch+32)
		} else if ch == ' ' || ch == '-' || ch == '_' {
			if len(out) > 0 && out[len(out)-1] != '-' {
				out = append(out, '-')
			}
		}
	}
	return string(out)
}

func splitLabels(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := []string{}
	for _, p := range splitByComma(s) {
		p = trimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitByComma(s string) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}
