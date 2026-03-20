package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/orchestra-mcp/web/internal/services"
)

// TestResolveConflict_ServerWinsWhenNoClientTimestamp verifies that when the
// client sends no client_timestamp, the server version always wins.
func TestResolveConflict_ServerWinsWhenNoClientTimestamp(t *testing.T) {
	info := &services.ConflictInfo{
		EntityType:      "feature",
		EntityID:        "FEAT-ABC",
		ServerVersion:   5,
		ClientVersion:   3,
		ServerUpdatedAt: time.Now().Add(-1 * time.Hour),
		ClientTimestamp: nil, // no timestamp — server wins by default
	}
	if services.ResolveConflict(info) {
		t.Error("expected server_wins (false) when client_timestamp is nil")
	}
}

// TestResolveConflict_ClientWinsWhenNewerTimestamp verifies that a client
// with a strictly newer timestamp wins the LWW resolution.
func TestResolveConflict_ClientWinsWhenNewerTimestamp(t *testing.T) {
	serverUpdated := time.Now().Add(-2 * time.Hour)
	clientTs := time.Now().Add(-1 * time.Hour) // client wrote after server
	info := &services.ConflictInfo{
		EntityType:      "note",
		EntityID:        "note-001",
		ServerVersion:   4,
		ClientVersion:   2,
		ServerUpdatedAt: serverUpdated,
		ClientTimestamp: &clientTs,
	}
	if !services.ResolveConflict(info) {
		t.Error("expected client_wins (true) when client_timestamp is after server updated_at")
	}
}

// TestResolveConflict_ServerWinsWhenOlderTimestamp verifies that a client
// with an older timestamp loses to the server.
func TestResolveConflict_ServerWinsWhenOlderTimestamp(t *testing.T) {
	serverUpdated := time.Now().Add(-30 * time.Minute) // server updated more recently
	clientTs := time.Now().Add(-2 * time.Hour)         // client wrote earlier
	info := &services.ConflictInfo{
		EntityType:      "doc",
		EntityID:        "doc-xyz",
		ServerVersion:   7,
		ClientVersion:   5,
		ServerUpdatedAt: serverUpdated,
		ClientTimestamp: &clientTs,
	}
	if services.ResolveConflict(info) {
		t.Error("expected server_wins (false) when client_timestamp is before server updated_at")
	}
}

// TestSyncRecord_ClientTimestampSerialization verifies that client_timestamp
// round-trips through JSON correctly (new field must be backward compatible).
func TestSyncRecord_ClientTimestampSerialization(t *testing.T) {
	ts := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	rec := services.SyncRecord{
		EntityType:      "feature",
		EntityID:        "FEAT-TST",
		Action:          "upsert",
		Version:         3,
		ClientTimestamp: &ts,
	}

	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var out services.SyncRecord
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if out.ClientTimestamp == nil {
		t.Fatal("client_timestamp was nil after round-trip")
	}
	if !out.ClientTimestamp.Equal(ts) {
		t.Errorf("client_timestamp = %v, want %v", out.ClientTimestamp, ts)
	}
}

// TestSyncRecord_ClientTimestampOptional verifies that records without
// client_timestamp deserialize cleanly (backward compatibility).
func TestSyncRecord_ClientTimestampOptional(t *testing.T) {
	raw := `{"entity_type":"note","entity_id":"note-1","action":"upsert","version":2}`
	var rec services.SyncRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if rec.ClientTimestamp != nil {
		t.Error("expected nil client_timestamp for legacy records without the field")
	}
}

// TestConflictInfo_Fields verifies the ConflictInfo struct fields are populated correctly.
func TestConflictInfo_Fields(t *testing.T) {
	serverAt := time.Now().Add(-3 * time.Hour)
	clientAt := time.Now()
	info := &services.ConflictInfo{
		EntityType:      "plan",
		EntityID:        "PLAN-001",
		ServerVersion:   10,
		ClientVersion:   8,
		ServerUpdatedAt: serverAt,
		ClientTimestamp: &clientAt,
	}

	if info.EntityType != "plan" {
		t.Errorf("EntityType = %q, want plan", info.EntityType)
	}
	if info.ServerVersion != 10 {
		t.Errorf("ServerVersion = %d, want 10", info.ServerVersion)
	}
	if info.ClientVersion != 8 {
		t.Errorf("ClientVersion = %d, want 8", info.ClientVersion)
	}
	if !info.ServerUpdatedAt.Equal(serverAt) {
		t.Errorf("ServerUpdatedAt mismatch")
	}
	if !info.ClientTimestamp.Equal(clientAt) {
		t.Errorf("ClientTimestamp mismatch")
	}
}
