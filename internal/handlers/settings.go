package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// SettingsHandler handles user settings endpoints.
type SettingsHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(db *gorm.DB, cfg ...*config.Config) *SettingsHandler {
	h := &SettingsHandler{db: db}
	if len(cfg) > 0 && cfg[0] != nil {
		h.cfg = cfg[0]
	}
	return h
}

// UpdateProfile handles PATCH /api/settings/profile (extended)
func (h *SettingsHandler) UpdateProfile(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name                 string        `json:"name"`
		Email                string        `json:"email"`
		Phone                string        `json:"phone"`
		Gender               string        `json:"gender"`
		Position             string        `json:"position"`
		Timezone             string        `json:"timezone"`
		Bio                  string        `json:"bio"`
		About                string        `json:"about"`
		PublicProfileEnabled *bool         `json:"public_profile_enabled,omitempty"`
		Handle               *string       `json:"handle,omitempty"`
		CoverURL             *string       `json:"cover_url,omitempty"`
		SocialLinks          interface{}   `json:"social_links,omitempty"`
		Sponsors             interface{}   `json:"sponsors,omitempty"`
		ShowBadges           *bool         `json:"show_badges,omitempty"`
		ShowWallet           *bool         `json:"show_wallet,omitempty"`
		ShowTeams            *bool         `json:"show_teams,omitempty"`
		ShowSponsors         *bool         `json:"show_sponsors,omitempty"`
		ShowCommentsOnProfile *bool        `json:"show_comments_on_profile,omitempty"`
		Appearance            interface{}  `json:"appearance,omitempty"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Load existing settings to preserve other keys (api_keys, preferences, etc.)
	var settings map[string]interface{}
	if err := json.Unmarshal(user.Settings, &settings); err != nil {
		settings = map[string]interface{}{}
	}

	// Merge profile metadata into settings
	settings["phone"] = body.Phone
	settings["gender"] = body.Gender
	settings["position"] = body.Position
	settings["timezone"] = body.Timezone
	settings["bio"] = body.Bio
	settings["about"] = body.About

	// Public profile fields (only set if provided)
	if body.PublicProfileEnabled != nil {
		settings["public_profile_enabled"] = *body.PublicProfileEnabled
	}
	if body.Handle != nil && *body.Handle != "" {
		settings["handle"] = *body.Handle
	}
	if body.CoverURL != nil {
		settings["cover_url"] = *body.CoverURL
	}
	if body.SocialLinks != nil {
		settings["social_links"] = body.SocialLinks
	}
	if body.Sponsors != nil {
		settings["sponsors"] = body.Sponsors
	}
	if body.ShowBadges != nil {
		settings["show_badges"] = *body.ShowBadges
	}
	if body.ShowWallet != nil {
		settings["show_wallet"] = *body.ShowWallet
	}
	if body.ShowTeams != nil {
		settings["show_teams"] = *body.ShowTeams
	}
	if body.ShowSponsors != nil {
		settings["show_sponsors"] = *body.ShowSponsors
	}
	if body.ShowCommentsOnProfile != nil {
		settings["show_comments_on_profile"] = *body.ShowCommentsOnProfile
	}
	if body.Appearance != nil {
		settings["appearance"] = body.Appearance
	}

	settingsJSON, _ := json.Marshal(settings)

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Email != "" && body.Email != user.Email {
		var existing models.User
		if err := h.db.Where("email = ? AND id != ?", body.Email, user.ID).First(&existing).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already in use"})
		}
		updates["email"] = body.Email
	}
	updates["settings"] = settingsJSON

	if len(updates) > 0 {
		if err := h.db.Model(user).Updates(updates).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update profile"})
		}
	}

	h.db.First(user, user.ID)
	return c.JSON(user)
}

// uploadDir returns the base upload directory (absolute path from config, or relative fallback).
func (h *SettingsHandler) uploadDir() string {
	if h.cfg != nil && h.cfg.UploadDir != "" {
		return h.cfg.UploadDir
	}
	return "uploads"
}

// UploadAvatar handles POST /api/settings/avatar
func (h *SettingsHandler) UploadAvatar(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "avatar file is required"})
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid file type, must be jpg/png/gif/webp"})
	}

	// Validate file size (max 2MB)
	if file.Size > 2*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file too large, max 2MB"})
	}

	// Ensure uploads directory exists (use absolute path from config).
	avatarDir := filepath.Join(h.uploadDir(), "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create upload directory"})
	}

	// Save file with unique name
	filename := fmt.Sprintf("%d-%d%s", user.ID, time.Now().Unix(), ext)
	savePath := filepath.Join(avatarDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}

	// Delete old avatar file if it exists
	if user.AvatarURL != "" && strings.HasPrefix(user.AvatarURL, "/uploads/avatars/") {
		oldFilename := strings.TrimPrefix(user.AvatarURL, "/uploads/avatars/")
		_ = os.Remove(filepath.Join(avatarDir, oldFilename))
	}

	// Update user record — URL path is always relative for the client.
	avatarURL := "/uploads/avatars/" + filename
	if err := h.db.Model(user).Update("avatar_url", avatarURL).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update avatar"})
	}

	return c.JSON(fiber.Map{"ok": true, "avatar_url": avatarURL})
}

// UploadCover handles POST /api/settings/cover
func (h *SettingsHandler) UploadCover(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	file, err := c.FormFile("cover")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cover file is required"})
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid file type, must be jpg/png/gif/webp"})
	}

	// Validate file size (max 5MB for cover images)
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file too large, max 5MB"})
	}

	// Ensure uploads directory exists (use absolute path from config).
	coverDir := filepath.Join(h.uploadDir(), "covers")
	if err := os.MkdirAll(coverDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create upload directory"})
	}

	// Save file with unique name
	filename := fmt.Sprintf("%d-%d%s", user.ID, time.Now().Unix(), ext)
	savePath := filepath.Join(coverDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
	}

	// Delete old cover file if it exists
	var settings map[string]interface{}
	if err := json.Unmarshal(user.Settings, &settings); err == nil {
		if oldCover, ok := settings["cover_url"].(string); ok && strings.HasPrefix(oldCover, "/uploads/covers/") {
			oldFilename := strings.TrimPrefix(oldCover, "/uploads/covers/")
			_ = os.Remove(filepath.Join(coverDir, oldFilename))
		}
	} else {
		settings = map[string]interface{}{}
	}

	// Update user settings with new cover URL — URL path is always relative for the client.
	coverURL := "/uploads/covers/" + filename
	settings["cover_url"] = coverURL
	settingsJSON, _ := json.Marshal(settings)

	if err := h.db.Model(user).Update("settings", gorm.Expr(
		"COALESCE(settings, '{}'::jsonb) || ?::jsonb",
		string(settingsJSON),
	)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update cover"})
	}

	return c.JSON(fiber.Map{"ok": true, "cover_url": coverURL})
}

// ListSessions handles GET /api/settings/sessions
func (h *SettingsHandler) ListSessions(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	ua := string(c.Request().Header.UserAgent())
	device, os, browser := parseUserAgent(ua)

	// Current session
	sessions := []fiber.Map{
		{
			"id":          "current",
			"device":      device,
			"os":          os,
			"browser":     browser,
			"device_type": classifyDevice(ua),
			"ip":          c.IP(),
			"user_agent":  ua,
			"last_seen":   time.Now().Format(time.RFC3339),
			"is_current":  true,
		},
	}

	// Include registered device tokens as sessions
	var devices []models.DeviceToken
	h.db.Where("user_id = ?", user.ID).Order("last_seen_at DESC").Find(&devices)
	for _, d := range devices {
		lastSeen := ""
		if d.LastSeenAt != nil {
			lastSeen = d.LastSeenAt.Format(time.RFC3339)
		}
		sess := fiber.Map{
			"id":          fmt.Sprintf("device:%s", d.ID),
			"device":      d.Name,
			"device_type": d.Platform,
			"ip":          "",
			"last_seen":   lastSeen,
			"is_current":  false,
		}
		// Cross-reference with active tunnels
		var tunnel models.Tunnel
		if err := h.db.Where("user_id = ? AND status = ?", user.ID, "online").First(&tunnel).Error; err == nil {
			sess["tunnel_active"] = true
		}
		sessions = append(sessions, sess)
	}

	return c.JSON(fiber.Map{"sessions": sessions})
}

// parseUserAgent extracts device name, OS, and browser from a user-agent string.
func parseUserAgent(ua string) (device, os, browser string) {
	lower := strings.ToLower(ua)
	// OS detection
	switch {
	case strings.Contains(lower, "iphone"):
		os, device = "iOS", "iPhone"
	case strings.Contains(lower, "ipad"):
		os, device = "iPadOS", "iPad"
	case strings.Contains(lower, "mac os") || strings.Contains(lower, "macos"):
		os, device = "macOS", "Mac"
	case strings.Contains(lower, "android"):
		os, device = "Android", "Android Device"
	case strings.Contains(lower, "windows"):
		os, device = "Windows", "PC"
	case strings.Contains(lower, "linux"):
		os, device = "Linux", "Linux PC"
	default:
		os, device = "Unknown", "Unknown Device"
	}
	// Browser detection
	switch {
	case strings.Contains(lower, "edg/"):
		browser = "Edge"
	case strings.Contains(lower, "chrome") && !strings.Contains(lower, "edg/"):
		browser = "Chrome"
	case strings.Contains(lower, "firefox"):
		browser = "Firefox"
	case strings.Contains(lower, "safari") && !strings.Contains(lower, "chrome"):
		browser = "Safari"
	case strings.Contains(lower, "orchestra"):
		browser = "Orchestra App"
	default:
		browser = "Unknown"
	}
	return
}

// classifyDevice returns the device type from a user-agent string.
func classifyDevice(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "android") && strings.Contains(lower, "mobile"):
		return "mobile"
	case strings.Contains(lower, "ipad") || strings.Contains(lower, "tablet"):
		return "tablet"
	default:
		return "desktop"
	}
}

// RevokeSession handles DELETE /api/settings/sessions/:id
func (h *SettingsHandler) RevokeSession(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	sessionID := c.Params("id")
	if sessionID == "current" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot revoke current session"})
	}

	// Handle device token sessions (format: "device:123")
	if strings.HasPrefix(sessionID, "device:") {
		idStr := strings.TrimPrefix(sessionID, "device:")
		if err := h.db.Where("id = ? AND user_id = ?", idStr, user.ID).Delete(&models.DeviceToken{}).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
		}
	}

	return c.JSON(fiber.Map{"ok": true})
}

type apiKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	Hash      string `json:"hash,omitempty"`
	CreatedAt string `json:"created_at"`
	LastUsed  string `json:"last_used"`
}

// hashAPIKey returns the SHA-256 hex digest of a raw API key token.
func hashAPIKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ListApiKeys handles GET /api/settings/api-keys
func (h *SettingsHandler) ListApiKeys(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Load from user.Settings JSON field
	var meta map[string]interface{}
	if err := json.Unmarshal(user.Settings, &meta); err != nil {
		meta = map[string]interface{}{}
	}

	keys := []apiKey{}
	if raw, ok := meta["api_keys"]; ok {
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &keys)
	}
	return c.JSON(fiber.Map{"api_keys": keys})
}

// CreateApiKey handles POST /api/settings/api-keys
func (h *SettingsHandler) CreateApiKey(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	b := make([]byte, 20)
	_, _ = rand.Read(b)
	token := "orch_" + hex.EncodeToString(b)

	newKey := apiKey{
		ID:        hex.EncodeToString(b[:8]),
		Name:      body.Name,
		Prefix:    token[:13] + "...",
		Hash:      hashAPIKey(token),
		CreatedAt: time.Now().Format(time.RFC3339),
		LastUsed:  "",
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(user.Settings, &meta); err != nil {
		meta = map[string]interface{}{}
	}
	existing := []apiKey{}
	if raw, ok := meta["api_keys"]; ok {
		b2, _ := json.Marshal(raw)
		_ = json.Unmarshal(b2, &existing)
	}
	existing = append(existing, newKey)
	meta["api_keys"] = existing
	metaJSON, _ := json.Marshal(meta)
	h.db.Model(user).Update("settings", metaJSON)

	return c.JSON(fiber.Map{"ok": true, "token": token, "key": newKey})
}

// RevokeApiKey handles DELETE /api/settings/api-keys/:id
func (h *SettingsHandler) RevokeApiKey(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	var meta map[string]interface{}
	if err := json.Unmarshal(user.Settings, &meta); err != nil {
		meta = map[string]interface{}{}
	}
	keys := []apiKey{}
	if raw, ok := meta["api_keys"]; ok {
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &keys)
	}
	filtered := []apiKey{}
	for _, k := range keys {
		if k.ID != id {
			filtered = append(filtered, k)
		}
	}
	meta["api_keys"] = filtered
	metaJSON, _ := json.Marshal(meta)
	h.db.Model(user).Update("settings", metaJSON)

	return c.JSON(fiber.Map{"ok": true})
}

// ListConnectedAccounts handles GET /api/settings/connected-accounts
func (h *SettingsHandler) ListConnectedAccounts(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var accounts []models.OAuthAccount
	h.db.Where("user_id = ?", user.ID).Find(&accounts)

	type AccountRow struct {
		Provider string `json:"provider"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Avatar   string `json:"avatar"`
	}
	rows := make([]AccountRow, len(accounts))
	for i, a := range accounts {
		rows[i] = AccountRow{Provider: a.Provider, Email: a.Email, Name: a.Name, Avatar: a.Avatar}
	}
	return c.JSON(fiber.Map{"accounts": rows})
}

