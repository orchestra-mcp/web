package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncRecord represents a single entity change from a client push.
type SyncRecord struct {
	EntityType     string          `json:"entity_type"`
	EntityID       string          `json:"entity_id"`
	Action         string          `json:"action"` // upsert | delete
	Payload        json.RawMessage `json:"payload"`
	Version        int64           `json:"version"`
	IdempotencyKey *string         `json:"idempotency_key"`
	TeamID         *string         `json:"team_id"`
}

// SyncService handles LWW sync logic for entity push/pull.
type SyncService struct{}

// NewSyncService creates a new SyncService.
func NewSyncService() *SyncService {
	return &SyncService{}
}

// Apply applies a single sync record to the database using last-write-wins.
func (s *SyncService) Apply(record SyncRecord, userID uint, db *gorm.DB) error {
	payload := datatypes.JSON(record.Payload)

	switch record.EntityType {
	case "project":
		return s.applyProject(record, userID, payload, db)
	case "epic":
		return s.applyEpic(record, userID, payload, db)
	case "story":
		return s.applyStory(record, userID, payload, db)
	case "task":
		return s.applyTask(record, userID, payload, db)
	case "feature":
		return s.applyFeature(record, userID, payload, db)
	case "note":
		return s.applyNote(record, userID, payload, db)
	case "ai_session":
		return s.applyAiSession(record, userID, payload, db)
	default:
		// Unknown entity type — skip silently
		return nil
	}
}

func (s *SyncService) applyProject(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	slug := record.EntityID
	if v, ok := data["slug"].(string); ok {
		slug = v
	}

	if record.Action == "delete" {
		return db.Where("slug = ? AND user_id = ?", slug, userID).
			Delete(&models.Project{}).Error
	}

	var project models.Project
	err := db.Where("slug = ? AND user_id = ?", slug, userID).First(&project).Error
	if err == nil && int64(project.Version) >= record.Version {
		return nil
	}

	project.UserID = userID
	project.Slug = slug
	project.Version = int(record.Version)
	project.Meta = payload
	if v, ok := data["name"].(string); ok {
		project.Name = v
	}
	if v, ok := data["description"].(string); ok {
		project.Description = v
	}
	if record.TeamID != nil {
		project.TeamID = record.TeamID
	}

	if project.Base.ID == "" {
		return db.Create(&project).Error
	}
	return db.Save(&project).Error
}

func (s *SyncService) applyEpic(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	if record.Action == "delete" {
		return db.Where("id = ? AND user_id = ?", record.EntityID, userID).
			Delete(&models.Epic{}).Error
	}

	var existing models.Epic
	err := db.Where("id = ? AND user_id = ?", record.EntityID, userID).First(&existing).Error
	if err == nil && int64(existing.Version) >= record.Version {
		return nil
	}

	epic := models.Epic{}
	epic.Base.ID = record.EntityID
	epic.UserID = userID
	epic.Version = int(record.Version)
	epic.Meta = payload

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"meta", "version", "updated_at"}),
	}).Create(&epic).Error
}

func (s *SyncService) applyStory(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	if record.Action == "delete" {
		return db.Where("id = ? AND user_id = ?", record.EntityID, userID).
			Delete(&models.Story{}).Error
	}

	var existing models.Story
	err := db.Where("id = ? AND user_id = ?", record.EntityID, userID).First(&existing).Error
	if err == nil && int64(existing.Version) >= record.Version {
		return nil
	}

	story := models.Story{}
	story.Base.ID = record.EntityID
	story.UserID = userID
	story.Version = int(record.Version)
	story.Meta = payload

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"meta", "version", "updated_at"}),
	}).Create(&story).Error
}

