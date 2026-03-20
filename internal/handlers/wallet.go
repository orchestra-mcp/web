package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/services"
)

// WalletHandler handles HTTP requests for the wallet and points system.
type WalletHandler struct {
	svc services.WalletService
}

// NewWalletHandler creates a WalletHandler with the given service.
func NewWalletHandler(svc services.WalletService) *WalletHandler {
	return &WalletHandler{svc: svc}
}

// GetWallet returns the wallet balance and recent transactions for the authenticated user.
// GET /api/wallet
func (h *WalletHandler) GetWallet(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "authentication required",
		})
	}

	wallet, err := h.svc.GetWallet(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	// Include 5 most recent transactions.
	txns, _, err := h.svc.GetTransactions(c.Context(), user.ID, 5, 0)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"wallet":              wallet,
			"recent_transactions": txns,
		},
	})
}

// ListTransactions returns paginated transaction history for the authenticated user.
// GET /api/wallet/transactions?limit=20&offset=0
func (h *WalletHandler) ListTransactions(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "authentication required",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	txns, total, err := h.svc.GetTransactions(c.Context(), user.ID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data":  txns,
		"total": total,
	})
}

// AdminGrantPoints adds points to a user's wallet (admin only).
// POST /api/admin/wallet/:userId/grant
func (h *WalletHandler) AdminGrantPoints(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "admin access required",
		})
	}

	userID, err := strconv.ParseUint(c.Params("userId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_params",
			"message": "invalid user ID",
		})
	}

	var req struct {
		Amount      int    `json:"amount"`
		Description string `json:"description"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "invalid_body",
			"message": err.Error(),
		})
	}
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_amount",
			"message": "amount must be positive",
		})
	}

	wallet, tx, err := h.svc.AddPoints(c.Context(), uint(userID), req.Amount, "admin_grant", "admin", req.Description)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": fiber.Map{
			"wallet":      wallet,
			"transaction": tx,
		},
	})
}

// AdminDeductPoints removes points from a user's wallet (admin only).
// POST /api/admin/wallet/:userId/deduct
func (h *WalletHandler) AdminDeductPoints(c fiber.Ctx) error {
	if !isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "admin access required",
		})
	}

	userID, err := strconv.ParseUint(c.Params("userId"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_params",
			"message": "invalid user ID",
		})
	}

	var req struct {
		Amount      int    `json:"amount"`
		Description string `json:"description"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "invalid_body",
			"message": err.Error(),
		})
	}
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_amount",
			"message": "amount must be positive",
		})
	}

	wallet, tx, err := h.svc.AddPoints(c.Context(), uint(userID), -req.Amount, "admin_deduct", "admin", req.Description)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"wallet":      wallet,
			"transaction": tx,
		},
	})
}
