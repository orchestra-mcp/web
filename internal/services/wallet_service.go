package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/repositories"
	"gorm.io/gorm"
)

// WalletService defines the business-logic contract for the wallet system.
type WalletService interface {
	GetWallet(ctx context.Context, userID uint) (*models.UserWallet, error)
	GetTransactions(ctx context.Context, userID uint, limit, offset int) ([]models.PointsTransaction, int64, error)
	AddPoints(ctx context.Context, userID uint, amount int, txType, source, description string) (*models.UserWallet, *models.PointsTransaction, error)
}

type walletService struct {
	repo repositories.WalletRepository
}

// NewWalletService returns a WalletService backed by the given repository.
func NewWalletService(repo repositories.WalletRepository) WalletService {
	return &walletService{repo: repo}
}

// GetWallet returns the wallet for the given user, creating a default if none exists.
func (s *walletService) GetWallet(ctx context.Context, userID uint) (*models.UserWallet, error) {
	w, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &models.UserWallet{UserID: userID, Balance: 0, LifetimeEarned: 0}, nil
		}
		return nil, err
	}
	return w, nil
}

// GetTransactions returns paginated transaction history.
func (s *walletService) GetTransactions(ctx context.Context, userID uint, limit, offset int) ([]models.PointsTransaction, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListTransactions(ctx, userID, limit, offset)
}

// AddPoints creates a transaction and updates the wallet balance.
func (s *walletService) AddPoints(ctx context.Context, userID uint, amount int, txType, source, description string) (*models.UserWallet, *models.PointsTransaction, error) {
	if amount == 0 {
		return nil, nil, fmt.Errorf("amount must be non-zero")
	}

	// Validate transaction type.
	switch txType {
	case "earn", "spend", "admin_grant", "admin_deduct":
		// valid
	default:
		return nil, nil, fmt.Errorf("invalid transaction type: %s", txType)
	}

	// Get or create wallet.
	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
		wallet = &models.UserWallet{UserID: userID, Balance: 0, LifetimeEarned: 0}
	}

	// Apply amount.
	wallet.Balance += amount
	if wallet.Balance < 0 {
		wallet.Balance = 0
	}
	if amount > 0 {
		wallet.LifetimeEarned += amount
	}
	now := time.Now().UTC()
	wallet.LastTransactionAt = &now

	// Persist wallet.
	if err := s.repo.UpsertWallet(ctx, wallet); err != nil {
		return nil, nil, fmt.Errorf("update wallet: %w", err)
	}

	// Create transaction record.
	tx := &models.PointsTransaction{
		UserID:       userID,
		Amount:       amount,
		Type:         txType,
		Source:       source,
		Description:  description,
		BalanceAfter: wallet.Balance,
	}
	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, nil, fmt.Errorf("create transaction: %w", err)
	}

	return wallet, tx, nil
}