func (s *SyncService) applyTask(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	if record.Action == "delete" {
		return db.Where("id = ? AND user_id = ?", record.EntityID, userID).
			Delete(&models.Task{}).Error
	}

	var existing models.Task
	err := db.Where("id = ? AND user_id = ?", record.EntityID, userID).First(&existing).Error
	if err == nil && int64(existing.Version) >= record.Version {
		return nil
	}

	task := models.Task{}
	task.Base.ID = record.EntityID
	task.UserID = userID
	task.Version = int(record.Version)
	task.Meta = payload

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"meta", "version", "updated_at"}),
	}).Create(&task).Error
}

func (s *SyncService) applyNote(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	if record.Action == "delete" {
		return db.Where("id = ? AND user_id = ?", record.EntityID, userID).
			Delete(&models.Note{}).Error
	}

	var existing models.Note
	err := db.Where("id = ? AND user_id = ?", record.EntityID, userID).First(&existing).Error

	if err == nil {
		if int64(existing.Version) >= record.Version {
			// Conflict: incoming is not newer — save revision for manual resolution
			var payloadMap map[string]interface{}
			_ = json.Unmarshal(record.Payload, &payloadMap)
			content, _ := payloadMap["content"].(string)

			revision := models.NoteRevision{
				NoteID:   record.EntityID,
				UserID:   userID,
				DeviceID: fmt.Sprintf("sync-conflict-%d", time.Now().UnixNano()),
				Content:  content,
				Version:  int(record.Version),
			}
			return db.Create(&revision).Error
		}
	}

	note := models.Note{}
	note.Base.ID = record.EntityID
	note.UserID = userID
	note.Version = int(record.Version)
	note.Meta = payload

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"meta", "version", "updated_at"}),
	}).Create(&note).Error
}

func (s *SyncService) applyAiSession(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	if record.Action == "delete" {
		return db.Where("id = ? AND user_id = ?", record.EntityID, userID).
			Delete(&models.AiSession{}).Error
	}

	var existing models.AiSession
	err := db.Where("id = ? AND user_id = ?", record.EntityID, userID).First(&existing).Error
	if err == nil && int64(existing.Version) >= record.Version {
		return nil
	}

	session := models.AiSession{}
	session.Base.ID = record.EntityID
	session.UserID = userID
	session.Version = int(record.Version)
	session.Meta = payload

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"meta", "version", "updated_at"}),
	}).Create(&session).Error
}

func (s *SyncService) applyFeature(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	// Parse typed fields from payload.
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	// Resolve feature ID — metadata uses "id" (e.g. FEAT-ATZ), fallback to entity ID from filename.
	featureID := record.EntityID
	if v, ok := data["id"].(string); ok {
		featureID = v
	} else if v, ok := data["feature_id"].(string); ok {
		featureID = v
	}

	if record.Action == "delete" {
		return db.Where("feature_id = ? AND user_id = ?", featureID, userID).
			Delete(&models.Feature{}).Error
	}

	// Look up by feature_id + user_id (not Base.ID which is a UUID).
	var feature models.Feature
	err := db.Where("feature_id = ? AND user_id = ?", featureID, userID).First(&feature).Error
	if err == nil && int64(feature.Version) >= record.Version {
		return nil
	}

	// Update fields.
	feature.UserID = userID
	feature.FeatureID = featureID
	feature.Version = int(record.Version)
	feature.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		feature.ProjectSlug = v
	}
	if v, ok := data["title"].(string); ok {
		feature.Title = v
	}
	if v, ok := data["description"].(string); ok {
		feature.Description = v
	}
	if v, ok := data["status"].(string); ok {
		feature.Status = v
	}
	if v, ok := data["priority"].(string); ok {
		feature.Priority = v
	}
	if v, ok := data["assignee"].(string); ok {
		feature.Assignee = v
	}
	if v, ok := data["estimate"].(string); ok {
		feature.Estimate = v
	}
	if v, ok := data["body"].(string); ok {
		feature.Body = v
	}
	if record.TeamID != nil {
		feature.TeamID = record.TeamID
	}

	if feature.Base.ID == "" {
		// New record — let Postgres generate UUID.
		return db.Create(&feature).Error
	}
	// Existing record — update in place.
	return db.Save(&feature).Error
}
