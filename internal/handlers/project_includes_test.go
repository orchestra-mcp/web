package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/orchestra-mcp/web/internal/models"
)

// ── buildClaudeMD ─────────────────────────────────────────────────────────────

func TestBuildClaudeMD_NoSkillsNoAgents(t *testing.T) {
	out := buildClaudeMD("My Project", nil, nil)
	if !strings.Contains(out, "# CLAUDE.md") {
		t.Error("expected CLAUDE.md header")
	}
	if !strings.Contains(out, "My Project") {
		t.Error("expected project name")
	}
	if !strings.Contains(out, "No skills configured") {
		t.Error("expected no-skills message")
	}
	if !strings.Contains(out, "No agents configured") {
		t.Error("expected no-agents message")
	}
}

func TestBuildClaudeMD_WithSkills(t *testing.T) {
	skills := []string{"docs", "qa-testing", "typescript-react"}
	out := buildClaudeMD("My Project", skills, nil)

	for _, s := range skills {
		if !strings.Contains(out, "/"+s) {
			t.Errorf("expected slash command /%s in output", s)
		}
		if !strings.Contains(out, ".claude/skills/"+s+"/") {
			t.Errorf("expected skill path .claude/skills/%s/ in output", s)
		}
	}
}

func TestBuildClaudeMD_WithAgents(t *testing.T) {
	agents := []string{"devops", "qa-playwright"}
	out := buildClaudeMD("My Project", nil, agents)

	for _, a := range agents {
		if !strings.Contains(out, a) {
			t.Errorf("expected agent %s in output", a)
		}
		if !strings.Contains(out, ".claude/agents/"+a+".md") {
			t.Errorf("expected agent file .claude/agents/%s.md in output", a)
		}
	}
}

func TestBuildClaudeMD_ContainsMandatoryWorkflowRule(t *testing.T) {
	out := buildClaudeMD("P", nil, nil)
	if !strings.Contains(out, "Mandatory Workflow Rule") {
		t.Error("expected Mandatory Workflow Rule section")
	}
	if !strings.Contains(out, "create_feature") {
		t.Error("expected create_feature mention")
	}
	if !strings.Contains(out, "advance_feature") {
		t.Error("expected advance_feature mention")
	}
}

func TestBuildClaudeMD_ContainsGitSync(t *testing.T) {
	out := buildClaudeMD("P", nil, nil)
	if !strings.Contains(out, "Git & Sync") {
		t.Error("expected Git & Sync section")
	}
	if !strings.Contains(out, "git_quick_commit") {
		t.Error("expected git_quick_commit mention")
	}
}

func TestBuildClaudeMD_SkillsPreserveOrder(t *testing.T) {
	// buildClaudeMD renders skills in the order given (caller sorts before calling)
	skills := []string{"a-skill", "m-skill", "z-skill"}
	out := buildClaudeMD("P", skills, nil)
	aIdx := strings.Index(out, "/a-skill")
	mIdx := strings.Index(out, "/m-skill")
	zIdx := strings.Index(out, "/z-skill")
	if !(aIdx < mIdx && mIdx < zIdx) {
		t.Error("expected skills to appear in given order")
	}
}

// ── buildAgentsMD ─────────────────────────────────────────────────────────────

func TestBuildAgentsMD_NoAgents(t *testing.T) {
	out := buildAgentsMD(nil)
	if !strings.Contains(out, "# AGENTS.md") {
		t.Error("expected AGENTS.md header")
	}
	if !strings.Contains(out, "No agents configured") {
		t.Error("expected no-agents message")
	}
}

func TestBuildAgentsMD_WithAgents(t *testing.T) {
	agents := []string{"devops", "scrum-master"}
	out := buildAgentsMD(agents)
	for _, a := range agents {
		if !strings.Contains(out, "## "+a) {
			t.Errorf("expected ## %s section header", a)
		}
		if !strings.Contains(out, ".claude/agents/"+a+".md") {
			t.Errorf("expected link to .claude/agents/%s.md", a)
		}
	}
}

// ── buildContextMD ────────────────────────────────────────────────────────────

func TestBuildContextMD_Header(t *testing.T) {
	project := models.Project{Name: "Orchestra", Slug: "orchestra-agents"}
	out := buildContextMD(project, nil, nil, nil)
	if !strings.Contains(out, "# CONTEXT.md") {
		t.Error("expected CONTEXT.md header")
	}
	if !strings.Contains(out, "Orchestra") {
		t.Error("expected project name")
	}
}

