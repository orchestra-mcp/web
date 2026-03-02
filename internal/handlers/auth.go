package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// deviceAuthRequest represents a pending device authentication request.
type deviceAuthRequest struct {
	DeviceCode string
	UserCode   string
	ExpiresAt  time.Time
	Approved   bool
	UserID     uint
	Token      string
}

// deviceAuthStore is an in-memory store for short-lived device auth requests.
var deviceAuthStore sync.Map

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	db          *gorm.DB
	cfg         *config.Config
	authService *services.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		db:          db,
		cfg:         cfg,
		authService: services.NewAuthService(db, cfg),
	}
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var user models.User
	if err := h.db.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if user.Status == "suspended" || user.Status == "blocked" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account is suspended"})
	}

	token, err := h.authService.GenerateJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(fiber.Map{"ok": true, "token": token, "user": user})
}

// Register handles POST /api/auth/register
func (h *AuthHandler) Register(c fiber.Ctx) error {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Email == "" || body.Password == "" || body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name, email, and password are required"})
	}

	var existing models.User
	if err := h.db.Where("email = ?", body.Email).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already registered"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	user := models.User{
		Name:        body.Name,
		Email:       body.Email,
		Password:    string(hashed),
		Status:      "active",
		PasswordSet: true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}

	// Generate OTP for email verification (fire and forget)
	_, _ = h.authService.GenerateOTP(&user, "email_verification")

	token, err := h.authService.GenerateJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"ok": true, "token": token, "user": user})
}

// Logout handles POST /api/auth/logout
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": true})
}

// Me handles GET /api/auth/me
func (h *AuthHandler) Me(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	return c.JSON(user)
}

// MyRole handles GET /api/auth/me/role
func (h *AuthHandler) MyRole(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	role := user.Role
	if role == "" {
		role = "user"
	}

	// Check if user has a team membership with a higher role.
	if role != "admin" {
		var membership models.Membership
		err := h.db.Where("user_id = ?", user.ID).Order("created_at ASC").First(&membership).Error
		if err == nil {
			// Map membership roles to frontend roles.
			switch membership.Role {
			case "owner":
				role = "team_owner"
			case "admin":
				role = "team_manager"
			}
		}
	}

	return c.JSON(fiber.Map{"role": role})
}

// SendOTP handles POST /api/auth/otp/send
func (h *AuthHandler) SendOTP(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var user models.User
	if err := h.db.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return c.JSON(fiber.Map{"ok": true, "message": "OTP sent if account exists"})
	}

	if _, err := h.authService.GenerateOTP(&user, "login"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send OTP"})
	}

	return c.JSON(fiber.Map{"ok": true, "message": "OTP sent"})
}

// VerifyOTP handles POST /api/auth/otp/verify
func (h *AuthHandler) VerifyOTP(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var otp models.OtpCode
	if err := h.db.Where(
		"email = ? AND code = ? AND type = ? AND used_at IS NULL AND expires_at > ?",
		body.Email, body.Code, body.Type, time.Now(),
	).First(&otp).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired OTP"})
	}

	now := time.Now()
	h.db.Model(&otp).Update("used_at", &now)

	var user models.User
	if err := h.db.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	switch body.Type {
	case "login":
		token, err := h.authService.GenerateJWT(&user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
		}
		return c.JSON(fiber.Map{"ok": true, "token": token, "user": user})

	case "email_verification":
		h.db.Model(&user).Update("email_verified_at", &now)
		return c.JSON(fiber.Map{"ok": true, "user": user})

	case "password_reset":
		return c.JSON(fiber.Map{"ok": true, "verified": true})

	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unknown OTP type"})
	}
}

