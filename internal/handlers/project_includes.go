package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// ProjectIncludeHandler handles project ↔ skill/agent pivot table endpoints.
type ProjectIncludeHandler struct {
	db *gorm.DB
}

// NewProjectIncludeHandler creates a new ProjectIncludeHandler.
func NewProjectIncludeHandler(db *gorm.DB) *ProjectIncludeHandler {
	return &ProjectIncludeHandler{db: db}
}

// findProject looks up a project by slug, scoped to the user's own projects
// or projects belonging to teams the user is a member of.
func (h *ProjectIncludeHandler) findProject(c fiber.Ctx, user *models.User) (*models.Project, error) {
	var project models.Project
	if err := teamScopedProjects(h.db, user.ID).
		Where("slug = ?", c.Params("slug")).
		First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// --- Skills ---

// ListProjectSkills handles GET /api/projects/:slug/skills
func (h *ProjectIncludeHandler) ListProjectSkills(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.findProject(c, user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var entries []models.ProjectSkill
	if err := h.db.Where("project_id = ?", project.ID).
		Preload("Skill").
		Find(&entries).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list project skills"})
	}

	return c.JSON(entries)
}

// IncludeSkill handles POST /api/projects/:slug/skills
func (h *ProjectIncludeHandler) IncludeSkill(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.findProject(c, user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var body struct {
		SkillID string `json:"skill_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.SkillID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "skill_id is required"})
	}

	entry := models.ProjectSkill{
		ProjectID: project.ID,
		SkillID:   body.SkillID,
		Enabled:   true,
	}

	if err := h.db.Create(&entry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to include skill"})
	}

	return c.Status(fiber.StatusCreated).JSON(entry)
}

// ExcludeSkill handles DELETE /api/projects/:slug/skills/:id
func (h *ProjectIncludeHandler) ExcludeSkill(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.findProject(c, user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	result := h.db.Where("project_id = ? AND skill_id = ?", project.ID, c.Params("id")).
		Delete(&models.ProjectSkill{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to exclude skill"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project skill not found"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// --- Agents ---

// ListProjectAgents handles GET /api/projects/:slug/agents
func (h *ProjectIncludeHandler) ListProjectAgents(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.findProject(c, user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var entries []models.ProjectAgent
	if err := h.db.Where("project_id = ?", project.ID).
		Preload("Agent").
		Find(&entries).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list project agents"})
	}

	return c.JSON(entries)
}

// IncludeAgent handles POST /api/projects/:slug/agents
func (h *ProjectIncludeHandler) IncludeAgent(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.findProject(c, user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	var body struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.AgentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "agent_id is required"})
	}

	entry := models.ProjectAgent{
		ProjectID: project.ID,
		AgentID:   body.AgentID,
		Enabled:   true,
	}

	if err := h.db.Create(&entry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to include agent"})
	}

	return c.Status(fiber.StatusCreated).JSON(entry)
}

// ExcludeAgent handles DELETE /api/projects/:slug/agents/:id
func (h *ProjectIncludeHandler) ExcludeAgent(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.findProject(c, user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	result := h.db.Where("project_id = ? AND agent_id = ?", project.ID, c.Params("id")).
		Delete(&models.ProjectAgent{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to exclude agent"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project agent not found"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// --- Generate Docs ---

// GenerateDocs handles POST /api/projects/:slug/generate-docs.
// It queries the project's enabled skills/agents and generates
// CLAUDE.md and AGENTS.md content from them.
func (h *ProjectIncludeHandler) GenerateDocs(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	project, err := h.findProject(c, user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	// Query enabled skills for this project.
	var skillEntries []models.ProjectSkill
	h.db.Where("project_id = ? AND enabled = ?", project.ID, true).
		Preload("Skill").Find(&skillEntries)

	var skills []string
	for _, e := range skillEntries {
		if e.Skill.Slug != "" {
			skills = append(skills, e.Skill.Slug)
		}
	}
	sort.Strings(skills)

	// Query enabled agents for this project.
	var agentEntries []models.ProjectAgent
	h.db.Where("project_id = ? AND enabled = ?", project.ID, true).
		Preload("Agent").Find(&agentEntries)

	var agents []string
	for _, e := range agentEntries {
		if e.Agent.Slug != "" {
			agents = append(agents, e.Agent.Slug)
		}
	}
	sort.Strings(agents)

	// Collect full skill models for CONTEXT.md.
	var fullSkills []models.Skill
	for _, e := range skillEntries {
		if e.Skill.Slug != "" {
			fullSkills = append(fullSkills, e.Skill)
		}
	}

	// Collect full agent models for CONTEXT.md.
	var fullAgents []models.Agent
	for _, e := range agentEntries {
		if e.Agent.Slug != "" {
			fullAgents = append(fullAgents, e.Agent)
		}
	}

	// Query features for this project.
	var features []models.Feature
	h.db.Where("project_slug = ?", project.Slug).Find(&features)

	claudeMD := buildClaudeMD(project.Name, skills, agents)
	agentsMD := buildAgentsMD(agents)
	contextMD := buildContextMD(*project, features, fullSkills, fullAgents)

	return c.JSON(fiber.Map{
		"project":    project.Slug,
		"claude_md":  claudeMD,
		"agents_md":  agentsMD,
		"context_md": contextMD,
		"skills":     skills,
		"agents":     agents,
	})
}

// buildClaudeMD generates CLAUDE.md content from a project's included skills/agents.
func buildClaudeMD(projectName string, skills, agents []string) string {
	var b strings.Builder

	b.WriteString("# CLAUDE.md\n\n")
	b.WriteString(fmt.Sprintf("This project (**%s**) uses [Orchestra MCP](https://github.com/orchestra-mcp/framework) for AI-powered project management.\n\n", projectName))

	b.WriteString("## Available Tools\n\n")
	b.WriteString("Orchestra provides tools via MCP. Run `orchestra serve` to start the MCP server. IDE config is in `.mcp.json`.\n\n")

	// Skills section.
	b.WriteString("## Skills (Slash Commands)\n\n")
	if len(skills) == 0 {
		b.WriteString("No skills configured for this project.\n\n")
	} else {
		b.WriteString("| Command | Source |\n")
		b.WriteString("|---------|--------|\n")
		for _, name := range skills {
			b.WriteString(fmt.Sprintf("| `/%s` | .claude/skills/%s/ |\n", name, name))
		}
		b.WriteString("\n")
	}

	// Agents section.
	b.WriteString("## Agents\n\n")
	if len(agents) == 0 {
		b.WriteString("No agents configured for this project.\n\n")
	} else {
		b.WriteString("Specialized agents in `.claude/agents/` auto-delegate based on task context.\n\n")
		b.WriteString("| Agent | File |\n")
		b.WriteString("|-------|------|\n")
		for _, name := range agents {
			b.WriteString(fmt.Sprintf("| `%s` | .claude/agents/%s.md |\n", name, name))
		}
		b.WriteString("\n")
	}

	// Mandatory workflow rule (same as CLI-generated version).
	b.WriteString("\n## Mandatory Workflow Rule\n\n")
	b.WriteString("**ALL work MUST go through Orchestra MCP tools.** When the user asks you to do ANY task:\n\n")
	b.WriteString("1. `search_features` / `list_features` — check for existing feature\n")
	b.WriteString("2. `create_feature` — create one if needed (with `kind`: feature/bug/hotfix/chore)\n")
	b.WriteString("3. `set_current_feature` — start work\n")
	b.WriteString("4. Do the work (each status = ONE activity only)\n")
	b.WriteString("5. `advance_feature` — pass gates with structured evidence\n")
	b.WriteString("6. At `in-review`: use `AskUserQuestion` for user approval\n")
	b.WriteString("7. `submit_review` — complete\n\n")
	b.WriteString("**Never do any work without an active feature.**\n\n")

	// Git & Sync.
	b.WriteString("## Git & Sync\n\n")
	b.WriteString("| User says | Action |\n")
	b.WriteString("|-----------|--------|\n")
	b.WriteString("| \"sync my changes\" | `git_quick_commit` → `git_push` |\n")
	b.WriteString("| \"get latest\" | `git_pull` |\n")
	b.WriteString("| \"save my work\" | `git_quick_commit` |\n")
	b.WriteString("| \"push\" | `git_push` |\n")
	b.WriteString("| \"create a branch for X\" | `git_create_branch` |\n")
	b.WriteString("| \"merge X\" | `git_merge_branch` |\n")
	b.WriteString("| \"git status\" | `git_status_summary` |\n")

	return b.String()
}

// buildAgentsMD generates AGENTS.md content from included agents.
func buildAgentsMD(agents []string) string {
	var b strings.Builder

	b.WriteString("# AGENTS.md\n\n")
	b.WriteString("Specialized agents installed via Orchestra. Each agent is a markdown file in `.claude/agents/` that provides domain-specific instructions.\n\n")

	if len(agents) == 0 {
		b.WriteString("No agents configured for this project.\n")
	} else {
		for _, name := range agents {
			b.WriteString(fmt.Sprintf("## %s\n\n", name))
			b.WriteString(fmt.Sprintf("See [.claude/agents/%s.md](.claude/agents/%s.md)\n\n", name, name))
		}
	}

	return b.String()
}

// buildContextMD generates CONTEXT.md content from a project's features, skills, and agents.
func buildContextMD(project models.Project, features []models.Feature, skills []models.Skill, agents []models.Agent) string {
	var b strings.Builder

	b.WriteString("# CONTEXT.md\n\n")

	// Project Overview.
	b.WriteString("## Project Overview\n")
	desc := project.Description
	if desc == "" {
		desc = "No description provided."
	}
	b.WriteString(fmt.Sprintf("**%s** — %s\n\n", project.Name, desc))

	// Detected Stacks — union of all stacks from included skills.
	stackSet := make(map[string]bool)
	for _, skill := range skills {
		if len(skill.Stacks) > 0 {
			var stacks []string
			if err := json.Unmarshal(skill.Stacks, &stacks); err == nil {
				for _, s := range stacks {
					stackSet[s] = true
				}
			}
		}
	}
	b.WriteString("## Detected Stacks\n")
	if len(stackSet) == 0 {
		b.WriteString("No stacks detected.\n\n")
	} else {
		var stackList []string
		for s := range stackSet {
			stackList = append(stackList, s)
		}
		sort.Strings(stackList)
		b.WriteString(strings.Join(stackList, ", "))
		b.WriteString("\n\n")
	}

	// Project Health.
	total := len(features)
	var doneCount, inProgressCount, inReviewCount, todoCount int
	for _, f := range features {
		switch f.Status {
		case "done":
			doneCount++
		case "in-progress":
			inProgressCount++
		case "in-review":
			inReviewCount++
		case "todo":
			todoCount++
		}
	}

	b.WriteString("## Project Health\n")
	donePct := 0
	if total > 0 {
		donePct = doneCount * 100 / total
	}
	b.WriteString(fmt.Sprintf("- **Total features:** %d\n", total))
	b.WriteString(fmt.Sprintf("- **Done:** %d (%d%%)\n", doneCount, donePct))
	b.WriteString(fmt.Sprintf("- **In Progress:** %d\n", inProgressCount))
	b.WriteString(fmt.Sprintf("- **In Review:** %d\n", inReviewCount))
	b.WriteString(fmt.Sprintf("- **To Do:** %d\n\n", todoCount))

	// Active Features — table of non-done features.
	b.WriteString("## Active Features\n")
	var active []models.Feature
	for _, f := range features {
		if f.Status != "done" {
			active = append(active, f)
		}
	}
	if len(active) == 0 {
		b.WriteString("No active features.\n\n")
	} else {
		b.WriteString("| ID | Title | Status | Priority | Assignee |\n")
		b.WriteString("|----|-------|--------|----------|----------|\n")
		for _, f := range active {
			assignee := f.Assignee
			if assignee == "" {
				assignee = "-"
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				f.FeatureID, f.Title, f.Status, f.Priority, assignee))
		}
		b.WriteString("\n")
	}

	// Included Skills.
	b.WriteString("## Included Skills\n")
	if len(skills) == 0 {
		b.WriteString("No skills configured for this project.\n\n")
	} else {
		for _, skill := range skills {
			desc := skill.Description
			if desc == "" {
				desc = "No description."
			}
			b.WriteString(fmt.Sprintf("- **%s** — %s\n", skill.Name, desc))
		}
		b.WriteString("\n")
	}

	// Included Agents.
	b.WriteString("## Included Agents\n")
	if len(agents) == 0 {
		b.WriteString("No agents configured for this project.\n")
	} else {
		for _, agent := range agents {
			desc := agent.Description
			if desc == "" {
				desc = "No description."
			}
			b.WriteString(fmt.Sprintf("- **%s** — %s\n", agent.Name, desc))
		}
	}

	return b.String()
}
