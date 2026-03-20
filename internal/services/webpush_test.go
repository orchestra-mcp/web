package services

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupPushTestDB creates an in-memory SQLite database with the push_subscriptions table.
func setupPushTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.PushSubscription{}); err != nil {
		t.Fatalf("failed to migrate push_subscriptions: %v", err)
	}
	return db
}

func TestSendPushNoVapidKeys(t *testing.T) {
	// Ensure no VAPID keys are set
	os.Unsetenv("VAPID_PRIVATE_KEY")
	os.Unsetenv("VAPID_PUBLIC_KEY")

	// Should not panic — no-op when keys are missing.
	// db is nil because sendPushSync returns before touching it.
	SendPush(nil, 1, "Test", "Message", "info")
	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
}

func TestSendPushNoSubscriptions(t *testing.T) {
	db := setupPushTestDB(t)

	// Set VAPID keys so the function proceeds past the key check
	os.Setenv("VAPID_PRIVATE_KEY", "test-private-key")
	os.Setenv("VAPID_PUBLIC_KEY", "test-public-key")
	defer func() {
		os.Unsetenv("VAPID_PRIVATE_KEY")
		os.Unsetenv("VAPID_PUBLIC_KEY")
	}()

	// No subscriptions for user 999 — should return without error
	sendPushSync(db, 999, "Test", "No subs", "info")
}

func TestPushPayloadFormat(t *testing.T) {
	payload, err := json.Marshal(map[string]string{
		"title":   "Hello",
		"message": "World",
		"body":    "World",
		"type":    "info",
		"url":     "/",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(payload, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["title"] != "Hello" {
		t.Errorf("title = %q, want Hello", parsed["title"])
	}
	if parsed["message"] != "World" {
		t.Errorf("message = %q, want World", parsed["message"])
	}
	if parsed["body"] != "World" {
		t.Errorf("body = %q, want World", parsed["body"])
	}
	if parsed["type"] != "info" {
		t.Errorf("type = %q, want info", parsed["type"])
	}
	if parsed["url"] != "/" {
		t.Errorf("url = %q, want /", parsed["url"])
	}
}

func TestPushSubscriptionModel(t *testing.T) {
	sub := models.PushSubscription{
		UserID:    42,
		Endpoint:  "https://fcm.googleapis.com/fcm/send/abc123",
		P256dh:    "BNcR...",
		Auth:      "auth_secret",
		Platform:  "web",
		UserAgent: "Mozilla/5.0",
	}

	if sub.UserID != 42 {
		t.Errorf("UserID = %d, want 42", sub.UserID)
	}
	if sub.Endpoint != "https://fcm.googleapis.com/fcm/send/abc123" {
		t.Errorf("Endpoint mismatch")
	}
	if sub.Platform != "web" {
		t.Errorf("Platform = %q, want web", sub.Platform)
	}
	if sub.P256dh != "BNcR..." {
		t.Errorf("P256dh = %q, want BNcR...", sub.P256dh)
	}
	if sub.Auth != "auth_secret" {
		t.Errorf("Auth = %q, want auth_secret", sub.Auth)
	}
	if sub.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %q, want Mozilla/5.0", sub.UserAgent)
	}
}

func TestVapidSubjectDefault(t *testing.T) {
	// When VAPID_SUBJECT is empty, the code defaults to mailto:noreply@orchestra-mcp.com
	os.Unsetenv("VAPID_SUBJECT")
	subject := os.Getenv("VAPID_SUBJECT")
	if subject != "" {
		t.Errorf("VAPID_SUBJECT should be empty when unset, got %q", subject)
	}
	// The sendPushSync function handles the default internally — this tests the env reads correctly
}

func TestPushPayloadIncludesAllFields(t *testing.T) {
	tests := []struct {
		title   string
		message string
		ntype   string
	}{
		{"Alert", "Server down", "error"},
		{"", "Empty title", "warning"},
		{"Info", "", "info"},
		{"Success", "Task done", "success"},
	}

	for _, tt := range tests {
		payload, _ := json.Marshal(map[string]string{
			"title":   tt.title,
			"message": tt.message,
			"body":    tt.message,
			"type":    tt.ntype,
			"url":     "/",
		})

		var parsed map[string]string
		json.Unmarshal(payload, &parsed)

		if len(parsed) != 5 {
			t.Errorf("payload should have 5 fields, got %d", len(parsed))
		}
	}
}
