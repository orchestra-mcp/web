package routes

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/handlers"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/repositories"
	"github.com/orchestra-mcp/web/internal/services"
	"gorm.io/gorm"
)

// Register wires all route handlers onto the Fiber app.
func Register(app *fiber.App, db *gorm.DB, cfg *config.Config, wsMgr ...*services.WorkspaceManager) {
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

	app.Use(middleware.Logger())
	app.Use(middleware.CORS(cfg.AllowedOrigins))

	// Serve uploaded files (avatars, covers, etc.) using absolute path from config.
	app.Get("/uploads*", static.New(cfg.UploadDir, static.Config{
		CacheDuration: 24 * time.Hour,
		MaxAge:        86400,
	}))

	// Create and start WebSocket hub.
	wsHub := hub.NewHub()
	go wsHub.Run()

	authSvc := services.NewAuthService(db, cfg)

	authHandler := handlers.NewAuthHandler(db, cfg)
	passkeyHandler := handlers.NewPasskeyHandler(db, cfg)
	oauthHandler := handlers.NewOAuthHandler(db, cfg)
	projectHandler := handlers.NewProjectHandler(db, wsHub)
	featureHandler := handlers.NewFeatureHandler(db, wsHub)
	epicHandler := handlers.NewEpicHandler(db)
	storyHandler := handlers.NewStoryHandler(db)
	taskHandler := handlers.NewTaskHandler(db)
	noteHandler := handlers.NewNoteHandler(db, wsHub)
	aiHandler := handlers.NewAiSessionHandler(db)
	syncHandler := handlers.NewSyncHandler(db, wsHub)
	hookEventHandler := handlers.NewHookEventHandler(db, wsHub)
	teamHandler := handlers.NewTeamHandler(db, cfg)
	adminHandler := handlers.NewAdminHandler(db)
	settingsHandler := handlers.NewSettingsHandler(db, cfg)
	adminCmsHandler := handlers.NewAdminCmsHandlerWithAuth(db, authSvc, wsHub)
	adminSettingsHandler := handlers.NewAdminSettingsHandler(db)
	tunnelHub := handlers.NewTunnelHub()
	tunnelHandler := handlers.NewTunnelHandler(db, tunnelHub)
	tunnelProxyHandler := handlers.NewTunnelProxyHandler(db, cfg, tunnelHub)
	tunnelReverseHandler := handlers.NewTunnelReverseHandler(db, tunnelHub)
	claudeCodeOAuthHandler := handlers.NewClaudeCodeOAuthHandler(db, cfg)
	teamScopedHandler := handlers.NewTeamScopedHandler(db)
	userIntegrationsHandler := handlers.NewUserIntegrationsHandler(db)
	searchHandler := handlers.NewSearchHandler(db)
	commentHandler := handlers.NewCommentHandler(db)
	docHandler := handlers.NewDocHandler(db, wsHub)
	workspaceHandler := handlers.NewWorkspaceHandler(db)
	skillHandler := handlers.NewSkillHandler(db, wsHub)
	agentHandler := handlers.NewAgentHandler(db, wsHub)
	planHandler := handlers.NewPlanHandler(db, wsHub)
	delegationHandler := handlers.NewDelegationHandler(db, wsHub)
	requestHandler := handlers.NewRequestHandler(db)
	personHandler := handlers.NewPersonHandler(db, wsHub)
	dashboardHandler := handlers.NewDashboardHandler(db)
	communityHandler := handlers.NewCommunityHandler(db, wsHub)
	sharingHandler := handlers.NewSharingHandler(db)
	workflowHandler := handlers.NewWorkflowHandler(db, wsHub)
	terminalHandler := handlers.NewTerminalHandler()
	projectIncludeHandler := handlers.NewProjectIncludeHandler(db)
	presenceHandler := handlers.NewPresenceHandler(db, wsHub)
	wsHandler := handlers.NewWebSocketHandler(wsHub, db, cfg)
	var workspaceManager *services.WorkspaceManager
	if len(wsMgr) > 0 && wsMgr[0] != nil {
		workspaceManager = wsMgr[0]
	} else {
		workspaceManager = services.NewWorkspaceManager(db, cfg.RepoBaseDir)
	}
	repoHandler := handlers.NewRepoWorkspaceHandler(db, workspaceManager)
	powersyncHandler := handlers.NewPowerSyncHandler(db)
	ogPreviewHandler := handlers.NewOgPreviewHandler()
	adminMarketplaceHandler := handlers.NewAdminMarketplaceHandler(db)
	adminBadgeHandler := handlers.NewAdminBadgeHandler(db)
	adminVerificationHandler := handlers.NewAdminVerificationHandler(db)
	adminGamificationHandler := handlers.NewAdminUserGamificationHandler(db, wsHub)
	adminContentHandler := handlers.NewAdminContentHandler(db)
	apiCollectionHandler := handlers.NewApiCollectionHandler(db)
	presentationHandler := handlers.NewPresentationHandler(db)
	contentAnalyticsHandler := handlers.NewContentAnalyticsHandler(db)
	teamContentHandler := handlers.NewTeamContentHandler(db)
	customDomainHandler := handlers.NewCustomDomainHandler(db)
	mcpHandler := handlers.NewMCPHandler(db)

	api := app.Group("/api")

	// MCP service-to-service endpoints (called by cloud-mcp, no user session needed — auth via user_id param).
	api.Get("/mcp/profile", mcpHandler.GetProfile)
	api.Patch("/mcp/profile", mcpHandler.PatchProfile)

	// Public settings (no auth — used by Next.js middleware for coming soon check).
	// Uses /public/settings/ prefix to avoid collision with /settings/preferences etc.
	api.Get("/public/settings/:key", adminSettingsHandler.GetPublicSetting)

	// Public system docs (from repo docs/ folder, no auth required).
	api.Get("/docs", docHandler.SystemList)
	api.Get("/docs/:id", docHandler.SystemShow)

	// Public skills/agents/notes (read-only, no auth — scope=public).
	api.Get("/skills/public/:slug", skillHandler.PublicShow)
	api.Get("/agents/public/:slug", agentHandler.PublicShow)
	api.Get("/notes/public/:slug", noteHandler.PublicShow)

	// Public project docs (published, no auth required).
	api.Get("/public/docs/:team/:project", docHandler.PublicDocList)
	api.Get("/public/docs/:team/:project/:slug", docHandler.PublicDocShow)

	// Public project view (no auth — read-only project health + kanban).
	api.Get("/public/projects/:user/:slug", projectHandler.PublicShow)

	// Public community (no auth)
	community := api.Group("/public/community")
	community.Get("/members", communityHandler.ListMembers)
	community.Get("/members/:handle", communityHandler.MemberProfile)
	community.Get("/members/:handle/posts", communityHandler.MemberPosts)
	community.Get("/members/:handle/activity", communityHandler.MemberActivity)
	community.Get("/posts/:id", communityHandler.ShowPost)
	community.Get("/posts/:id/comments", communityHandler.PostComments)
	community.Get("/posts/:id/related", communityHandler.RelatedPosts)
	community.Get("/shares/:handle", sharingHandler.ListPublicShares)
	community.Get("/shares/:handle/:type/:slug", sharingHandler.PublicShare)
	community.Get("/shares/:id/comments", sharingHandler.ListShareComments)
	community.Post("/shares/:id/view", contentAnalyticsHandler.RecordView)

	// Public API collections (no auth)
	api.Get("/public/api-collections/:handle", apiCollectionHandler.PublicList)
	api.Get("/public/api-collections/:handle/:slug", apiCollectionHandler.PublicShow)

	// Public presentations (no auth)
	api.Get("/public/presentations/:handle", presentationHandler.PublicList)
	api.Get("/public/presentations/:handle/:slug", presentationHandler.PublicShow)

	// Public sponsors & issues (no auth)
	api.Get("/public/sponsors", adminCmsHandler.PublicSponsors)
	api.Get("/public/issues", adminCmsHandler.PublicIssues)

	// Public Open Graph preview (no auth — fetches OG metadata for link cards).
	api.Get("/og-preview", ogPreviewHandler.Preview)

	// Public search (no auth — searches published posts, docs, profiles).
	api.Get("/search/public", searchHandler.PublicSearch)

	// Blog comments (list is public, create requires auth — handled in protected group).
	api.Get("/blog/:slug/comments", commentHandler.List)

	// PowerSync JWKS endpoint (public — PowerSync service calls this to verify client JWTs).
	api.Get("/powersync/keys", powersyncHandler.Keys)

	// WebSocket routes (before auth middleware — auth done via token query param).
	api.Get("/ws", wsHandler.Handle)
	api.Get("/tunnels/:id/ws", tunnelProxyHandler.Handle)
	api.Get("/tunnels/reverse", tunnelReverseHandler.Handle)

	// Tunnel claim (no JWT auth — nonce is the secret).
	api.Post("/tunnels/claim", tunnelHandler.Claim)

	// Public auth routes
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/register", authHandler.Register)
	auth.Post("/otp/send", authHandler.SendOTP)
	auth.Post("/otp/verify", authHandler.VerifyOTP)
	auth.Post("/verify-otp", authHandler.VerifyOTP) // alias for frontend compat
	auth.Post("/magic-link/send", authHandler.SendMagicLink)
	auth.Post("/magic-link", authHandler.SendMagicLink) // alias for frontend compat
	auth.Post("/magic-link/verify", authHandler.VerifyMagicLink)
	auth.Post("/reset-password", authHandler.ResetPassword)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/api-key-exchange", authHandler.APIKeyExchange)
	auth.Post("/device/request", authHandler.DeviceRequest)
	auth.Post("/device/poll", authHandler.DevicePoll)
	auth.Post("/passkey/authenticate/begin", passkeyHandler.BeginAuthentication)
	auth.Post("/passkey/authenticate/finish", passkeyHandler.FinishAuthentication)
	auth.Post("/2fa/verify", authHandler.Verify2FA)

	// OAuth routes (public — initiates redirect and handles callback)
	auth.Get("/oauth/:provider", oauthHandler.Redirect)
	auth.Get("/oauth/:provider/callback", oauthHandler.Callback)
	// OAuth connect (reads JWT from cookie — browser redirects don't send Authorization header)
	auth.Get("/oauth/:provider/connect", oauthHandler.Redirect)

	// Authenticated routes
	protected := api.Group("", middleware.Auth(db, cfg))

	// PowerSync token endpoint (authenticated — issues PowerSync JWT for sync connection).
	protected.Post("/powersync/token", powersyncHandler.Token)

	// PowerSync CRUD upload — handles all local writes from PowerSync clients.
	powersyncCrudHandler := handlers.NewPowerSyncCrudHandler(db)
	protected.Post("/powersync/crud", powersyncCrudHandler.Upload)

	// Blog comments (create requires auth).
	protected.Post("/blog/:slug/comments", commentHandler.Create)

	// System docs (auth required for editing)
	protected.Post("/docs", docHandler.SystemCreate)
	protected.Put("/docs/:id", docHandler.SystemUpdate)
	protected.Patch("/docs/:id", docHandler.SystemUpdate) // alias for frontend compat
	protected.Patch("/docs/:id/pin", docHandler.SystemPin)
	protected.Delete("/docs/:id", docHandler.SystemDelete)

	// Auth (protected)
	protected.Get("/auth/me", authHandler.Me)
	protected.Get("/auth/me/role", authHandler.MyRole)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Patch("/auth/profile", authHandler.UpdateProfile)
	protected.Post("/auth/device/approve", authHandler.DeviceApprove)
	protected.Post("/auth/passkey/register/begin", passkeyHandler.BeginRegistration)
	protected.Post("/auth/passkey/register/finish", passkeyHandler.FinishRegistration)
	protected.Post("/auth/2fa/setup", authHandler.Setup2FA)
	protected.Get("/auth/2fa/setup", authHandler.Setup2FA) // alias for frontend compat
	protected.Post("/auth/2fa/confirm", authHandler.Confirm2FA)
	protected.Post("/auth/2fa/disable", authHandler.Disable2FA)
	protected.Patch("/auth/2fa/disable", authHandler.Disable2FA) // alias for frontend compat
	protected.Post("/auth/change-password", authHandler.ChangePassword)
	protected.Patch("/auth/change-password", authHandler.ChangePassword) // alias for frontend compat
	protected.Delete("/auth/account", authHandler.DeleteAccount)

	// Dashboard (batch endpoint)
	protected.Get("/dashboard", dashboardHandler.Dashboard)

	// My Tasks (features assigned to current user)
	protected.Get("/my-tasks", teamScopedHandler.MyTasks)

	// Persons
	protected.Get("/persons", personHandler.List)
	protected.Get("/persons/:id", personHandler.Show)

	// Requests (by ID — flat alias)
	protected.Get("/requests/:id", requestHandler.Show)

	// Delegations
	protected.Get("/delegations", delegationHandler.MyPending)
	protected.Get("/delegations/:id", delegationHandler.Show)
	protected.Post("/delegations/:id/respond", delegationHandler.Respond)

	// Community (authenticated)
	communityAuth := protected.Group("/community")
	communityAuth.Get("/posts", communityHandler.MyPosts)
	communityAuth.Post("/posts", communityHandler.CreatePost)
	communityAuth.Put("/posts/:id", communityHandler.UpdatePost)
	communityAuth.Delete("/posts/:id", communityHandler.DeletePost)
	communityAuth.Post("/posts/:id/comments", communityHandler.AddComment)
	communityAuth.Post("/posts/:id/like", communityHandler.ToggleLike)
	communityAuth.Post("/shares", sharingHandler.CreateShare)
	communityAuth.Get("/shares", sharingHandler.ListMyShares)
	communityAuth.Put("/shares/:id", sharingHandler.UpdateShare)
	communityAuth.Delete("/shares/:id", sharingHandler.DeleteShare)
	communityAuth.Post("/shares/:id/clone", sharingHandler.CloneShare)
	communityAuth.Post("/shares/:id/comments", sharingHandler.AddShareComment)
	communityAuth.Get("/shares/:id/export", sharingHandler.ExportShare)
	communityAuth.Get("/shares/:id/analytics", contentAnalyticsHandler.GetAnalytics)
	communityAuth.Get("/shares/:id/inline-comments", teamContentHandler.InlineComments)
	communityAuth.Post("/shares/:id/inline-comments", teamContentHandler.AddInlineComment)
	communityAuth.Get("/teams/:teamId/content", teamContentHandler.ListTeamContent)
	communityAuth.Post("/teams/:teamId/content/:id/share", teamContentHandler.ShareWithTeam)
	communityAuth.Delete("/teams/:teamId/content/:id/share", teamContentHandler.UnshareFromTeam)
	communityAuth.Get("/teams/:teamId/activity", teamContentHandler.TeamActivity)

	// API Collections
	apiCols := protected.Group("/api-collections")
	apiCols.Get("/", apiCollectionHandler.List)
	apiCols.Post("/", apiCollectionHandler.Create)
	apiCols.Get("/:id", apiCollectionHandler.Show)
	apiCols.Put("/:id", apiCollectionHandler.Update)
	apiCols.Delete("/:id", apiCollectionHandler.Delete)
	apiCols.Post("/:id/endpoints", apiCollectionHandler.CreateEndpoint)
	apiCols.Put("/:id/endpoints/:epId", apiCollectionHandler.UpdateEndpoint)
	apiCols.Delete("/:id/endpoints/:epId", apiCollectionHandler.DeleteEndpoint)
	apiCols.Post("/:id/environments", apiCollectionHandler.CreateEnvironment)
	apiCols.Put("/:id/environments/:envId", apiCollectionHandler.UpdateEnvironment)
	apiCols.Delete("/:id/environments/:envId", apiCollectionHandler.DeleteEnvironment)

	// Presentations
	pres := protected.Group("/presentations")
	pres.Get("/", presentationHandler.List)
	pres.Post("/", presentationHandler.Create)
	pres.Get("/:id", presentationHandler.Show)
	pres.Put("/:id", presentationHandler.Update)
	pres.Delete("/:id", presentationHandler.Delete)
	pres.Post("/:id/slides", presentationHandler.CreateSlide)
	pres.Put("/:id/slides/reorder", presentationHandler.ReorderSlides)
	pres.Put("/:id/slides/:slideId", presentationHandler.UpdateSlide)
	pres.Delete("/:id/slides/:slideId", presentationHandler.DeleteSlide)
	pres.Get("/:id/export", presentationHandler.Export)

	// Projects
	projects := protected.Group("/projects")
	projects.Get("/", projectHandler.List)
	projects.Post("/", projectHandler.Create)
	projects.Get("/:slug", projectHandler.Show)
	projects.Put("/:slug", projectHandler.Update)
	projects.Patch("/:slug", projectHandler.Update)
	projects.Delete("/:slug", projectHandler.Delete)
	projects.Post("/:slug/share", projectHandler.Share)
	projects.Delete("/:slug/share", projectHandler.Unshare)
	projects.Get("/:slug/tree", projectHandler.Tree)
	projects.Get("/:slug/stats", projectHandler.Stats)

	// Epics (nested under project)
	projects.Get("/:slug/epics", epicHandler.List)
	projects.Post("/:slug/epics", epicHandler.Create)
	projects.Get("/:slug/epics/:epicId", epicHandler.Show)
	projects.Put("/:slug/epics/:epicId", epicHandler.Update)
	projects.Delete("/:slug/epics/:epicId", epicHandler.Delete)

	// Plans (project-scoped)
	projects.Get("/:slug/plans", planHandler.List)
	projects.Get("/:slug/plans/:planId", planHandler.Show)

	// Features (flat, nested under project by slug — used by MCP sync)
	projects.Get("/:slug/features", featureHandler.List)
	projects.Get("/:slug/features/:id", featureHandler.Show)
	projects.Put("/:slug/features/:id", featureHandler.Update)
	projects.Delete("/:slug/features/:id", featureHandler.Delete)

	// Requests (project-scoped)
	projects.Get("/:slug/requests", requestHandler.List)

	// Delegations (project-scoped)
	projects.Get("/:slug/delegations", delegationHandler.ProjectList)

	// Docs (wiki pages, synced from MCP docs plugin)
	projects.Get("/:slug/docs", docHandler.List)
	projects.Get("/:slug/docs/:id", docHandler.Show)

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
	notes.Patch("/:id", noteHandler.Update) // alias for frontend compat
	notes.Delete("/:id", noteHandler.Delete)

	// Workspaces
	workspaces := protected.Group("/workspaces")
	workspaces.Get("/", workspaceHandler.List)
	workspaces.Post("/", workspaceHandler.Create)
	workspaces.Post("/sync", workspaceHandler.Sync)
	workspaces.Get("/:id", workspaceHandler.Show)
	workspaces.Put("/:id", workspaceHandler.Update)
	workspaces.Delete("/:id", workspaceHandler.Delete)
	workspaces.Post("/:id/teams", workspaceHandler.AddTeam)
	workspaces.Delete("/:id/teams/:teamId", workspaceHandler.RemoveTeam)
	workspaces.Post("/:id/projects", teamScopedHandler.CreateWorkspaceProject)

	// Skills (team + personal + public)
	skills := protected.Group("/skills")
	skills.Get("/", skillHandler.List)
	skills.Post("/", skillHandler.Create)
	skills.Get("/:id", skillHandler.Show)
	skills.Put("/:id", skillHandler.Update)
	skills.Delete("/:id", skillHandler.Delete)

	// Agents (team + personal + public)
	agents := protected.Group("/agents")
	agents.Get("/", agentHandler.List)
	agents.Post("/", agentHandler.Create)
	agents.Get("/:id", agentHandler.Show)
	agents.Put("/:id", agentHandler.Update)
	agents.Delete("/:id", agentHandler.Delete)

	// Project ↔ Skill/Agent inclusions
	projects.Get("/:slug/skills", projectIncludeHandler.ListProjectSkills)
	projects.Post("/:slug/skills", projectIncludeHandler.IncludeSkill)
	projects.Delete("/:slug/skills/:id", projectIncludeHandler.ExcludeSkill)
	projects.Get("/:slug/agents", projectIncludeHandler.ListProjectAgents)
	projects.Post("/:slug/agents", projectIncludeHandler.IncludeAgent)
	projects.Delete("/:slug/agents/:id", projectIncludeHandler.ExcludeAgent)
	projects.Post("/:slug/generate-docs", projectIncludeHandler.GenerateDocs)

	// Tunnels
	smartActionHandler := handlers.NewSmartActionHandler(db, tunnelHub)
	fileEventHandler := handlers.NewFileEventHandler(db, wsHub)
	tunnels := protected.Group("/tunnels")
	tunnels.Get("/", tunnelHandler.List)
	tunnels.Post("/register", tunnelHandler.Register)
	tunnels.Post("/auto-register", tunnelHandler.AutoRegister)
	tunnels.Post("/heartbeat", tunnelHandler.Heartbeat)
	tunnels.Get("/:id", tunnelHandler.Show)
	tunnels.Put("/:id", tunnelHandler.Update)
	tunnels.Delete("/:id", tunnelHandler.Delete)
	tunnels.Get("/:id/status", tunnelHandler.Status)
	tunnels.Get("/:id/actions", smartActionHandler.SupportedActions)
	tunnels.Post("/:id/actions", smartActionHandler.Execute)
	tunnels.Post("/:id/action", smartActionHandler.Dispatch)
	tunnels.Get("/:id/actions/history", smartActionHandler.History)
	tunnels.Get("/:id/action-log", smartActionHandler.ActionLog)
	tunnels.Post("/:id/file-events", fileEventHandler.Report)

	// Action history (across all tunnels)
	protected.Get("/actions/history", smartActionHandler.AllHistory)

	// Repos (workspace management)
	repos := protected.Group("/repos")
	repos.Get("/github", repoHandler.GitHubRepos)
	repos.Get("/", repoHandler.List)
	repos.Post("/", repoHandler.Create)
	repos.Get("/:id", repoHandler.Show)
	repos.Post("/:id/sync", repoHandler.Sync)
	repos.Post("/:id/chat", repoHandler.Chat)
	repos.Delete("/:id", repoHandler.Delete)

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
	sync.Get("/export", syncHandler.Export)
	sync.Get("/status", syncHandler.Status)
	sync.Post("/share", syncHandler.ShareEntity)
	sync.Get("/team-updates", syncHandler.TeamUpdates)
	sync.Get("/share/:entityType/:entityId", syncHandler.GetEntityShares)
	sync.Delete("/share/:shareId", syncHandler.RevokeShare)
	sync.Get("/delta", syncHandler.Delta)
	sync.Get("/conflicts", syncHandler.Conflicts)

	// Hook events (MCP → WebSocket bridge)
	hooks := protected.Group("/hooks")
	hooks.Post("/events", hookEventHandler.Receive)
	hooks.Get("/events", hookEventHandler.List)

	// Flutter sync aliases (TeamSyncService expected paths)
	protected.Get("/team/updates", syncHandler.TeamUpdates)
	protected.Post("/team/pull", syncHandler.TeamPull)
	protected.Post("/:entityType/:entityId/share", syncHandler.ShareEntity)

	// Team (current user's team — matches Next.js frontend endpoints)
	protected.Get("/team", teamHandler.MyTeam)
	protected.Patch("/team", teamHandler.UpdateMyTeam)
	protected.Post("/team/avatar", teamHandler.UploadTeamAvatar)
	protected.Get("/team/members", teamHandler.MyTeamMembers)
	protected.Get("/team/members/:id", teamHandler.ShowMember)
	protected.Patch("/team/members/:id/role", teamHandler.UpdateMemberRole)
	protected.Delete("/team/members/:id", teamHandler.RemoveMember)

	// Teams (multi-team API)
	teams := protected.Group("/teams")
	teams.Get("/", teamHandler.List)
	teams.Post("/", teamHandler.Create)
	teams.Get("/:id", teamHandler.Show)
	teams.Delete("/:id", teamHandler.Delete)
	teams.Post("/:id/invite", teamHandler.Invite)

	// Team-scoped endpoints (used by the dashboard frontend)
	teams.Get("/:id/workspaces", teamScopedHandler.TeamWorkspaces)
	teams.Post("/:id/workspaces", teamScopedHandler.CreateTeamWorkspace)
	teams.Get("/:id/projects", teamScopedHandler.TeamProjects)
	teams.Get("/:id/features", teamScopedHandler.TeamFeatures)
	teams.Get("/:id/analytics", teamScopedHandler.TeamAnalytics)
	teams.Get("/:id/activity", teamScopedHandler.TeamActivity)
	teams.Get("/:id/notes", teamScopedHandler.TeamNotes)
	teams.Get("/:id/skills", teamScopedHandler.TeamSkills)
	teams.Get("/:id/agents", teamScopedHandler.TeamAgents)
	teams.Get("/:id/members", teamScopedHandler.TeamMembers)
	teams.Get("/:id/presence", presenceHandler.TeamPresence)
	teams.Get("/:id/workflows", workflowHandler.List)

	// Workspace-scoped projects
	workspaces.Get("/:id/projects", teamScopedHandler.WorkspaceProjects)

	// Feature CRUD by ID (used by frontend stores — flat aliases for project-scoped routes)
	protected.Get("/features/:id", teamScopedHandler.ShowFeature)
	protected.Patch("/features/:id", teamScopedHandler.UpdateFeature)
	protected.Put("/features/:id", teamScopedHandler.UpdateFeature)
	protected.Delete("/features/:id", teamScopedHandler.DeleteFeature)

	// Workspace PATCH (frontend uses PATCH, existing handler only has PUT)
	workspaces.Patch("/:id", teamScopedHandler.UpdateWorkspace)

	// Workflows
	wf := protected.Group("/workflows")
	wf.Get("/", workflowHandler.List)
	wf.Post("/", workflowHandler.Create)
	wf.Get("/:id", workflowHandler.Show)
	wf.Patch("/:id", workflowHandler.Update)
	wf.Delete("/:id", workflowHandler.Delete)

	// Terminal WebSocket (stub — proxied through tunnels)
	protected.Get("/terminal/ws", terminalHandler.WS)

	// Claude Code OAuth
	claudeCode := protected.Group("/oauth/claude-code")
	claudeCode.Get("/start", claudeCodeOAuthHandler.Start)
	claudeCode.Post("/exchange", claudeCodeOAuthHandler.Exchange)

	// Global search
	protected.Get("/search", searchHandler.Search)
	protected.Get("/search/suggestions", searchHandler.Suggestions)
	protected.Post("/search/ai", searchHandler.AiSearch)

	// Health Debug API
	healthRepo := repositories.NewHealthRepository(db)
	healthSvc := services.NewHealthService(healthRepo)
	healthHandler := handlers.NewHealthHandler(healthSvc)
	handlers.RegisterHealthRoutes(protected, healthHandler)

	// Wallet & Points
	walletRepo := repositories.NewWalletRepository(db)
	walletSvc := services.NewWalletService(walletRepo)
	walletHandler := handlers.NewWalletHandler(walletSvc)
	protected.Get("/wallet", walletHandler.GetWallet)
	protected.Get("/wallet/transactions", walletHandler.ListTransactions)

	// Notifications (user)
	protected.Get("/notifications", settingsHandler.ListNotifications)
	protected.Patch("/notifications/read-all", settingsHandler.MarkAllNotificationsRead)
	protected.Patch("/notifications/:id/read", settingsHandler.MarkNotificationRead)
	protected.Delete("/notifications/:id", settingsHandler.DeleteNotification)
	protected.Post("/notifications/push/subscribe", settingsHandler.RegisterPushSubscription)
	protected.Delete("/notifications/push/unsubscribe", settingsHandler.UnregisterPushSubscription)
	protected.Get("/notifications/push/subscriptions", settingsHandler.ListPushSubscriptions)

	// Issues (user)
	protected.Post("/issues", settingsHandler.CreateIssue)
	protected.Get("/issues", settingsHandler.ListMyIssues)

	// Settings
	settingsGroup := protected.Group("/settings")
	settingsGroup.Patch("/profile", settingsHandler.UpdateProfile)
	settingsGroup.Post("/avatar", settingsHandler.UploadAvatar)
	settingsGroup.Post("/cover", settingsHandler.UploadCover)
	settingsGroup.Get("/sessions", settingsHandler.ListSessions)
	settingsGroup.Delete("/sessions/:id", settingsHandler.RevokeSession)
	settingsGroup.Get("/api-keys", settingsHandler.ListApiKeys)
	settingsGroup.Post("/api-keys", settingsHandler.CreateApiKey)
	settingsGroup.Delete("/api-keys/:id", settingsHandler.RevokeApiKey)
	settingsGroup.Get("/connected-accounts", settingsHandler.ListConnectedAccounts)
	settingsGroup.Delete("/connected-accounts/:provider", settingsHandler.UnlinkAccount)
	settingsGroup.Get("/passkeys", passkeyHandler.ListPasskeys)
	settingsGroup.Patch("/passkeys/:id", passkeyHandler.RenamePasskey)
	settingsGroup.Delete("/passkeys/:id", passkeyHandler.DeletePasskey)
	settingsGroup.Get("/preferences", settingsHandler.GetPreferences)
	settingsGroup.Patch("/preferences", settingsHandler.UpdatePreferences)
	settingsGroup.Get("/mcp-permissions", mcpHandler.GetPermissions)
	settingsGroup.Patch("/mcp-permissions", mcpHandler.PatchPermissions)
	settingsGroup.Get("/mcp-token", mcpHandler.GetMCPToken)
	settingsGroup.Post("/mcp-token/regenerate", mcpHandler.RegenerateMCPToken)
	settingsGroup.Get("/integrations/user", userIntegrationsHandler.List)
	settingsGroup.Put("/integrations/user/:provider", userIntegrationsHandler.Upsert)
	settingsGroup.Delete("/integrations/user/:provider", userIntegrationsHandler.Delete)
	settingsGroup.Get("/integrations/apps", userIntegrationsHandler.AppInstallURLs)

	// Custom domains
	settingsGroup.Get("/custom-domains", customDomainHandler.List)
	settingsGroup.Post("/custom-domains", customDomainHandler.Add)
	settingsGroup.Post("/custom-domains/:id/verify", customDomainHandler.Verify)
	settingsGroup.Delete("/custom-domains/:id", customDomainHandler.Delete)

	// Admin base
	admin := protected.Group("/admin")
	admin.Get("/teams", teamHandler.ListAll)
	admin.Post("/teams", teamHandler.Create) // admin can also create teams
	admin.Get("/teams/:id", teamHandler.AdminShowTeam)
	admin.Patch("/teams/:id", teamHandler.AdminUpdateTeam)
	admin.Delete("/teams/:id", teamHandler.AdminDeleteTeam)
	admin.Post("/teams/:id/members", teamHandler.AdminAddMember)
	admin.Delete("/teams/:id/members/:user_id", teamHandler.AdminRemoveMember)
	admin.Patch("/teams/:id/members/:user_id", teamHandler.AdminUpdateMemberRole)
	admin.Get("/users", adminHandler.ListUsers)
	admin.Patch("/users/:id", adminHandler.UpdateUser)
	admin.Patch("/users/:id/role", adminHandler.UpdateUserRole)
	admin.Post("/users/:id/role", adminHandler.UpdateUserRole) // alias for frontend compat
	admin.Patch("/users/:id/suspend", adminHandler.SuspendUser)
	admin.Post("/users/:id/suspend", adminHandler.SuspendUser) // alias for frontend compat
	admin.Patch("/users/:id/unsuspend", adminHandler.UnsuspendUser)
	admin.Post("/users/:id/unsuspend", adminHandler.UnsuspendUser) // alias for frontend compat
	admin.Patch("/users/:id/verify", adminHandler.VerifyUser)
	admin.Patch("/users/:id/unverify", adminHandler.UnverifyUser)

	// Admin extended user management
	admin.Get("/users/:id", adminCmsHandler.GetUser)
	admin.Get("/users/:id/projects", adminCmsHandler.UserProjects)
	admin.Get("/users/:id/notes", adminCmsHandler.UserNotes)
	admin.Get("/users/:id/sessions", adminCmsHandler.UserSessions)
	admin.Get("/users/:id/teams", adminCmsHandler.UserTeams)
	admin.Get("/users/:id/issues", adminCmsHandler.UserIssues)
	admin.Get("/users/:id/memberships", teamHandler.AdminUserTeams)
	admin.Delete("/users/:id/memberships/:team_id", teamHandler.AdminRemoveUserFromTeam)
	admin.Get("/users/:id/otp", adminCmsHandler.GetLastOTP)
	admin.Post("/users/:id/password", adminCmsHandler.ForceResetPassword)
	admin.Post("/users/:id/impersonate", adminCmsHandler.Impersonate)
	admin.Post("/users/:id/notify", adminCmsHandler.NotifyUser)
	admin.Post("/users/:id/subscription", adminCmsHandler.UpsertSubscription)

	// Admin user gamification (badges + points)
	admin.Get("/users/:id/badges", adminGamificationHandler.ListUserBadges)
	admin.Post("/users/:id/badges", adminGamificationHandler.AwardBadge)
	admin.Delete("/users/:id/badges/:badge_id", adminGamificationHandler.RevokeBadge)
	admin.Get("/users/:id/points", adminGamificationHandler.GetPoints)
	admin.Post("/users/:id/points", adminGamificationHandler.AddPoints)

	// Admin badge definitions CRUD
	admin.Get("/badges", adminBadgeHandler.List)
	admin.Post("/badges", adminBadgeHandler.Create)
	admin.Put("/badges/:id", adminBadgeHandler.Update)
	admin.Delete("/badges/:id", adminBadgeHandler.Delete)

	// Admin verification types CRUD
	admin.Get("/verifications", adminVerificationHandler.List)
	admin.Post("/verifications", adminVerificationHandler.Create)
	admin.Put("/verifications/:id", adminVerificationHandler.Update)
	admin.Delete("/verifications/:id", adminVerificationHandler.Delete)

	// Admin wallet (grant/deduct points)
	admin.Post("/wallet/:userId/grant", walletHandler.AdminGrantPoints)
	admin.Post("/wallet/:userId/deduct", walletHandler.AdminDeductPoints)

	// Admin marketplace approval
	admin.Get("/marketplace/pending", adminMarketplaceHandler.ListPending)
	admin.Post("/marketplace/:id/approve", adminMarketplaceHandler.Approve)
	admin.Post("/marketplace/:id/reject", adminMarketplaceHandler.Reject)

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
	admin.Post("/notifications/seed", adminCmsHandler.SeedNotifications)
	admin.Get("/notifications", adminCmsHandler.ListNotificationsSent)

	// Admin sponsors
	admin.Get("/sponsors", adminCmsHandler.ListSponsors)
	admin.Post("/sponsors", adminCmsHandler.CreateSponsor)
	admin.Put("/sponsors/:id", adminCmsHandler.UpdateSponsor)
	admin.Delete("/sponsors/:id", adminCmsHandler.DeleteSponsor)

	// Admin community posts
	admin.Get("/community/posts", adminCmsHandler.ListCommunityPosts)
	admin.Patch("/community/posts/:id", adminCmsHandler.UpdateCommunityPost)
	admin.Delete("/community/posts/:id", adminCmsHandler.DeleteCommunityPost)

	// Admin GitHub issues
	admin.Get("/github/repos", adminCmsHandler.ListGitHubRepos)
	admin.Get("/github/issues", adminCmsHandler.ListGitHubIssues)
	admin.Post("/github/sync", adminCmsHandler.SyncGitHubIssues)

	// Admin system settings
	admin.Get("/settings/:key", adminSettingsHandler.GetSetting)
	admin.Patch("/settings/:key", adminSettingsHandler.UpdateSetting)
	admin.Post("/settings/seed", adminSettingsHandler.SeedSettings)
	admin.Post("/settings/test-email", adminSettingsHandler.TestEmail)
	admin.Post("/settings/generate-sitemap", adminSettingsHandler.GenerateSitemap)

	// Content management
	admin.Get("/content", adminContentHandler.ListContent)
	admin.Patch("/content/:id/visibility", adminContentHandler.UpdateVisibility)
	admin.Delete("/content/:id", adminContentHandler.DeleteContent)
	admin.Post("/content/bulk", adminContentHandler.BulkAction)

	// ── OAuth2 Provider (Orchestra as authorization server) ───────────
	oauthProviderHandler := handlers.NewOAuthProviderHandler(db, cfg)
	handlers.RegisterOAuthProviderRoutes(app, oauthProviderHandler, middleware.Auth(db, cfg))
	handlers.RegisterOAuthProviderAdminRoutes(admin, oauthProviderHandler)
	handlers.RegisterOAuthProviderUserRoutes(protected, oauthProviderHandler)
}

