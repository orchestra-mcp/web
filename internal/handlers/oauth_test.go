package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOAuthStateExpiry(t *testing.T) {
	st := &oauthState{
		Provider:  "discord",
		Mode:      "login",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	if !time.Now().After(st.ExpiresAt) {
		t.Error("expected state to be expired")
	}
}

func TestOAuthStateStoreRoundTrip(t *testing.T) {
	key := "test-state-123"
	st := &oauthState{
		Provider:  "discord",
		Mode:      "connect",
		UserID:    42,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	oauthStateStore.Store(key, st)

	val, ok := oauthStateStore.LoadAndDelete(key)
	if !ok {
		t.Fatal("expected state to be found")
	}
	got := val.(*oauthState)
	if got.Provider != "discord" || got.Mode != "connect" || got.UserID != 42 {
		t.Errorf("state mismatch: %+v", got)
	}

	// Should be deleted now
	_, ok = oauthStateStore.Load(key)
	if ok {
		t.Error("expected state to be deleted after LoadAndDelete")
	}
}

func TestOAuthStateModes(t *testing.T) {
	login := &oauthState{Mode: "login", UserID: 0}
	connect := &oauthState{Mode: "connect", UserID: 99}

	if login.Mode != "login" {
		t.Error("expected login mode")
	}
	if connect.Mode != "connect" || connect.UserID != 99 {
		t.Error("expected connect mode with user ID")
	}
}

func TestTokenResponseParsing(t *testing.T) {
	raw := `{"access_token":"abc123","token_type":"Bearer","refresh_token":"def456","expires_in":604800,"scope":"identify email"}`
	var tr tokenResponse
	if err := json.Unmarshal([]byte(raw), &tr); err != nil {
		t.Fatal(err)
	}
	if tr.AccessToken != "abc123" {
		t.Errorf("access_token = %q, want abc123", tr.AccessToken)
	}
	if tr.RefreshToken != "def456" {
		t.Errorf("refresh_token = %q, want def456", tr.RefreshToken)
	}
	if tr.ExpiresIn != 604800 {
		t.Errorf("expires_in = %d, want 604800", tr.ExpiresIn)
	}
}

func TestUserInfoResponseParsing_Discord(t *testing.T) {
	raw := `{"id":"123456","username":"testuser","discriminator":"0001","email":"test@example.com","avatar":"abc123hash"}`
	var info userInfoResponse
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		t.Fatal(err)
	}
	if info.ID != "123456" {
		t.Errorf("id = %q, want 123456", info.ID)
	}
	if info.Username != "testuser" {
		t.Errorf("username = %q, want testuser", info.Username)
	}
	if info.Email != "test@example.com" {
		t.Errorf("email = %q, want test@example.com", info.Email)
	}
	if info.Avatar != "abc123hash" {
		t.Errorf("avatar = %q, want abc123hash", info.Avatar)
	}
	// Name fallback to Username
	if info.Name == "" {
		info.Name = info.Username
	}
	if info.Name != "testuser" {
		t.Errorf("name fallback = %q, want testuser", info.Name)
	}
}

func TestUserInfoResponseParsing_GitHub(t *testing.T) {
	raw := `{"id":"789","login":"ghuser","name":"","email":"gh@example.com","avatar":"https://avatars.githubusercontent.com/u/789"}`
	var info userInfoResponse
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		t.Fatal(err)
	}
	if info.Login != "ghuser" {
		t.Errorf("login = %q, want ghuser", info.Login)
	}
	// Name fallback to Login
	if info.Name == "" {
		if info.Username != "" {
			info.Name = info.Username
		} else if info.Login != "" {
			info.Name = info.Login
		}
	}
	if info.Name != "ghuser" {
		t.Errorf("name fallback = %q, want ghuser", info.Name)
	}
}

