package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
)

// TestSyncLogDeltaFields verifies SyncLog fields used by the Delta endpoint serialize correctly.
func TestSyncLogDeltaFields(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	payload, _ := json.Marshal(map[string]string{"title": "My Feature"})

	log := models.SyncLog{
		ID:         "log-001",
		UserID:     42,
		DeviceID:   "device-abc",
		EntityType: "feature",
		EntityID:   "FEAT-XYZ",
		Action:     "upsert",
		Payload:    datatypes.JSON(payload),
		Version:    7,
		CreatedAt:  now,
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal SyncLog: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal SyncLog JSON: %v", err)
	}

	if parsed["entity_type"] != "feature" {
		t.Errorf("entity_type = %v, want feature", parsed["entity_type"])
	}
	if parsed["entity_id"] != "FEAT-XYZ" {
		t.Errorf("entity_id = %v, want FEAT-XYZ", parsed["entity_id"])
	}
	if parsed["action"] != "upsert" {
		t.Errorf("action = %v, want upsert", parsed["action"])
	}
	if int(parsed["version"].(float64)) != 7 {
		t.Errorf("version = %v, want 7", parsed["version"])
	}
}

// TestSyncLogDeleteAction verifies delete action serializes correctly for delta tracking.
func TestSyncLogDeleteAction(t *testing.T) {
	log := models.SyncLog{
		ID:         "log-002",
		UserID:     1,
		EntityType: "plan",
		EntityID:   "PLAN-ABC",
		Action:     "delete",
		Version:    3,
	}

	data, _ := json.Marshal(log)
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["action"] != "delete" {
		t.Errorf("action = %v, want delete", parsed["action"])
	}
	if parsed["entity_type"] != "plan" {
		t.Errorf("entity_type = %v, want plan", parsed["entity_type"])
	}
}

// TestSyncLogTeamScoping verifies team_id field is present when set.
func TestSyncLogTeamScoping(t *testing.T) {
	teamID := "team-xyz"
	log := models.SyncLog{
		ID:         "log-003",
		UserID:     5,
		EntityType: "note",
		EntityID:   "note-001",
		Action:     "upsert",
		TeamID:     &teamID,
		Version:    1,
	}

	data, _ := json.Marshal(log)
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["team_id"] != "team-xyz" {
		t.Errorf("team_id = %v, want team-xyz", parsed["team_id"])
	}
}

// TestSyncLogIdempotencyKey verifies idempotency_key is present when set.
func TestSyncLogIdempotencyKey(t *testing.T) {
	key := "idem-key-abc123"
	log := models.SyncLog{
		ID:             "log-004",
		UserID:         10,
		EntityType:     "feature",
		EntityID:       "FEAT-001",
		Action:         "upsert",
		IdempotencyKey: &key,
		Version:        2,
	}

	data, _ := json.Marshal(log)
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["idempotency_key"] != "idem-key-abc123" {
		t.Errorf("idempotency_key = %v, want idem-key-abc123", parsed["idempotency_key"])
	}
}

// TestSyncLogEntityTypes verifies all supported entity types for delta tracking.
func TestSyncLogEntityTypes(t *testing.T) {
	entityTypes := []string{
		"project", "feature", "plan", "note", "skill", "agent", "hook",
		"person", "delegation", "workflow", "request", "doc",
	}

	for _, et := range entityTypes {
		log := models.SyncLog{
			EntityType: et,
			EntityID:   "entity-001",
			Action:     "upsert",
			Version:    1,
		}
		data, _ := json.Marshal(log)
		var parsed map[string]interface{}
		json.Unmarshal(data, &parsed)

		if parsed["entity_type"] != et {
			t.Errorf("entity_type = %v, want %s", parsed["entity_type"], et)
		}
	}
}
