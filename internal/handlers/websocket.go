package handlers

import (
	"log"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/hub"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"
)

// WebSocketHandler handles WebSocket upgrade requests.
type WebSocketHandler struct {
	hub      *hub.Hub
	db       *gorm.DB
	cfg      *config.Config
	upgrader websocket.FastHTTPUpgrader
}

// NewWebSocketHandler creates a new WebSocketHandler.
func NewWebSocketHandler(h *hub.Hub, db *gorm.DB, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		hub: h,
		db:  db,
		cfg: cfg,
		upgrader: websocket.FastHTTPUpgrader{
			CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
				return true // Allow all origins — CORS is handled at HTTP level.
			},
		},
	}
}

// Handle upgrades an HTTP request to a WebSocket connection.
// Authentication is done via JWT token passed as a query parameter: ?token=...
func (wsh *WebSocketHandler) Handle(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing token query parameter",
		})
	}

	// Parse and validate JWT.
	claims := &middleware.Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "unexpected signing method")
		}
		return []byte(wsh.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid or expired token",
		})
	}

	// Look up the user.
	var user models.User
	if err := wsh.db.First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	if user.Status == "blocked" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "account is blocked",
		})
	}

	// Upgrade to WebSocket.
	err = wsh.upgrader.Upgrade(c.Context(), func(conn *websocket.Conn) {
		client := hub.NewClient(wsh.hub, conn, user.ID)
		wsh.hub.Register(client)

		go client.WritePump()
		client.ReadPump() // Blocks until disconnect.
	})
	if err != nil {
		log.Printf("websocket upgrade error for user %d: %v", user.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "websocket upgrade failed",
		})
	}

	return nil
}