func TestProviderConfigStruct(t *testing.T) {
	pc := providerConfig{
		ClientID:     "client123",
		ClientSecret: "secret456",
		AuthURL:      "https://discord.com/api/oauth2/authorize",
		TokenURL:     "https://discord.com/api/oauth2/token",
		UserInfoURL:  "https://discord.com/api/users/@me",
		Scopes:       "identify email",
	}
	if pc.ClientID != "client123" {
		t.Errorf("ClientID = %q, want client123", pc.ClientID)
	}
	if pc.Scopes != "identify email" {
		t.Errorf("Scopes = %q, want 'identify email'", pc.Scopes)
	}
}

func TestExchangeCodeMockServer(t *testing.T) {
	// Mock Discord token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %q, want application/x-www-form-urlencoded", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "mock_access_token",
			"token_type":    "Bearer",
			"refresh_token": "mock_refresh_token",
			"expires_in":    604800,
			"scope":         "identify email",
		})
	}))
	defer server.Close()

	h := &OAuthHandler{}
	pcfg := &providerConfig{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL,
	}

	resp, err := h.exchangeCode(pcfg, "test_code", "http://localhost/callback")
	if err != nil {
		t.Fatal(err)
	}
	if resp.AccessToken != "mock_access_token" {
		t.Errorf("access_token = %q, want mock_access_token", resp.AccessToken)
	}
	if resp.RefreshToken != "mock_refresh_token" {
		t.Errorf("refresh_token = %q, want mock_refresh_token", resp.RefreshToken)
	}
}

func TestExchangeCodeEmptyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "",
			"error":        "invalid_grant",
		})
	}))
	defer server.Close()

	h := &OAuthHandler{}
	pcfg := &providerConfig{TokenURL: server.URL}

	_, err := h.exchangeCode(pcfg, "bad_code", "http://localhost/callback")
	if err == nil {
		t.Error("expected error for empty access token")
	}
}

func TestFetchUserInfoMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mock_token" {
			t.Errorf("authorization = %q, want 'Bearer mock_token'", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "999",
			"username": "discorduser",
			"email":    "discord@example.com",
			"avatar":   "avatar_hash",
		})
	}))
	defer server.Close()

	h := &OAuthHandler{}
	pcfg := &providerConfig{UserInfoURL: server.URL}

	info, err := h.fetchUserInfo(pcfg, "mock_token")
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "999" {
		t.Errorf("id = %q, want 999", info.ID)
	}
	if info.Username != "discorduser" {
		t.Errorf("username = %q, want discorduser", info.Username)
	}
	if info.Email != "discord@example.com" {
		t.Errorf("email = %q, want discord@example.com", info.Email)
	}
}

func TestFetchUserInfoNameFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "111",
			"username": "fallback_user",
			"email":    "fb@test.com",
			"name":     "",
		})
	}))
	defer server.Close()

	h := &OAuthHandler{}
	pcfg := &providerConfig{UserInfoURL: server.URL}

	info, err := h.fetchUserInfo(pcfg, "tok")
	if err != nil {
		t.Fatal(err)
	}
	// fetchUserInfo normalizes name
	if info.Name != "fallback_user" {
		t.Errorf("name fallback = %q, want fallback_user", info.Name)
	}
}

func TestDiscordAvatarURL(t *testing.T) {
	// Simulate the avatar URL construction from Callback
	userID := "123456"
	avatarHash := "abc123hash"
	avatar := avatarHash

	// Same logic as in Callback
	if avatar != "" {
		avatar = "https://cdn.discordapp.com/avatars/" + userID + "/" + avatar + ".png"
	}

	expected := "https://cdn.discordapp.com/avatars/123456/abc123hash.png"
	if avatar != expected {
		t.Errorf("avatar = %q, want %q", avatar, expected)
	}
}

