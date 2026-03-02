package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/handlers"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/services"
	"gorm.io/gorm"
)

// Register wires all route handlers onto the Fiber app.
func Register(app *fiber.App, db *gorm.DB, cfg *config.Config) {
	// Health check (unauthenticated, used by deploy scripts and monitoring).
	app.Get("/health", func(c fiber.Ctx) error {
		sqlDB, err := db.DB()
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  "database connection unavailable",
			})
		}
		if err := sqlDB.Ping(); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  "database ping failed",
			})
		}
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	app.Use(middleware.CORS())

	// Create and start WebSocket hub.
	wsHub := hub.NewHub()
	go wsHub.Run()

	authSvc := services.NewAuthService(db, cfg)

	authHandler := handlers.NewAuthHandler(db, cfg)
	projectHandler := handlers.NewProjectHandler(db)
	featureHandler := handlers.NewFeatureHandler(db)
	epicHandler := handlers.NewEpicHandler(db)
	storyHandler := handlers.NewStoryHandler(db)
	taskHandler := handlers.NewTaskHandler(db)
	noteHandler := handlers.NewNoteHandler(db)
	aiHandler := handlers.NewAiSessionHandler(db)
	syncHandler := handlers.NewSyncHandler(db, wsHub)
	teamHandler := handlers.NewTeamHandler(db)
	adminHandler := handlers.NewAdminHandler(db)
	settingsHandler := handlers.NewSettingsHandler(db)
	adminCmsHandler := handlers.NewAdminCmsHandlerWithAuth(db, authSvc)
	adminSettingsHandler := handlers.NewAdminSettingsHandler(db)
	wsHandler := handlers.NewWebSocketHandler(wsHub, db, cfg)

	api := app.Group("/api")

	// WebSocket route (before auth middleware — auth done via token query param).
	api.Get("/ws", wsHandler.Handle)

	// Public auth routes
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/register", authHandler.Register)
	auth.Post("/otp/send", authHandler.SendOTP)
	auth.Post("/otp/verify", authHandler.VerifyOTP)
	auth.Post("/magic-link/send", authHandler.SendMagicLink)
	auth.Post("/magic-link/verify", authHandler.VerifyMagicLink)
	auth.Post("/reset-password", authHandler.ResetPassword)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/api-key-exchange", authHandler.APIKeyExchange)
	auth.Post("/device/request", authHandler.DeviceRequest)
	auth.Post("/device/poll", authHandler.DevicePoll)

	// Authenticated routes
	protected := api.Group("", middleware.Auth(db, cfg))

	// Auth (protected)
	protected.Get("/auth/me", authHandler.Me)
	protected.Get("/auth/me/role", authHandler.MyRole)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Patch("/auth/profile", authHandler.UpdateProfile)
	protected.Post("/auth/device/approve", authHandler.DeviceApprove)

	// Projects
	projects := protected.Group("/projects")
	projects.Get("/", projectHandler.List)
	projects.Post("/", projectHandler.Create)
	projects.Get("/:slug", projectHandler.Show)
	projects.Put("/:slug", projectHandler.Update)
	projects.Delete("/:slug", projectHandler.Delete)
	projects.Get("/:slug/tree", projectHandler.Tree)

	// Epics (nested under project)
	projects.Get("/:slug/epics", epicHandler.List)
	projects.Post("/:slug/epics", epicHandler.Create)
	projects.Get("/:slug/epics/:epicId", epicHandler.Show)
	projects.Put("/:slug/epics/:epicId", epicHandler.Update)
	projects.Delete("/:slug/epics/:epicId", epicHandler.Delete)

	// Features (flat, nested under project by slug — used by MCP sync)
	projects.Get("/:slug/features", featureHandler.List)
	projects.Get("/:slug/features/:id", featureHandler.Show)
	projects.Put("/:slug/features/:id", featureHandler.Update)
	projects.Delete("/:slug/features/:id", featureHandler.Delete)

	// Stories (nested under epic)
	projects.Get("/:slug/epics/:epicId/stories", storyHandler.List)
	projects.Post("/:slug/epics/:epicId/stories", storyHandler.Create)
	projects.Get("/:slug/epics/:epicId/stories/:storyId", storyHandler.Show)
	projects.Put("/:slug/epics/:epicId/stories/:storyId", storyHandler.Update)
	projects.Delete("/:slug/epics/:epicId/stories/:storyId", storyHandler.Delete)

	// Tasks (nested under story)
	projects.Get("/:slug/epics/:epicId/stories/:storyId/tasks", taskHandler.List)
	projects.Post("/:slug/epics/:epicId/stories/:storyId/tasks", taskHandler.Create)
	projects.Get("/:slug/epics/:epicId/stories/:storyId/tasks/:taskId", taskHandler.Show)
	projects.Put("/:slug/epics/:epicId/stories/:storyId/tasks/:taskId", taskHandler.Update)
	projects.Delete("/:slug/epics/:epicId/stories/:storyId/tasks/:taskId", taskHandler.Delete)

	// Notes
	notes := protected.Group("/notes")
	notes.Get("/", noteHandler.List)
	notes.Post("/", noteHandler.Create)
	notes.Get("/:id", noteHandler.Show)
	notes.Put("/:id", noteHandler.Update)
	notes.Delete("/:id", noteHandler.Delete)

	// AI Sessions
	ai := protected.Group("/ai/sessions")
	ai.Get("/", aiHandler.List)
	ai.Post("/", aiHandler.Create)
	ai.Patch("/:id/rename", aiHandler.Rename)
	ai.Delete("/:id", aiHandler.Delete)

	// Sync
	sync := protected.Group("/sync")
	sync.Post("/devices/register", syncHandler.RegisterDevice)
	sync.Post("/push", syncHandler.Push)
	sync.Get("/pull", syncHandler.Pull)
	sync.Get("/status", syncHandler.Status)

	// Team (current user's team — matches Next.js frontend endpoints)
	protected.Get("/team", teamHandler.MyTeam)
	protected.Patch("/team", teamHandler.UpdateMyTeam)
	protected.Get("/team/members", teamHandler.MyTeamMembers)

	// Teams (multi-team API)
	teams := protected.Group("/teams")
	teams.Get("/", teamHandler.List)
	teams.Post("/", teamHandler.Create)
	teams.Get("/:id", teamHandler.Show)
	teams.Delete("/:id", teamHandler.Delete)
	teams.Post("/:id/invite", teamHandler.Invite)

	// Notifications (user)
	protected.Get("/notifications", settingsHandler.ListNotifications)
	protected.Patch("/notifications/:id/read", settingsHandler.MarkNotificationRead)

	// Settings
	settingsGroup := protected.Group("/settings")
	settingsGroup.Patch("/profile", settingsHandler.UpdateProfile)
	settingsGroup.Get("/sessions", settingsHandler.ListSessions)
	settingsGroup.Delete("/sessions/:id", settingsHandler.RevokeSession)
	settingsGroup.Get("/api-keys", settingsHandler.ListApiKeys)
	settingsGroup.Post("/api-keys", settingsHandler.CreateApiKey)
	settingsGroup.Delete("/api-keys/:id", settingsHandler.RevokeApiKey)
	settingsGroup.Get("/connected-accounts", settingsHandler.ListConnectedAccounts)
	settingsGroup.Delete("/connected-accounts/:provider", settingsHandler.UnlinkAccount)
	settingsGroup.Get("/preferences", settingsHandler.GetPreferences)
	settingsGroup.Patch("/preferences", settingsHandler.UpdatePreferences)

	// Admin base
	admin := protected.Group("/admin")
	admin.Get("/users", adminHandler.ListUsers)
	admin.Patch("/users/:id/role", adminHandler.UpdateUserRole)
	admin.Patch("/users/:id/suspend", adminHandler.SuspendUser)
	admin.Patch("/users/:id/unsuspend", adminHandler.UnsuspendUser)

	// Admin extended user management
	admin.Get("/users/:id", adminCmsHandler.GetUser)
	admin.Get("/users/:id/projects", adminCmsHandler.UserProjects)
	admin.Get("/users/:id/notes", adminCmsHandler.UserNotes)
	admin.Get("/users/:id/sessions", adminCmsHandler.UserSessions)
	admin.Get("/users/:id/teams", adminCmsHandler.UserTeams)
	admin.Get("/users/:id/issues", adminCmsHandler.UserIssues)
	admin.Get("/users/:id/otp", adminCmsHandler.GetLastOTP)
	admin.Post("/users/:id/password", adminCmsHandler.ForceResetPassword)
	admin.Post("/users/:id/impersonate", adminCmsHandler.Impersonate)
	admin.Post("/users/:id/notify", adminCmsHandler.NotifyUser)
	admin.Post("/users/:id/subscription", adminCmsHandler.UpsertSubscription)

	// Admin CMS
	admin.Get("/pages", adminCmsHandler.ListPages)
	admin.Post("/pages", adminCmsHandler.CreatePage)
	admin.Put("/pages/:id", adminCmsHandler.UpdatePage)
	admin.Delete("/pages/:id", adminCmsHandler.DeletePage)
	admin.Get("/posts", adminCmsHandler.ListPosts)
	admin.Post("/posts", adminCmsHandler.CreatePost)
	admin.Put("/posts/:id", adminCmsHandler.UpdatePost)
	admin.Delete("/posts/:id", adminCmsHandler.DeletePost)
	admin.Get("/marketplace", adminCmsHandler.ListMarketplace)
	admin.Get("/categories", adminCmsHandler.ListCategories)
	admin.Post("/categories", adminCmsHandler.CreateCategory)
	admin.Delete("/categories/:id", adminCmsHandler.DeleteCategory)
	admin.Get("/contact", adminCmsHandler.ListContact)
	admin.Delete("/contact/:id", adminCmsHandler.DeleteContact)
	admin.Get("/issues", adminCmsHandler.ListIssues)
	admin.Patch("/issues/:id", adminCmsHandler.UpdateIssue)
	admin.Post("/notifications/send", adminCmsHandler.SendNotification)
	admin.Get("/notifications", adminCmsHandler.ListNotificationsSent)

	// Admin system settings
	admin.Get("/settings/:key", adminSettingsHandler.GetSetting)
	admin.Patch("/settings/:key", adminSettingsHandler.UpdateSetting)
}

