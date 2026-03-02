# Orchestra Web API

Go backend API server for the Orchestra platform, built with [Fiber v3](https://gofiber.io/), [GORM](https://gorm.io/), and PostgreSQL.

Part of the [Orchestra MCP](https://github.com/orchestra-mcp) framework.

## Prerequisites

- **Go 1.23+**
- **PostgreSQL 15+**

## Quick Start

```bash
# Build the binary
go build -o orchestra-web ./cmd/

# Run with default settings (listens on :8080)
./orchestra-web

# Run with flags
./orchestra-web --port 3000 --db-url "postgres://user:pass@localhost:5432/orchestra?sslmode=disable"
```

Set configuration via environment variables or flags:

| Variable       | Flag       | Default                                                              | Description          |
|----------------|------------|----------------------------------------------------------------------|----------------------|
| `PORT`         | `--port`   | `8080`                                                               | HTTP listen port     |
| `DATABASE_URL` | `--db-url` | `postgres://orchestra:orchestra@localhost:5432/orchestra?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET`   | `--jwt-secret` | _(required)_                                                     | Secret for JWT signing |

## Project Structure

```
apps/web/
├── cmd/
│   └── main.go            # Binary entry point, graceful shutdown
├── internal/
│   ├── config/            # Configuration loading (env, flags)
│   ├── database/          # GORM database connection and migrations
│   ├── handlers/          # HTTP handlers (controllers)
│   ├── middleware/         # Fiber middleware (JWT auth, logging, CORS)
│   ├── models/            # GORM models
│   ├── routes/            # Route registration
│   └── services/          # Business logic
├── go.mod                 # Module: github.com/orchestra-mcp/web
├── go.sum
└── README.md
```

## Development

```bash
# Run tests
go test ./...

# Run with hot reload (requires air)
air

# Vet
go vet ./...
```

## License

MIT License. See [LICENSE](LICENSE) for details.
