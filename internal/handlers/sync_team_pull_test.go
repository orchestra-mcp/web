package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
)

// TestTeamPullSyncLogPayload verifies SyncLog fields used by TeamPull serialize correctly.
func TestTeamPullSyncLogPayload(t *testing.T) {
	teamID := "team-abc"
	payload, _ := json.Marshal(map[string]string{"title": "Feature X"})

	log := models.SyncLog{
		ID:         "log-team-001",
		UserID:     10,
		DeviceID:   "device-mobile",
		EntityType: "feature",
		EntityID:   "FEAT-MOB",
		Action:     "upsert",
		Payload:    datatypes.JSON(payload),
		Version:    4,
		TeamID:     &teamID,
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed["entity_type"] != "feature" {
		t.Errorf("entity_type = %v, want feature", parsed["entity_type"])
	}
	if parsed["entity_id"] != "FEAT-MOB" {
		t.Errorf("entity_id = %v, want FEAT-MOB", parsed["entity_id"])
	}
	if parsed["team_id"] != "team-abc" {
		t.Errorf("team_id = %v, want team-abc", parsed["team_id"])
	}
	if parsed["action"] != "upsert" {
		t.Errorf("action = %v, want upsert", parsed["action"])
	}
}

// TestTeamPullDeleteRecord verifies delete action records work for TeamPull.
func TestTeamPullDeleteRecord(t *testing.T) {
	teamID := "team-xyz"
	log := models.SyncLog{
		ID:         "log-team-002",
		UserID:     5,
		EntityType: "note",
		EntityID:   "note-deleted",
		Action:     "delete",
		TeamID:     &teamID,
		Version:    2,
	}

	data, _ := json.Marshal(log)
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["action"] != "delete" {
		t.Errorf("action = %v, want delete", parsed["action"])
	}
	if parsed["team_id"] != "team-xyz" {
		t.Errorf("team_id = %v, want team-xyz", parsed["team_id"])
	}
}

// TestTeamShareRequestFields verifies the body fields expected by ShareEntity handler.
func TestTeamShareRequestFields(t *testing.T) {
	memberIDs, _ := json.Marshal([]string{"m1", "m2"})
	entityData, _ := json.Marshal(map[string]string{"title": "Shared Feature"})

	share := models.TeamShare{
		ID:           "share-flutter-001",
		UserID:       3,
		TeamID:       "team-flutter",
		EntityType:   "feature",
		EntityID:     "FEAT-SHR",
		ShareWithAll: true,
		MemberIDs:    datatypes.JSON(memberIDs),
		Permission:   "write",
		ContentHash:  "hash123",
		EntityData:   datatypes.JSON(entityData),
		Version:      1,
		SharedAt:     time.Now(),
	}

	data, _ := json.Marshal(share)
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["entity_type"] != "feature" {
		t.Errorf("entity_type = %v, want feature", parsed["entity_type"])
	}
	if parsed["share_with_all"] != true {
		t.Errorf("share_with_all = %v, want true", parsed["share_with_all"])
	}
	if parsed["permission"] != "write" {
		t.Errorf("permission = %v, want write", parsed["permission"])
	}
}
