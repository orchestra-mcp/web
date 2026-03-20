package repositories

import (
	"context"
	"fmt"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// WalletRepository defines the data-access contract for wallet entities.
type WalletRepository interface {
	GetWallet(ctx context.Context, userID uint) (*models.UserWallet, error)
	UpsertWallet(ctx context.Context, wallet *models.UserWallet) error
	CreateTransaction(ctx context.Context, tx *models.PointsTransaction) error
	ListTransactions(ctx context.Context, userID uint, limit, offset int) ([]models.PointsTransaction, int64, error)
}

type walletRepository struct {
	db *gorm.DB
}

// NewWalletRepository returns a GORM-backed WalletRepository.
func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) GetWallet(ctx context.Context, userID uint) (*models.UserWallet, error) {
	var w models.UserWallet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&w).Error
	if err != nil {
		return nil, fmt.Errorf("wallet get: %w", err)
	}
	return &w, nil
}

func (r *walletRepository) UpsertWallet(ctx context.Context, wallet *models.UserWallet) error {
	var existing models.UserWallet
	err := r.db.WithContext(ctx).Where("user_id = ?", wallet.UserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if err := r.db.WithContext(ctx).Create(wallet).Error; err != nil {
			return fmt.Errorf("wallet create: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("wallet lookup: %w", err)
	}
	wallet.ID = existing.ID
	if err := r.db.WithContext(ctx).Save(wallet).Error; err != nil {
		return fmt.Errorf("wallet update: %w", err)
	}
	return nil
}

func (r *walletRepository) CreateTransaction(ctx context.Context, tx *models.PointsTransaction) error {
	if err := r.db.WithContext(ctx).Create(tx).Error; err != nil {
		return fmt.Errorf("transaction create: %w", err)
	}
	return nil
}

func (r *walletRepository) ListTransactions(ctx context.Context, userID uint, limit, offset int) ([]models.PointsTransaction, int64, error) {
	var txns []models.PointsTransaction
	var total int64

	r.db.WithContext(ctx).Model(&models.PointsTransaction{}).Where("user_id = ?", userID).Count(&total)

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&txns).Error
	if err != nil {
		return nil, 0, fmt.Errorf("transactions list: %w", err)
	}
	return txns, total, nil
}
