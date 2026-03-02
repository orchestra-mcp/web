package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// SettingsHandler handles user settings endpoints.
type SettingsHandler struct {
	db *gorm.DB
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(db *gorm.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

// UpdateProfile handles PATCH /api/settings/profile (extended)
func (h *SettingsHandler) UpdateProfile(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Position string `json:"position"`
		Timezone string `json:"timezone"`
		Bio      string `json:"bio"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Merge into Settings JSON
	type profileMeta struct {
		Phone    string `json:"phone"`
		Position string `json:"position"`
		Timezone string `json:"timezone"`
		Bio      string `json:"bio"`
	}
	meta := profileMeta{
		Phone:    body.Phone,
		Position: body.Position,
		Timezone: body.Timezone,
		Bio:      body.Bio,
	}
	metaJSON, _ := json.Marshal(meta)

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
	updates["settings"] = metaJSON

	if len(updates) > 0 {
		if err := h.db.Model(user).Updates(updates).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update profile"})
		}
	}

	h.db.First(user, user.ID)
	return c.JSON(user)
}

// ListSessions handles GET /api/settings/sessions
func (h *SettingsHandler) ListSessions(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Return a synthetic "current session" based on the request
	sessions := []fiber.Map{
		{
			"id":         "current",
			"device":     "Web Browser",
			"ip":         c.IP(),
			"user_agent": string(c.Request().Header.UserAgent()),
			"last_seen":  time.Now().Format(time.RFC3339),
			"is_current": true,
		},
	}
	return c.JSON(fiber.Map{"sessions": sessions})
}

// RevokeSession handles DELETE /api/settings/sessions/:id
func (h *SettingsHandler) RevokeSession(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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
