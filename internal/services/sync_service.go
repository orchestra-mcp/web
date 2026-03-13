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
	TunnelID       *string         `json:"tunnel_id"`
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
	case "plan":
		return s.applyPlan(record, userID, payload, db)
	case "person":
		return s.applyPerson(record, userID, payload, db)
	case "request":
		return s.applyRequest(record, userID, payload, db)
	case "assignment_rule":
		return s.applyAssignmentRule(record, userID, payload, db)
	case "session_turn":
		return s.applySessionTurn(record, userID, payload, db)
	case "doc":
		return s.applyDoc(record, userID, payload, db)
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

	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	note := models.Note{}
	note.Base.ID = record.EntityID
	note.UserID = userID
	note.Version = int(record.Version)
	note.Meta = payload
	if v, ok := data["title"].(string); ok {
		note.Title = v
	}
	if v, ok := data["content"].(string); ok {
		note.Content = v
	}
	if v, ok := data["body"].(string); ok && note.Content == "" {
		note.Content = v
	}
	if v, ok := data["pinned"].(bool); ok {
		note.Pinned = v
	}
	if record.TeamID != nil {
		note.TeamID = record.TeamID
	}

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "content", "pinned", "meta", "version", "updated_at"}),
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

	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	session := models.AiSession{}
	session.Base.ID = record.EntityID
	session.UserID = userID
	session.Version = int(record.Version)
	session.Meta = payload
	if v, ok := data["name"].(string); ok {
		session.Name = v
	}
	if v, ok := data["model"].(string); ok {
		session.Model = v
	}
	if v, ok := data["pinned"].(bool); ok {
		session.Pinned = v
	}
	if v, ok := data["message_count"].(float64); ok {
		session.MessageCount = int(v)
	}

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "model", "pinned", "message_count", "meta", "version", "updated_at"}),
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

func (s *SyncService) applyPlan(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	planID := record.EntityID
	if v, ok := data["id"].(string); ok {
		planID = v
	}

	if record.Action == "delete" {
		return db.Where("plan_id = ? AND user_id = ?", planID, userID).
			Delete(&models.Plan{}).Error
	}

	var plan models.Plan
	err := db.Where("plan_id = ? AND user_id = ?", planID, userID).First(&plan).Error
	if err == nil && int64(plan.Version) >= record.Version {
		return nil
	}

	plan.UserID = userID
	plan.PlanID = planID
	plan.Version = int(record.Version)
	plan.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		plan.ProjectSlug = v
	}
	if v, ok := data["title"].(string); ok {
		plan.Title = v
	}
	if v, ok := data["description"].(string); ok {
		plan.Description = v
	}
	if v, ok := data["status"].(string); ok {
		plan.Status = v
	}
	if v, ok := data["body"].(string); ok {
		plan.Body = v
	}
	if record.TeamID != nil {
		plan.TeamID = record.TeamID
	}

	if plan.Base.ID == "" {
		return db.Create(&plan).Error
	}
	return db.Save(&plan).Error
}

func (s *SyncService) applyPerson(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	personID := record.EntityID
	if v, ok := data["id"].(string); ok {
		personID = v
	}

	if record.Action == "delete" {
		return db.Where("person_id = ? AND user_id = ?", personID, userID).
			Delete(&models.Person{}).Error
	}

	var person models.Person
	err := db.Where("person_id = ? AND user_id = ?", personID, userID).First(&person).Error
	if err == nil && int64(person.Version) >= record.Version {
		return nil
	}

	person.UserID = userID
	person.PersonID = personID
	person.Version = int(record.Version)
	person.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		person.ProjectSlug = v
	}
	if v, ok := data["name"].(string); ok {
		person.Name = v
	}
	if v, ok := data["email"].(string); ok {
		person.Email = v
	}
	if v, ok := data["role"].(string); ok {
		person.Role = v
	}
	if v, ok := data["status"].(string); ok {
		person.Status = v
	}
	if v, ok := data["bio"].(string); ok {
		person.Bio = v
	}
	if v, ok := data["body"].(string); ok {
		person.Body = v
	}
	if record.TeamID != nil {
		person.TeamID = record.TeamID
	}

	if person.Base.ID == "" {
		return db.Create(&person).Error
	}
	return db.Save(&person).Error
}

