package handlers

import (
	"github.com/gofiber/fiber/v3"
)

// TerminalHandler handles terminal WebSocket connections.
type TerminalHandler struct{}

func NewTerminalHandler() *TerminalHandler {
	return &TerminalHandler{}
}

// WS handles GET /api/terminal/ws — WebSocket terminal proxy stub.
func (h *TerminalHandler) WS(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": "Terminal WebSocket is proxied through tunnel connections. Use /api/tunnels/:id/ws instead.",
	})
}
