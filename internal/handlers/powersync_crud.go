package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"gorm.io/gorm"
)

// columnType holds the PostgreSQL type name and whether it is an array.
type columnType struct {
	pgType  string // e.g. "bool", "int8", "float4", "text", "_text"
	isArray bool
}

// PowerSyncCrudHandler handles batch CRUD uploads from PowerSync clients.
// This replaces per-table API mapping — all local writes go through one endpoint.
type PowerSyncCrudHandler struct {
	db         *gorm.DB
	schemaMu   sync.RWMutex
	schemaCache map[string]map[string]columnType // table → column → type
}

func NewPowerSyncCrudHandler(db *gorm.DB) *PowerSyncCrudHandler {
	return &PowerSyncCrudHandler{
		db:          db,
		schemaCache: make(map[string]map[string]columnType),
	}
}

// tableSchema returns the column type map for a table, fetching from PostgreSQL if needed.
func (h *PowerSyncCrudHandler) tableSchema(table string) map[string]columnType {
	h.schemaMu.RLock()
	m, ok := h.schemaCache[table]
	h.schemaMu.RUnlock()
	if ok {
		return m
	}

	type row struct {
		Column  string `gorm:"column:column_name"`
		PgType  string `gorm:"column:udt_name"`
	}
	var rows []row
	h.db.Raw(`
		SELECT column_name, udt_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ?`, table).Scan(&rows)

	m = make(map[string]columnType, len(rows))
	for _, r := range rows {
		m[r.Column] = columnType{
			pgType:  r.PgType,
			isArray: strings.HasPrefix(r.PgType, "_"),
		}
	}

	h.schemaMu.Lock()
	h.schemaCache[table] = m
	h.schemaMu.Unlock()
	return m
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

	schema := h.tableSchema(op.Table)

	// Stable key ordering so cols/vals/placeholders/updateCols all align.
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	cols := make([]string, 0, len(keys))
	insertVals := make([]interface{}, 0, len(keys))
	placeholders := make([]string, 0, len(keys))
	updateCols := make([]string, 0, len(keys))
	fallbackSets := make([]string, 0, len(keys))
	fallbackVals := make([]interface{}, 0, len(keys))

	for _, k := range keys {
		v, ph := coerce(data[k], schema[k])
		cols = append(cols, quote(k))
		if v == nil {
			placeholders = append(placeholders, "NULL")
		} else {
			placeholders = append(placeholders, ph)
			insertVals = append(insertVals, v)
		}
		if k != "id" {
			updateCols = append(updateCols, fmt.Sprintf("%s = EXCLUDED.%s", quote(k), quote(k)))
			if v == nil {
				fallbackSets = append(fallbackSets, fmt.Sprintf("%s = NULL", quote(k)))
			} else {
				fallbackSets = append(fallbackSets, fmt.Sprintf("%s = %s", quote(k), ph))
				fallbackVals = append(fallbackVals, v)
			}
		}
	}

	// UPSERT: INSERT ... ON CONFLICT (id) DO UPDATE SET ... (EXCLUDED is valid here).
	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (id) DO UPDATE SET %s",
		quote(op.Table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updateCols, ", "),
	)

	// If the upsert fails (rare: unique constraint on non-id columns), fall back to plain UPDATE.
	if err := tx.Exec(sql, insertVals...).Error; err != nil {
		updateSql := fmt.Sprintf(
			"UPDATE %s SET %s WHERE id = ?",
			quote(op.Table),
			strings.Join(fallbackSets, ", "),
		)
		fallbackVals = append(fallbackVals, op.ID)
		return tx.Exec(updateSql, fallbackVals...).Error
	}
	return nil
}

func (h *PowerSyncCrudHandler) handlePatch(tx *gorm.DB, userID uint, op CrudOp) error {
	data := cleanData(op.Data)
	if len(data) == 0 {
		return nil
	}

	schema := h.tableSchema(op.Table)
	sets := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data)+2)

	for k, v := range data {
		p, ph := coerce(v, schema[k])
		if p == nil {
			sets = append(sets, fmt.Sprintf("%s = NULL", quote(k)))
		} else {
			sets = append(sets, fmt.Sprintf("%s = %s", quote(k), ph))
			vals = append(vals, p)
		}
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

// coerce converts a JSON value from Flutter SQLite into a Go value and the
// correct SQL placeholder expression for the target PostgreSQL column type.
//
// Returns (nil, "NULL") for null/empty values.
// Returns (value, "?::pgtype") for typed values — the explicit cast tells pgx
// which OID to use, avoiding binary-protocol mismatches (e.g. int8→bool, int8→float4).
//
// For array columns (udt_name starts with "_"): CSV strings like "a,b" are
// converted to PostgreSQL array literals {"a","b"}.
func coerce(v interface{}, ct columnType) (interface{}, string) {
	if v == nil {
		return nil, "NULL"
	}

	// Array columns: udt_name starts with "_" (e.g. "_text", "_int4").
	// elementType is the base type without the leading "_".
	if ct.isArray {
		s := fmt.Sprint(v)
		if s == "" {
			return nil, "NULL"
		}
		// Already in {} form (e.g. synced back from server).
		if strings.HasPrefix(s, "{") {
			return s, "?::" + ct.pgType
		}
		// Convert CSV "a,b" → {"a","b"}.
		parts := strings.Split(s, ",")
		quoted := make([]string, len(parts))
		for i, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.ReplaceAll(p, `"`, `\"`)
			quoted[i] = `"` + p + `"`
		}
		return "{" + strings.Join(quoted, ",") + "}", "?::" + ct.pgType
	}

	// Scalar columns — use the PostgreSQL udt_name for the cast.
	pgType := ct.pgType
	if pgType == "" {
		pgType = "text" // unknown column: let PostgreSQL decide
	}

	switch val := v.(type) {
	case float64:
		// Whole numbers: format as integer string so PostgreSQL can cast to bool, int, float.
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val)), "?::" + pgType
		}
		return fmt.Sprintf("%g", val), "?::" + pgType
	case bool:
		if val {
			return "true", "?::" + pgType
		}
		return "false", "?::" + pgType
	case string:
		if val == "" {
			return nil, "NULL"
		}
		return val, "?::" + pgType
	default:
		return fmt.Sprint(v), "?::" + pgType
	}
}
