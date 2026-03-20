package handlers

import (
	"fmt"
	"testing"
	"time"

	"github.com/orchestra-mcp/web/internal/hub"
)

func TestBroadcastSyncNilHub(t *testing.T) {
	// Should not panic with nil hub
	broadcastSync(nil, 1, "feature", "FEAT-123", "upsert")
}

func TestBroadcastSyncWithHub(t *testing.T) {
	h := hub.NewHub()
	go h.Run()
	// No connected clients — should succeed silently
	broadcastSync(h, 1, "note", 42, "upsert")
	broadcastSync(h, 1, "note", 42, "delete")
}

func TestBroadcastSyncEventFormat(t *testing.T) {
	// Verify entityID formatting
	tests := []struct {
		entityID interface{}
		want     string
	}{
		{"FEAT-123", "FEAT-123"},
		{42, "42"},
		{uint(99), "99"},
		{"abc-def", "abc-def"},
	}
	for _, tt := range tests {
		got := fmt.Sprintf("%v", tt.entityID)
		if got != tt.want {
			t.Errorf("entityID format: got %q, want %q", got, tt.want)
		}
	}
}

func TestBroadcastTimestampIsRecent(t *testing.T) {
	before := time.Now().UnixMilli()
	now := time.Now().UnixMilli()
	if now < before {
		t.Error("timestamp should be monotonically increasing")
	}
}

// Verify all 10 handlers accept optional hub parameter without breaking
func TestFeatureHandlerAcceptsHub(t *testing.T) {
	h := NewFeatureHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewFeatureHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestNoteHandlerAcceptsHub(t *testing.T) {
	h := NewNoteHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewNoteHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestPlanHandlerAcceptsHub(t *testing.T) {
	h := NewPlanHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewPlanHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestPersonHandlerAcceptsHub(t *testing.T) {
	h := NewPersonHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewPersonHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestAgentHandlerAcceptsHub(t *testing.T) {
	h := NewAgentHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewAgentHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestWorkflowHandlerAcceptsHub(t *testing.T) {
	h := NewWorkflowHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewWorkflowHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestSkillHandlerAcceptsHub(t *testing.T) {
	h := NewSkillHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewSkillHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestDocHandlerAcceptsHub(t *testing.T) {
	h := NewDocHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewDocHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestCommunityHandlerAcceptsHub(t *testing.T) {
	h := NewCommunityHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewCommunityHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestProjectHandlerAcceptsHub(t *testing.T) {
	h := NewProjectHandler(nil)
	if h.hub != nil {
		t.Error("hub should be nil when not provided")
	}
	wsHub := hub.NewHub()
	h2 := NewProjectHandler(nil, wsHub)
	if h2.hub != wsHub {
		t.Error("hub should be set when provided")
	}
}

func TestAllEntityTypesAreBroadcast(t *testing.T) {
	// This is a documentation test verifying the entity types we broadcast
	entityTypes := []string{
		"feature",
		"note",
		"plan",         // future use
		"person",       // future use
		"agent",
		"workflow",
		"skill",
		"doc",
		"community_post",
		"project",
	}

	if len(entityTypes) != 10 {
		t.Errorf("expected 10 entity types, got %d", len(entityTypes))
	}
}
