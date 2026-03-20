package handlers

import (
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/helpers"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdminSettingsHandler handles system settings key-value endpoints.
type AdminSettingsHandler struct {
	db *gorm.DB
}

// NewAdminSettingsHandler creates a new AdminSettingsHandler.
func NewAdminSettingsHandler(db *gorm.DB) *AdminSettingsHandler {
	return &AdminSettingsHandler{db: db}
}

// validKeys lists the allowed settings keys.
var validKeys = map[string]bool{
	"general":          true,
	"features":         true,
	"homepage":         true,
	"agents":           true,
	"contact":          true,
	"pricing":          true,
	"download":         true,
	"integrations":     true,
	"smtp":             true,
	"seo":              true,
	"discord":          true,
	"slack":            true,
	"coming_soon":      true,
	"marketplace":      true,
	"plugins":          true,
	"docs":             true,
	"blog":             true,
	"aimodels":         true,
	"social_platforms": true,
	"sponsors":         true,
	"community":        true,
	"github":           true,
	"smart_prompts":    true,
}

// publicKeys lists settings keys readable without authentication.
var publicKeys = map[string]bool{
	"coming_soon":  true,
	"marketplace":  true,
	"plugins":      true,
	"homepage":     true,
	"pricing":      true,
	"download":     true,
	"docs":         true,
	"blog":         true,
	"general":      true,
	"features":     true,
	"integrations": true,
}

// defaultSettings returns default values for each key.
func defaultSettings(key string) interface{} {
	defaults := map[string]interface{}{
		"general": map[string]interface{}{
			"site_name":        "Orchestra",
			"tagline":          "The AI-Agentic IDE",
			"url":              "https://orchestra-mcp.dev",
			"support_email":    "support@orchestra-mcp.dev",
			"maintenance_mode": false,
			"version":          "v1.0.4",
			"github_url":       "https://github.com/orchestra-mcp/framework",
			"discord_url":      "https://discord.gg/orchestra",
		},
		"features": map[string]interface{}{
			"rag":         true,
			"multi_agent": true,
			"marketplace": true,
			"quic_bridge": true,
			"web_gateway": true,
			"packs":       true,
			"projects":    true,
			"notes":       true,
			"plans":       true,
			"wiki":        true,
			"devtools":    true,
		},
		"homepage": map[string]interface{}{
			"hero_headline": "The AI-native IDE for every platform",
			"hero_subtext":  "290 MCP tools across 36 plugins. Single-process in-process routing. macOS, Windows, Linux, Mobile, and Extensions.",
			"hero_cta":      "Get started free",
			"stats": []map[string]interface{}{
				{"label": "MCP Tools", "value": "290"},
				{"label": "Plugins", "value": "36"},
				{"label": "Official Packs", "value": "17"},
				{"label": "Platforms", "value": "5"},
			},
			"terminal_lines": []map[string]interface{}{
				{"name": "orchestrator", "note": "in-process router", "color": "cyan"},
				{"name": "storage.markdown", "note": ".projects/", "color": "cyan"},
				{"name": "tools.features", "note": "70 tools", "color": "cyan"},
				{"name": "tools.marketplace", "note": "15 tools + 5 prompts", "color": "cyan"},
				{"name": "engine.rag", "note": "22 tools (Rust)", "color": "purple"},
				{"name": "bridge.claude", "note": "5 tools + streaming", "color": "purple"},
				{"name": "agent.orchestrator", "note": "20 tools", "color": "purple"},
				{"name": "transport.stdio", "note": "MCP server", "color": "green"},
			},
			"total_tools": "290",
		},
		"agents": map[string]interface{}{
			"headline": "Multi-agent orchestration built in",
			"subtext":  "Define agents, run workflows, compare providers. 20 tools for multi-model coordination.",
		},
		"contact": map[string]interface{}{
			"headline": "Get in touch",
			"email":    "hello@orchestra-mcp.dev",
			"hours":    "Mon–Fri, 9am–5pm PST",
		},
		"pricing": map[string]interface{}{
			"plans": []map[string]interface{}{
				{
					"name": "Free", "price": "0", "period": "forever", "highlighted": false,
					"features": []string{
						"All 290 MCP tools",
						"4 core plugins bundled",
						"1 GB RAG memory",
						"5 projects",
						"Claude Code, Cursor, VS Code",
						"Community support",
					},
				},
				{
					"name": "Pro", "price": "19", "period": "month", "highlighted": true,
					"features": []string{
						"Everything in Free",
						"All 36 plugins",
						"Unlimited projects",
						"50 GB RAG memory",
						"All 17 official packs",
						"Multi-model AI bridges",
						"Team collaboration",
						"Priority support",
					},
				},
				{
					"name": "Enterprise", "price": "custom", "period": "", "highlighted": false,
					"features": []string{
						"Everything in Pro",
						"Self-hosted deployment",
						"Custom plugins",
						"SSO / SAML",
						"SLA guarantee",
						"Dedicated support",
						"Audit logs",
					},
				},
			},
		},
		"download": map[string]interface{}{
			"version":      "1.0.4",
			"release_date": "2026-03-12",
			"install_script": "curl -fsSL https://orchestra-mcp.dev/install.sh | sh",
			"macos":   map[string]string{"url": "", "version": "1.0.4", "release_date": "2026-03-12", "arch": "Apple Silicon + Intel"},
			"windows": map[string]string{"url": "", "version": "1.0.4", "release_date": "2026-03-12", "arch": "x64"},
			"linux":   map[string]string{"url": "", "version": "1.0.4", "release_date": "2026-03-12", "arch": "x64 + arm64"},
		},
		"integrations": map[string]interface{}{
			"google_enabled":       false,
			"google_client_id":     "",
			"google_client_secret": "",
			"github_enabled":       false,
			"github_client_id":     "",
			"github_client_secret": "",
			"discord_enabled":      false,
			"discord_client_id":    "",
			"discord_client_secret":"",
			"slack_enabled":        false,
			"slack_client_id":      "",
			"slack_client_secret":  "",
		},
		"discord": map[string]interface{}{
			"enabled":        false,
			"bot_token":      "",
			"application_id": "",
			"client_id":      "",
			"client_secret":  "",
			"guild_id":       "",
			"channel_id":     "",
			"command_prefix": "!",
			"webhook_url":    "",
			"allowed_users":  []string{},
		},
		"slack": map[string]interface{}{
			"enabled":        false,
			"bot_token":      "",
			"app_token":      "",
			"signing_secret": "",
			"app_id":         "",
			"channel_id":     "",
			"command_prefix": "!",
			"webhook_url":    "",
			"allowed_users":  []string{},
			"team_id":        "",
		},
		"smtp": map[string]interface{}{
			"host":       "smtp.gmail.com",
			"port":       587,
			"username":   "",
			"password":   "",
			"from_name":  "Orchestra",
			"from_email": "noreply@orchestra-mcp.dev",
		},
		"seo": map[string]interface{}{
			"title_template":   "%s | Orchestra",
			"meta_description": "Orchestra MCP — the AI-native developer workspace. 290 tools, 36 plugins, 17 packs, 5 platforms.",
			"og_image_url":     "/og-image.png",
			"robots_txt":       "User-agent: *\nAllow: /",
			"sitemap_url":      "/sitemap.xml",
		},
		"coming_soon": map[string]interface{}{
			"enabled": false,
			"message": "We're putting the finishing touches on something amazing. Stay tuned!",
			"title":   "Coming Soon",
		},
		"marketplace": map[string]interface{}{
			"total_packs": 17,
			"packs": []map[string]interface{}{
				{"name": "go-backend", "display_name": "Go Backend", "desc": "Fiber v3, GORM, JWT, asynq, gocron, stripe-go, zerolog, go-mail, validator. Full Go API development patterns.", "skills": 12, "agents": 3, "hooks": 2, "color": "#00acd7", "tags": []string{"go", "api", "backend"}},
				{"name": "rust-engine", "display_name": "Rust Engine", "desc": "Tonic gRPC, Tree-sitter, Tantivy, rusqlite, tower-lsp, ropey, dashmap, ring. Rust plugin development.", "skills": 10, "agents": 2, "hooks": 1, "color": "#f74c00", "tags": []string{"rust", "grpc", "search"}},
				{"name": "typescript-react", "display_name": "TypeScript + React", "desc": "React, Zustand, React Query, Vite, shadcn/ui, Tailwind CSS v4, Monaco, xterm.js. Modern frontend patterns.", "skills": 14, "agents": 4, "hooks": 3, "color": "#3178c6", "tags": []string{"react", "typescript", "frontend"}},
				{"name": "database-sync", "display_name": "Database & Sync", "desc": "PostgreSQL, pgvector, SQLite, Redis, sync protocol, migrations, conflict resolution.", "skills": 8, "agents": 2, "hooks": 1, "color": "#336791", "tags": []string{"postgres", "sqlite", "redis", "sync"}},
				{"name": "ai-agentic", "display_name": "AI & Agentic", "desc": "Anthropic SDK, OpenAI, langchaingo, chromem-go, pgvector, RAG, vector search, embeddings, tool use.", "skills": 16, "agents": 6, "hooks": 2, "color": "#a900ff", "tags": []string{"ai", "llm", "rag", "agents"}},
				{"name": "chrome-extension", "display_name": "Chrome Extension", "desc": "Manifest V3, service worker, content scripts, side panel, Chrome APIs.", "skills": 7, "agents": 1, "hooks": 2, "color": "#4285f4", "tags": []string{"chrome", "extension", "browser"}},
				{"name": "macos-integration", "display_name": "macOS Integration", "desc": "CGo, Spotlight, Keychain, iCloud, Touch Bar, Notifications, file associations, URL schemes.", "skills": 9, "agents": 2, "hooks": 1, "color": "#888", "tags": []string{"macos", "swift", "cgo"}},
				{"name": "gcp-infrastructure", "display_name": "GCP Infrastructure", "desc": "Cloud Run, Cloud SQL, CDN, Cloud Build, Artifact Registry, Secret Manager, Docker, nginx, Sentry, PostHog.", "skills": 11, "agents": 3, "hooks": 4, "color": "#4285f4", "tags": []string{"gcp", "docker", "devops"}},
				{"name": "proto-grpc", "display_name": "Proto + gRPC", "desc": "Protobuf, Buf, tonic-build, Go/Rust code generation, service contracts.", "skills": 6, "agents": 1, "hooks": 1, "color": "#00e5ff", "tags": []string{"protobuf", "grpc", "codegen"}},
				{"name": "wails-desktop", "display_name": "Wails Desktop", "desc": "Wails v3, Go-React bindings, system tray, window management, native desktop features.", "skills": 8, "agents": 2, "hooks": 1, "color": "#22c55e", "tags": []string{"wails", "desktop", "go"}},
				{"name": "react-native-mobile", "display_name": "React Native Mobile", "desc": "React Native, WatermelonDB, React Navigation, offline sync, iOS/Android native code.", "skills": 9, "agents": 2, "hooks": 2, "color": "#61dafb", "tags": []string{"mobile", "react-native", "ios", "android"}},
				{"name": "native-extensions", "display_name": "Native Extensions", "desc": "Extension lifecycle, commands, editor API, AI API, filesystem, UI, permissions, sandbox.", "skills": 10, "agents": 2, "hooks": 2, "color": "#f97316", "tags": []string{"extensions", "plugins", "sdk"}},
				{"name": "native-widgets", "display_name": "Native Widgets", "desc": "macOS WidgetKit, Windows Adaptive Cards, Linux GNOME/KDE. Cross-platform OS widget system.", "skills": 7, "agents": 1, "hooks": 1, "color": "#ec4899", "tags": []string{"widgets", "macos", "windows", "linux"}},
				{"name": "vscode-compat", "display_name": "VS Code Compat", "desc": "LSP/DAP, theme migration, snippets, grammars. Run VS Code extensions in Orchestra (~85% compat).", "skills": 8, "agents": 2, "hooks": 1, "color": "#007acc", "tags": []string{"vscode", "lsp", "dap"}},
				{"name": "raycast-compat", "display_name": "Raycast Compat", "desc": "List/Detail/Form/Action components. Run Raycast extensions in Orchestra (~95% compat).", "skills": 6, "agents": 1, "hooks": 1, "color": "#ff6363", "tags": []string{"raycast", "extensions"}},
				{"name": "extension-marketplace", "display_name": "Extension Marketplace", "desc": "Publishing, search, CLI, versioning, reviews, auto-updates. Full extension store pipeline.", "skills": 8, "agents": 2, "hooks": 2, "color": "#a855f7", "tags": []string{"marketplace", "publishing"}},
				{"name": "project-manager", "display_name": "Project Manager", "desc": "Sprint planning, feature breakdown, ADRs, cross-team coordination. Scrum + cyclical delivery.", "skills": 10, "agents": 3, "hooks": 2, "color": "#eab308", "tags": []string{"scrum", "agile", "pm"}},
			},
		},
		"plugins": map[string]interface{}{
			"total":       36,
			"total_tools": 290,
			"core_plugins": []map[string]interface{}{
				{"name": "storage.markdown", "tools": 1, "desc": "Protobuf metadata + Markdown body storage. Source of truth for all project data."},
				{"name": "tools.features", "tools": 34, "desc": "290-tool feature-driven workflow: projects, sprints, PRDs, tasks, dependencies, WIP limits."},
				{"name": "transport.stdio", "tools": 0, "desc": "MCP stdio JSON-RPC transport for Claude Code, Cursor, VS Code, Cline, Windsurf, Codex, Zed."},
				{"name": "tools.marketplace", "tools": 20, "desc": "Pack install/remove/update/list/search + 5 prompts. 17 official packs."},
			},
			"optional_plugins": []map[string]interface{}{
				{"name": "engine.rag", "tools": 22, "lang": "Rust", "desc": "Tree-sitter parsing (14 langs), Tantivy search, SQLite memory + cosine similarity embeddings."},
				{"name": "bridge.claude", "tools": 6, "lang": "Go", "desc": "Anthropic Claude — spawn sessions, stream, async polling, budget tracking."},
				{"name": "bridge.openai", "tools": 5, "lang": "Go", "desc": "OpenAI + compatible (grok, perplexity, deepseek, qwen, kimi)."},
				{"name": "bridge.gemini", "tools": 5, "lang": "Go", "desc": "Google Gemini Pro / Ultra via Generative Language API."},
				{"name": "bridge.ollama", "tools": 5, "lang": "Go", "desc": "Local Ollama models — llama3, mistral, codestral, etc."},
				{"name": "bridge.firecrawl", "tools": 5, "lang": "Go", "desc": "Web scraping and crawling via Firecrawl API."},
				{"name": "agent.orchestrator", "tools": 20, "lang": "Go", "desc": "Define agents, run workflows, compare providers, test suites."},
				{"name": "tools.agentops", "tools": 8, "lang": "Go", "desc": "Account management, budget tracking, usage reporting across providers."},
				{"name": "tools.sessions", "tools": 6, "lang": "Go", "desc": "Persistent AI conversation sessions with turn storage."},
				{"name": "tools.workspace", "tools": 8, "lang": "Go", "desc": "Multi-workspace registry, folder management, workspace switching."},
				{"name": "tools.notes", "tools": 8, "lang": "Go", "desc": "Markdown notes with tags, search, and versioning."},
				{"name": "tools.docs", "tools": 10, "lang": "Go", "desc": "Documentation browser, search, and generation."},
				{"name": "tools.markdown", "tools": 8, "lang": "Go", "desc": "Markdown parsing, rendering, and transformation utilities."},
				{"name": "devtools.git", "tools": 20, "lang": "Go", "desc": "Git operations — commit, branch, diff, log, stash, rebase."},
				{"name": "devtools.docker", "tools": 10, "lang": "Go", "desc": "Container lifecycle, images, volumes, compose."},
				{"name": "devtools.terminal", "tools": 6, "lang": "Go", "desc": "Integrated terminal sessions with output streaming."},
				{"name": "devtools.file-explorer", "tools": 17, "lang": "Go", "desc": "File system CRUD, search, tree navigation."},
				{"name": "devtools.database", "tools": 8, "lang": "Go", "desc": "Database query runner, schema explorer, migration runner."},
				{"name": "devtools.debugger", "tools": 9, "lang": "Go", "desc": "DAP-compatible debugger — breakpoints, step, watch, evaluate."},
				{"name": "devtools.test-runner", "tools": 8, "lang": "Go", "desc": "Run tests, watch mode, coverage reports across frameworks."},
				{"name": "devtools.services", "tools": 6, "lang": "Go", "desc": "Service process management, health checks, logs."},
				{"name": "devtools.log-viewer", "tools": 5, "lang": "Go", "desc": "Structured log viewing, filtering, and tailing."},
				{"name": "devtools.devops", "tools": 8, "lang": "Go", "desc": "CI/CD pipeline management, deployment, secrets."},
				{"name": "devtools.ssh", "tools": 7, "lang": "Go", "desc": "SSH connection management, remote execution, tunnels."},
				{"name": "devtools.components", "tools": 6, "lang": "Go", "desc": "UI component generation and storybook management."},
				{"name": "ai.browser-context", "tools": 7, "lang": "Go", "desc": "Browser context extraction for AI awareness."},
				{"name": "ai.screenshot", "tools": 6, "lang": "Go", "desc": "Screen capture and visual context for AI."},
				{"name": "ai.screen-reader", "tools": 6, "lang": "Go", "desc": "Accessibility tree extraction for AI understanding."},
				{"name": "ai.vision", "tools": 6, "lang": "Go", "desc": "Image analysis and visual reasoning tools."},
				{"name": "integration.figma", "tools": 6, "lang": "Go", "desc": "Figma file access, component export, design tokens."},
				{"name": "services.notifications", "tools": 8, "lang": "Go", "desc": "Push notifications — macOS, Windows, Linux, mobile."},
				{"name": "services.voice", "tools": 8, "lang": "Go", "desc": "Voice input/output, speech-to-text, text-to-speech."},
				{"name": "sync.cloud", "tools": 5, "lang": "Go", "desc": "Sync local projects to Orchestra cloud dashboard for team visibility."},
				{"name": "tools.extension-generator", "tools": 8, "lang": "Go", "desc": "Scaffold new plugins from templates with full SDK wiring."},
			},
		},
		"docs": map[string]interface{}{
			"version": "v1.0.0",
			"intro": map[string]interface{}{
				"title":    "Introduction to Orchestra",
				"subtitle": "Orchestra is the AI-native IDE platform. 290 tools, 36 plugins, 17 packs, 5 platforms. One protocol.",
				"description": "Orchestra solves the fragmentation problem in AI tooling. Instead of gluing together separate tools for project management, vector search, agent orchestration, and file storage — Orchestra provides all of these as MCP-compatible tools via a unified in-process plugin backbone.",
				"core_concepts": []map[string]interface{}{
					{"term": "Plugin", "def": "A Go, Rust, Swift, Kotlin, or C# binary that registers tools with the orchestrator. 4 core plugins bundled, 32+ optional."},
					{"term": "MCP Tool", "def": "A callable function exposed via the Model Context Protocol. Claude, Cursor, VS Code, Cline, Windsurf, Codex, Zed, Gemini all call these automatically."},
					{"term": "Pack", "def": "A curated bundle of skills (slash commands), agents, and hooks. Install with: orchestra pack install <name>"},
					{"term": "Orchestrator", "def": "The central in-process router that manages plugin discovery, tool dispatch, and lifecycle. Runs stdio + TCP (port 50101) + optional QUIC (port 9200)."},
				},
			},
			"installation": map[string]interface{}{
				"script": "curl -fsSL https://orchestra-mcp.dev/install.sh | sh",
				"verify": "orchestra version",
				"init":   "orchestra init",
				"serve":  "orchestra serve",
				"requirements": []map[string]interface{}{
					{"os": "macOS", "req": "13.0+ (Ventura), Apple Silicon or Intel"},
					{"os": "Linux", "req": "kernel 5.4+, glibc 2.31+"},
					{"os": "Windows", "req": "Windows 10 19041+ (WSL2 for CLI)"},
				},
			},
			"quick_start": map[string]interface{}{
				"steps": []map[string]interface{}{
					{"step": "1", "title": "Install Orchestra", "code": "curl -fsSL https://orchestra-mcp.dev/install.sh | sh"},
					{"step": "2", "title": "Initialize your project", "code": "cd your-project\norchestra init"},
					{"step": "3", "title": "Start the server", "code": "orchestra serve\n# ✓ orchestrator      listening :9100\n# ✓ storage.markdown  .projects/\n# ✓ tools.features    34 tools\n# ✓ tools.marketplace 20 tools\n# ⚡ 290 tools ready  · stdio + tcp · in-process"},
					{"step": "4", "title": "Connect your AI client", "code": "# .mcp.json is configured by orchestra init\n# Open your project in Claude Code, Cursor, or VS Code\n# MCP tools are available automatically"},
					{"step": "5", "title": "Install a pack (optional)", "code": "orchestra pack install go-backend\norchestra pack install ai-agentic\norchestra pack list"},
				},
			},
		},
		"blog": map[string]interface{}{
			"posts": []map[string]interface{}{
				{
					"slug":    "orchestra-v1-release",
					"title":   "Orchestra v1.0.0 — GA Release",
					"excerpt": "290 tools, 36 plugins, 17 official packs, 5 platforms. Single-process in-process architecture. Available now.",
					"tag":     "Announcement",
					"date":    "2026-03-02",
					"read_time": "4 min read",
				},
				{
					"slug":    "rag-memory-engine",
					"title":   "The RAG Memory Engine",
					"excerpt": "How Orchestra's Rust-powered engine.rag plugin delivers Tree-sitter parsing, Tantivy search, and SQLite vector memory.",
					"tag":     "Engineering",
					"date":    "2026-02-27",
					"read_time": "8 min read",
				},
				{
					"slug":    "multi-agent-orchestration",
					"title":   "Multi-Agent Orchestration",
					"excerpt": "Define agents, run workflows, compare providers. How agent.orchestrator manages Claude, OpenAI, Gemini, and Ollama in parallel.",
					"tag":     "Tutorial",
					"date":    "2026-02-24",
					"read_time": "6 min read",
				},
				{
					"slug":    "packs-system",
					"title":   "The Packs System",
					"excerpt": "17 official packs ship with curated skills, agents, and hooks for every stack. Here's how they work and how to build your own.",
					"tag":     "Product",
					"date":    "2026-02-20",
					"read_time": "5 min read",
				},
				{
					"slug":    "selective-plugin-system",
					"title":   "Selective Plugin Architecture",
					"excerpt": "4 core plugins bundled, 32+ optional. Install only what you need. How the new in-process plugin system works.",
					"tag":     "Engineering",
					"date":    "2026-03-01",
					"read_time": "7 min read",
				},
				{
					"slug":    "five-platforms",
					"title":   "One Protocol, Five Platforms",
					"excerpt": "macOS, Windows, Linux, Chrome Extension, and Mobile — how Orchestra delivers the same 290 tools everywhere.",
					"tag":     "Engineering",
					"date":    "2026-02-15",
					"read_time": "9 min read",
				},
			},
		},
		"social_platforms": map[string]interface{}{
			"platforms": []map[string]interface{}{
				{"value": "github", "label": "GitHub", "icon": "bxl-github", "placeholder": "https://github.com/..."},
				{"value": "twitter", "label": "Twitter / X", "icon": "bxl-twitter", "placeholder": "https://twitter.com/..."},
				{"value": "linkedin", "label": "LinkedIn", "icon": "bxl-linkedin", "placeholder": "https://linkedin.com/in/..."},
				{"value": "website", "label": "Website", "icon": "bx-globe", "placeholder": "https://..."},
			},
		},
		"sponsors": map[string]interface{}{
			"headline": "Our Sponsors",
			"subtext":  "Thanks to our amazing sponsors",
		},
		"community": map[string]interface{}{
			"headline": "Our Community",
			"subtext":  "Meet the people behind Orchestra",
		},
		"github": map[string]interface{}{
			"token":         "",
			"default_repos": "orchestra-mcp/framework",
			"sync_interval": 60,
		},
		"smart_prompts": map[string]interface{}{
			"prompts": []map[string]interface{}{
				{
					"key":         "agent",
					"label":       "Agent",
					"description": "System prompt for generating Agent definitions",
					"prompt":      "You are creating an Orchestra Agent. Output ONLY a valid JSON object with these fields:\n{\"name\": string (required), \"description\": string, \"system_prompt\": string (detailed markdown content)}\n\nRules:\n- Output ONLY the raw JSON object. No markdown code fences. No explanation.\n- All values must be strings.\n- For the markdown content field, use proper markdown formatting.",
				},
				{
					"key":         "skill",
					"label":       "Skill",
					"description": "System prompt for generating Skill definitions",
					"prompt":      "You are creating an Orchestra Skill. Output ONLY a valid JSON object with these fields:\n{\"name\": string (required), \"description\": string, \"body\": string (detailed markdown content)}\n\nRules:\n- Output ONLY the raw JSON object. No markdown code fences. No explanation.\n- All values must be strings.\n- For the markdown content field, use proper markdown formatting.",
				},
				{
					"key":         "workflow",
					"label":       "Workflow",
					"description": "System prompt for generating Workflow definitions with states, transitions, and gates",
					"prompt":      "You are creating an Orchestra Workflow definition. Output ONLY a valid JSON object with these fields:\n{\n  \"name\": string (required, workflow name),\n  \"description\": string (brief description),\n  \"initial_state\": string (the starting state key),\n  \"states\": { \"<key>\": {\"label\": string, \"terminal\": bool, \"active_work\": bool} },\n  \"transitions\": [{\"from\": \"<state_key>\", \"to\": \"<state_key>\", \"gate\": \"<gate_key or empty>\"}],\n  \"gates\": { \"<key>\": {\"label\": string, \"required_section\": string, \"skippable_for\": []} }\n}\n\nExample state keys: \"todo\", \"in-progress\", \"in-review\", \"done\".\nterminal=true means the workflow ends at that state.\nactive_work=true means active development happens in that state.\n\nRules:\n- Output ONLY the raw JSON object. No markdown code fences. No explanation.\n- states, transitions, and gates must be proper JSON (not strings).\n- Every state referenced in transitions must exist in states.\n- initial_state must be a valid state key.",
				},
				{
					"key":         "doc",
					"label":       "Document",
					"description": "System prompt for generating Document content",
					"prompt":      "You are creating an Orchestra Document. Output ONLY a valid JSON object with these fields:\n{\"name\": string (required), \"description\": string, \"body\": string (detailed markdown content)}\n\nRules:\n- Output ONLY the raw JSON object. No markdown code fences. No explanation.\n- All values must be strings.\n- For the markdown content field, use proper markdown formatting.",
				},
				{
					"key":         "feature",
					"label":       "Feature",
					"description": "System prompt for generating Feature definitions",
					"prompt":      "You are creating an Orchestra Feature. Output ONLY a valid JSON object with these fields:\n{\"name\": string (required), \"description\": string, \"kind\": string (one of: feature, bug, hotfix, chore), \"priority\": string (one of: low, medium, high, critical)}\n\nRules:\n- Output ONLY the raw JSON object. No markdown code fences. No explanation.\n- All values must be strings.",
				},
				{
					"key":         "plan",
					"label":       "Plan",
					"description": "System prompt for generating Plan definitions",
					"prompt":      "You are creating an Orchestra Plan. Output ONLY a valid JSON object with these fields:\n{\"name\": string (required), \"description\": string, \"body\": string (detailed markdown content)}\n\nRules:\n- Output ONLY the raw JSON object. No markdown code fences. No explanation.\n- All values must be strings.\n- For the markdown content field, use proper markdown formatting.",
				},
				{
					"key":         "request",
					"label":       "Request",
					"description": "System prompt for generating Request definitions",
					"prompt":      "You are creating an Orchestra Request. Output ONLY a valid JSON object with these fields:\n{\"name\": string (required), \"description\": string, \"kind\": string (one of: feature, bug, hotfix), \"priority\": string (one of: low, medium, high, critical)}\n\nRules:\n- Output ONLY the raw JSON object. No markdown code fences. No explanation.\n- All values must be strings.",
				},
				{
					"key":         "person",
					"label":       "Person",
					"description": "System prompt for generating Person profiles",
					"prompt":      "You are creating an Orchestra Person profile. Output ONLY a valid JSON object with these fields:\n{\"name\": string (required), \"role\": string, \"bio\": string, \"github_email\": string}\n\nRules:\n- Output ONLY the raw JSON object. No markdown code fences. No explanation.\n- All values must be strings.",
				},
			},
		},
	}
	if v, ok := defaults[key]; ok {
		return v
	}
	return map[string]interface{}{}
}

// GetPublicSetting handles GET /api/settings/:key — no auth required, only for public keys.
func (h *AdminSettingsHandler) GetPublicSetting(c fiber.Ctx) error {
	key := c.Params("key")
	if !publicKeys[key] {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	var setting models.SystemSetting
	var value interface{}
	if err := h.db.Where("key = ?", key).First(&setting).Error; err != nil {
		value = defaultSettings(key)
	} else {
		_ = json.Unmarshal(setting.Value, &value)
	}

	// Strip secrets from integrations — only expose *_enabled flags publicly
	if key == "integrations" {
		if m, ok := value.(map[string]interface{}); ok {
			public := map[string]interface{}{}
			for k, v := range m {
				if strings.HasSuffix(k, "_enabled") {
					public[k] = v
				}
			}
			value = public
		}
	}

	// Overlay locale-specific translations for public content
	locale := helpers.ContentLocale(c)
	if locale != helpers.DefaultLocale {
		if m, ok := value.(map[string]interface{}); ok {
			if tr, ok := m["translations"].(map[string]interface{}); ok {
				if localeData, ok := tr[locale].(map[string]interface{}); ok {
					for k, v := range localeData {
						m[k] = v
					}
				}
			}
			delete(m, "translations")
		}
	}

	return c.JSON(fiber.Map{"key": key, "value": value})
}

// GetSetting handles GET /api/admin/settings/:key
func (h *AdminSettingsHandler) GetSetting(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	key := c.Params("key")
	if !validKeys[key] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid settings key"})
	}

	var setting models.SystemSetting
	if err := h.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return c.JSON(fiber.Map{"key": key, "value": defaultSettings(key)})
	}

	var value interface{}
	_ = json.Unmarshal(setting.Value, &value)

	// Overlay locale-specific translations when requested
	locale := helpers.ContentLocale(c)
	if locale != helpers.DefaultLocale {
		if m, ok := value.(map[string]interface{}); ok {
			if tr, ok := m["translations"].(map[string]interface{}); ok {
				if localeData, ok := tr[locale].(map[string]interface{}); ok {
					for k, v := range localeData {
						m[k] = v
					}
				}
			}
			delete(m, "translations")
		}
	}

	return c.JSON(fiber.Map{"key": key, "value": value, "updated_at": setting.UpdatedAt})
}

