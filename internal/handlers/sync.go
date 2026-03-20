package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/services"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SyncHandler handles sync push/pull endpoints.
type SyncHandler struct {
	db          *gorm.DB
	syncService *services.SyncService
	hub         *hub.Hub
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(db *gorm.DB, h *hub.Hub) *SyncHandler {
	return &SyncHandler{
		db:          db,
		syncService: services.NewSyncService(),
		hub:         h,
	}
}

// RegisterDevice handles POST /api/sync/devices/register
func (h *SyncHandler) RegisterDevice(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		DeviceID string `json:"device_id"`
		Name     string `json:"name"`
		Platform string `json:"platform"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.DeviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "device_id is required"})
	}

	now := time.Now()
	var device models.DeviceToken
	result := h.db.Where("device_id = ? AND user_id = ?", body.DeviceID, user.ID).First(&device)
	if result.Error != nil {
		// Create new device record
		device = models.DeviceToken{
			UserID:     user.ID,
			DeviceID:   body.DeviceID,
			Name:       body.Name,
			Platform:   body.Platform,
			LastSeenAt: &now,
		}
		if err := h.db.Create(&device).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to register device"})
		}
	} else {
		// Update last seen
		h.db.Model(&device).Updates(map[string]interface{}{
			"last_seen_at": &now,
			"name":         body.Name,
			"platform":     body.Platform,
		})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// Push handles POST /api/sync/push
func (h *SyncHandler) Push(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		DeviceID string                `json:"device_id"`
		TunnelID string                `json:"tunnel_id"`
		Records  []services.SyncRecord `json:"records"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	type result struct {
		EntityID string `json:"entity_id"`
		Status   string `json:"status"`
		Error    string `json:"error,omitempty"`
	}

	results := make([]result, 0, len(body.Records))

	for _, record := range body.Records {
		// Check idempotency key uniqueness
		if record.IdempotencyKey != nil && *record.IdempotencyKey != "" {
			var count int64
			h.db.Model(&models.SyncLog{}).
				Where("idempotency_key = ?", *record.IdempotencyKey).
				Count(&count)
			if count > 0 {
				results = append(results, result{EntityID: record.EntityID, Status: "skipped"})
				continue
			}
		}

		// Write sync log entry
		var tunnelIDPtr *string
		if body.TunnelID != "" {
			tunnelIDPtr = &body.TunnelID
		}
		syncLog := models.SyncLog{
			UserID:         user.ID,
			DeviceID:       body.DeviceID,
			EntityType:     record.EntityType,
			EntityID:       record.EntityID,
			Action:         record.Action,
			Payload:        datatypes.JSON(record.Payload),
			Version:        record.Version,
			IdempotencyKey: record.IdempotencyKey,
			TeamID:         record.TeamID,
			TunnelID:       tunnelIDPtr,
		}
		h.db.Create(&syncLog)

		// Detect version conflict and log it before applying.
		// A conflict occurs when the server has a strictly newer version than
		// the client's record. LWW resolution (client_timestamp vs server
		// updated_at) determines the winning payload, but the server always
		// stores the resolved state and logs the conflict for audit.
		if conflict, _ := h.syncService.DetectConflict(record, user.ID, h.db); conflict != nil {
			resolution := "server_wins"
			if services.ResolveConflict(conflict) {
				resolution = "client_wins"
			}
			h.db.Create(&models.ConflictLog{
				UserID:        user.ID,
				EntityType:    record.EntityType,
				EntityID:      record.EntityID,
				LocalPayload:  datatypes.JSON(record.Payload), // "local" = what client sent
				RemotePayload: datatypes.JSON(nil),            // server payload not fetched (LWW applies anyway)
				Resolution:    resolution,
			})
		}

		// Apply LWW upsert
		if err := h.syncService.Apply(record, user.ID, h.db); err != nil {
			results = append(results, result{EntityID: record.EntityID, Status: "error", Error: err.Error()})
			continue
		}

		// Broadcast sync event to user + all team members.
		if h.hub != nil {
			event := hub.Event{
				Type:       "sync",
				EntityType: record.EntityType,
				EntityID:   record.EntityID,
				Action:     record.Action,
				UserID:     user.ID,
				Timestamp:  time.Now().Unix(),
			}
			if record.TeamID != nil && *record.TeamID != "" {
				// Broadcast to all members of the team.
				var memberIDs []uint
				h.db.Model(&models.Membership{}).
					Where("team_id = ?", *record.TeamID).
					Pluck("user_id", &memberIDs)
				if len(memberIDs) > 0 {
					h.hub.BroadcastToUsers(memberIDs, event)
				}
			} else {
				h.hub.BroadcastToUser(user.ID, event)
			}
		}

		// Send real-time notification for new pending delegations.
		if h.hub != nil && record.EntityType == "delegation" && record.Action == "upsert" {
			var delPayload map[string]interface{}
			_ = json.Unmarshal(record.Payload, &delPayload)
			if status, _ := delPayload["status"].(string); status == "pending" {
				toPerson, _ := delPayload["to_person"].(string)
				question, _ := delPayload["question"].(string)
				fromPerson, _ := delPayload["from_person"].(string)
				// Resolve to_person's email → user ID for notification.
				var personEmail string
				h.db.Model(&models.Person{}).Where("person_id = ?", toPerson).Pluck("email", &personEmail)
				if personEmail != "" {
					var targetUser models.User
					if h.db.Where("email = ?", personEmail).First(&targetUser).Error == nil {
						h.hub.BroadcastToUser(targetUser.ID, hub.Event{
							Type:       "notification",
							EntityType: "delegation",
							EntityID:   record.EntityID,
							Action:     "created",
							UserID:     user.ID,
							Timestamp:  time.Now().Unix(),
							Title:      "New Delegation",
							Message:    fmt.Sprintf("%s needs your input: %s", fromPerson, truncate(question, 100)),
							NType:      "info",
						})
					}
				}
			}
		}

		results = append(results, result{EntityID: record.EntityID, Status: "applied"})
	}

	return c.JSON(fiber.Map{"results": results})
}

// Pull handles GET /api/sync/pull
func (h *SyncHandler) Pull(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	sinceStr := c.Query("since")
	deviceID := c.Query("device_id")
	limit := 500
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid since format, use RFC3339"})
		}
	} else {
		since = time.Time{} // Beginning of time
	}

	// Include changes from all team members (not just current user).
	teamIDs := userTeamIDs(h.db, user.ID)
	query := h.db.Model(&models.SyncLog{}).
		Where("created_at > ?", since).
		Limit(limit).
		Order("created_at asc")

	if len(teamIDs) > 0 {
		query = query.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		query = query.Where("user_id = ?", user.ID)
	}

	if deviceID != "" {
		query = query.Where("device_id != ?", deviceID)
	}

	var records []models.SyncLog
	if err := query.Find(&records).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to pull records"})
	}

	return c.JSON(fiber.Map{
		"records": records,
		"count":   len(records),
	})
}

