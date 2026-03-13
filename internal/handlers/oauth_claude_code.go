package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	claudeCodeOAuthAuthorizeURL = "https://claude.ai/oauth/authorize"
	claudeCodeOAuthTokenURL     = "https://console.anthropic.com/v1/oauth/token"
	claudeCodeOAuthClientID     = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	claudeCodeOAuthRedirectURI  = "https://console.anthropic.com/oauth/code/callback"
	claudeCodeOAuthScopes       = "org:create_api_key user:profile user:inference"
)

// pendingClaudeOAuth holds PKCE state for an in-progress web OAuth flow.
type pendingClaudeOAuth struct {
	CodeVerifier string
	State        string
	UserID       uint
	ExpiresAt    time.Time
}

var pendingClaudeOAuths sync.Map

// ClaudeCodeOAuthHandler handles Claude Code OAuth flows on the web dashboard.
type ClaudeCodeOAuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewClaudeCodeOAuthHandler creates a new ClaudeCodeOAuthHandler.
func NewClaudeCodeOAuthHandler(db *gorm.DB, cfg *config.Config) *ClaudeCodeOAuthHandler {
	return &ClaudeCodeOAuthHandler{db: db, cfg: cfg}
}

// Start handles GET /api/oauth/claude-code/start — generates PKCE and returns the authorize URL.
func (h *ClaudeCodeOAuthHandler) Start(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Generate code_verifier (32 random bytes, base64url-encoded).
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate verifier"})
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// code_challenge = SHA256(code_verifier), base64url-encoded.
	ch := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(ch[:])

	// Generate state.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate state"})
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Store pending flow.
	pendingClaudeOAuths.Store(state, &pendingClaudeOAuth{
		CodeVerifier: codeVerifier,
		State:        state,
		UserID:       user.ID,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	})

	// Build authorize URL.
	params := url.Values{
		"client_id":             {claudeCodeOAuthClientID},
		"response_type":        {"code"},
		"redirect_uri":          {claudeCodeOAuthRedirectURI},
		"scope":                 {claudeCodeOAuthScopes},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"code":                  {"true"},
	}
	authorizeURL := claudeCodeOAuthAuthorizeURL + "?" + params.Encode()

	return c.JSON(fiber.Map{
		"authorize_url": authorizeURL,
		"state":         state,
	})
}

// ExchangeRequest is the request body for the exchange endpoint.
type ExchangeRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// Exchange handles POST /api/oauth/claude-code/exchange — exchanges the code for tokens.
func (h *ClaudeCodeOAuthHandler) Exchange(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body ExchangeRequest
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Code == "" || body.State == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code and state are required"})
	}

	// Look up pending flow.
	val, ok := pendingClaudeOAuths.LoadAndDelete(body.State)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid or expired state"})
	}
	pending := val.(*pendingClaudeOAuth)

	if time.Now().After(pending.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "OAuth flow expired"})
	}

	if pending.UserID != user.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "state belongs to another user"})
	}

	// Exchange code for tokens.
	tokenResp, err := exchangeClaudeCodeWeb(body.Code, pending.CodeVerifier, body.State)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": fmt.Sprintf("token exchange failed: %v", err)})
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Store as OAuthAccount linked to user.
	meta, _ := json.Marshal(map[string]any{
		"scope":      tokenResp.Scope,
		"expires_at": expiresAt.Format(time.RFC3339),
	})

	// Upsert: update if user already has a claude_code OAuth account.
	var existing models.OAuthAccount
	if err := h.db.Where("user_id = ? AND provider = ?", user.ID, "claude_code").First(&existing).Error; err == nil {
		h.db.Model(&existing).Updates(map[string]interface{}{
			"access_token":  tokenResp.AccessToken,
			"refresh_token": tokenResp.RefreshToken,
			"meta":          datatypes.JSON(meta),
		})
	} else {
		oauthAcct := models.OAuthAccount{
			UserID:         user.ID,
			Provider:       "claude_code",
			ProviderUserID: fmt.Sprintf("claude_%d", user.ID),
			Email:          user.Email,
			Name:           user.Name,
			AccessToken:    tokenResp.AccessToken,
			RefreshToken:   tokenResp.RefreshToken,
			Meta:           datatypes.JSON(meta),
		}
		h.db.Create(&oauthAcct)
	}

	// Mask token for response.
	maskedToken := tokenResp.AccessToken[:10] + "..." + tokenResp.AccessToken[len(tokenResp.AccessToken)-4:]

	return c.JSON(fiber.Map{
		"success":    true,
		"token":      maskedToken,
		"expires_at": expiresAt.Format(time.RFC3339),
		"scope":      tokenResp.Scope,
	})
}

// claudeCodeTokenResponse is the response from Anthropic's OAuth token endpoint.
type claudeCodeTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// exchangeClaudeCodeWeb exchanges an authorization code for tokens.
func exchangeClaudeCodeWeb(code, codeVerifier, state string) (*claudeCodeTokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     claudeCodeOAuthClientID,
		"code":          code,
		"redirect_uri":  claudeCodeOAuthRedirectURI,
		"code_verifier": codeVerifier,
		"state":         state,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", claudeCodeOAuthTokenURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://claude.ai/")
	req.Header.Set("Origin", "https://claude.ai")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp claudeCodeTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("empty access token in response")
	}

	return &tokenResp, nil
}