// SendMagicLink handles POST /api/auth/magic-link/send
func (h *AuthHandler) SendMagicLink(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var user models.User
	if err := h.db.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return c.JSON(fiber.Map{"ok": true})
	}

	if _, err := h.authService.GenerateMagicLink(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate magic link"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// VerifyMagicLink handles POST /api/auth/magic-link/verify
func (h *AuthHandler) VerifyMagicLink(c fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var ml models.MagicLinkToken
	if err := h.db.Where(
		"token = ? AND used_at IS NULL AND expires_at > ?",
		body.Token, time.Now(),
	).First(&ml).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired magic link"})
	}

	now := time.Now()
	h.db.Model(&ml).Update("used_at", &now)

	var user models.User
	if err := h.db.First(&user, ml.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	token, err := h.authService.GenerateJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(fiber.Map{"ok": true, "token": token, "user": user})
}

// ResetPassword handles POST /api/auth/reset-password
func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var body struct {
		Email                string `json:"email"`
		Code                 string `json:"code"`
		Password             string `json:"password"`
		PasswordConfirmation string `json:"password_confirmation"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Password != body.PasswordConfirmation {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "passwords do not match"})
	}

	var otp models.OtpCode
	if err := h.db.Where(
		"email = ? AND code = ? AND type = ? AND used_at IS NULL AND expires_at > ?",
		body.Email, body.Code, "password_reset", time.Now(),
	).First(&otp).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired code"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), 12)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	var user models.User
	if err := h.db.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	h.db.Model(&user).Updates(map[string]interface{}{
		"password":     string(hashed),
		"password_set": true,
	})

	now := time.Now()
	h.db.Model(&otp).Update("used_at", &now)

	token, err := h.authService.GenerateJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(fiber.Map{"ok": true, "token": token, "user": user})
}

// ForgotPassword handles POST /api/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	var user models.User
	if err := h.db.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return c.JSON(fiber.Map{"ok": true, "message": "If the email exists, a reset code has been sent"})
	}

	_, _ = h.authService.GenerateOTP(&user, "password_reset")

	return c.JSON(fiber.Map{"ok": true, "message": "If the email exists, a reset code has been sent"})
}

// APIKeyExchange handles POST /api/auth/api-key-exchange
// Accepts an orch_* API key via X-API-Key header and returns a JWT.
func (h *AuthHandler) APIKeyExchange(c fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "X-API-Key header is required"})
	}

	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	var users []models.User
	if err := h.db.Where("settings LIKE ?", "%"+keyHash+"%").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid API key"})
	}

	for _, user := range users {
		var meta map[string]interface{}
		if json.Unmarshal(user.Settings, &meta) != nil {
			continue
		}
		raw, ok := meta["api_keys"]
		if !ok {
			continue
		}
		b, _ := json.Marshal(raw)
		var keys []struct {
			Hash string `json:"hash"`
		}
		if json.Unmarshal(b, &keys) != nil {
			continue
		}
		for _, k := range keys {
			if k.Hash == keyHash {
				if user.Status == "blocked" {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account is blocked"})
				}
				token, err := h.authService.GenerateJWT(&user)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
				}
				return c.JSON(fiber.Map{"ok": true, "token": token, "user": user})
			}
		}
	}

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid API key"})
}

// DeviceRequest handles POST /api/auth/device/request
// Creates a device auth request for CLI/MCP tools to authenticate via browser.
func (h *AuthHandler) DeviceRequest(c fiber.Ctx) error {
	var body struct {
		DeviceName string `json:"device_name"`
	}
	// body is optional, ignore parse errors
	_ = json.Unmarshal(c.Body(), &body)

	// Generate random device_code (32 hex chars = 16 random bytes)
	deviceCodeBytes := make([]byte, 16)
	if _, err := rand.Read(deviceCodeBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate device code"})
	}
	deviceCode := hex.EncodeToString(deviceCodeBytes)

	// Generate user_code (XXXX-XXXX, uppercase alphanumeric)
	const alphanumeric = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I to avoid confusion
	codeBytes := make([]byte, 8)
	if _, err := rand.Read(codeBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate user code"})
	}
	for i := range codeBytes {
		codeBytes[i] = alphanumeric[int(codeBytes[i])%len(alphanumeric)]
	}
	userCode := fmt.Sprintf("%s-%s", string(codeBytes[:4]), string(codeBytes[4:]))

	req := &deviceAuthRequest{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	deviceAuthStore.Store(deviceCode, req)

	return c.JSON(fiber.Map{
		"device_code":      deviceCode,
		"user_code":        userCode,
		"verification_url": "/cli-auth",
		"expires_in":       300,
	})
}

// DeviceApprove handles POST /api/auth/device/approve (authenticated)
// Links the current user to a pending device auth request by user_code.
func (h *AuthHandler) DeviceApprove(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		UserCode string `json:"user_code"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	userCode := strings.TrimSpace(strings.ToUpper(body.UserCode))
	if userCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_code is required"})
	}

	// Find the device auth request by user_code
	var found *deviceAuthRequest
	deviceAuthStore.Range(func(key, value interface{}) bool {
		req := value.(*deviceAuthRequest)
		if req.UserCode == userCode {
			found = req
			return false // stop iteration
		}
		return true
	})

	if found == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "invalid or expired user code"})
	}

	if time.Now().After(found.ExpiresAt) {
		deviceAuthStore.Delete(found.DeviceCode)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "invalid or expired user code"})
	}

	token, err := h.authService.GenerateJWT(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	found.Approved = true
	found.UserID = user.ID
	found.Token = token

	return c.JSON(fiber.Map{"ok": true})
}

// DevicePoll handles POST /api/auth/device/poll
// MCP/CLI polls with device_code to check if the user has approved.
func (h *AuthHandler) DevicePoll(c fiber.Ctx) error {
	var body struct {
		DeviceCode string `json:"device_code"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.DeviceCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "device_code is required"})
	}

	val, ok := deviceAuthStore.Load(body.DeviceCode)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "expired_or_not_found"})
	}

	req := val.(*deviceAuthRequest)

	if time.Now().After(req.ExpiresAt) {
		deviceAuthStore.Delete(body.DeviceCode)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "expired_or_not_found"})
	}

	if !req.Approved {
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "pending"})
	}

	// Approved — fetch the user and return the token
	var user models.User
	if err := h.db.First(&user, req.UserID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load user"})
	}

	// Clean up the store entry
	deviceAuthStore.Delete(body.DeviceCode)

	return c.JSON(fiber.Map{
		"status": "approved",
		"token":  req.Token,
		"user":   user,
	})
}

// UpdateProfile handles PATCH /api/auth/profile
func (h *AuthHandler) UpdateProfile(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	updates := map[string]interface{}{}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Email != "" && body.Email != user.Email {
		var existing models.User
		if err := h.db.Where("email = ? AND id != ?", body.Email, user.ID).First(&existing).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already in use"})
		}
		updates["email"] = body.Email
	}

	if len(updates) > 0 {
		if err := h.db.Model(user).Updates(updates).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update profile"})
		}
	}

	h.db.First(user, user.ID)
	return c.JSON(user)
}