// Export handles GET /api/sync/export — bulk exports all user/team data for local SQLite population.
func (h *SyncHandler) Export(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	teamIDs := userTeamIDs(h.db, user.ID)

	// scope returns a GORM scope that filters by user_id OR team_id membership.
	scope := func(db *gorm.DB) *gorm.DB {
		if len(teamIDs) > 0 {
			return db.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
		}
		return db.Where("user_id = ?", user.ID)
	}

	var projects []models.Project
	h.db.Scopes(scope).Find(&projects)

	var features []models.Feature
	h.db.Scopes(scope).Find(&features)

	var notes []models.Note
	h.db.Scopes(scope).Find(&notes)

	var plans []models.Plan
	h.db.Scopes(scope).Find(&plans)

	var persons []models.Person
	h.db.Scopes(scope).Find(&persons)

	var docs []models.Doc
	h.db.Scopes(scope).Find(&docs)

	var skills []models.Skill
	h.db.Scopes(scope).Find(&skills)

	var agents []models.Agent
	h.db.Scopes(scope).Find(&agents)

	var delegations []models.Delegation
	h.db.Scopes(scope).Find(&delegations)

	var workflows []models.Workflow
	h.db.Scopes(scope).Find(&workflows)

	var requests []models.Request
	h.db.Scopes(scope).Find(&requests)

	// Fetch team members (real cloud users) for all user's teams.
	type memberRow struct {
		MembershipID string `json:"membership_id"`
		Name         string `json:"name"`
		Email        string `json:"email"`
		AvatarURL    string `json:"avatar_url"`
		Role         string `json:"role"`
		Status       string `json:"status"`
	}
	var members []memberRow
	if len(teamIDs) > 0 {
		var memberships []models.Membership
		h.db.Where("team_id IN ?", teamIDs).Preload("User").Find(&memberships)
		for _, m := range memberships {
			members = append(members, memberRow{
				MembershipID: m.ID,
				Name:         m.User.Name,
				Email:        m.User.Email,
				AvatarURL:    m.User.AvatarURL,
				Role:         m.Role,
				Status:       m.User.Status,
			})
		}
	}

	return c.JSON(fiber.Map{
		"projects":    projects,
		"features":    features,
		"notes":       notes,
		"plans":       plans,
		"persons":     persons,
		"docs":        docs,
		"skills":      skills,
		"agents":      agents,
		"delegations": delegations,
		"workflows":   workflows,
		"requests":    requests,
		"members":     members,
		"exported_at": time.Now().Format(time.RFC3339),
	})
}