func TestBuildContextMD_NoFeatures(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	out := buildContextMD(project, nil, nil, nil)
	if !strings.Contains(out, "No active features") {
		t.Error("expected no-active-features message")
	}
}

func TestBuildContextMD_ProjectHealth(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	features := []models.Feature{
		{Status: "done"},
		{Status: "done"},
		{Status: "in-progress"},
		{Status: "todo"},
	}
	out := buildContextMD(project, features, nil, nil)
	if !strings.Contains(out, "Total features:** 4") {
		t.Error("expected total features = 4")
	}
	if !strings.Contains(out, "Done:** 2") {
		t.Error("expected done = 2")
	}
	if !strings.Contains(out, "In Progress:** 1") {
		t.Error("expected in-progress = 1")
	}
	if !strings.Contains(out, "To Do:** 1") {
		t.Error("expected todo = 1")
	}
}

func TestBuildContextMD_DonePercentage(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	features := []models.Feature{
		{Status: "done"},
		{Status: "todo"},
		{Status: "todo"},
		{Status: "todo"},
	}
	out := buildContextMD(project, features, nil, nil)
	// 1/4 = 25%
	if !strings.Contains(out, "25%") {
		t.Error("expected 25% completion")
	}
}

func TestBuildContextMD_ActiveFeaturesTable(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	features := []models.Feature{
		{FeatureID: "FEAT-ABC", Title: "Active one", Status: "in-progress", Priority: "P1", Assignee: "alice"},
		{FeatureID: "FEAT-XYZ", Title: "Done one", Status: "done", Priority: "P2"},
	}
	out := buildContextMD(project, features, nil, nil)
	// Active feature should appear
	if !strings.Contains(out, "FEAT-ABC") {
		t.Error("expected active feature FEAT-ABC in table")
	}
	// Done feature should NOT appear in active table
	if strings.Contains(out, "FEAT-XYZ") {
		t.Error("expected done feature FEAT-XYZ to be excluded from active table")
	}
}

func TestBuildContextMD_SkillStacksDetected(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	stacksJSON, _ := json.Marshal([]string{"go", "react"})
	skills := []models.Skill{
		{Name: "My Skill", Slug: "my-skill", Stacks: stacksJSON},
	}
	out := buildContextMD(project, nil, skills, nil)
	if !strings.Contains(out, "go") {
		t.Error("expected 'go' stack detected")
	}
	if !strings.Contains(out, "react") {
		t.Error("expected 'react' stack detected")
	}
}

func TestBuildContextMD_NoSkillsMessage(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	out := buildContextMD(project, nil, nil, nil)
	if !strings.Contains(out, "No skills configured") {
		t.Error("expected no-skills message")
	}
}

func TestBuildContextMD_NoAgentsMessage(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	out := buildContextMD(project, nil, nil, nil)
	if !strings.Contains(out, "No agents configured") {
		t.Error("expected no-agents message")
	}
}

func TestBuildContextMD_IncludedSkillsListed(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	skills := []models.Skill{
		{Name: "TypeScript React", Slug: "typescript-react", Description: "React with TS"},
	}
	out := buildContextMD(project, nil, skills, nil)
	if !strings.Contains(out, "TypeScript React") {
		t.Error("expected skill name in output")
	}
	if !strings.Contains(out, "React with TS") {
		t.Error("expected skill description in output")
	}
}

func TestBuildContextMD_IncludedAgentsListed(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	agents := []models.Agent{
		{Name: "DevOps Engineer", Slug: "devops", Description: "Docker and CI/CD"},
	}
	out := buildContextMD(project, nil, nil, agents)
	if !strings.Contains(out, "DevOps Engineer") {
		t.Error("expected agent name in output")
	}
	if !strings.Contains(out, "Docker and CI/CD") {
		t.Error("expected agent description in output")
	}
}

func TestBuildContextMD_ZeroFeaturesPercentage(t *testing.T) {
	project := models.Project{Name: "P", Slug: "p"}
	// No features — should not divide by zero
	out := buildContextMD(project, nil, nil, nil)
	if !strings.Contains(out, "Total features:** 0") {
		t.Error("expected total features = 0")
	}
}
