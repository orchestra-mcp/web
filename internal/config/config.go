package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	DSN              string
	JWTSecret        string
	OTPExpiry        time.Duration
	MagicLinkExpiry  time.Duration
	Env              string
	RepoBaseDir      string
	UploadDir        string
	AllowedOrigins   []string
}

// Load reads configuration from environment variables and applies defaults.
func Load() *Config {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/orchestra_web?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "orchestra-secret-change-in-production"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	repoBaseDir := os.Getenv("REPO_BASE_DIR")
	if repoBaseDir == "" {
		repoBaseDir = "/var/orchestra/repos"
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "uploads"
	}
	// Resolve to absolute path so file serving works regardless of working directory.
	if !filepath.IsAbs(uploadDir) {
		if abs, err := filepath.Abs(uploadDir); err == nil {
			uploadDir = abs
		}
	}

	return &Config{
		DSN:             dsn,
		JWTSecret:       jwtSecret,
		OTPExpiry:       10 * time.Minute,
		MagicLinkExpiry: 15 * time.Minute,
		Env:             env,
		RepoBaseDir:     repoBaseDir,
		UploadDir:       uploadDir,
		AllowedOrigins:  parseOrigins(os.Getenv("ALLOWED_ORIGINS")),
	}
}

// parseOrigins splits a comma-separated list of origins.
// Returns defaults for local development if the input is empty.
func parseOrigins(raw string) []string {
	if raw == "" {
		return []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:8080",
		}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}