// Status handles GET /api/sync/status
func (h *SyncHandler) Status(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	deviceID := c.Query("device_id")

	// If no device_id provided, return zero pending (client must identify itself).
	if deviceID == "" {
		return c.JSON(fiber.Map{
			"last_sync_at":  nil,
			"pending_count": 0,
			"devices":       []fiber.Map{},
		})
	}

	// Get last sync time for device
	var lastLog models.SyncLog
	var lastSyncAt *time.Time
	if err := h.db.Where("user_id = ? AND device_id = ?", user.ID, deviceID).
		Order("created_at desc").
		First(&lastLog).Error; err == nil {
		t := lastLog.CreatedAt
		lastSyncAt = &t
	}

	// Count pending records since last sync
	var pendingCount int64
	query := h.db.Model(&models.SyncLog{}).
		Where("user_id = ?", user.ID).
		Where("device_id != ?", deviceID)
	if lastSyncAt != nil {
		query = query.Where("created_at > ?", lastSyncAt)
	}
	query.Count(&pendingCount)

	// Get all devices for this user
	var devices []models.DeviceToken
	h.db.Where("user_id = ?", user.ID).Find(&devices)

	return c.JSON(fiber.Map{
		"last_sync_at":  lastSyncAt,
		"pending_count": pendingCount,
		"devices":       devices,
	})
}

