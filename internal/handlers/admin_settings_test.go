package handlers

import "testing"

func TestValidKeys_IncludesDiscord(t *testing.T) {
	if !validKeys["discord"] {
		t.Error("validKeys should include 'discord'")
	}
}

func TestValidKeys_AllExpected(t *testing.T) {
	expected := []string{
		"general", "features", "homepage", "agents",
		"contact", "pricing", "download", "integrations",
		"smtp", "seo", "discord", "slack", "coming_soon",
		"marketplace", "plugins", "docs", "blog",
	}
	for _, key := range expected {
		if !validKeys[key] {
			t.Errorf("validKeys missing %q", key)
		}
	}
}

func TestPublicKeys_ExcludesDiscord(t *testing.T) {
	if publicKeys["discord"] {
		t.Error("discord should NOT be a public key (contains secrets)")
	}
}

func TestDefaultSettings_Discord(t *testing.T) {
	val := defaultSettings("discord")
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map for discord defaults, got %T", val)
	}

	checks := map[string]interface{}{
		"enabled":        false,
		"bot_token":      "",
		"application_id": "",
		"client_id":      "",
		"client_secret":  "",
		"guild_id":       "",
		"channel_id":     "",
		"command_prefix": "!",
		"webhook_url":    "",
	}
	for key, want := range checks {
		got, exists := m[key]
		if !exists {
			t.Errorf("missing key %q in discord defaults", key)
			continue
		}
		if got != want {
			t.Errorf("discord default %q = %v, want %v", key, got, want)
		}
	}

	// allowed_users should be a slice
	if _, ok := m["allowed_users"]; !ok {
		t.Error("missing 'allowed_users' in discord defaults")
	}
}

func TestValidKeys_IncludesSlack(t *testing.T) {
	if !validKeys["slack"] {
		t.Error("validKeys should include 'slack'")
	}
}

func TestPublicKeys_ExcludesSlack(t *testing.T) {
	if publicKeys["slack"] {
		t.Error("slack should NOT be a public key (contains secrets)")
	}
}

func TestDefaultSettings_Slack(t *testing.T) {
	val := defaultSettings("slack")
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map for slack defaults, got %T", val)
	}

	checks := map[string]interface{}{
		"enabled":        false,
		"bot_token":      "",
		"app_token":      "",
		"signing_secret": "",
		"app_id":         "",
		"channel_id":     "",
		"command_prefix": "!",
		"webhook_url":    "",
		"team_id":        "",
	}
	for key, want := range checks {
		got, exists := m[key]
		if !exists {
			t.Errorf("missing key %q in slack defaults", key)
			continue
		}
		if got != want {
			t.Errorf("slack default %q = %v, want %v", key, got, want)
		}
	}

	// allowed_users should be a slice
	if _, ok := m["allowed_users"]; !ok {
		t.Error("missing 'allowed_users' in slack defaults")
	}
}

func TestDefaultSettings_UnknownKey(t *testing.T) {
	val := defaultSettings("nonexistent_key")
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("expected empty map for unknown key, got %T", val)
	}
	if len(m) != 0 {
		t.Error("expected empty map for unknown key")
	}
}
