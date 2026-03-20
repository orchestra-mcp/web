package services

import (
	"context"
	"testing"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// mockWalletRepo is an in-memory mock of WalletRepository for unit testing.
type mockWalletRepo struct {
	wallet *models.UserWallet
	txns   []models.PointsTransaction
}

func (m *mockWalletRepo) GetWallet(_ context.Context, userID uint) (*models.UserWallet, error) {
	if m.wallet != nil && m.wallet.UserID == userID {
		return m.wallet, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockWalletRepo) UpsertWallet(_ context.Context, wallet *models.UserWallet) error {
	m.wallet = wallet
	return nil
}

func (m *mockWalletRepo) CreateTransaction(_ context.Context, tx *models.PointsTransaction) error {
	tx.ID = "tx-test"
	m.txns = append(m.txns, *tx)
	return nil
}

func (m *mockWalletRepo) ListTransactions(_ context.Context, userID uint, limit, offset int) ([]models.PointsTransaction, int64, error) {
	var result []models.PointsTransaction
	for _, tx := range m.txns {
		if tx.UserID == userID {
			result = append(result, tx)
		}
	}
	total := int64(len(result))
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	} else {
		result = nil
	}
	return result, total, nil
}

func TestAddPoints_Grant(t *testing.T) {
	repo := &mockWalletRepo{}
	svc := NewWalletService(repo)

	wallet, tx, err := svc.AddPoints(context.Background(), 1, 100, "admin_grant", "admin", "welcome bonus")
	if err != nil {
		t.Fatalf("AddPoints() error = %v", err)
	}
	if wallet.Balance != 100 {
		t.Errorf("Balance = %d, want 100", wallet.Balance)
	}
	if wallet.LifetimeEarned != 100 {
		t.Errorf("LifetimeEarned = %d, want 100", wallet.LifetimeEarned)
	}
	if tx.Amount != 100 {
		t.Errorf("Transaction.Amount = %d, want 100", tx.Amount)
	}
	if tx.Type != "admin_grant" {
		t.Errorf("Transaction.Type = %s, want admin_grant", tx.Type)
	}
	if tx.BalanceAfter != 100 {
		t.Errorf("Transaction.BalanceAfter = %d, want 100", tx.BalanceAfter)
	}
}

func TestAddPoints_Deduct(t *testing.T) {
	repo := &mockWalletRepo{
		wallet: &models.UserWallet{UserID: 1, Balance: 50, LifetimeEarned: 100},
	}
	svc := NewWalletService(repo)

	wallet, tx, err := svc.AddPoints(context.Background(), 1, -30, "admin_deduct", "admin", "penalty")
	if err != nil {
		t.Fatalf("AddPoints() error = %v", err)
	}
	if wallet.Balance != 20 {
		t.Errorf("Balance = %d, want 20", wallet.Balance)
	}
	if tx.Amount != -30 {
		t.Errorf("Transaction.Amount = %d, want -30", tx.Amount)
	}
	if tx.BalanceAfter != 20 {
		t.Errorf("Transaction.BalanceAfter = %d, want 20", tx.BalanceAfter)
	}
}

func TestAddPoints_FloorAtZero(t *testing.T) {
	repo := &mockWalletRepo{
		wallet: &models.UserWallet{UserID: 1, Balance: 10, LifetimeEarned: 10},
	}
	svc := NewWalletService(repo)

	wallet, _, err := svc.AddPoints(context.Background(), 1, -50, "spend", "system", "big spend")
	if err != nil {
		t.Fatalf("AddPoints() error = %v", err)
	}
	if wallet.Balance != 0 {
		t.Errorf("Balance = %d, want 0 (should floor at zero)", wallet.Balance)
	}
}

func TestAddPoints_ZeroAmountReturnsError(t *testing.T) {
	repo := &mockWalletRepo{}
	svc := NewWalletService(repo)

	_, _, err := svc.AddPoints(context.Background(), 1, 0, "earn", "system", "nothing")
	if err == nil {
		t.Error("AddPoints(0) should return an error")
	}
}

func TestAddPoints_InvalidTypeReturnsError(t *testing.T) {
	repo := &mockWalletRepo{}
	svc := NewWalletService(repo)

	_, _, err := svc.AddPoints(context.Background(), 1, 10, "invalid_type", "system", "test")
	if err == nil {
		t.Error("AddPoints with invalid type should return an error")
	}
}

func TestGetWallet_DefaultsForNewUser(t *testing.T) {
	repo := &mockWalletRepo{}
	svc := NewWalletService(repo)

	wallet, err := svc.GetWallet(context.Background(), 99)
	if err != nil {
		t.Fatalf("GetWallet() error = %v", err)
	}
	if wallet.Balance != 0 {
		t.Errorf("Default balance = %d, want 0", wallet.Balance)
	}
	if wallet.UserID != 99 {
		t.Errorf("UserID = %d, want 99", wallet.UserID)
	}
}

func TestGetTransactions_Pagination(t *testing.T) {
	repo := &mockWalletRepo{}
	svc := NewWalletService(repo)

	// Add 3 transactions
	for i := 0; i < 3; i++ {
		svc.AddPoints(context.Background(), 1, 10, "earn", "system", "test")
	}

	txns, total, err := svc.GetTransactions(context.Background(), 1, 2, 0)
	if err != nil {
		t.Fatalf("GetTransactions() error = %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(txns) != 2 {
		t.Errorf("len(txns) = %d, want 2 (limit=2)", len(txns))
	}
}
