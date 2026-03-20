package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncRecord represents a single entity change from a client push.
type SyncRecord struct {
	EntityType      string          `json:"entity_type"`
	EntityID        string          `json:"entity_id"`
	Action          string          `json:"action"` // upsert | delete
	Payload         json.RawMessage `json:"payload"`
	Version         int64           `json:"version"`
	ClientTimestamp *time.Time      `json:"client_timestamp,omitempty"` // device-local write timestamp for LWW
	IdempotencyKey  *string         `json:"idempotency_key"`
	TeamID          *string         `json:"team_id"`
	TunnelID        *string         `json:"tunnel_id"`
}

// ConflictInfo describes a detected version conflict for a single push record.
type ConflictInfo struct {
	EntityType      string
	EntityID        string
	ServerVersion   int64
	ClientVersion   int64
	ServerUpdatedAt time.Time
	ClientTimestamp *time.Time
}

// DetectConflict queries the server-side version of the entity and returns
// conflict information when the server has a strictly newer version than the
// incoming client record. Returns nil if no conflict or entity is new.
//
// A conflict means: server_version > client_version — the server has changes
// the client hasn't seen yet. LWW resolution uses client_timestamp vs
// server updated_at to decide which payload wins.
func (s *SyncService) DetectConflict(record SyncRecord, userID uint, db *gorm.DB) (*ConflictInfo, error) {
	if record.Action == "delete" {
		return nil, nil // deletes are always authoritative
	}
	sv, sat, err := s.serverVersion(record, userID, db)
	if err != nil || sv == 0 {
		return nil, nil // entity doesn't exist yet — no conflict
	}
	if sv <= record.Version {
		return nil, nil // server is at same or older version — no conflict
	}
	return &ConflictInfo{
		EntityType:      record.EntityType,
		EntityID:        record.EntityID,
		ServerVersion:   sv,
		ClientVersion:   record.Version,
		ServerUpdatedAt: sat,
		ClientTimestamp: record.ClientTimestamp,
	}, nil
}

