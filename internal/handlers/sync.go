package handlers

import (
	"strconv"
	"time"

	"encoding/json"

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

		// Apply LWW upsert
		if err := h.syncService.Apply(record, user.ID, h.db); err != nil {
			results = append(results, result{EntityID: record.EntityID, Status: "error", Error: err.Error()})
			continue
		}

		// Broadcast sync event to user's WebSocket clients.
		if h.hub != nil {
			h.hub.BroadcastToUser(user.ID, hub.Event{
				Type:       "sync",
				EntityType: record.EntityType,
				EntityID:   record.EntityID,
				Action:     record.Action,
				UserID:     user.ID,
				Timestamp:  time.Now().Unix(),
			})
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

	query := h.db.Model(&models.SyncLog{}).
		Where("user_id = ? AND created_at > ?", user.ID, since).
		Limit(limit).
		Order("created_at asc")

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

// Status handles GET /api/sync/status
func (h *SyncHandler) Status(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	deviceID := c.Query("device_id")

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
		Where("user_id = ?", user.ID)
	if lastSyncAt != nil {
		query = query.Where("created_at > ?", lastSyncAt)
	}
	if deviceID != "" {
		query = query.Where("device_id != ?", deviceID)
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
