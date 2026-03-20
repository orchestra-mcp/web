package handlers

import (
	"fmt"
	"time"

	"github.com/orchestra-mcp/web/internal/hub"
)

// broadcastSync is a convenience helper to broadcast a sync event to a user.
func broadcastSync(h *hub.Hub, userID uint, entityType string, entityID interface{}, action string) {
	if h == nil {
		return
	}
	h.BroadcastToUser(userID, hub.Event{
		Type:       "sync",
		EntityType: entityType,
		EntityID:   fmt.Sprintf("%v", entityID),
		Action:     action,
		UserID:     userID,
		Timestamp:  time.Now().UnixMilli(),
	})
}