// serverVersion returns the server-stored version and updated_at for the entity.
// Returns (0, zero, nil) when the entity doesn't exist.
func (s *SyncService) serverVersion(record SyncRecord, userID uint, db *gorm.DB) (int64, time.Time, error) {
	scope := ownerScope(db, userID, record.TeamID)

	switch record.EntityType {
	case "project":
		var r struct {
			Version   int       `gorm:"column:version"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		err := scope.Model(&models.Project{}).Where("slug = ?", record.EntityID).
			Select("version, updated_at").Scan(&r).Error
		return int64(r.Version), r.UpdatedAt, err
	case "feature":
		var r struct {
			Version   int       `gorm:"column:version"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		err := scope.Model(&models.Feature{}).Where("id = ?", record.EntityID).
			Select("version, updated_at").Scan(&r).Error
		return int64(r.Version), r.UpdatedAt, err
	case "note":
		var r struct {
			Version   int       `gorm:"column:version"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		err := scope.Model(&models.Note{}).Where("id = ?", record.EntityID).
			Select("version, updated_at").Scan(&r).Error
		return int64(r.Version), r.UpdatedAt, err
	case "plan":
		var r struct {
			Version   int       `gorm:"column:version"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		err := scope.Model(&models.Plan{}).Where("id = ?", record.EntityID).
			Select("version, updated_at").Scan(&r).Error
		return int64(r.Version), r.UpdatedAt, err
	case "doc":
		var r struct {
			Version   int       `gorm:"column:version"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		err := scope.Model(&models.Doc{}).Where("id = ?", record.EntityID).
			Select("version, updated_at").Scan(&r).Error
		return int64(r.Version), r.UpdatedAt, err
	case "person":
		var r struct {
			Version   int       `gorm:"column:version"`
			UpdatedAt time.Time `gorm:"column:updated_at"`
		}
		err := scope.Model(&models.Person{}).Where("id = ?", record.EntityID).
			Select("version, updated_at").Scan(&r).Error
		return int64(r.Version), r.UpdatedAt, err
	default:
		// For entity types without tracked version, report no conflict.
		return 0, time.Time{}, nil
	}
}

// ResolveConflict determines the winning payload using last-write-wins.
// If client_timestamp is provided and is strictly newer than server updated_at,
// the client wins (returns true). Otherwise the server wins (returns false).
func ResolveConflict(info *ConflictInfo) bool {
	if info.ClientTimestamp == nil {
		return false // no client timestamp — server wins
	}
	return info.ClientTimestamp.After(info.ServerUpdatedAt)
}

// SyncService handles LWW sync logic for entity push/pull.
type SyncService struct{}

// NewSyncService creates a new SyncService.
func NewSyncService() *SyncService {
	return &SyncService{}
}

// ownerScope returns a GORM scope that matches records owned by userID
// OR belonging to the record's team (when teamID is set).
// This ensures team members can read/update each other's records.
func ownerScope(db *gorm.DB, userID uint, teamID *string) *gorm.DB {
	if teamID != nil && *teamID != "" {
		return db.Where("user_id = ? OR team_id = ?", userID, *teamID)
	}
	return db.Where("user_id = ?", userID)
}

// ensureProject creates a project row for the given slug if one doesn't exist yet.
// Called by project-scoped apply* methods so the projects list is always in sync.
func ensureProject(db *gorm.DB, userID uint, teamID *string, slug string) {
	if slug == "" {
		return
	}
	var count int64
	ownerScope(db, userID, teamID).Model(&models.Project{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		return
	}
	// Derive a human-readable name from the slug (e.g. "orchestra-agents" → "Orchestra Agents").
	words := strings.Fields(strings.ReplaceAll(slug, "-", " "))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	p := models.Project{
		UserID:     userID,
		TeamID:     teamID,
		Slug:       slug,
		Name:       strings.Join(words, " "),
		SyncStatus: "not_synced",
		Version:    1,
	}
	db.Create(&p)
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
	case "skill":
		return s.applySkill(record, userID, payload, db)
	case "agent":
		return s.applyAgent(record, userID, payload, db)
	case "delegation":
		return s.applyDelegation(record, userID, payload, db)
	case "workflow":
		return s.applyWorkflow(record, userID, payload, db)
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
		return ownerScope(db, userID, record.TeamID).
			Where("slug = ?", slug).Delete(&models.Project{}).Error
	}

	var project models.Project
	err := ownerScope(db, userID, record.TeamID).Where("slug = ?", slug).First(&project).Error
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
		return ownerScope(db, userID, record.TeamID).
			Where("id = ?", record.EntityID).Delete(&models.Note{}).Error
	}

	var existing models.Note
	err := ownerScope(db, userID, record.TeamID).Where("id = ?", record.EntityID).First(&existing).Error

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
		return ownerScope(db, userID, record.TeamID).
			Where("feature_id = ?", featureID).Delete(&models.Feature{}).Error
	}

	// Look up by feature_id scoped to owner/team.
	var feature models.Feature
	err := ownerScope(db, userID, record.TeamID).Where("feature_id = ?", featureID).First(&feature).Error
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
		ensureProject(db, userID, record.TeamID, v)
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
		return ownerScope(db, userID, record.TeamID).
			Where("plan_id = ?", planID).Delete(&models.Plan{}).Error
	}

	var plan models.Plan
	err := ownerScope(db, userID, record.TeamID).Where("plan_id = ?", planID).First(&plan).Error
	if err == nil && int64(plan.Version) >= record.Version {
		return nil
	}

	plan.UserID = userID
	plan.PlanID = planID
	plan.Version = int(record.Version)
	plan.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		plan.ProjectSlug = v
		ensureProject(db, userID, record.TeamID, v)
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
		return ownerScope(db, userID, record.TeamID).
			Where("person_id = ?", personID).Delete(&models.Person{}).Error
	}

	var person models.Person
	err := ownerScope(db, userID, record.TeamID).Where("person_id = ?", personID).First(&person).Error
	if err == nil && int64(person.Version) >= record.Version {
		return nil
	}

	person.UserID = userID
	person.PersonID = personID
	person.Version = int(record.Version)
	person.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		person.ProjectSlug = v
		ensureProject(db, userID, record.TeamID, v)
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
		return ownerScope(db, userID, record.TeamID).
			Where("request_id = ?", requestID).Delete(&models.Request{}).Error
	}

	var request models.Request
	err := ownerScope(db, userID, record.TeamID).Where("request_id = ?", requestID).First(&request).Error
	if err == nil && int64(request.Version) >= record.Version {
		return nil
	}

	request.UserID = userID
	request.RequestID = requestID
	request.Version = int(record.Version)
	request.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		request.ProjectSlug = v
		ensureProject(db, userID, record.TeamID, v)
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
		return ownerScope(db, userID, record.TeamID).
			Where("rule_id = ?", ruleID).Delete(&models.AssignmentRule{}).Error
	}

	var rule models.AssignmentRule
	err := ownerScope(db, userID, record.TeamID).Where("rule_id = ?", ruleID).First(&rule).Error
	if err == nil && int64(rule.Version) >= record.Version {
		return nil
	}

	rule.UserID = userID
	rule.RuleID = ruleID
	rule.Version = int(record.Version)
	rule.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		rule.ProjectSlug = v
		ensureProject(db, userID, record.TeamID, v)
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

func (s *SyncService) applySkill(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	slug := record.EntityID
	if v, ok := data["slug"].(string); ok {
		slug = v
	}

	if record.Action == "delete" {
		return ownerScope(db, userID, record.TeamID).
			Where("slug = ?", slug).Delete(&models.Skill{}).Error
	}

	var skill models.Skill
	err := ownerScope(db, userID, record.TeamID).Where("slug = ?", slug).First(&skill).Error
	if err == nil && int64(skill.Version) >= record.Version {
		return nil
	}

	skill.UserID = userID
	skill.Slug = slug
	skill.Version = int(record.Version)
	skill.Meta = payload
	if v, ok := data["name"].(string); ok {
		skill.Name = v
	}
	if v, ok := data["description"].(string); ok {
		skill.Description = v
	}
	if v, ok := data["content"].(string); ok {
		skill.Content = v
	}
	if v, ok := data["scope"].(string); ok {
		skill.Scope = v
	}
	if v, ok := data["icon"].(string); ok {
		skill.Icon = v
	}
	if v, ok := data["color"].(string); ok {
		skill.Color = v
	}
	if record.TeamID != nil {
		skill.TeamID = record.TeamID
	}

	if skill.Base.ID == "" {
		return db.Create(&skill).Error
	}
	return db.Save(&skill).Error
}

func (s *SyncService) applyAgent(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	slug := record.EntityID
	if v, ok := data["slug"].(string); ok {
		slug = v
	}

	if record.Action == "delete" {
		return ownerScope(db, userID, record.TeamID).
			Where("slug = ?", slug).Delete(&models.Agent{}).Error
	}

	var agent models.Agent
	err := ownerScope(db, userID, record.TeamID).Where("slug = ?", slug).First(&agent).Error
	if err == nil && int64(agent.Version) >= record.Version {
		return nil
	}

	agent.UserID = userID
	agent.Slug = slug
	agent.Version = int(record.Version)
	agent.Meta = payload
	if v, ok := data["name"].(string); ok {
		agent.Name = v
	}
	if v, ok := data["description"].(string); ok {
		agent.Description = v
	}
	if v, ok := data["content"].(string); ok {
		agent.Content = v
	}
	if v, ok := data["scope"].(string); ok {
		agent.Scope = v
	}
	if v, ok := data["icon"].(string); ok {
		agent.Icon = v
	}
	if v, ok := data["color"].(string); ok {
		agent.Color = v
	}
	if record.TeamID != nil {
		agent.TeamID = record.TeamID
	}

	if agent.Base.ID == "" {
		return db.Create(&agent).Error
	}
	return db.Save(&agent).Error
}

func (s *SyncService) applyDelegation(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	delegationID := record.EntityID
	if v, ok := data["id"].(string); ok {
		delegationID = v
	} else if v, ok := data["delegation_id"].(string); ok {
		delegationID = v
	}

	if record.Action == "delete" {
		return ownerScope(db, userID, record.TeamID).
			Where("delegation_id = ?", delegationID).Delete(&models.Delegation{}).Error
	}

	var delegation models.Delegation
	err := ownerScope(db, userID, record.TeamID).Where("delegation_id = ?", delegationID).First(&delegation).Error
	if err == nil && int64(delegation.Version) >= record.Version {
		return nil
	}

	delegation.UserID = userID
	delegation.DelegationID = delegationID
	delegation.Version = int(record.Version)
	delegation.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		delegation.ProjectSlug = v
		ensureProject(db, userID, record.TeamID, v)
	}
	if v, ok := data["feature_id"].(string); ok {
		delegation.FeatureID = v
	}
	if v, ok := data["from_person"].(string); ok {
		delegation.FromPerson = v
	}
	if v, ok := data["to_person"].(string); ok {
		delegation.ToPerson = v
	}
	if v, ok := data["question"].(string); ok {
		delegation.Question = v
	}
	if v, ok := data["context"].(string); ok {
		delegation.Context = v
	}
	if v, ok := data["response"].(string); ok {
		delegation.Response = v
	}
	if v, ok := data["status"].(string); ok {
		delegation.Status = v
	}
	if v, ok := data["body"].(string); ok {
		delegation.Body = v
	}
	if v, ok := data["responded_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			delegation.RespondedAt = &t
		}
	}
	if record.TeamID != nil {
		delegation.TeamID = record.TeamID
	}

	if delegation.Base.ID == "" {
		return db.Create(&delegation).Error
	}
	return db.Save(&delegation).Error
}

func (s *SyncService) applyWorkflow(record SyncRecord, userID uint, payload datatypes.JSON, db *gorm.DB) error {
	var data map[string]interface{}
	_ = json.Unmarshal(record.Payload, &data)

	workflowID := record.EntityID
	if v, ok := data["id"].(string); ok {
		workflowID = v
	} else if v, ok := data["workflow_id"].(string); ok {
		workflowID = v
	}

	if record.Action == "delete" {
		return ownerScope(db, userID, record.TeamID).
			Where("workflow_id = ?", workflowID).Delete(&models.Workflow{}).Error
	}

	var wf models.Workflow
	err := ownerScope(db, userID, record.TeamID).Where("workflow_id = ?", workflowID).First(&wf).Error
	if err == nil && int64(wf.Version) >= record.Version {
		return nil
	}

	wf.UserID = userID
	wf.WorkflowID = workflowID
	wf.Version = int(record.Version)
	wf.Meta = payload
	if v, ok := data["project_id"].(string); ok {
		wf.ProjectSlug = v
		ensureProject(db, userID, record.TeamID, v)
	}
	if v, ok := data["name"].(string); ok {
		wf.Name = v
	}
	if v, ok := data["description"].(string); ok {
		wf.Description = v
	}
	if v, ok := data["initial_state"].(string); ok {
		wf.InitialState = v
	}
	if v, ok := data["is_default"].(bool); ok {
		wf.IsDefault = v
	}
	if v, ok := data["states"]; ok {
		b, _ := json.Marshal(v)
		wf.States = datatypes.JSON(b)
	}
	if v, ok := data["transitions"]; ok {
		b, _ := json.Marshal(v)
		wf.Transitions = datatypes.JSON(b)
	}
	if v, ok := data["gates"]; ok {
		b, _ := json.Marshal(v)
		wf.Gates = datatypes.JSON(b)
	}
	if record.TeamID != nil {
		wf.TeamID = record.TeamID
	}

	if wf.Base.ID == "" {
		return db.Create(&wf).Error
	}
	return db.Save(&wf).Error
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
		return ownerScope(db, userID, record.TeamID).
			Where("doc_id = ?", docID).Delete(&models.Doc{}).Error
	}

	var doc models.Doc
	err := ownerScope(db, userID, record.TeamID).Where("doc_id = ?", docID).First(&doc).Error
	if err == nil && int64(doc.Version) >= record.Version {
		return nil
	}

	doc.UserID = userID
	doc.DocID = docID
	doc.Version = int(record.Version)
	doc.Meta = payload
	if v, ok := data["project_slug"].(string); ok {
		doc.ProjectSlug = v
		ensureProject(db, userID, record.TeamID, v)
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