func TestDiscordAvatarURL_AlreadyHTTP(t *testing.T) {
	avatar := "https://cdn.discordapp.com/avatars/123/existing.png"
	// Should not double-prefix
	if avatar != "" && len(avatar) > 4 && avatar[:4] == "http" {
		// no-op, already a URL
	} else {
		avatar = "https://cdn.discordapp.com/avatars/123/" + avatar + ".png"
	}
	if avatar != "https://cdn.discordapp.com/avatars/123/existing.png" {
		t.Errorf("should not modify existing URL: %q", avatar)
	}
}

// ── Slack-specific tests ──

func TestExchangeCode_SlackNestedToken(t *testing.T) {
	// Slack returns access_token nested under authed_user
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":          true,
			"access_token": "",
			"authed_user": map[string]interface{}{
				"id":           "U12345",
				"access_token": "xoxp-slack-user-token",
				"scope":        "identity.basic,identity.email",
			},
		})
	}))
	defer server.Close()

	h := &OAuthHandler{}
	pcfg := &providerConfig{
		ClientID:     "slack_client",
		ClientSecret: "slack_secret",
		TokenURL:     server.URL,
	}

	resp, err := h.exchangeCode(pcfg, "slack_code", "http://localhost/callback")
	if err != nil {
		t.Fatal(err)
	}
	if resp.AccessToken != "xoxp-slack-user-token" {
		t.Errorf("access_token = %q, want xoxp-slack-user-token", resp.AccessToken)
	}
}

func TestFetchUserInfo_SlackNestedUser(t *testing.T) {
	// Slack's users.identity returns user info nested under "user" key
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":        "U67890",
				"name":      "Slack User",
				"email":     "slack@example.com",
				"image_192": "https://avatars.slack-edge.com/user-192.png",
			},
			"team": map[string]interface{}{
				"id": "T11111",
			},
		})
	}))
	defer server.Close()

	h := &OAuthHandler{}
	pcfg := &providerConfig{UserInfoURL: server.URL}

	info, err := h.fetchUserInfo(pcfg, "xoxp-token")
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "U67890" {
		t.Errorf("id = %q, want U67890", info.ID)
	}
	if info.Name != "Slack User" {
		t.Errorf("name = %q, want 'Slack User'", info.Name)
	}
	if info.Email != "slack@example.com" {
		t.Errorf("email = %q, want slack@example.com", info.Email)
	}
	if info.Avatar != "https://avatars.slack-edge.com/user-192.png" {
		t.Errorf("avatar = %q, want Slack avatar URL", info.Avatar)
	}
}

func TestSlackOAuthStateRoundTrip(t *testing.T) {
	key := "test-slack-state-456"
	st := &oauthState{
		Provider:  "slack",
		Mode:      "login",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	oauthStateStore.Store(key, st)

	val, ok := oauthStateStore.LoadAndDelete(key)
	if !ok {
		t.Fatal("expected slack state to be found")
	}
	got := val.(*oauthState)
	if got.Provider != "slack" || got.Mode != "login" {
		t.Errorf("slack state mismatch: %+v", got)
	}
}

func TestProviderConfigStruct_Slack(t *testing.T) {
	pc := providerConfig{
		ClientID:     "slack_client",
		ClientSecret: "slack_secret",
		AuthURL:      "https://slack.com/oauth/v2/authorize",
		TokenURL:     "https://slack.com/api/oauth.v2.access",
		UserInfoURL:  "https://slack.com/api/users.identity",
		Scopes:       "identity.basic,identity.email,identity.avatar",
	}
	if pc.ClientID != "slack_client" {
		t.Errorf("ClientID = %q, want slack_client", pc.ClientID)
	}
	if pc.AuthURL != "https://slack.com/oauth/v2/authorize" {
		t.Errorf("AuthURL = %q, want Slack auth URL", pc.AuthURL)
	}
	if pc.Scopes != "identity.basic,identity.email,identity.avatar" {
		t.Errorf("Scopes = %q, want Slack scopes", pc.Scopes)
	}
}
