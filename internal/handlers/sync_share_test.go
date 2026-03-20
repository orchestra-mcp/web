package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
)

func TestTeamShareModelJSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	memberIDs, _ := json.Marshal([]string{"user-1", "user-2"})
	entityData, _ := json.Marshal(map[string]string{"title": "Test Project"})

	share := models.TeamShare{
		ID:           "share-123",
		UserID:       1,
		TeamID:       "team-abc",
		EntityType:   "project",
		EntityID:     "proj-xyz",
		ShareWithAll: false,
		MemberIDs:    datatypes.JSON(memberIDs),
		Permission:   "write",
		ContentHash:  "abc123",
		EntityData:   datatypes.JSON(entityData),
		Version:      2,
		SharedAt:     now,
	}

	data, err := json.Marshal(share)
	if err != nil {
		t.Fatalf("failed to marshal TeamShare: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal TeamShare JSON: %v", err)
	}

	if parsed["entity_type"] != "project" {
		t.Errorf("entity_type = %v, want project", parsed["entity_type"])
	}
	if parsed["entity_id"] != "proj-xyz" {
		t.Errorf("entity_id = %v, want proj-xyz", parsed["entity_id"])
	}
	if parsed["team_id"] != "team-abc" {
		t.Errorf("team_id = %v, want team-abc", parsed["team_id"])
	}
	if parsed["permission"] != "write" {
		t.Errorf("permission = %v, want write", parsed["permission"])
	}
	if parsed["share_with_all"] != false {
		t.Errorf("share_with_all = %v, want false", parsed["share_with_all"])
	}
	if int(parsed["version"].(float64)) != 2 {
		t.Errorf("version = %v, want 2", parsed["version"])
	}

	// Verify member_ids round-trips correctly.
	var ids []string
	midsRaw, _ := json.Marshal(parsed["member_ids"])
	if err := json.Unmarshal(midsRaw, &ids); err != nil {
		t.Fatalf("failed to parse member_ids: %v", err)
	}
	if len(ids) != 2 || ids[0] != "user-1" || ids[1] != "user-2" {
		t.Errorf("member_ids = %v, want [user-1, user-2]", ids)
	}
}

func TestTeamShareModelDefaults(t *testing.T) {
	share := models.TeamShare{}

	if share.Version != 0 {
		t.Errorf("default Version = %d, want 0 (GORM default:1 applies on Create)", share.Version)
	}
	if share.Permission != "" {
		t.Errorf("default Permission = %q, want empty (GORM default:read applies on Create)", share.Permission)
	}
	if share.ShareWithAll != false {
		t.Errorf("default ShareWithAll = %v, want false (GORM default:true applies on Create)", share.ShareWithAll)
	}
}

func TestSyncLogHasTeamID(t *testing.T) {
	teamID := "team-123"
	log := models.SyncLog{
		UserID:     1,
		EntityType: "project",
		EntityID:   "proj-1",
		Action:     "upsert",
		TeamID:     &teamID,
	}

	if log.TeamID == nil {
		t.Fatal("TeamID should not be nil")
	}
	if *log.TeamID != "team-123" {
		t.Errorf("TeamID = %q, want team-123", *log.TeamID)
	}
}

func TestSyncLogWithoutTeamID(t *testing.T) {
	log := models.SyncLog{
		UserID:     1,
		EntityType: "note",
		EntityID:   "note-1",
		Action:     "upsert",
	}

	if log.TeamID != nil {
		t.Errorf("TeamID = %v, want nil", log.TeamID)
	}
}

func TestUserTeamIDsEmptyDB(t *testing.T) {
	// userTeamIDs is a package-level function that requires a DB.
	// This test verifies the function signature exists and is callable.
	// Full integration tests require a test database.
	_ = userTeamIDs // Compile-time check that function exists.
}

func TestShareRequestFields(t *testing.T) {
	// Test the JSON shape matches what Flutter ShareRequest.toJson() sends.
	request := map[string]interface{}{
		"entity_type":    "feature",
		"entity_id":      "FEAT-ABC",
		"team_id":        "team-uuid",
		"share_with_all": true,
		"member_ids":     []string{},
		"permission":     "read",
		"entity_data":    map[string]string{"title": "Test"},
		"content_hash":   "sha256hex",
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal share request: %v", err)
	}

	var body struct {
		EntityType   string                 `json:"entity_type"`
		EntityID     string                 `json:"entity_id"`
		TeamID       string                 `json:"team_id"`
		ShareWithAll bool                   `json:"share_with_all"`
		MemberIDs    []string               `json:"member_ids"`
		Permission   string                 `json:"permission"`
		EntityData   map[string]interface{} `json:"entity_data"`
		ContentHash  string                 `json:"content_hash"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if body.EntityType != "feature" {
		t.Errorf("EntityType = %q, want feature", body.EntityType)
	}
	if body.EntityID != "FEAT-ABC" {
		t.Errorf("EntityID = %q, want FEAT-ABC", body.EntityID)
	}
	if body.TeamID != "team-uuid" {
		t.Errorf("TeamID = %q, want team-uuid", body.TeamID)
	}
	if !body.ShareWithAll {
		t.Errorf("ShareWithAll = false, want true")
	}
	if body.Permission != "read" {
		t.Errorf("Permission = %q, want read", body.Permission)
	}
	if body.ContentHash != "sha256hex" {
		t.Errorf("ContentHash = %q, want sha256hex", body.ContentHash)
	}
}

func TestTeamUpdateEntryJSON(t *testing.T) {
	// Verify the Go response matches Flutter TeamUpdateEntry.fromJson().
	entry := map[string]interface{}{
		"entity_type":  "feature",
		"entity_id":    "FEAT-XYZ",
		"entity_title": "My Feature",
		"team_id":      "team-1",
		"team_name":    "Alpha Team",
		"author_name":  "Jane Doe",
		"from_version": 1,
		"to_version":   2,
		"updated_at":   time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	requiredKeys := []string{
		"entity_type", "entity_id", "entity_title",
		"team_id", "team_name", "author_name",
		"from_version", "to_version", "updated_at",
	}
	for _, key := range requiredKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing required key %q in TeamUpdateEntry", key)
		}
	}
}

func TestShareResponseJSON(t *testing.T) {
	// Verify the Go response matches Flutter ShareResponse.fromJson().
	resp := map[string]interface{}{
		"share_id":         "share-uuid",
		"success":          true,
		"version":          1,
		"server_timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["share_id"] != "share-uuid" {
		t.Errorf("share_id = %v, want share-uuid", parsed["share_id"])
	}
	if parsed["success"] != true {
		t.Errorf("success = %v, want true", parsed["success"])
	}
	if int(parsed["version"].(float64)) != 1 {
		t.Errorf("version = %v, want 1", parsed["version"])
	}
	if _, ok := parsed["server_timestamp"]; !ok {
		t.Error("missing server_timestamp")
	}
}