// UnlinkAccount handles DELETE /api/settings/connected-accounts/:provider
func (h *SettingsHandler) UnlinkAccount(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	provider := c.Params("provider")
	if err := h.db.Where("user_id = ? AND provider = ?", user.ID, provider).Delete(&models.OAuthAccount{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to unlink account"})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// defaultPreferences — returned when user has no saved preferences.
var defaultPreferences = map[string]interface{}{
	"theme":                "dark",
	"language":             "en",
	"sidebar_collapsed":    false,
	"notifications_email":  true,
	"notifications_push":   true,
	"notifications_sound":  false,
	"editor_font_size":     14,
	"editor_tab_size":      2,
	"editor_word_wrap":     true,
	"compact_mode":         false,
}

// GetPreferences handles GET /api/settings/preferences
func (h *SettingsHandler) GetPreferences(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(user.Settings, &settings); err != nil {
		return c.JSON(fiber.Map{"preferences": defaultPreferences})
	}

	prefs, ok := settings["preferences"]
	if !ok || prefs == nil {
		return c.JSON(fiber.Map{"preferences": defaultPreferences})
	}

	// Merge defaults with saved preferences so new keys are always present.
	saved, _ := prefs.(map[string]interface{})
	merged := make(map[string]interface{}, len(defaultPreferences))
	for k, v := range defaultPreferences {
		merged[k] = v
	}
	for k, v := range saved {
		merged[k] = v
	}

	return c.JSON(fiber.Map{"preferences": merged})
}

// UpdatePreferences handles PATCH /api/settings/preferences
func (h *SettingsHandler) UpdatePreferences(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var incoming map[string]interface{}
	if err := json.Unmarshal(c.Body(), &incoming); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Load existing settings.
	var settings map[string]interface{}
	if err := json.Unmarshal(user.Settings, &settings); err != nil {
		settings = map[string]interface{}{}
	}

	// Merge: defaults → existing → incoming.
	existing, _ := settings["preferences"].(map[string]interface{})
	merged := make(map[string]interface{}, len(defaultPreferences))
	for k, v := range defaultPreferences {
		merged[k] = v
	}
	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range incoming {
		merged[k] = v
	}

	settings["preferences"] = merged
	settingsJSON, _ := json.Marshal(settings)

	if err := h.db.Model(user).Update("settings", gorm.Expr(
		"COALESCE(settings, '{}'::jsonb) || ?::jsonb",
		string(settingsJSON),
	)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save preferences"})
	}

	return c.JSON(fiber.Map{"preferences": merged})
}

// ListNotifications handles GET /api/notifications
func (h *SettingsHandler) ListNotifications(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var notifications []models.Notification
	h.db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&notifications)

	return c.JSON(fiber.Map{"notifications": notifications})
}

// MarkNotificationRead handles PATCH /api/notifications/:id/read
func (h *SettingsHandler) MarkNotificationRead(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	now := time.Now()
	h.db.Model(&models.Notification{}).Where("id = ? AND user_id = ?", id, user.ID).Update("read_at", &now)
	return c.JSON(fiber.Map{"ok": true})
}

// CreateIssue handles POST /api/issues
func (h *SettingsHandler) CreateIssue(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Title == "" || body.Description == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title and description are required"})
	}

	priority := body.Priority
	if priority == "" {
		priority = "medium"
	}

	issue := models.Issue{
		UserID:      user.ID,
		Title:       body.Title,
		Description: body.Description,
		Priority:    priority,
		Status:      "open",
	}
	if err := h.db.Create(&issue).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create issue"})
	}

	return c.Status(fiber.StatusCreated).JSON(issue)
}

