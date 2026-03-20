package main

import (
	"fmt"
	"log"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dsn := "postgres://user:pass@localhost:5432/orchestra?sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	allModels := []interface{}{
		&models.SystemSetting{},
		&models.Page{},
		&models.Post{},
		&models.Issue{},
		&models.Notification{},
		&models.ContactMessage{},
		&models.Tunnel{},
		&models.Request{},
		&models.AssignmentRule{},
		&models.SessionTurn{},
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
		&models.ProjectSkill{},
		&models.ProjectAgent{},
		&models.ActionHistory{},
		&models.CommunityLike{},
		&models.HealthProfile{},
		&models.WaterLog{},
		&models.MealLog{},
		&models.CaffeineLog{},
		&models.PomodoroSession{},
		&models.SleepConfig{},
		&models.HealthSnapshot{},
	}
	for _, m := range allModels {
		if err := db.AutoMigrate(m); err != nil {
			fmt.Printf("ERROR %-40T %v\n", m, err)
		} else {
			fmt.Printf("OK    %T\n", m)
		}
	}
}
