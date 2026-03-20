package hub

// Event represents a real-time event sent to WebSocket clients.
type Event struct {
	Type       string `json:"type"`        // "sync", "presence", "notification", "file_change"
	EntityType string `json:"entity_type"` // "project", "feature", "note", "notification", "file"
	EntityID   string `json:"entity_id"`
	Action     string `json:"action"` // "upsert", "delete", "online", "offline", "created", "modified", "deleted"
	UserID     uint   `json:"user_id"`
	Timestamp  int64  `json:"timestamp"`

	// Notification-specific fields (only set when Type == "notification")
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`
	NType   string `json:"ntype,omitempty"` // info, success, warning, error

	// File change fields (only set when Type == "file_change")
	FilePath  string `json:"file_path,omitempty"`
	TunnelID  string `json:"tunnel_id,omitempty"`
	Workspace string `json:"workspace,omitempty"`

	// MCP hook event fields (only set when Type == "mcp")
	ToolName  string `json:"tool_name,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	AgentType string `json:"agent_type,omitempty"`
}
