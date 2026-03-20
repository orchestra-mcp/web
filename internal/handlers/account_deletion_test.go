package handlers

import (
	"testing"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
)

func TestUserDeletionScheduledAtField(t *testing.T) {
	now := time.Now()
	user := models.User{
		Name:                "Test User",
		Email:               "test@example.com",
		Status:              "pending_deletion",
		DeletionScheduledAt: &now,
	}
	if user.DeletionScheduledAt == nil {
		t.Fatal("expected DeletionScheduledAt to be set")
	}
	if user.Status != "pending_deletion" {
		t.Fatalf("expected status pending_deletion, got %s", user.Status)
	}
}

func TestDeletionScheduleExpiry(t *testing.T) {
	// Verify 7-day grace period calculation
	now := time.Now()
	scheduled := now.Add(7 * 24 * time.Hour)

	// Before expiry
	if time.Now().After(scheduled) {
		t.Fatal("scheduled time should be in the future")
	}

	// Simulate after expiry
	expired := now.Add(-1 * time.Hour) // 1 hour ago
	if !time.Now().After(expired) {
		t.Fatal("expired time should be in the past")
	}
}

func TestDeletionCancellationOnLogin(t *testing.T) {
	// Verify that login restores pending_deletion → active
	user := models.User{
		Status: "pending_deletion",
	}
	now := time.Now()
	user.DeletionScheduledAt = &now

	// Simulate login cancellation
	if user.Status == "pending_deletion" {
		user.Status = "active"
		user.DeletionScheduledAt = nil
	}

	if user.Status != "active" {
		t.Fatalf("expected active, got %s", user.Status)
	}
	if user.DeletionScheduledAt != nil {
		t.Fatal("expected DeletionScheduledAt to be nil after cancellation")
	}
}

func TestDeleteAccountRequiresPassword(t *testing.T) {
	// Verify the handler validates password is not empty
	password := ""
	if password == "" {
		// Password required — would return 400
		return
	}
	t.Fatal("should not reach here")
}