// UpdateSetting handles PATCH /api/admin/settings/:key
func (h *AdminSettingsHandler) UpdateSetting(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	key := c.Params("key")
	if !validKeys[key] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid settings key"})
	}

	rawValue := c.Body()
	if len(rawValue) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body is required"})
	}

	locale := helpers.ContentLocale(c)

	var finalValue []byte
	if locale == helpers.DefaultLocale {
		finalValue = rawValue
	} else {
		// Load existing setting, merge translation into it
		var existing models.SystemSetting
		var currentValue map[string]interface{}
		if err := h.db.Where("key = ?", key).First(&existing).Error; err == nil {
			_ = json.Unmarshal(existing.Value, &currentValue)
		}
		if currentValue == nil {
			currentValue = map[string]interface{}{}
		}

		var newFields map[string]interface{}
		_ = json.Unmarshal(rawValue, &newFields)

		translations, _ := currentValue["translations"].(map[string]interface{})
		if translations == nil {
			translations = map[string]interface{}{}
		}
		translations[locale] = newFields
		currentValue["translations"] = translations

		finalValue, _ = json.Marshal(currentValue)
	}

	setting := models.SystemSetting{
		Key:   key,
		Value: finalValue,
	}

	if err := h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&setting).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save setting"})
	}

	return c.JSON(fiber.Map{"ok": true, "key": key})
}

