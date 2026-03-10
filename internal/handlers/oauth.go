package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// oauthStateStore holds short-lived CSRF state tokens.
var oauthStateStore sync.Map

type oauthState struct {
	Provider  string
	Mode      string // "login" or "connect" (connect = link to existing user)
	UserID    uint   // only set when mode=connect
	ExpiresAt time.Time
}

// OAuthHandler handles OAuth2 login/connect flows.
type OAuthHandler struct {
	db          *gorm.DB
	cfg         *config.Config
	authService *services.AuthService
}

// NewOAuthHandler creates a new OAuthHandler.
func NewOAuthHandler(db *gorm.DB, cfg *config.Config) *OAuthHandler {
	return &OAuthHandler{
		db:          db,
		cfg:         cfg,
		authService: services.NewAuthService(db, cfg),
	}
}

// providerConfig returns the OAuth2 settings for a provider from admin SystemSettings.
type providerConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       string
}

func (h *OAuthHandler) getProviderConfig(provider string) (*providerConfig, error) {
	var setting models.SystemSetting
	if err := h.db.Where("key = ?", "integrations").First(&setting).Error; err != nil {
		return nil, fmt.Errorf("integrations not configured")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(setting.Value, &data); err != nil {
		return nil, fmt.Errorf("invalid integrations data")
	}

	// The integrations settings are flat keys like "discord_client_id", "discord_client_secret"
	// OR nested like {"discord": {"client_id": "...", "client_secret": "..."}}
	// Support both patterns:
	clientID := ""
	clientSecret := ""

	// Try flat keys first
	if v, ok := data[provider+"_client_id"]; ok {
		clientID = fmt.Sprint(v)
	}
	if v, ok := data[provider+"_client_secret"]; ok {
		clientSecret = fmt.Sprint(v)
	}

	// Try nested
	if clientID == "" {
		if nested, ok := data[provider]; ok {
			if m, ok := nested.(map[string]interface{}); ok {
				if v, ok := m["client_id"]; ok {
					clientID = fmt.Sprint(v)
				}
				if v, ok := m["client_secret"]; ok {
					clientSecret = fmt.Sprint(v)
				}
			}
		}
	}

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("%s OAuth not configured", provider)
	}

	switch provider {
	case "discord":
		return &providerConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AuthURL:      "https://discord.com/api/oauth2/authorize",
			TokenURL:     "https://discord.com/api/oauth2/token",
			UserInfoURL:  "https://discord.com/api/users/@me",
			Scopes:       "identify email",
		}, nil
	case "google":
		return &providerConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:       "openid email profile",
		}, nil
	case "github":
		return &providerConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			Scopes:       "user:email repo",
		}, nil
	case "slack":
		return &providerConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AuthURL:      "https://slack.com/oauth/v2/authorize",
			TokenURL:     "https://slack.com/api/oauth.v2.access",
			UserInfoURL:  "https://slack.com/api/users.identity",
			Scopes:       "identity.basic,identity.email,identity.avatar",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// Redirect handles GET /api/auth/oauth/:provider — initiates the OAuth login flow.
