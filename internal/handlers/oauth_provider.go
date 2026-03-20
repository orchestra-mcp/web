package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// OAuthProviderHandler implements Orchestra as an OAuth2 authorization server.
type OAuthProviderHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewOAuthProviderHandler creates a new OAuthProviderHandler.
func NewOAuthProviderHandler(db *gorm.DB, cfg *config.Config) *OAuthProviderHandler {
	return &OAuthProviderHandler{db: db, cfg: cfg}
}

// ---------------------------------------------------------------------------
// GET /oauth/authorize — show consent page / return authorization form data
// ---------------------------------------------------------------------------

func (h *OAuthProviderHandler) Authorize(c fiber.Ctx) error {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	scope := c.Query("scope")
	responseType := c.Query("response_type")

	if responseType != "code" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported_response_type"})
	}
	if clientID == "" || redirectURI == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "message": "client_id and redirect_uri are required"})
	}

	// Look up the app
	var app models.OAuthApp
	if err := h.db.Where("client_id = ?", clientID).First(&app).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_client"})
	}

	// Validate redirect URI
	var uris []string
	if err := json.Unmarshal(app.RedirectURIs, &uris); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}
	validURI := false
	for _, u := range uris {
		if u == redirectURI {
			validURI = true
			break
		}
	}
	if !validURI {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_redirect_uri"})
	}

	// Parse scopes
	scopes := strings.Fields(scope)
	if len(scopes) == 0 {
		scopes = []string{"profile"}
	}

	// For GET: return app info for consent page
	// For POST: process approval/denial (handled by ApproveAuthorization)
	return c.JSON(fiber.Map{
		"app": fiber.Map{
			"name":     app.Name,
			"icon_url": app.IconURL,
		},
		"scopes":       scopes,
		"client_id":    clientID,
		"redirect_uri": redirectURI,
	})
}

// ---------------------------------------------------------------------------
// POST /oauth/authorize — user approves or denies the authorization
// ---------------------------------------------------------------------------

func (h *OAuthProviderHandler) ApproveAuthorization(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		ClientID    string `json:"client_id"`
		RedirectURI string `json:"redirect_uri"`
		Scope       string `json:"scope"`
		Approved    bool   `json:"approved"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	if !body.Approved {
		redirectURL := fmt.Sprintf("%s?error=access_denied", body.RedirectURI)
		return c.JSON(fiber.Map{"redirect": redirectURL})
	}

	// Look up the app
	var app models.OAuthApp
	if err := h.db.Where("client_id = ?", body.ClientID).First(&app).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_client"})
	}

	// Generate authorization code
	codeBytes := make([]byte, 32)
	if _, err := rand.Read(codeBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}
	code := hex.EncodeToString(codeBytes)

	scopes := strings.Fields(body.Scope)
	scopesJSON, _ := json.Marshal(scopes)

	auth := models.OAuthAuthorization{
		AppID:       app.ID,
		UserID:      user.ID,
		Code:        code,
		Scopes:      scopesJSON,
		RedirectURI: body.RedirectURI,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	if err := h.db.Create(&auth).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}

	redirectURL := fmt.Sprintf("%s?code=%s", body.RedirectURI, url.QueryEscape(code))
	return c.JSON(fiber.Map{"redirect": redirectURL, "code": code})
}

// ---------------------------------------------------------------------------
// POST /oauth/token — exchange code for tokens or refresh
// ---------------------------------------------------------------------------

func (h *OAuthProviderHandler) Token(c fiber.Ctx) error {
	grantType := c.FormValue("grant_type")
	if grantType == "" {
		// Also try JSON body
		var body struct {
			GrantType    string `json:"grant_type"`
			Code         string `json:"code"`
			RedirectURI  string `json:"redirect_uri"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.Unmarshal(c.Body(), &body); err == nil {
			grantType = body.GrantType
			if c.FormValue("code") == "" {
				c.Request().PostArgs().Set("code", body.Code)
				c.Request().PostArgs().Set("redirect_uri", body.RedirectURI)
				c.Request().PostArgs().Set("client_id", body.ClientID)
				c.Request().PostArgs().Set("client_secret", body.ClientSecret)
				c.Request().PostArgs().Set("refresh_token", body.RefreshToken)
			}
		}
	}

	switch grantType {
	case "authorization_code":
		return h.tokenAuthorizationCode(c)
	case "refresh_token":
		return h.tokenRefresh(c)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported_grant_type"})
	}
}

