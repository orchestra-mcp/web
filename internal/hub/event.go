package hub

// Event represents a real-time event sent to WebSocket clients.
type Event struct {
	Type       string `json:"type"`        // "sync", "presence", "notification"
	EntityType string `json:"entity_type"` // "project", "feature", "note", "notification"
	EntityID   string `json:"entity_id"`
	Action     string `json:"action"` // "upsert", "delete"
	UserID     uint   `json:"user_id"`
	Timestamp  int64  `json:"timestamp"`

	// Notification-specific fields (only set when Type == "notification")
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`
	NType   string `json:"ntype,omitempty"` // info, success, warning, error
}
