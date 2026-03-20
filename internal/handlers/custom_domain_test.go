package handlers

import (
	"strings"
	"testing"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCustomDomainTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.CustomDomain{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestCustomDomainModel(t *testing.T) {
	db := setupCustomDomainTestDB(t)

	cd := models.CustomDomain{
		UserID:       1,
		Domain:       "example.com",
		DNSTxtRecord: "orchestra-verify=abc123",
	}

	if err := db.Create(&cd).Error; err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	if cd.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
	if cd.Verified {
		t.Error("expected verified to be false by default")
	}
}

func TestCustomDomainUniqueDomain(t *testing.T) {
	db := setupCustomDomainTestDB(t)

	cd1 := models.CustomDomain{UserID: 1, Domain: "example.com", DNSTxtRecord: "token1"}
	cd2 := models.CustomDomain{UserID: 2, Domain: "example.com", DNSTxtRecord: "token2"}

	if err := db.Create(&cd1).Error; err != nil {
		t.Fatalf("failed to create first domain: %v", err)
	}

	err := db.Create(&cd2).Error
	if err == nil {
		t.Fatal("expected error for duplicate domain, got nil")
	}
	if !strings.Contains(err.Error(), "UNIQUE") {
		t.Errorf("expected UNIQUE constraint error, got: %v", err)
	}
}

func TestCustomDomainListByUser(t *testing.T) {
	db := setupCustomDomainTestDB(t)

	db.Create(&models.CustomDomain{UserID: 1, Domain: "a.com", DNSTxtRecord: "t1"})
	db.Create(&models.CustomDomain{UserID: 1, Domain: "b.com", DNSTxtRecord: "t2"})
	db.Create(&models.CustomDomain{UserID: 2, Domain: "c.com", DNSTxtRecord: "t3"})

	var domains []models.CustomDomain
	db.Where("user_id = ?", 1).Find(&domains)

	if len(domains) != 2 {
		t.Errorf("expected 2 domains for user 1, got %d", len(domains))
	}
}

func TestCustomDomainVerify(t *testing.T) {
	db := setupCustomDomainTestDB(t)

	cd := models.CustomDomain{UserID: 1, Domain: "test.com", DNSTxtRecord: "orchestra-verify=xyz"}
	db.Create(&cd)

	// Mark as verified
	db.Model(&cd).Updates(map[string]any{
		"verified": true,
	})

	var loaded models.CustomDomain
	db.First(&loaded, cd.ID)

	if !loaded.Verified {
		t.Error("expected domain to be verified after update")
	}
}

func TestCustomDomainDelete(t *testing.T) {
	db := setupCustomDomainTestDB(t)

	cd := models.CustomDomain{UserID: 1, Domain: "del.com", DNSTxtRecord: "token"}
	db.Create(&cd)

	result := db.Where("id = ? AND user_id = ?", cd.ID, 1).Delete(&models.CustomDomain{})
	if result.RowsAffected != 1 {
		t.Errorf("expected 1 row deleted, got %d", result.RowsAffected)
	}

	// Verify it's gone
	var count int64
	db.Model(&models.CustomDomain{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 domains after delete, got %d", count)
	}
}

func TestCustomDomainDeleteWrongUser(t *testing.T) {
	db := setupCustomDomainTestDB(t)

	cd := models.CustomDomain{UserID: 1, Domain: "mine.com", DNSTxtRecord: "token"}
	db.Create(&cd)

	// User 2 tries to delete user 1's domain
	result := db.Where("id = ? AND user_id = ?", cd.ID, 2).Delete(&models.CustomDomain{})
	if result.RowsAffected != 0 {
		t.Error("should not delete another user's domain")
	}

	// Original still exists
	var count int64
	db.Model(&models.CustomDomain{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 domain still present, got %d", count)
	}
}

func TestCustomDomainLookupVerified(t *testing.T) {
	db := setupCustomDomainTestDB(t)

	db.Create(&models.CustomDomain{UserID: 1, Domain: "verified.com", DNSTxtRecord: "t1", Verified: true})
	db.Create(&models.CustomDomain{UserID: 2, Domain: "pending.com", DNSTxtRecord: "t2", Verified: false})

	// Lookup like the middleware does
	var cd models.CustomDomain
	err := db.Where("domain = ? AND verified = ?", "verified.com", true).First(&cd).Error
	if err != nil {
		t.Fatalf("expected to find verified domain: %v", err)
	}
	if cd.UserID != 1 {
		t.Errorf("expected user_id 1, got %d", cd.UserID)
	}

	// Unverified domain should not be found
	err = db.Where("domain = ? AND verified = ?", "pending.com", true).First(&models.CustomDomain{}).Error
	if err == nil {
		t.Error("expected error for unverified domain lookup, got nil")
	}

	// Unknown domain should not be found
	err = db.Where("domain = ? AND verified = ?", "unknown.com", true).First(&models.CustomDomain{}).Error
	if err == nil {
		t.Error("expected error for unknown domain lookup, got nil")
	}
}

func TestNewCustomDomainHandler(t *testing.T) {
	db := setupCustomDomainTestDB(t)
	h := NewCustomDomainHandler(db)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.db != db {
		t.Error("handler db should match provided db")
	}
}