func (h *OAuthProviderHandler) tokenAuthorizationCode(c fiber.Ctx) error {
	code := c.FormValue("code")
	redirectURI := c.FormValue("redirect_uri")
	clientID := c.FormValue("client_id")
	clientSecret := c.FormValue("client_secret")

	if code == "" || clientID == "" || clientSecret == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	// Validate client
	var app models.OAuthApp
	if err := h.db.Where("client_id = ?", clientID).First(&app).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_client"})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(app.ClientSecret), []byte(clientSecret)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_client"})
	}

	// Find and validate authorization code
	var auth models.OAuthAuthorization
	if err := h.db.Where("code = ? AND app_id = ?", code, app.ID).First(&auth).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant"})
	}
	if auth.Used {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant", "message": "code already used"})
	}
	if time.Now().After(auth.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant", "message": "code expired"})
	}
	if redirectURI != "" && auth.RedirectURI != redirectURI {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant", "message": "redirect_uri mismatch"})
	}

	// Mark code as used
	h.db.Model(&auth).Update("used", true)

	// Generate tokens
	accessTokenBytes := make([]byte, 32)
	refreshTokenBytes := make([]byte, 32)
	rand.Read(accessTokenBytes)
	rand.Read(refreshTokenBytes)

	accessToken := hex.EncodeToString(accessTokenBytes)
	refreshToken := hex.EncodeToString(refreshTokenBytes)

	token := models.OAuthAccessToken{
		AppID:        app.ID,
		UserID:       auth.UserID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Scopes:       auth.Scopes,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	if err := h.db.Create(&token).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refreshToken,
		"scope":         scopeString(auth.Scopes),
	})
}

func (h *OAuthProviderHandler) tokenRefresh(c fiber.Ctx) error {
	refreshToken := c.FormValue("refresh_token")
	clientID := c.FormValue("client_id")
	clientSecret := c.FormValue("client_secret")

	if refreshToken == "" || clientID == "" || clientSecret == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	// Validate client
	var app models.OAuthApp
	if err := h.db.Where("client_id = ?", clientID).First(&app).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_client"})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(app.ClientSecret), []byte(clientSecret)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_client"})
	}

	// Find existing token
	var existing models.OAuthAccessToken
	if err := h.db.Where("refresh_token = ? AND app_id = ?", refreshToken, app.ID).First(&existing).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_grant"})
	}

	// Rotate tokens
	newAccessBytes := make([]byte, 32)
	newRefreshBytes := make([]byte, 32)
	rand.Read(newAccessBytes)
	rand.Read(newRefreshBytes)

	newAccess := hex.EncodeToString(newAccessBytes)
	newRefresh := hex.EncodeToString(newRefreshBytes)

	h.db.Model(&existing).Updates(map[string]any{
		"access_token":  newAccess,
		"refresh_token": newRefresh,
		"expires_at":    time.Now().Add(1 * time.Hour),
	})

	return c.JSON(fiber.Map{
		"access_token":  newAccess,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": newRefresh,
		"scope":         scopeString(existing.Scopes),
	})
}

// ---------------------------------------------------------------------------
// GET /oauth/userinfo — OIDC user profile
// ---------------------------------------------------------------------------

