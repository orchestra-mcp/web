package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"gorm.io/gorm"
)

// PowerSyncCrudHandler handles batch CRUD uploads from PowerSync clients.
// This replaces per-table API mapping — all local writes go through one endpoint.
type PowerSyncCrudHandler struct {
	db *gorm.DB
}

func NewPowerSyncCrudHandler(db *gorm.DB) *PowerSyncCrudHandler {
	return &PowerSyncCrudHandler{db: db}
}

// CrudOp represents a single CRUD operation from PowerSync.
type CrudOp struct {
	Table string                 `json:"table"`
	Op    string                 `json:"op"`    // "PUT", "PATCH", "DELETE"
	ID    string                 `json:"id"`
	Data  map[string]interface{} `json:"data"`
}

// CrudBatch is the request body for the batch endpoint.
type CrudBatch struct {
	Operations []CrudOp `json:"operations"`
}

// Tables that PowerSync clients are allowed to write to.
var allowedTables = map[string]bool{
	"water_logs":        true,
	"caffeine_logs":     true,
	"meal_logs":         true,
	"pomodoro_sessions": true,
	"sleep_configs":     true,
	"health_snapshots":  true,
	"health_profiles":   true,
	"notes":             true,
	"projects":          true,
	"features":          true,
	"plans":             true,
	"requests":          true,
	"persons":           true,
	"agents":            true,
	"skills":            true,
	"workflows":         true,
	"docs":              true,
	"delegations":       true,
	"sessions":          true,
	"user_settings":     true,
	"sleep_logs":        true,
	"workspaces":        true,
	"teams":             true,
	"memberships":          true,
	"api_collections":      true,
	"api_endpoints":        true,
	"api_environments":     true,
	"presentations":        true,
	"presentation_slides":  true,
}

// Upload handles POST /api/powersync/crud
// Accepts a batch of CRUD operations and applies them directly to PostgreSQL.
func (h *PowerSyncCrudHandler) Upload(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var batch CrudBatch
	if err := json.Unmarshal(c.Body(), &batch); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	results := make([]map[string]interface{}, 0, len(batch.Operations))

	// Process each operation independently (no wrapping transaction).
	// A single failure must NOT poison the entire batch — PostgreSQL transactions
	// mark themselves as aborted on the first error, causing "commit unexpectedly
	// resulted in rollback" for subsequent operations.
	for _, op := range batch.Operations {
		if !allowedTables[op.Table] {
			results = append(results, map[string]interface{}{
				"table": op.Table, "id": op.ID, "status": "skipped", "reason": "table not allowed",
			})
			continue
		}

		var err error
		switch op.Op {
		case "PUT":
			err = h.handlePut(h.db, user.ID, op)
		case "PATCH":
			err = h.handlePatch(h.db, user.ID, op)
		case "DELETE":
			err = h.handleDelete(h.db, user.ID, op)
		default:
			results = append(results, map[string]interface{}{
				"table": op.Table, "id": op.ID, "status": "skipped", "reason": "unknown op",
			})
			continue
		}

		if err != nil {
			results = append(results, map[string]interface{}{
				"table": op.Table, "id": op.ID, "status": "error", "error": err.Error(),
			})
		} else {
			results = append(results, map[string]interface{}{
				"table": op.Table, "id": op.ID, "status": "ok",
			})
		}
	}

	return c.JSON(fiber.Map{"results": results})
}

func (h *PowerSyncCrudHandler) handlePut(tx *gorm.DB, userID uint, op CrudOp) error {
	data := cleanData(op.Data)
	data["id"] = op.ID
	data["user_id"] = userID

	cols := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data))
	placeholders := make([]string, 0, len(data))
	updateCols := make([]string, 0, len(data))

	for k, v := range data {
		cols = append(cols, quote(k))
		vals = append(vals, v)
		placeholders = append(placeholders, "?")
		if k != "id" {
			updateCols = append(updateCols, fmt.Sprintf("%s = EXCLUDED.%s", quote(k), quote(k)))
		}
	}

	// UPSERT: INSERT ... ON CONFLICT (id) DO UPDATE
	// Use DO UPDATE SET ... to handle both id conflicts and unique constraint conflicts.
	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (id) DO UPDATE SET %s",
		quote(op.Table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateCols, ", "),
	)

	// If the initial upsert fails (e.g. unique constraint on non-id columns),
	// fall back to a plain UPDATE.
	if err := tx.Exec(sql, vals...).Error; err != nil {
		updateSql := fmt.Sprintf(
			"UPDATE %s SET %s WHERE id = ?",
			quote(op.Table),
			strings.Join(updateCols, ", "),
		)
		updateVals := make([]interface{}, 0, len(data))
		for k, v := range data {
			if k != "id" {
				updateVals = append(updateVals, v)
			}
		}
		updateVals = append(updateVals, op.ID)
		return tx.Exec(updateSql, updateVals...).Error
	}
	return nil
}

func (h *PowerSyncCrudHandler) handlePatch(tx *gorm.DB, userID uint, op CrudOp) error {
	data := cleanData(op.Data)
	if len(data) == 0 {
		return nil
	}

	sets := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data)+2)

	for k, v := range data {
		sets = append(sets, fmt.Sprintf("%s = ?", quote(k)))
		vals = append(vals, v)
	}

	vals = append(vals, op.ID, userID)

	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = ? AND user_id = ?",
		quote(op.Table),
		strings.Join(sets, ", "),
	)

	return tx.Exec(sql, vals...).Error
}

func (h *PowerSyncCrudHandler) handleDelete(tx *gorm.DB, userID uint, op CrudOp) error {
	// Soft-delete if the table has deleted_at column, otherwise hard-delete.
	sql := fmt.Sprintf(
		"UPDATE %s SET deleted_at = NOW() WHERE id = ? AND user_id = ?",
		quote(op.Table),
	)
	result := tx.Exec(sql, op.ID, userID)

	if result.RowsAffected == 0 {
		// Table might not have deleted_at or user_id — try hard delete.
		sql = fmt.Sprintf("DELETE FROM %s WHERE id = ?", quote(op.Table))
		return tx.Exec(sql, op.ID).Error
	}

	return result.Error
}

// cleanData removes internal PowerSync fields that shouldn't be written to PostgreSQL.
func cleanData(data map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{}, len(data))
	for k, v := range data {
		// Skip PowerSync internal fields.
		if k == "user_id" || k == "deleted_at" {
			continue
		}
		clean[k] = v
	}
	return clean
}

// quote wraps a column/table name in double quotes for PostgreSQL.
func quote(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
