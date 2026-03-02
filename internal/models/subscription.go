package models

import (
	"time"

	"gorm.io/datatypes"
)

// Subscription tracks billing plan for a user or team.
type Subscription struct {
	Base
	UserID             *uint          `json:"user_id"`
	TeamID             *string        `gorm:"type:uuid" json:"team_id"`
	Plan               string         `gorm:"default:free" json:"plan"`
	Status             string         `gorm:"default:active" json:"status"` // active | canceled | past_due | trialing
	StripeCustomerID   string         `json:"stripe_customer_id"`
	StripeSubscriptionID string       `json:"stripe_subscription_id"`
	CurrentPeriodStart *time.Time     `json:"current_period_start"`
	CurrentPeriodEnd   *time.Time     `json:"current_period_end"`
	CanceledAt         *time.Time     `json:"canceled_at"`
	Meta               datatypes.JSON `json:"meta"`
}
