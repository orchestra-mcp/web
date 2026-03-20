package database

import (
	"log"

	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a PostgreSQL connection using GORM.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Warn

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// AutoMigrate runs GORM auto-migration for all models.
// Each model is migrated individually so a single failure doesn't block the rest.
func AutoMigrate(db *gorm.DB) error {
	allModels := []interface{}{
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
		&models.Tunnel{},
		&models.Plan{},
		&models.Person{},
		&models.Request{},
		&models.AssignmentRule{},
		&models.SessionTurn{},
		&models.Doc{},
		&models.RepoWorkspace{},
		&models.Passkey{},
		&models.UserIntegration{},
		&models.Comment{},
		&models.Sponsor{},
		&models.CommunityPost{},
		&models.GitHubIssue{},
		&models.GitHubRepo{},
		&models.Workspace{},
		&models.WorkspaceTeam{},
		&models.Skill{},
		&models.Agent{},
		&models.ProjectSkill{},
		&models.ProjectAgent{},
		&models.ActionHistory{},
		&models.ActionLog{},
		&models.CommunityLike{},
		&models.HealthProfile{},
		&models.WaterLog{},
		&models.MealLog{},
		&models.CaffeineLog{},
		&models.PomodoroSession{},
		&models.SleepConfig{},
		&models.HealthSnapshot{},
		&models.Workflow{},
		&models.Delegation{},
		&models.MCPEventLog{},
		&models.SharedContent{},
		&models.ShareComment{},
		&models.PushSubscription{},
		&models.UserSetting{},
		&models.SleepLog{},
		&models.BadgeDefinition{},
		&models.UserBadge{},
		&models.VerificationType{},
		&models.UserVerification{},
		&models.ApiCollection{},
		&models.ApiEndpoint{},
		&models.ApiEnvironment{},
		&models.Presentation{},
		&models.PresentationSlide{},
		&models.UserWallet{},
		&models.PointsTransaction{},
		&models.TeamShare{},
		&models.ContentView{},
		&models.CustomDomain{},
	}
	for _, m := range allModels {
		if err := db.AutoMigrate(m); err != nil {
			log.Printf("migrate warning: %T: %v", m, err)
		}
	}
	return nil
}
