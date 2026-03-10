package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/database"
	"github.com/orchestra-mcp/web/internal/routes"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbDSN := flag.String("db-dsn", "", "PostgreSQL DSN (overrides DATABASE_URL env var)")
	flag.Parse()

	cfg := config.Load()
	if *dbDSN != "" {
		cfg.DSN = *dbDSN
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	database.SeedDefaults(db)
	database.ScanDocs(db)

	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		AppName:      "Orchestra Web API",
	})

	routes.Register(app, db, cfg)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Orchestra Web API listening on %s", *addr)
		if err := app.Listen(*addr, fiber.ListenConfig{
			DisableStartupMessage: true,
		}); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited gracefully")
}