func (s *SyncService) applyRequest(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	requestID := record.EntityID
	if v, ok := data["id"].(string); ok {
		requestID = v
	}

	if record.Action == "delete" {
		return db.Where("request_id = ? AND user_id = ?", requestID, userID).
			Delete(&models.Request{}).Error
	}

	var request models.Request
	err := db.Where("request_id = ? AND user_id = ?", requestID, userID).First(&request).Error
	if err == nil && int64(request.Version) >= record.Version {
		return nil
	}

	request.UserID = userID
	request.RequestID = requestID
	request.Version = int(record.Version)
	request.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		request.ProjectSlug = v
	}
	if v, ok := data["title"].(string); ok {
		request.Title = v
	}
	if v, ok := data["description"].(string); ok {
		request.Description = v
	}
	if v, ok := data["kind"].(string); ok {
		request.Kind = v
	}
	if v, ok := data["priority"].(string); ok {
		request.Priority = v
	}
	if v, ok := data["status"].(string); ok {
		request.Status = v
	}
	if v, ok := data["body"].(string); ok {
		request.Body = v
	}
	if record.TeamID != nil {
		request.TeamID = record.TeamID
	}

	if request.Base.ID == "" {
		return db.Create(&request).Error
	}
	return db.Save(&request).Error
}

func (s *SyncService) applyAssignmentRule(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	ruleID := record.EntityID
	if v, ok := data["id"].(string); ok {
		ruleID = v
	}

	if record.Action == "delete" {
		return db.Where("rule_id = ? AND user_id = ?", ruleID, userID).
			Delete(&models.AssignmentRule{}).Error
	}

	var rule models.AssignmentRule
	err := db.Where("rule_id = ? AND user_id = ?", ruleID, userID).First(&rule).Error
	if err == nil && int64(rule.Version) >= record.Version {
		return nil
	}

	rule.UserID = userID
	rule.RuleID = ruleID
	rule.Version = int(record.Version)
	rule.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		rule.ProjectSlug = v
	}
	if v, ok := data["kind"].(string); ok {
		rule.Kind = v
	}
	if v, ok := data["person_id"].(string); ok {
		rule.PersonID = v
	}
	if v, ok := data["body"].(string); ok {
		rule.Body = v
	}
	if record.TeamID != nil {
		rule.TeamID = record.TeamID
	}

	if rule.Base.ID == "" {
		return db.Create(&rule).Error
	}
	return db.Save(&rule).Error
}

func (s *SyncService) applySessionTurn(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	turnID := record.EntityID
	sessionID := ""
	if v, ok := data["session_id"].(string); ok {
		sessionID = v
	}

	if record.Action == "delete" {
		return db.Where("turn_id = ? AND user_id = ?", turnID, userID).
			Delete(&models.SessionTurn{}).Error
	}

	var turn models.SessionTurn
	err := db.Where("turn_id = ? AND session_id = ? AND user_id = ?", turnID, sessionID, userID).First(&turn).Error
	if err == nil && int64(turn.Version) >= record.Version {
		return nil
	}

	turn.UserID = userID
	turn.TurnID = turnID
	turn.SessionID = sessionID
	turn.Version = int(record.Version)
	turn.Meta = payload
	if v, ok := data["body"].(string); ok {
		turn.Body = v
	}

	if turn.Base.ID == "" {
		return db.Create(&turn).Error
	}
	return db.Save(&turn).Error
}

func (s *SyncService) applyDoc(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	docID := record.EntityID
	if v, ok := data["id"].(string); ok {
		docID = v
	} else if v, ok := data["slug"].(string); ok {
		docID = v
	}

	if record.Action == "delete" {
		return db.Where("doc_id = ? AND user_id = ?", docID, userID).
			Delete(&models.Doc{}).Error
	}

	var doc models.Doc
	err := db.Where("doc_id = ? AND user_id = ?", docID, userID).First(&doc).Error
	if err == nil && int64(doc.Version) >= record.Version {
		return nil
	}

	doc.UserID = userID
	doc.DocID = docID
	doc.Version = int(record.Version)
	doc.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		doc.ProjectSlug = v
	}
	if v, ok := data["title"].(string); ok {
		doc.Title = v
	}
	if v, ok := data["category"].(string); ok {
		doc.Category = v
	}
	if v, ok := data["parent_id"].(string); ok {
		doc.ParentID = v
	}
	if v, ok := data["body"].(string); ok {
		doc.Body = v
	}
	if record.TeamID != nil {
		doc.TeamID = record.TeamID
	}

	if doc.Base.ID == "" {
		return db.Create(&doc).Error
	}
	return db.Save(&doc).Error
}