// ShareEntity handles POST /api/sync/share
func (h *SyncHandler) ShareEntity(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.EntityType == "" || body.EntityID == "" || body.TeamID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "entity_type, entity_id, and team_id are required"})
	}

	// Verify user is a member of the target team.
	var membership models.Membership
	if err := h.db.Where("user_id = ? AND team_id = ?", user.ID, body.TeamID).
		First(&membership).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a member of this team"})
	}

	if body.Permission == "" {
		body.Permission = "read"
	}

	memberIDsJSON, _ := json.Marshal(body.MemberIDs)
	entityDataJSON, _ := json.Marshal(body.EntityData)
	now := time.Now()

	// Upsert: update if same entity+team share exists.
	var existing models.TeamShare
	result := h.db.Where("entity_type = ? AND entity_id = ? AND team_id = ?",
		body.EntityType, body.EntityID, body.TeamID).First(&existing)

	var share models.TeamShare
	if result.Error == nil {
		// Update existing share.
		existing.ShareWithAll = body.ShareWithAll
		existing.MemberIDs = datatypes.JSON(memberIDsJSON)
		existing.Permission = body.Permission
		existing.ContentHash = body.ContentHash
		existing.EntityData = datatypes.JSON(entityDataJSON)
		existing.Version++
		h.db.Save(&existing)
		share = existing
	} else {
		// Create new share.
		share = models.TeamShare{
			UserID:       user.ID,
			TeamID:       body.TeamID,
			EntityType:   body.EntityType,
			EntityID:     body.EntityID,
			ShareWithAll: body.ShareWithAll,
			MemberIDs:    datatypes.JSON(memberIDsJSON),
			Permission:   body.Permission,
			ContentHash:  body.ContentHash,
			EntityData:   datatypes.JSON(entityDataJSON),
			Version:      1,
			SharedAt:     now,
		}
		h.db.Create(&share)
	}

	// Write sync log entry for team propagation.
	syncLog := models.SyncLog{
		UserID:     user.ID,
		EntityType: body.EntityType,
		EntityID:   body.EntityID,
		Action:     "upsert",
		Payload:    datatypes.JSON(entityDataJSON),
		Version:    int64(share.Version),
		TeamID:     &body.TeamID,
	}
	h.db.Create(&syncLog)

	// Broadcast to team members.
	if h.hub != nil {
		var memberIDs []uint
		h.db.Model(&models.Membership{}).
			Where("team_id = ?", body.TeamID).
			Pluck("user_id", &memberIDs)
		if len(memberIDs) > 0 {
			h.hub.BroadcastToUsers(memberIDs, hub.Event{
				Type:       "sync",
				EntityType: body.EntityType,
				EntityID:   body.EntityID,
				Action:     "shared",
				UserID:     user.ID,
				Timestamp:  now.Unix(),
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"share_id":         share.ID,
		"success":          true,
		"version":          share.Version,
		"server_timestamp": now.UTC().Format(time.RFC3339),
	})
}

// TeamUpdates handles GET /api/sync/team-updates
func (h *SyncHandler) TeamUpdates(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	teamIDs := userTeamIDs(h.db, user.ID)
	now := time.Now()

	if len(teamIDs) == 0 {
		return c.JSON(fiber.Map{
			"available_updates": 0,
			"updates":           []fiber.Map{},
			"checked_at":        now.UTC().Format(time.RFC3339),
		})
	}

	// Find team-scoped sync log entries not authored by the current user.
	var logs []models.SyncLog
	h.db.Where("team_id IN ? AND user_id != ?", teamIDs, user.ID).
		Order("created_at desc").
		Limit(50).
		Find(&logs)

	type updateEntry struct {
		EntityType  string `json:"entity_type"`
		EntityID    string `json:"entity_id"`
		EntityTitle string `json:"entity_title"`
		TeamID      string `json:"team_id"`
		TeamName    string `json:"team_name"`
		AuthorName  string `json:"author_name"`
		FromVersion int    `json:"from_version"`
		ToVersion   int    `json:"to_version"`
		UpdatedAt   string `json:"updated_at"`
	}

	// Cache team names and user names.
	teamNames := map[string]string{}
	userNames := map[uint]string{}
	updates := make([]updateEntry, 0, len(logs))

	for _, log := range logs {
		teamID := ""
		if log.TeamID != nil {
			teamID = *log.TeamID
		}

		// Resolve team name.
		if _, ok := teamNames[teamID]; !ok && teamID != "" {
			var team models.Team
			if h.db.Where("id = ?", teamID).First(&team).Error == nil {
				teamNames[teamID] = team.Name
			}
		}

		// Resolve author name.
		if _, ok := userNames[log.UserID]; !ok {
			var u models.User
			if h.db.Where("id = ?", log.UserID).First(&u).Error == nil {
				userNames[log.UserID] = u.Name
			}
		}

		// Extract title from payload.
		var payload map[string]interface{}
		_ = json.Unmarshal(log.Payload, &payload)
		title, _ := payload["title"].(string)
		if title == "" {
			title, _ = payload["name"].(string)
		}

		updates = append(updates, updateEntry{
			EntityType:  log.EntityType,
			EntityID:    log.EntityID,
			EntityTitle: title,
			TeamID:      teamID,
			TeamName:    teamNames[teamID],
			AuthorName:  userNames[log.UserID],
			FromVersion: int(log.Version) - 1,
			ToVersion:   int(log.Version),
			UpdatedAt:   log.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return c.JSON(fiber.Map{
		"available_updates": len(updates),
		"updates":           updates,
		"checked_at":        now.UTC().Format(time.RFC3339),
	})
}

// GetEntityShares handles GET /api/sync/share/:entityType/:entityId
func (h *SyncHandler) GetEntityShares(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	entityType := c.Params("entityType")
	entityID := c.Params("entityId")

	var shares []models.TeamShare
	h.db.Where("entity_type = ? AND entity_id = ? AND user_id = ?",
		entityType, entityID, user.ID).Find(&shares)

	return c.JSON(fiber.Map{"shares": shares})
}

// RevokeShare handles DELETE /api/sync/share/:shareId
func (h *SyncHandler) RevokeShare(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	shareID := c.Params("shareId")

	var share models.TeamShare
	if err := h.db.Where("id = ? AND user_id = ?", shareID, user.ID).First(&share).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "share not found"})
	}

	h.db.Delete(&share)

	return c.JSON(fiber.Map{"ok": true})
}

// TeamPull handles POST /api/team/pull — Flutter alias for pulling pending team updates.
// Downloads all sync records authored by team members since the device's last pull.
func (h *SyncHandler) TeamPull(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		DeviceID string `json:"device_id"`
		Since    string `json:"since"` // RFC3339; optional
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	limit := 500
	if body.Limit > 0 {
		limit = body.Limit
	}

	var since time.Time
	if body.Since != "" {
		var err error
		since, err = time.Parse(time.RFC3339, body.Since)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid since format, use RFC3339"})
		}
	}

	teamIDs := userTeamIDs(h.db, user.ID)
	if len(teamIDs) == 0 {
		return c.JSON(fiber.Map{"records": []interface{}{}, "count": 0})
	}

	query := h.db.Model(&models.SyncLog{}).
		Where("team_id IN ? AND user_id != ?", teamIDs, user.ID).
		Order("created_at asc").
		Limit(limit)

	if !since.IsZero() {
		query = query.Where("created_at > ?", since)
	}
	if body.DeviceID != "" {
		query = query.Where("device_id != ?", body.DeviceID)
	}

	var records []models.SyncLog
	if err := query.Find(&records).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to pull team records"})
	}

	return c.JSON(fiber.Map{
		"records": records,
		"count":   len(records),
	})
}

