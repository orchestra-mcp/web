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

// MCPHandler handles cloud-mcp service-to-service endpoints.
type MCPHandler struct {
	db *gorm.DB
}

// NewMCPHandler creates a new MCPHandler.
func NewMCPHandler(db *gorm.DB) *MCPHandler {
	return &MCPHandler{db: db}
}

// UserMCPPermission mirrors the table in cloud-mcp.
type UserMCPPermission struct {
	UserID     uint      `gorm:"primaryKey;column:user_id"`
	Permission string    `gorm:"primaryKey;column:permission;size:64"`
	Enabled    bool      `gorm:"default:true;column:enabled"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (UserMCPPermission) TableName() string { return "user_mcp_permissions" }

// GetProfile handles GET /api/mcp/profile?user_id=N
// Called by cloud-mcp service to retrieve user profile data.
func (h *MCPHandler) GetProfile(c fiber.Ctx) error {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id required"})
	}

	var user models.User
	if err := h.db.First(&user, userIDStr).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	// Parse settings JSON for extra profile fields.
	var settings map[string]interface{}
	if len(user.Settings) > 0 {
		json.Unmarshal(user.Settings, &settings)
	}
	if settings == nil {
		settings = map[string]interface{}{}
	}

	plan := "free"
	if p, ok := settings["plan"].(string); ok && p != "" {
		plan = p
	}
	timezone := ""
	if tz, ok := settings["timezone"].(string); ok {
		timezone = tz
	}
	githubUsername := ""
	if gh, ok := settings["github_username"].(string); ok {
		githubUsername = gh
	}
	bio := ""
	if b, ok := settings["bio"].(string); ok {
		bio = b
	}
	handle := ""
	if h2, ok := settings["handle"].(string); ok {
		handle = h2
	}

	return c.JSON(fiber.Map{
		"id":              user.ID,
		"name":            user.Name,
		"email":           user.Email,
		"role":            user.Role,
		"plan":            plan,
		"created_at":      user.CreatedAt.Format("January 2006"),
		"timezone":        timezone,
		"github_username": githubUsername,
		"bio":             bio,
		"handle":          handle,
		"avatar_url":      user.AvatarURL,
		"is_verified":     user.IsVerified,
	})
}

// PatchProfile handles PATCH /api/mcp/profile
// Called by cloud-mcp service to update profile fields.
func (h *MCPHandler) PatchProfile(c fiber.Ctx) error {
	// Use a raw map so we can detect which fields were actually provided.
	var body map[string]interface{}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	userIDRaw, ok := body["user_id"]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id required"})
	}
	// JSON numbers decode as float64.
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	}
	if userID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id required"})
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	// Update top-level user columns.
	updates := map[string]interface{}{}
	if v, ok := body["name"].(string); ok && v != "" {
		updates["name"] = v
	}
	if len(updates) > 0 {
		h.db.Model(&user).Updates(updates)
	}

	// Load existing settings JSON so we don't overwrite unrelated keys.
	var settings map[string]interface{}
	if len(user.Settings) > 0 {
		json.Unmarshal(user.Settings, &settings)
	}
	if settings == nil {
		settings = map[string]interface{}{}
	}

	changed := false

	// String fields stored in settings.
	for _, key := range []string{"timezone", "github_username", "bio", "handle", "cover_url", "phone", "gender", "position"} {
		if v, ok := body[key].(string); ok && v != "" {
			settings[key] = v
			changed = true
		}
	}

	// Boolean fields stored in settings.
	for _, key := range []string{
		"public_profile_enabled",
		"show_badges", "show_wallet", "show_teams",
		"show_sponsors", "show_comments_on_profile",
	} {
		if v, ok := body[key]; ok && v != nil {
			settings[key] = v
			changed = true
		}
	}

	// Object fields stored in settings (social_links, appearance).
	for _, key := range []string{"social_links", "appearance"} {
		if v, ok := body[key]; ok && v != nil {
			settings[key] = v
			changed = true
		}
	}

	if changed {
		b, _ := json.Marshal(settings)
		h.db.Model(&user).Update("settings", string(b))
	}

	return c.JSON(fiber.Map{"ok": true})
}

// GetPermissions handles GET /api/settings/mcp-permissions
// Returns the current user's MCP permission toggles.
func (h *MCPHandler) GetPermissions(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	uid := user.ID

	defaults := map[string]bool{
		"mcp.install":       true,
		"mcp.status":        true,
		"mcp.profile.read":  true,
		"mcp.profile.write": false,
		"mcp.marketplace":   true,
	}

	var rows []UserMCPPermission
	h.db.Where("user_id = ?", uid).Find(&rows)
	for _, r := range rows {
		defaults[r.Permission] = r.Enabled
	}

	return c.JSON(fiber.Map{"permissions": defaults})
}

// PatchPermissions handles PATCH /api/settings/mcp-permissions
// Updates the current user's MCP permission toggles.
func (h *MCPHandler) PatchPermissions(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	uid := user.ID

	var body struct {
		Permissions map[string]bool `json:"permissions"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Permissions == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	for perm, enabled := range body.Permissions {
		row := UserMCPPermission{
			UserID:     uid,
			Permission: perm,
			Enabled:    enabled,
			UpdatedAt:  time.Now(),
		}
		h.db.Save(&row)
	}

	return c.JSON(fiber.Map{"ok": true})
}

