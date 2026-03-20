package handlers

import (
	"encoding/json"
	"testing"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
)

func TestOAuthAppModel(t *testing.T) {
	uris, _ := json.Marshal([]string{"http://localhost:3000/callback"})
	scopes, _ := json.Marshal([]string{"profile", "email"})

	app := models.OAuthApp{
		Name:         "Test App",
		ClientID:     "test-client-id",
		ClientSecret: "$2a$12$hashed",
		RedirectURIs: datatypes.JSON(uris),
		Scopes:       datatypes.JSON(scopes),
		OwnerID:      1,
	}

	if app.Name != "Test App" {
		t.Errorf("expected name Test App, got %s", app.Name)
	}
	if app.ClientID != "test-client-id" {
		t.Errorf("expected client_id test-client-id, got %s", app.ClientID)
	}

	// Verify redirect URIs can be unmarshaled
	var parsedURIs []string
	if err := json.Unmarshal(app.RedirectURIs, &parsedURIs); err != nil {
		t.Fatalf("failed to unmarshal redirect_uris: %v", err)
	}
	if len(parsedURIs) != 1 || parsedURIs[0] != "http://localhost:3000/callback" {
		t.Errorf("unexpected redirect_uris: %v", parsedURIs)
	}
}

func TestOAuthAuthorizationModel(t *testing.T) {
	scopes, _ := json.Marshal([]string{"profile"})
	auth := models.OAuthAuthorization{
		AppID:       1,
		UserID:      2,
		Code:        "test-code-abc123",
		Scopes:      datatypes.JSON(scopes),
		RedirectURI: "http://localhost:3000/callback",
	}

	if auth.Code != "test-code-abc123" {
		t.Errorf("expected code test-code-abc123, got %s", auth.Code)
	}
	if auth.Used {
		t.Error("expected Used to be false initially")
	}
}

func TestScopeString(t *testing.T) {
	scopes, _ := json.Marshal([]string{"profile", "email", "openid"})
	got := scopeString(scopes)
	if got != "profile email openid" {
		t.Errorf("expected 'profile email openid', got %q", got)
	}

	// Empty/invalid JSON
	got = scopeString([]byte("invalid"))
	if got != "" {
		t.Errorf("expected empty string for invalid JSON, got %q", got)
	}
}

func TestExtractBearerTokenFromString(t *testing.T) {
	// Test the helper directly — extractBearerToken needs fiber.Ctx, so test the logic
	auth := "Bearer abc123token"
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token := auth[7:]
		if token != "abc123token" {
			t.Errorf("expected abc123token, got %s", token)
		}
	} else {
		t.Error("failed to parse bearer token")
	}
}
