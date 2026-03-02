package database

import (
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a PostgreSQL connection using GORM.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Warn
	if cfg.Env == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// AutoMigrate runs GORM auto-migration for all models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Team{},
		&models.Membership{},
		&models.Project{},
		&models.Epic{},
		&models.Story{},
		&models.Task{},
		&models.Feature{},
		&models.Note{},
		&models.NoteRevision{},
		&models.AiSession{},
		&models.Subscription{},
		&models.SyncLog{},
		&models.DeviceToken{},
		&models.OtpCode{},
		&models.MagicLinkToken{},
		&models.OAuthAccount{},
		&models.ConflictLog{},
		&models.SystemSetting{},
		&models.Page{},
		&models.Post{},
		&models.Issue{},
		&models.Notification{},
		&models.ContactMessage{},
	)
}
