package models

import "time"

// UserWallet stores the current point balance for a user.
// One row per user (unique constraint on user_id).
type UserWallet struct {
	Base
	UserID          uint      `gorm:"not null;uniqueIndex:idx_wallets_user,where:deleted_at IS NULL" json:"user_id"`
	Balance         int       `gorm:"not null;default:0"               json:"balance"`
	LifetimeEarned  int       `gorm:"not null;default:0"               json:"lifetime_earned"`
	LastTransactionAt *time.Time `gorm:"type:timestamptz"              json:"last_transaction_at,omitempty"`
}

func (UserWallet) TableName() string { return "user_wallets" }

// PointsTransaction records a single points change event.
type PointsTransaction struct {
	Base
	UserID      uint   `gorm:"not null;index"                    json:"user_id"`
	Amount      int    `gorm:"not null"                          json:"amount"`
	Type        string `gorm:"type:varchar(20);not null"         json:"type"` // earn, spend, admin_grant, admin_deduct
	Source      string `gorm:"type:varchar(100);not null;default:'system'" json:"source"`
	ReferenceID string `gorm:"type:varchar(255)"                 json:"reference_id,omitempty"`
	Description string `gorm:"type:text;not null;default:''"     json:"description"`
	BalanceAfter int   `gorm:"not null;default:0"                json:"balance_after"`
}

func (PointsTransaction) TableName() string { return "points_transactions" }
