package database

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// ScanDocs reads markdown files from the docs/ directory and upserts them
// into the docs table as system-level docs (user_id=0, project_slug="_system").
// Safe to call on every boot — uses content hash to skip unchanged files.
func ScanDocs(db *gorm.DB) {
	docsDir := os.Getenv("DOCS_DIR")
	if docsDir == "" {
		// Try common locations in order of priority.
		candidates := []string{
			"../../docs",                  // dev: running from apps/web/
			"/opt/orchestra/repo/docs",    // production: repo checkout
			"docs",                        // fallback: current directory
		}
		for _, c := range candidates {
			abs, err := filepath.Abs(c)
			if err != nil {
				continue
			}
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				docsDir = abs
				break
			}
		}
		if docsDir == "" {
			log.Printf("docs-scanner: no docs directory found")
			return
		}
	}

	absDir, err := filepath.Abs(docsDir)
	if err != nil {
		log.Printf("docs-scanner: failed to resolve docs dir: %v", err)
		return
	}

	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		log.Printf("docs-scanner: docs dir not found: %s", absDir)
		return
	}

	var count int
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable files
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("docs-scanner: failed to read %s: %v", path, err)
			return nil
		}

		// Derive slug from relative path: "artifacts/00-architecture-overview.md" -> "artifacts--00-architecture-overview"
		relPath, _ := filepath.Rel(absDir, path)
		relPath = filepath.ToSlash(relPath)
		slug := strings.TrimSuffix(relPath, ".md")
		slug = strings.ReplaceAll(slug, "/", "--")

		// Derive category from subdirectory
		parts := strings.Split(relPath, "/")
		category := ""
		if len(parts) > 1 {
			category = parts[0]
		}

		// Extract title from first H1 or filename
		title := extractTitle(string(content), info.Name())

		// Content hash to detect changes
		hash := fmt.Sprintf("%x", md5.Sum(content))

		// Check if exists and unchanged
		var existing models.Doc
		result := db.Where("doc_id = ? AND user_id = 0", slug).First(&existing)
		if result.Error == nil {
			// Exists — check if content changed via meta hash
			if existing.Body == string(content) {
				return nil // unchanged, skip
			}
			// Update
			existing.Title = title
			existing.Category = category
			existing.Body = string(content)
			existing.Version = existing.Version + 1
			_ = hash // used for future optimization
			if err := db.Save(&existing).Error; err != nil {
				log.Printf("docs-scanner: failed to update %s: %v", slug, err)
			} else {
				count++
			}
			return nil
		}

		// Create new
		doc := models.Doc{
			UserID:      0,
			ProjectSlug: "_system",
			DocID:       slug,
			Title:       title,
			Category:    category,
			Body:        string(content),
			Version:     1,
		}
		if err := db.Create(&doc).Error; err != nil {
			log.Printf("docs-scanner: failed to create %s: %v", slug, err)
		} else {
			count++
		}
		return nil
	})

	if err != nil {
		log.Printf("docs-scanner: walk error: %v", err)
	}
	if count > 0 {
		log.Printf("docs-scanner: upserted %d docs from %s", count, absDir)
	}
}

// extractTitle pulls the first H1 heading from markdown, or falls back to filename.
func extractTitle(content, filename string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	// Fallback: clean filename
	name := strings.TrimSuffix(filename, ".md")
	name = strings.TrimLeft(name, "0123456789-")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	if len(name) > 0 {
		return strings.ToUpper(name[:1]) + name[1:]
	}
	return filename
}