// ListMyIssues handles GET /api/issues
func (h *SettingsHandler) ListMyIssues(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var issues []models.Issue
	h.db.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&issues)

	return c.JSON(fiber.Map{"issues": issues})
}

// DeleteNotification handles DELETE /api/notifications/:id
func (h *SettingsHandler) DeleteNotification(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id := c.Params("id")
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Notification{}).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "notification not found"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// MarkAllNotificationsRead handles PATCH /api/notifications/read-all
func (h *SettingsHandler) MarkAllNotificationsRead(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	now := time.Now()
	h.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", user.ID).
		Update("read_at", &now)

	return c.JSON(fiber.Map{"ok": true})
}

// ── Push Subscriptions ──────────────────────────────────────────────────────

// RegisterPushSubscription handles POST /api/notifications/push/subscribe
func (h *SettingsHandler) RegisterPushSubscription(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Endpoint  string `json:"endpoint"`
		P256dh    string `json:"p256dh"`
		Auth      string `json:"auth"`
		Platform  string `json:"platform"`
		UserAgent string `json:"user_agent"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Endpoint == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "endpoint is required"})
	}

	platform := body.Platform
	if platform == "" {
		platform = "web"
	}

	// Upsert — update if endpoint already exists for this user.
	var sub models.PushSubscription
	result := h.db.Where("endpoint = ?", body.Endpoint).First(&sub)
	if result.Error != nil {
		sub = models.PushSubscription{
			UserID:    user.ID,
			Endpoint:  body.Endpoint,
			P256dh:    body.P256dh,
			Auth:      body.Auth,
			Platform:  platform,
			UserAgent: body.UserAgent,
		}
		h.db.Create(&sub)
	} else {
		h.db.Model(&sub).Updates(map[string]any{
			"user_id":    user.ID,
			"p256dh":     body.P256dh,
			"auth":       body.Auth,
			"platform":   platform,
			"user_agent": body.UserAgent,
		})
	}

	return c.JSON(fiber.Map{"ok": true, "id": sub.ID})
}

// UnregisterPushSubscription handles DELETE /api/notifications/push/unsubscribe
func (h *SettingsHandler) UnregisterPushSubscription(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	h.db.Where("user_id = ? AND endpoint = ?", user.ID, body.Endpoint).
		Delete(&models.PushSubscription{})

	return c.JSON(fiber.Map{"ok": true})
}

// ListPushSubscriptions handles GET /api/notifications/push/subscriptions
func (h *SettingsHandler) ListPushSubscriptions(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var subs []models.PushSubscription
	h.db.Where("user_id = ?", user.ID).Find(&subs)

	return c.JSON(fiber.Map{"subscriptions": subs, "count": len(subs)})
}