func (h *OAuthHandler) Redirect(c fiber.Ctx) error {
	provider := c.Params("provider")
	pcfg, err := h.getProviderConfig(provider)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Generate CSRF state
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	// Determine mode: "connect" if the route ends with /connect and user is authenticated
	mode := "login"
	var userID uint
	if strings.HasSuffix(c.Path(), "/connect") {
		user := h.currentUserFromCookie(c)
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		mode = "connect"
		userID = user.ID
	}

	oauthStateStore.Store(state, &oauthState{
		Provider:  provider,
		Mode:      mode,
		UserID:    userID,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	redirectURI := h.redirectURI(c, provider)

	// Slack uses user_scope instead of scope for Sign in with Slack
	scopeParam := "scope"
	if provider == "slack" {
		scopeParam = "user_scope"
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&%s=%s&state=%s",
		pcfg.AuthURL,
		url.QueryEscape(pcfg.ClientID),
		url.QueryEscape(redirectURI),
		scopeParam,
		url.QueryEscape(pcfg.Scopes),
		url.QueryEscape(state),
	)

	// Discord-specific: add prompt=consent for re-auth
	if provider == "discord" {
		authURL += "&prompt=consent"
	}

	return c.Redirect().To(authURL)
}

// Callback handles GET /api/auth/oauth/:provider/callback — processes the OAuth callback.
func (h *OAuthHandler) Callback(c fiber.Ctx) error {
	provider := c.Params("provider")
	code := c.Query("code")
	state := c.Query("state")
	oauthErr := c.Query("error")

	if oauthErr != "" {
		return c.Redirect().To("/login?error=" + url.QueryEscape(oauthErr))
	}

	if code == "" || state == "" {
		return c.Redirect().To("/login?error=invalid_callback")
	}

	// Validate state
	val, ok := oauthStateStore.LoadAndDelete(state)
	if !ok {
		return c.Redirect().To("/login?error=invalid_state")
	}
	st := val.(*oauthState)
	if time.Now().After(st.ExpiresAt) || st.Provider != provider {
		return c.Redirect().To("/login?error=expired_state")
	}

	pcfg, err := h.getProviderConfig(provider)
	if err != nil {
		return c.Redirect().To("/login?error=" + url.QueryEscape(err.Error()))
	}

	// Exchange code for token
	redirectURI := h.redirectURI(c, provider)
	tokenResp, err := h.exchangeCode(pcfg, code, redirectURI)
	if err != nil {
		return c.Redirect().To("/login?error=" + url.QueryEscape("token_exchange_failed"))
	}

	// Fetch user info
	userInfo, err := h.fetchUserInfo(pcfg, tokenResp.AccessToken)
	if err != nil {
		return c.Redirect().To("/login?error=" + url.QueryEscape("userinfo_failed"))
	}

	// Build avatar URL for Discord
	avatar := userInfo.Avatar
	if provider == "discord" && avatar != "" && !strings.HasPrefix(avatar, "http") {
		avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userInfo.ID, avatar)
	}

	if st.Mode == "connect" {
		// Link to existing user
		return h.handleConnect(c, st.UserID, provider, userInfo, tokenResp, avatar)
	}
	return h.handleLogin(c, provider, userInfo, tokenResp, avatar)
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type userInfoResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Login         string `json:"login"` // GitHub
}

func (h *OAuthHandler) exchangeCode(pcfg *providerConfig, code, redirectURI string) (*tokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", pcfg.ClientID)
	data.Set("client_secret", pcfg.ClientSecret)

	req, _ := http.NewRequest("POST", pcfg.TokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response")
	}

	// Slack returns access token nested under authed_user for Sign in with Slack
	if token.AccessToken == "" {
		var slackResp struct {
			AuthedUser struct {
				AccessToken string `json:"access_token"`
			} `json:"authed_user"`
		}
		if err := json.Unmarshal(body, &slackResp); err == nil && slackResp.AuthedUser.AccessToken != "" {
			token.AccessToken = slackResp.AuthedUser.AccessToken
		}
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("empty access token")
	}
	return &token, nil
}

func (h *OAuthHandler) fetchUserInfo(pcfg *providerConfig, accessToken string) (*userInfoResponse, error) {
	req, _ := http.NewRequest("GET", pcfg.UserInfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var info userInfoResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	// Slack returns user info nested under "user" key
	if info.ID == "" {
		var slackResp struct {
			User struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
				Image string `json:"image_192"`
			} `json:"user"`
		}
		if err := json.Unmarshal(body, &slackResp); err == nil && slackResp.User.ID != "" {
			info.ID = slackResp.User.ID
			info.Name = slackResp.User.Name
			info.Email = slackResp.User.Email
			info.Avatar = slackResp.User.Image
		}
	}

	// Normalize name: Discord uses "username", GitHub uses "login"
	if info.Name == "" {
		if info.Username != "" {
			info.Name = info.Username
		} else if info.Login != "" {
			info.Name = info.Login
		}
	}
	return &info, nil
}

func (h *OAuthHandler) handleLogin(c fiber.Ctx, provider string, info *userInfoResponse, token *tokenResponse, avatar string) error {
	// Check if OAuth account already exists
	var existing models.OAuthAccount
	if err := h.db.Where("provider = ? AND provider_user_id = ?", provider, info.ID).First(&existing).Error; err == nil {
		// Account exists — log in the linked user
		var user models.User
		if err := h.db.First(&user, existing.UserID).Error; err != nil {
			return c.Redirect().To("/login?error=user_not_found")
		}
		if user.Status == "suspended" || user.Status == "blocked" {
			return c.Redirect().To("/login?error=account_suspended")
		}
		// Update tokens
		h.db.Model(&existing).Updates(map[string]interface{}{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
			"avatar":        avatar,
			"name":          info.Name,
		})
		jwtToken, err := h.authService.GenerateJWT(&user)
		if err != nil {
			return c.Redirect().To("/login?error=token_generation_failed")
		}
		return c.Redirect().To("/auth/oauth-callback?token=" + jwtToken)
	}

	// No existing OAuth account — check if a user with that email exists
	var user models.User
	if info.Email != "" {
		if err := h.db.Where("email = ?", info.Email).First(&user).Error; err == nil {
			// Link to existing user
			h.createOAuthAccount(user.ID, provider, info, token, avatar)
			jwtToken, err := h.authService.GenerateJWT(&user)
			if err != nil {
				return c.Redirect().To("/login?error=token_generation_failed")
			}
			return c.Redirect().To("/auth/oauth-callback?token=" + jwtToken)
		}
	}

	// New user — create account
	randomPass := make([]byte, 16)
	_, _ = rand.Read(randomPass)
	hashed, _ := bcrypt.GenerateFromPassword(randomPass, 12)

	displayName := info.Name
	if displayName == "" {
		displayName = info.Email
	}

	user = models.User{
		Name:      displayName,
		Email:     info.Email,
		Password:  string(hashed),
		AvatarURL: avatar,
		Status:    "active",
	}
	if err := h.db.Create(&user).Error; err != nil {
		return c.Redirect().To("/login?error=failed_to_create_account")
	}

	h.createOAuthAccount(user.ID, provider, info, token, avatar)

	jwtToken, err := h.authService.GenerateJWT(&user)
	if err != nil {
		return c.Redirect().To("/login?error=token_generation_failed")
	}
	return c.Redirect().To("/auth/oauth-callback?token=" + jwtToken)
}

func (h *OAuthHandler) handleConnect(c fiber.Ctx, userID uint, provider string, info *userInfoResponse, token *tokenResponse, avatar string) error {
	// Check if already connected by another user
	var existing models.OAuthAccount
	if err := h.db.Where("provider = ? AND provider_user_id = ?", provider, info.ID).First(&existing).Error; err == nil {
		if existing.UserID != userID {
			return c.Redirect().To("/settings?tab=social&error=account_linked_to_another_user")
		}
		// Already connected to this user — update tokens
		h.db.Model(&existing).Updates(map[string]interface{}{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
			"avatar":        avatar,
			"name":          info.Name,
		})
		return c.Redirect().To("/settings?tab=social&success=updated")
	}

	h.createOAuthAccount(userID, provider, info, token, avatar)
	return c.Redirect().To("/settings?tab=social&success=connected")
}

func (h *OAuthHandler) createOAuthAccount(userID uint, provider string, info *userInfoResponse, token *tokenResponse, avatar string) {
	account := models.OAuthAccount{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: info.ID,
		Email:          info.Email,
		Name:           info.Name,
		Avatar:         avatar,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
	}
	h.db.Create(&account)
}

func (h *OAuthHandler) redirectURI(c fiber.Ctx, provider string) string {
	proto := "https"
	if h.cfg.Env == "development" {
		proto = "http"
	}
	host := string(c.Request().Host())
	return fmt.Sprintf("%s://%s/api/auth/oauth/%s/callback", proto, host, provider)
}

// currentUserFromCookie reads the JWT from the orchestra_token cookie and returns the
// authenticated user. This is needed for OAuth connect flows which are browser
// redirects (GET) where the Authorization header is not available.
func (h *OAuthHandler) currentUserFromCookie(c fiber.Ctx) *models.User {
	// First try the middleware-set user (if this route goes through Auth middleware)
	if u := middleware.CurrentUser(c); u != nil {
		return u
	}

	// Fall back to reading the cookie directly
	tokenStr := c.Cookies("orchestra_token")
	if tokenStr == "" {
		return nil
	}

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(h.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil
	}

	var user models.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		return nil
	}

	if user.Status == "blocked" {
		return nil
	}

	return &user
}