// SeedSettings handles POST /api/admin/settings/seed — populates all keys with defaults.
func (h *AdminSettingsHandler) SeedSettings(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	seeded := []string{}
	for key := range validKeys {
		defaults := defaultSettings(key)
		raw, err := json.Marshal(defaults)
		if err != nil {
			continue
		}
		setting := models.SystemSetting{
			Key:   key,
			Value: raw,
		}
		if err := h.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&setting).Error; err != nil {
			continue
		}
		seeded = append(seeded, key)
	}

	return c.JSON(fiber.Map{"ok": true, "seeded": seeded, "count": len(seeded)})
}

// TestEmail handles POST /api/admin/settings/test-email — sends a test email using current SMTP config.
func (h *AdminSettingsHandler) TestEmail(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	// Get current user email from context
	userEmail, _ := c.Locals("email").(string)
	if userEmail == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no user email found"})
	}

	// Load SMTP settings
	var setting models.SystemSetting
	smtpCfg := map[string]interface{}{}
	if err := h.db.Where("key = ?", "smtp").First(&setting).Error; err == nil {
		_ = json.Unmarshal(setting.Value, &smtpCfg)
	} else {
		defaults := defaultSettings("smtp")
		if m, ok := defaults.(map[string]interface{}); ok {
			smtpCfg = m
		}
	}

	host, _ := smtpCfg["host"].(string)
	portRaw := smtpCfg["port"]
	username, _ := smtpCfg["username"].(string)
	password, _ := smtpCfg["password"].(string)
	fromName, _ := smtpCfg["from_name"].(string)
	fromEmail, _ := smtpCfg["from_email"].(string)

	if host == "" || fromEmail == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "SMTP not configured — set host and from_email first"})
	}

	port := 587
	switch v := portRaw.(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	subject := "Orchestra — Test Email"
	body := fmt.Sprintf("Hello!\n\nThis is a test email from Orchestra sent to %s.\n\nYour SMTP configuration is working correctly.\n\n— Orchestra", userEmail)

	msg := strings.Join([]string{
		"From: " + fromName + " <" + fromEmail + ">",
		"To: " + userEmail,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	if err := smtp.SendMail(addr, auth, fromEmail, []string{userEmail}, []byte(msg)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send: " + err.Error()})
	}

	return c.JSON(fiber.Map{"ok": true, "sent_to": userEmail})
}

// GenerateSitemap handles POST /api/admin/settings/generate-sitemap — returns a sitemap.xml string.
func (h *AdminSettingsHandler) GenerateSitemap(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin access required"})
	}

	// Load general settings for site URL
	siteURL := "https://orchestra-mcp.dev"
	var generalSetting models.SystemSetting
	if err := h.db.Where("key = ?", "general").First(&generalSetting).Error; err == nil {
		var val map[string]interface{}
		if json.Unmarshal(generalSetting.Value, &val) == nil {
			if u, ok := val["url"].(string); ok && u != "" {
				siteURL = strings.TrimRight(u, "/")
			}
		}
	}

	// Load feature flags to know which pages are enabled
	enabledPages := map[string]bool{
		"blog": true, "docs": true, "download": true,
		"solutions": true, "marketplace": true, "pricing": true, "contact": true,
	}
	var featureSetting models.SystemSetting
	if err := h.db.Where("key = ?", "features").First(&featureSetting).Error; err == nil {
		var val map[string]interface{}
		if json.Unmarshal(featureSetting.Value, &val) == nil {
			for k, v := range val {
				if b, ok := v.(bool); ok {
					enabledPages[k] = b
				}
			}
		}
	}

	// Static routes
	routes := []string{"/", "/login", "/register"}
	pageRoutes := []string{"/blog", "/docs", "/download", "/solutions", "/marketplace", "/pricing", "/contact"}
	for _, r := range pageRoutes {
		key := strings.TrimPrefix(r, "/")
		if enabled, ok := enabledPages[key]; !ok || enabled {
			routes = append(routes, r)
		}
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, route := range routes {
		sb.WriteString("  <url>\n")
		sb.WriteString("    <loc>" + siteURL + route + "</loc>\n")
		sb.WriteString("    <changefreq>weekly</changefreq>\n")
		sb.WriteString("  </url>\n")
	}
	sb.WriteString("</urlset>\n")

	return c.JSON(fiber.Map{"ok": true, "sitemap": sb.String()})
}
