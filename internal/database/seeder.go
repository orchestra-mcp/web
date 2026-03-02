package database

import (
	"encoding/json"
	"log"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// SeedDefaults inserts default system settings if they don't already exist.
// Safe to call on every boot — uses FirstOrCreate for idempotency.
func SeedDefaults(db *gorm.DB) {
	settings := map[string]interface{}{
		"general": map[string]interface{}{
			"site_name":        "Orchestra",
			"tagline":          "The AI-Agentic IDE",
			"url":              "https://orchestra.dev",
			"support_email":    "support@orchestra.dev",
			"maintenance_mode": false,
			"version":          "v1.0.0",
			"github_url":       "https://github.com/orchestra-mcp",
			"discord_url":      "https://discord.gg/orchestra",
		},
		"features": []map[string]interface{}{
			{"key": "blog", "label": "Blog", "enabled": true, "admin_only": false},
			{"key": "marketplace", "label": "Marketplace", "enabled": true, "admin_only": false},
			{"key": "docs", "label": "Docs", "enabled": true, "admin_only": false},
			{"key": "download", "label": "Download", "enabled": true, "admin_only": false},
			{"key": "solutions", "label": "Solutions", "enabled": true, "admin_only": false},
			{"key": "pricing", "label": "Pricing", "enabled": true, "admin_only": false},
		},
		"homepage": map[string]interface{}{
			"hero_headline": "The AI-native IDE for every platform",
			"hero_subtext":  "Orchestra connects your AI agents to 290 tools across 36 plugins and 5 platforms — over QUIC + Protobuf with mTLS.",
			"hero_cta":      "Get started free",
			"stats": []map[string]interface{}{
				{"label": "MCP Tools", "value": "290"},
				{"label": "Plugins", "value": "36"},
				{"label": "Official Packs", "value": "17"},
				{"label": "Platforms", "value": "5"},
			},
			"terminal_lines": []map[string]interface{}{
				{"name": "orchestrator", "note": "listening :9100", "color": "cyan"},
				{"name": "storage.markdown", "note": ".projects/", "color": "cyan"},
				{"name": "tools.features", "note": "34 tools", "color": "cyan"},
				{"name": "engine.rag", "note": "22 tools  (Rust)", "color": "cyan"},
				{"name": "bridge.claude", "note": "5 tools + 1 stream", "color": "purple"},
				{"name": "agent.orchestrator", "note": "20 tools", "color": "purple"},
				{"name": "devtools.git", "note": "20 tools", "color": "cyan"},
				{"name": "transport.quic-bridge", "note": "port 9200", "color": "green"},
			},
			"total_tools": "290",
		},
		"agents": map[string]interface{}{
			"headline": "Multi-agent orchestration built in",
			"subtext":  "Define agents, run workflows, compare providers. 20 tools for multi-model coordination.",
		},
		"contact": map[string]interface{}{
			"headline": "Get in touch",
			"email":    "hello@orchestra.dev",
			"hours":    "Mon–Fri, 9am–5pm PST",
		},
		"pricing": map[string]interface{}{
			"plans": []map[string]interface{}{
				{
					"name": "Free", "price": "0", "period": "forever", "highlighted": false,
					"cta": "Get started", "href": "/register",
					"desc": "Perfect for solo developers and open source projects.",
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
					"cta": "Start free trial", "href": "/register?plan=pro",
					"desc": "For teams building production AI-native applications.",
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
					"cta": "Contact us", "href": "/contact",
					"desc": "Custom deployments, SLAs, and dedicated support.",
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
			"version":         "1.0.0",
			"release_date":    "2026-03-02",
			"install_script":  "curl -fsSL https://orchestra-mcp.dev/install.sh | sh",
			"macos":           map[string]string{"url": "https://github.com/orchestra-mcp/orchestra/releases/latest", "version": "1.0.0", "release_date": "2026-03-02", "arch": "Apple Silicon + Intel", "desc": "Universal binary. Requires macOS 13+.", "file": ".dmg"},
			"windows":         map[string]string{"url": "https://github.com/orchestra-mcp/orchestra/releases/latest", "version": "1.0.0", "release_date": "2026-03-02", "arch": "x64", "desc": "MSIX installer. Requires Windows 10 19041+.", "file": ".msix"},
			"linux":           map[string]string{"url": "https://github.com/orchestra-mcp/orchestra/releases/latest", "version": "1.0.0", "release_date": "2026-03-02", "arch": "x64 + arm64", "desc": "Flatpak (recommended), .deb, .rpm, AppImage.", "file": "Flatpak"},
		},
		"integrations": map[string]interface{}{
			"google": map[string]string{"client_id": "", "client_secret": ""},
			"github": map[string]string{"client_id": "", "client_secret": ""},
		},
		"smtp": map[string]interface{}{
			"host":       "smtp.gmail.com",
			"port":       587,
			"username":   "",
			"password":   "",
			"from_name":  "Orchestra",
			"from_email": "noreply@orchestra.dev",
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
				{"name": "macos-integration", "display_name": "macOS Integration", "desc": "CGo, Spotlight, Keychain, iCloud, Touch Bar, Notifications, file associations, URL schemes.", "skills": 9, "agents": 2, "hooks": 1, "color": "#888888", "tags": []string{"macos", "swift", "cgo"}},
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
				{"name": "tools.features", "tools": 34, "desc": "34-tool feature-driven workflow: projects, sprints, PRDs, tasks, dependencies, WIP limits."},
				{"name": "transport.stdio", "tools": 0, "desc": "MCP stdio JSON-RPC transport for Claude Code, Cursor, VS Code, Cline, Windsurf, Codex, Zed."},
				{"name": "tools.marketplace", "tools": 20, "desc": "Pack install/remove/update/list/search + 5 prompts. 17 official packs."},
			},
			"optional_plugins": []map[string]interface{}{
				{"name": "engine.rag", "tools": 22, "lang": "Rust", "desc": "Tree-sitter parsing (14 langs), Tantivy search, SQLite memory + cosine similarity embeddings."},
				{"name": "bridge.claude", "tools": 6, "lang": "Go", "desc": "Anthropic Claude — spawn sessions, stream, async polling, budget tracking."},
				{"name": "bridge.openai", "tools": 5, "lang": "Go", "desc": "OpenAI + compatible APIs (grok, perplexity, deepseek, qwen, kimi)."},
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
			"nav": []map[string]interface{}{
				{"title": "Getting Started", "items": []map[string]string{
					{"title": "Introduction", "slug": ""},
					{"title": "Installation", "slug": "installation"},
					{"title": "Quick Start", "slug": "quick-start"},
				}},
				{"title": "Core Concepts", "items": []map[string]string{
					{"title": "Plugins", "slug": "plugins"},
					{"title": "MCP Tools", "slug": "mcp-tools"},
					{"title": "Packs", "slug": "packs"},
					{"title": "Orchestrator", "slug": "orchestrator"},
				}},
				{"title": "Guides", "items": []map[string]string{
					{"title": "Connect your AI client", "slug": "connect-ai-client"},
					{"title": "Install plugins", "slug": "install-plugins"},
					{"title": "Build a pack", "slug": "build-a-pack"},
					{"title": "Multi-agent workflows", "slug": "multi-agent"},
				}},
			},
			"intro": map[string]interface{}{
				"title":       "Introduction to Orchestra",
				"version":     "v1.0.0",
				"description": "Orchestra is an AI-native IDE platform that connects your AI agents to a unified plugin backbone. 290 tools, 36 plugins, 17 packs, 5 platforms, one protocol.",
				"what_is": "Orchestra solves the fragmentation problem in AI tooling. Instead of gluing together separate tools for project management, vector search, agent orchestration, and file storage, Orchestra provides all of these as MCP-compatible tools via a unified plugin mesh.",
				"concepts": []map[string]string{
					{"term": "Plugin", "def": "A standalone binary that connects to the orchestrator over QUIC + Protobuf with mTLS. Written in Go, Rust, Swift, Kotlin, or C#."},
					{"term": "MCP Tool", "def": "A callable function exposed by a plugin via the Model Context Protocol. Your AI client (Claude, Cursor, etc.) calls these automatically."},
					{"term": "Pack", "def": "A curated collection of skills (slash commands), agents, and hooks. Installed with orchestra pack install <name>."},
					{"term": "Orchestrator", "def": "The central process that manages plugin discovery, routing, and lifecycle. Runs on port 9100 by default."},
				},
				"install_cmd": "curl -fsSL https://orchestra-mcp.dev/install.sh | sh\norchestra init\norchestra serve",
			},
			"installation": map[string]interface{}{
				"title":       "Installation",
				"description": "Install Orchestra using the install script (recommended) or download a platform-specific binary.",
				"install_cmd": "curl -fsSL https://orchestra-mcp.dev/install.sh | sh",
				"verify_cmd":  "orchestra version\n# Orchestra v1.0.0",
				"init_cmd":    "cd your-project\norchestra init",
				"requirements": []map[string]string{
					{"os": "macOS", "req": "13.0+ (Ventura), Apple Silicon or Intel"},
					{"os": "Linux", "req": "kernel 5.4+, glibc 2.31+"},
					{"os": "Windows", "req": "Windows 10 19041+ (WSL2 for CLI)"},
				},
			},
			"quick_start": map[string]interface{}{
				"title":       "Quick Start",
				"description": "Get Orchestra running and connected to your AI client in under 5 minutes.",
				"steps": []map[string]string{
					{"step": "1", "title": "Install Orchestra", "code": "curl -fsSL https://orchestra-mcp.dev/install.sh | sh", "lang": "bash"},
					{"step": "2", "title": "Initialize your project", "code": "cd your-project\norchestra init", "lang": "bash"},
					{"step": "3", "title": "Start the server", "code": "orchestra serve\n# ✓ orchestrator  listening :9100\n# ✓ tools.features 34 tools\n# ✓ engine.rag     22 tools\n# ⚡ 290 tools ready", "lang": "bash"},
					{"step": "4", "title": "Connect your AI client", "code": "# .mcp.json is already configured by orchestra init\n# In Claude Code, Cursor, or VS Code:\n# Open your project — MCP tools are available automatically", "lang": "bash"},
					{"step": "5", "title": "Install a pack (optional)", "code": "orchestra pack install go-backend\norchestra pack install typescript-react", "lang": "bash"},
				},
				"ready_msg": "Orchestra is running. Your AI client now has access to 290 tools across project management, RAG memory, agent orchestration, and more.",
			},
		},
		"blog": map[string]interface{}{
			"posts": []map[string]interface{}{
				{"slug": "orchestra-v1-release", "title": "Orchestra v1.0.0 — GA release", "excerpt": "After months of development, Orchestra v1.0.0 is here. 290 tools, 36 plugins, 17 packs, 5 platforms — single binary, in-process architecture.", "date": "Mar 02, 2026", "read_time": "5 min read", "tag": "Announcement"},
				{"slug": "in-process-architecture", "title": "Why we switched to in-process architecture", "excerpt": "We replaced our QUIC plugin mesh with an in-process single-binary design for local IDE use. Here's why and what we gained.", "date": "Mar 01, 2026", "read_time": "7 min read", "tag": "Engineering"},
				{"slug": "selective-plugin-system", "title": "Introducing selective plugins: install only what you need", "excerpt": "4 core plugins ship bundled in the 17MB binary. Install any of the 32 optional plugins on demand via orchestra plugin install.", "date": "Feb 28, 2026", "read_time": "4 min read", "tag": "Product"},
				{"slug": "rag-memory-engine", "title": "How our RAG Memory Engine works", "excerpt": "A deep dive into Tantivy full-text search, SQLite vector embeddings, and how we got 22 tools running in a Rust plugin.", "date": "Feb 27, 2026", "read_time": "8 min read", "tag": "Engineering"},
				{"slug": "multi-agent-orchestration", "title": "Multi-agent orchestration with Orchestra", "excerpt": "Running Claude, GPT-4o, and Gemini in parallel agent workflows using our agent.orchestrator plugin — 20 tools, zero lock-in.", "date": "Feb 25, 2026", "read_time": "6 min read", "tag": "Tutorial"},
				{"slug": "packs-system", "title": "17 official packs: skills, agents, and hooks in one command", "excerpt": "Install curated workflows with orchestra pack install go-backend. A walkthrough of all 17 packs and how to build your own.", "date": "Feb 20, 2026", "read_time": "5 min read", "tag": "Product"},
			},
		},
	}

	for key, val := range settings {
		raw, err := json.Marshal(val)
		if err != nil {
			log.Printf("seeder: failed to marshal %s: %v", key, err)
			continue
		}

		s := models.SystemSetting{Key: key}
		result := db.Where("key = ?", key).FirstOrCreate(&s, models.SystemSetting{
			Key:   key,
			Value: raw,
		})
		if result.Error != nil {
			log.Printf("seeder: failed to seed %s: %v", key, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("seeder: seeded default %s", key)
		}
	}
}
