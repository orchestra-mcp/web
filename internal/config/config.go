package config

import (
	"os"
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

	return &Config{
		DSN:             dsn,
		JWTSecret:       jwtSecret,
		OTPExpiry:       10 * time.Minute,
		MagicLinkExpiry: 15 * time.Minute,
		Env:             env,
		RepoBaseDir:     repoBaseDir,
	}
}