func (h *OAuthProviderHandler) UserInfo(c fiber.Ctx) error {
	accessToken := extractBearerToken(c)
	if accessToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_token"})
	}

	var token models.OAuthAccessToken
	if err := h.db.Where("access_token = ?", accessToken).First(&token).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_token"})
	}
	if time.Now().After(token.ExpiresAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token_expired"})
	}

	var user models.User
	if err := h.db.First(&user, token.UserID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "server_error"})
	}

	return c.JSON(fiber.Map{
		"sub":            fmt.Sprintf("%d", user.ID),
		"name":           user.Name,
		"email":          user.Email,
		"email_verified": user.EmailVerifiedAt != nil,
		"picture":        user.AvatarURL,
	})
}

// ---------------------------------------------------------------------------
// POST /oauth/revoke — revoke a token
// ---------------------------------------------------------------------------

func (h *OAuthProviderHandler) Revoke(c fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request"})
	}

	// Try access token first, then refresh token
	result := h.db.Where("access_token = ?", body.Token).Delete(&models.OAuthAccessToken{})
	if result.RowsAffected == 0 {
		h.db.Where("refresh_token = ?", body.Token).Delete(&models.OAuthAccessToken{})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// ---------------------------------------------------------------------------
// Admin CRUD: POST/GET/PUT/DELETE /api/admin/oauth-apps
// ---------------------------------------------------------------------------

func (h *OAuthProviderHandler) ListApps(c fiber.Ctx) error {
	var apps []models.OAuthApp
	h.db.Order("created_at DESC").Find(&apps)
	return c.JSON(fiber.Map{"apps": apps})
}

func (h *OAuthProviderHandler) CreateApp(c fiber.Ctx) error {
	var body struct {
		Name         string   `json:"name"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes"`
		IconURL      string   `json:"icon_url"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.Name == "" || len(body.RedirectURIs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and redirect_uris are required"})
	}

	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Generate client ID and secret
	clientIDBytes := make([]byte, 16)
	clientSecretBytes := make([]byte, 32)
	rand.Read(clientIDBytes)
	rand.Read(clientSecretBytes)

	clientID := hex.EncodeToString(clientIDBytes)
	clientSecretPlain := hex.EncodeToString(clientSecretBytes)
	clientSecretHash, _ := bcrypt.GenerateFromPassword([]byte(clientSecretPlain), 12)

	redirectJSON, _ := json.Marshal(body.RedirectURIs)
	scopesJSON, _ := json.Marshal(body.Scopes)

	app := models.OAuthApp{
		Name:         body.Name,
		ClientID:     clientID,
		ClientSecret: string(clientSecretHash),
		RedirectURIs: redirectJSON,
		Scopes:       scopesJSON,
		OwnerID:      user.ID,
		IconURL:      body.IconURL,
	}
	if err := h.db.Create(&app).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create app"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"app":           app,
		"client_secret": clientSecretPlain, // Only shown once
	})
}

func (h *OAuthProviderHandler) GetApp(c fiber.Ctx) error {
	id := c.Params("id")
	var app models.OAuthApp
	if err := h.db.First(&app, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "app not found"})
	}
	return c.JSON(app)
}

func (h *OAuthProviderHandler) UpdateApp(c fiber.Ctx) error {
	id := c.Params("id")
	var app models.OAuthApp
	if err := h.db.First(&app, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "app not found"})
	}

	var body struct {
		Name         string   `json:"name"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       []string `json:"scopes"`
		IconURL      string   `json:"icon_url"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]any{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if len(body.RedirectURIs) > 0 {
		j, _ := json.Marshal(body.RedirectURIs)
		updates["redirect_uris"] = j
	}
	if len(body.Scopes) > 0 {
		j, _ := json.Marshal(body.Scopes)
		updates["scopes"] = j
	}
	if body.IconURL != "" {
		updates["icon_url"] = body.IconURL
	}

	h.db.Model(&app).Updates(updates)
	h.db.First(&app, id)
	return c.JSON(app)
}

func (h *OAuthProviderHandler) DeleteApp(c fiber.Ctx) error {
	id := c.Params("id")
	var app models.OAuthApp
	if err := h.db.First(&app, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "app not found"})
	}

	// Cascade: delete authorizations and tokens
	h.db.Where("app_id = ?", app.ID).Delete(&models.OAuthAuthorization{})
	h.db.Where("app_id = ?", app.ID).Delete(&models.OAuthAccessToken{})
	h.db.Delete(&app)

	return c.JSON(fiber.Map{"ok": true})
}

// ---------------------------------------------------------------------------
// User: list authorized apps + revoke
// ---------------------------------------------------------------------------

func (h *OAuthProviderHandler) ListUserAuthorizedApps(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var tokens []models.OAuthAccessToken
	h.db.Where("user_id = ?", user.ID).Preload("App").Find(&tokens)

	apps := make([]fiber.Map, 0, len(tokens))
	seen := map[uint]bool{}
	for _, t := range tokens {
		if seen[t.AppID] {
			continue
		}
		seen[t.AppID] = true
		apps = append(apps, fiber.Map{
			"app_id":       t.AppID,
			"app_name":     t.App.Name,
			"app_icon":     t.App.IconURL,
			"scopes":       scopeString(t.Scopes),
			"authorized_at": t.CreatedAt.Format(time.RFC3339),
		})
	}

	return c.JSON(fiber.Map{"apps": apps})
}

func (h *OAuthProviderHandler) RevokeUserApp(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	appID := c.Params("app_id")
	h.db.Where("user_id = ? AND app_id = ?", user.ID, appID).Delete(&models.OAuthAccessToken{})
	h.db.Where("user_id = ? AND app_id = ?", user.ID, appID).Delete(&models.OAuthAuthorization{})

	return c.JSON(fiber.Map{"ok": true})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func extractBearerToken(c fiber.Ctx) string {
	auth := c.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}

func scopeString(scopesJSON []byte) string {
	var scopes []string
	if err := json.Unmarshal(scopesJSON, &scopes); err != nil {
		return ""
	}
	return strings.Join(scopes, " ")
}

// AutoMigrate registers the OAuth2 provider models for GORM auto-migration.
func AutoMigrateOAuth(db *gorm.DB) {
	db.AutoMigrate(&models.OAuthApp{}, &models.OAuthAuthorization{}, &models.OAuthAccessToken{})
}

// NewOAuthProviderRoutes registers all OAuth2 provider routes (call from routes.go).
func RegisterOAuthProviderRoutes(
	app *fiber.App,
	handler *OAuthProviderHandler,
	authMiddleware fiber.Handler,
) {
	// Public endpoints
	oauth := app.Group("/oauth")
	oauth.Get("/authorize", handler.Authorize)
	oauth.Post("/authorize", authMiddleware, handler.ApproveAuthorization)
	oauth.Post("/token", handler.Token)
	oauth.Get("/userinfo", handler.UserInfo)
	oauth.Post("/revoke", handler.Revoke)
}

// RegisterOAuthProviderAdminRoutes registers admin CRUD routes for OAuth apps.
func RegisterOAuthProviderAdminRoutes(
	adminGroup fiber.Router,
	handler *OAuthProviderHandler,
) {
	adminGroup.Get("/oauth-apps", handler.ListApps)
	adminGroup.Post("/oauth-apps", handler.CreateApp)
	adminGroup.Get("/oauth-apps/:id", handler.GetApp)
	adminGroup.Put("/oauth-apps/:id", handler.UpdateApp)
	adminGroup.Delete("/oauth-apps/:id", handler.DeleteApp)
}

// RegisterOAuthProviderUserRoutes registers user-facing connected apps routes.
func RegisterOAuthProviderUserRoutes(
	protectedGroup fiber.Router,
	handler *OAuthProviderHandler,
) {
	protectedGroup.Get("/settings/connected-apps", handler.ListUserAuthorizedApps)
	protectedGroup.Delete("/settings/connected-apps/:app_id", handler.RevokeUserApp)
}
