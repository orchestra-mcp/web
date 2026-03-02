package hub

// Event represents a real-time event sent to WebSocket clients.
type Event struct {
	Type       string `json:"type"`        // "sync", "presence"
	EntityType string `json:"entity_type"` // "project", "feature", "note"
	EntityID   string `json:"entity_id"`
	Action     string `json:"action"` // "upsert", "delete"
	UserID     uint   `json:"user_id"`
	Timestamp  int64  `json:"timestamp"`
}