// GetMCPToken handles GET /api/settings/mcp-token
// Returns the user's dedicated MCP API token (creates one if missing).
func (h *MCPHandler) GetMCPToken(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	token, err := h.getOrCreateMCPToken(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load token"})
	}
	return c.JSON(fiber.Map{"token": token})
}

// RegenerateMCPToken handles POST /api/settings/mcp-token/regenerate
// Replaces the user's MCP API token with a new one.
func (h *MCPHandler) RegenerateMCPToken(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	token, err := h.rotateMCPToken(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to regenerate token"})
	}
	return c.JSON(fiber.Map{"token": token})
}

// getOrCreateMCPToken returns the existing mcp_token from settings, or creates one.
func (h *MCPHandler) getOrCreateMCPToken(user *models.User) (string, error) {
	var meta map[string]interface{}
	if len(user.Settings) > 0 {
		json.Unmarshal(user.Settings, &meta)
	}
	if meta == nil {
		meta = map[string]interface{}{}
	}
	if t, ok := meta["mcp_token"].(string); ok && t != "" {
		return t, nil
	}
	return h.rotateMCPToken(user)
}

// rotateMCPToken generates a new orch_* token, stores its hash, and returns the raw token.
func (h *MCPHandler) rotateMCPToken(user *models.User) (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := "orch_" + hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	hashHex := hex.EncodeToString(sum[:])

	var meta map[string]interface{}
	if len(user.Settings) > 0 {
		json.Unmarshal(user.Settings, &meta)
	}
	if meta == nil {
		meta = map[string]interface{}{}
	}

	// Remove old mcp key from api_keys list.
	oldHash, _ := meta["mcp_token_hash"].(string)
	if raw2, ok := meta["api_keys"]; ok {
		b, _ := json.Marshal(raw2)
		var keys []map[string]interface{}
		if json.Unmarshal(b, &keys) == nil {
			filtered := make([]map[string]interface{}, 0, len(keys))
			for _, k := range keys {
				if k["hash"] != oldHash {
					filtered = append(filtered, k)
				}
			}
			meta["api_keys"] = filtered
		}
	}

	// Store new key.
	meta["mcp_token"] = token
	meta["mcp_token_hash"] = hashHex

	// Append to api_keys so ValidateToken can find it.
	var keys []map[string]interface{}
	if raw2, ok := meta["api_keys"]; ok {
		b, _ := json.Marshal(raw2)
		json.Unmarshal(b, &keys)
	}
	keys = append(keys, map[string]interface{}{
		"id":   "mcp",
		"name": "MCP Access",
		"hash": hashHex,
	})
	meta["api_keys"] = keys

	b, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	if err := h.db.Model(user).Update("settings", string(b)).Error; err != nil {
		return "", err
	}
	return token, nil
}
