package services

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// WorkspaceManager handles cloning repos, running prompts via Claude Code,
// and auto-syncing changes back to GitHub.
type WorkspaceManager struct {
	db      *gorm.DB
	mu      sync.RWMutex
	baseDir string
}

// NewWorkspaceManager creates a manager using baseDir for cloned repos.
func NewWorkspaceManager(db *gorm.DB, baseDir string) *WorkspaceManager {
	if baseDir == "" {
		baseDir = "/var/orchestra/repos"
	}
	os.MkdirAll(baseDir, 0755)
	return &WorkspaceManager{db: db, baseDir: baseDir}
}

// CloneRepo clones a GitHub repo to the local filesystem.
func (m *WorkspaceManager) CloneRepo(ws *models.RepoWorkspace, token string) error {
	cloneDir := filepath.Join(m.baseDir, ws.ID)
	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, ws.RepoOwner, ws.RepoName)

	m.db.Model(ws).Updates(map[string]any{"status": "cloning", "error": ""})

	args := []string{"clone", "--depth", "1", "--branch", ws.Branch, cloneURL, cloneDir}
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		m.db.Model(ws).Updates(map[string]any{"status": "error", "error": errMsg})
		return fmt.Errorf("git clone: %s", errMsg)
	}

	// Get HEAD commit SHA.
	sha := m.gitSHA(cloneDir)

	// Initialize orchestra MCP in the workspace.
	initCmd := exec.Command("orchestra", "init", "--workspace", cloneDir)
	initCmd.Dir = cloneDir
	_ = initCmd.Run() // best-effort

	m.db.Model(ws).Updates(map[string]any{
		"status":     "ready",
		"clone_path": cloneDir,
		"commit_sha": sha,
	})

	return nil
}

// RunPrompt executes a Claude Code prompt in the workspace directory.
func (m *WorkspaceManager) RunPrompt(ws *models.RepoWorkspace, prompt string) (string, error) {
	if ws.Status != "ready" {
		return "", fmt.Errorf("workspace not ready (status: %s)", ws.Status)
	}

	cloneDir := filepath.Join(m.baseDir, ws.ID)
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		return "", fmt.Errorf("workspace directory not found")
	}

	cmd := exec.Command("claude", "-p", prompt, "--output-format", "text")
	cmd.Dir = cloneDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude: %s (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// SyncToGitHub stages, commits, and pushes changes.
func (m *WorkspaceManager) SyncToGitHub(ws *models.RepoWorkspace, token string) error {
	cloneDir := filepath.Join(m.baseDir, ws.ID)

	// Check for changes.
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = cloneDir
	out, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return nil // nothing to sync
	}

	m.db.Model(ws).Update("status", "syncing")

	// Set remote URL with token for push.
	remoteURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, ws.RepoOwner, ws.RepoName)
	m.gitExec(cloneDir, "remote", "set-url", "origin", remoteURL)

	// Stage, commit, push.
	m.gitExec(cloneDir, "add", "-A")

	commitCmd := exec.Command("git", "commit", "-m", "chore: auto-sync from Orchestra")
	commitCmd.Dir = cloneDir
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Orchestra Bot",
		"GIT_AUTHOR_EMAIL=bot@orchestra-mcp.dev",
		"GIT_COMMITTER_NAME=Orchestra Bot",
		"GIT_COMMITTER_EMAIL=bot@orchestra-mcp.dev",
	)
	_ = commitCmd.Run()

	pushCmd := exec.Command("git", "push", "origin", ws.Branch)
	pushCmd.Dir = cloneDir
	var pushErr bytes.Buffer
	pushCmd.Stderr = &pushErr
	if err := pushCmd.Run(); err != nil {
		m.db.Model(ws).Update("status", "ready")
		return fmt.Errorf("git push: %s", strings.TrimSpace(pushErr.String()))
	}

	now := time.Now()
	sha := m.gitSHA(cloneDir)
	m.db.Model(ws).Updates(map[string]any{
		"status":       "ready",
		"last_sync_at": now,
		"commit_sha":   sha,
	})

	return nil
}

// PullFromGitHub pulls latest changes from remote.
func (m *WorkspaceManager) PullFromGitHub(ws *models.RepoWorkspace, token string) error {
	cloneDir := filepath.Join(m.baseDir, ws.ID)

	remoteURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, ws.RepoOwner, ws.RepoName)
	m.gitExec(cloneDir, "remote", "set-url", "origin", remoteURL)

	pullCmd := exec.Command("git", "pull", "--rebase", "origin", ws.Branch)
	pullCmd.Dir = cloneDir
	var stderr bytes.Buffer
	pullCmd.Stderr = &stderr
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("git pull: %s", strings.TrimSpace(stderr.String()))
	}

	sha := m.gitSHA(cloneDir)
	m.db.Model(ws).Update("commit_sha", sha)
	return nil
}

// DeleteWorkspace removes the cloned directory.
func (m *WorkspaceManager) DeleteWorkspace(ws *models.RepoWorkspace) error {
	cloneDir := filepath.Join(m.baseDir, ws.ID)
	return os.RemoveAll(cloneDir)
}

// AutoSyncLoop runs in the background, syncing dirty workspaces every 5 minutes.
func (m *WorkspaceManager) AutoSyncLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			m.syncAll()
		}
	}
}

func (m *WorkspaceManager) syncAll() {
	var workspaces []models.RepoWorkspace
	m.db.Where("status = ?", "ready").Find(&workspaces)

	for _, ws := range workspaces {
		cloneDir := filepath.Join(m.baseDir, ws.ID)

		// Check if dirty.
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Dir = cloneDir
		out, err := statusCmd.Output()
		if err != nil || len(strings.TrimSpace(string(out))) == 0 {
			continue
		}

		// Need the user's GitHub token.
		var oauth models.OAuthAccount
		if err := m.db.Where("user_id = ? AND provider = ?", ws.UserID, "github").First(&oauth).Error; err != nil {
			continue
		}

		if err := m.SyncToGitHub(&ws, oauth.AccessToken); err != nil {
			log.Printf("[workspace-manager] sync %s failed: %v", ws.ID, err)
		}
	}
}

func (m *WorkspaceManager) gitExec(dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	_ = cmd.Run()
}

func (m *WorkspaceManager) gitSHA(dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
