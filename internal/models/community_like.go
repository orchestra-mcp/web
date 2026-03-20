package models

import "time"

// CommunityLike tracks a user's like on a community post.
type CommunityLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_like_user_post" json:"user_id"`
	PostID    uint      `gorm:"uniqueIndex:idx_like_user_post" json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}