// Delta handles GET /api/sync/delta — returns changed entity IDs since a given timestamp.
// Query params: since (RFC3339, required), entity_type (optional filter), limit (default 1000).
func (h *SyncHandler) Delta(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	sinceStr := c.Query("since")
	if sinceStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "since is required (RFC3339)"})
	}
	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid since format, use RFC3339"})
	}

	entityType := c.Query("entity_type")
	limit := 1000
	if l, err2 := strconv.Atoi(c.Query("limit")); err2 == nil && l > 0 {
		limit = l
	}

	teamIDs := userTeamIDs(h.db, user.ID)

	type deltaRow struct {
		EntityType string    `json:"entity_type"`
		EntityID   string    `json:"entity_id"`
		Action     string    `json:"action"`
		Version    int64     `json:"version"`
		ChangedAt  time.Time `json:"changed_at"`
	}

	query := h.db.Model(&models.SyncLog{}).
		Select("entity_type, entity_id, action, version, created_at as changed_at").
		Where("created_at > ?", since).
		Order("created_at asc").
		Limit(limit)

	if len(teamIDs) > 0 {
		query = query.Where("user_id = ? OR team_id IN ?", user.ID, teamIDs)
	} else {
		query = query.Where("user_id = ?", user.ID)
	}

	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}

	var rows []deltaRow
	if err := query.Scan(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch delta"})
	}

	return c.JSON(fiber.Map{
		"changes": rows,
		"count":   len(rows),
		"since":   since.UTC().Format(time.RFC3339),
	})
}

// Conflicts handles GET /api/sync/conflicts
// Returns all conflict log entries for the authenticated user, newest first.
// Query params: entity_type (optional filter), limit (default 100), resolution (optional: pending|server_wins|client_wins).
func (h *SyncHandler) Conflicts(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	query := h.db.Model(&models.ConflictLog{}).
		Where("user_id = ?", user.ID).
		Order("created_at desc").
		Limit(limit)

	if et := c.Query("entity_type"); et != "" {
		query = query.Where("entity_type = ?", et)
	}
	if res := c.Query("resolution"); res != "" {
		query = query.Where("resolution = ?", res)
	}

	var conflicts []models.ConflictLog
	if err := query.Find(&conflicts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch conflicts"})
	}

	return c.JSON(fiber.Map{
		"conflicts": conflicts,
		"count":     len(conflicts),
	})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
