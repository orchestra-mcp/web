package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// CheckAndAwardBadges checks if a user qualifies for any auto-award badges
// based on their current points. Awards any badges not yet earned and creates
// a notification for each new badge. Optionally broadcasts via WebSocket hub.
func CheckAndAwardBadges(db *gorm.DB, userID uint, currentPoints int, wsHub ...*hub.Hub) []models.BadgeDefinition {
	// Find all auto-award badges where points_required <= current points
	var eligibleBadges []models.BadgeDefinition
	db.Where("auto_award = ? AND points_required <= ? AND points_required > 0", true, currentPoints).
		Order("points_required asc").
		Find(&eligibleBadges)

	if len(eligibleBadges) == 0 {
		return nil
	}

	// Get user's existing badges
	var existingBadges []models.UserBadge
	db.Where("user_id = ?", userID).Find(&existingBadges)

	existingSet := make(map[string]bool, len(existingBadges))
	for _, ub := range existingBadges {
		existingSet[ub.BadgeDefinitionID] = true
	}

	// Award missing badges
	var awarded []models.BadgeDefinition
	for _, badge := range eligibleBadges {
		if existingSet[badge.ID] {
			continue // already has this badge
		}

		// Award the badge
		ub := models.UserBadge{
			UserID:            userID,
			BadgeDefinitionID: badge.ID,
			AwardedBy:         "", // empty = auto-awarded
			Note:              "Auto-awarded at " + time.Now().Format("2006-01-02") + " (" + badge.Name + ")",
		}
		if err := db.Create(&ub).Error; err != nil {
			log.Printf("badge auto-award: failed to award %s to user %d: %v", badge.Slug, userID, err)
			continue
		}

		// Create notification
		notification := models.Notification{
			UserID:  userID,
			Title:   "Badge Earned: " + badge.Name,
			Message: badge.Description,
			Type:    "badge_earned",
		}
		if err := db.Create(&notification).Error; err != nil {
			log.Printf("badge auto-award: failed to create notification for user %d: %v", userID, err)
		}

		// Push realtime notification via WebSocket
		if len(wsHub) > 0 && wsHub[0] != nil {
			wsHub[0].BroadcastToUser(userID, hub.Event{
				Type:       "notification",
				EntityType: "notification",
				EntityID:   fmt.Sprintf("%d", notification.ID),
				Action:     "upsert",
				UserID:     userID,
				Timestamp:  notification.CreatedAt.UnixMilli(),
				Title:      notification.Title,
				Message:    notification.Message,
				NType:      "badge_earned",
			})
		}

		awarded = append(awarded, badge)
		log.Printf("badge auto-award: awarded '%s' to user %d (points: %d, required: %d)", badge.Name, userID, currentPoints, badge.PointsRequired)
	}

	return awarded
}
