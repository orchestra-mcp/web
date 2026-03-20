package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
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

// validateTOTP validates a 6-digit TOTP code against a base32-encoded secret.
// It checks the current time step and ±1 step to allow for clock skew.
func validateTOTP(secret string, code string) bool {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	for _, offset := range []int64{0, -1, 1} {
		counter := uint64((now / 30) + offset)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, counter)
		mac := hmac.New(sha1.New, key)
		mac.Write(buf)
		sum := mac.Sum(nil)
		off := sum[len(sum)-1] & 0x0F
		trunc := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7FFFFFFF
		otp := fmt.Sprintf("%06d", trunc%uint32(math.Pow10(6)))
		if otp == code {
			return true
		}
	}
	return false
}

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

	// Cancel pending deletion on login
	if user.Status == "pending_deletion" {
		h.db.Model(&user).Updates(map[string]any{
			"status":                "active",
			"deletion_scheduled_at": nil,
		})
		user.Status = "active"
		user.DeletionScheduledAt = nil
	}

	token, err := h.authService.GenerateJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(h.loginResponse(&user, token))
}

// Register handles POST /api/auth/register
func (h *AuthHandler) Register(c fiber.Ctx) error {
	// Check if registration is allowed
	var generalSetting models.SystemSetting
	if err := h.db.Where("key = ?", "general").First(&generalSetting).Error; err == nil {
		var settings map[string]interface{}
		if json.Unmarshal(generalSetting.Value, &settings) == nil {
			if allow, ok := settings["allow_register"]; ok {
				if allow == false {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "registration is currently disabled"})
				}
			}
		}
	}

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

	return c.Status(fiber.StatusCreated).JSON(h.loginResponse(&user, token))
}

// loginResponse builds a standard login response with optional team_id.
func (h *AuthHandler) loginResponse(user *models.User, token string) fiber.Map {
	resp := fiber.Map{"ok": true, "token": token, "user": user}
	var membership models.Membership
	if err := h.db.Where("user_id = ?", user.ID).Order("created_at ASC").First(&membership).Error; err == nil {
		resp["team_id"] = membership.TeamID
	}
	return resp
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
		return c.JSON(h.loginResponse(&user, token))

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

	return c.JSON(h.loginResponse(&user, token))
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

	return c.JSON(h.loginResponse(&user, token))
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
				return c.JSON(h.loginResponse(&user, token))
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

	resp := fiber.Map{
		"status": "approved",
		"token":  req.Token,
		"user":   user,
	}
	var membership models.Membership
	if err := h.db.Where("user_id = ?", user.ID).Order("created_at ASC").First(&membership).Error; err == nil {
		resp["team_id"] = membership.TeamID
	}
	return c.JSON(resp)
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

// Setup2FA handles POST /api/auth/2fa/setup
func (h *AuthHandler) Setup2FA(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate secret"})
	}
	b32Secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)

	if err := h.db.Model(user).Update("two_factor_secret", b32Secret).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save secret"})
	}

	qrURL := fmt.Sprintf("otpauth://totp/Orchestra:%s?secret=%s&issuer=Orchestra", user.Email, b32Secret)

	return c.JSON(fiber.Map{
		"qr_url": qrURL,
		"secret": b32Secret,
	})
}

// Confirm2FA handles POST /api/auth/2fa/confirm
func (h *AuthHandler) Confirm2FA(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if len(body.Code) != 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code must be 6 digits"})
	}

	// Ensure user has a pending secret
	if err := h.db.First(user, user.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load user"})
	}
	if user.TwoFactorSecret == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "2FA setup not initiated"})
	}

	if !validateTOTP(user.TwoFactorSecret, body.Code) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid verification code"})
	}

	now := time.Now()
	if err := h.db.Model(user).Updates(map[string]interface{}{
		"two_factor_enabled":    true,
		"two_factor_verified_at": &now,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to enable 2FA"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// Disable2FA handles POST /api/auth/2fa/disable
func (h *AuthHandler) Disable2FA(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Reload user to get password hash
	if err := h.db.First(user, user.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load user"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid password"})
	}

	if err := h.db.Model(user).Updates(map[string]interface{}{
		"two_factor_secret":      "",
		"two_factor_enabled":     false,
		"two_factor_verified_at": nil,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to disable 2FA"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// Verify2FA handles POST /api/auth/2fa/verify
func (h *AuthHandler) Verify2FA(c fiber.Ctx) error {
	var body struct {
		Code  string `json:"code"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if len(body.Code) != 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "code must be 6 digits"})
	}

	var user models.User
	if err := h.db.Where("email = ?", body.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if !user.TwoFactorEnabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "2FA is not enabled for this account"})
	}

	if !validateTOTP(user.TwoFactorSecret, body.Code) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid 2FA code"})
	}

	token, err := h.authService.GenerateJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(fiber.Map{"token": token, "user": user})
}

// ChangePassword handles POST /api/auth/change-password
func (h *AuthHandler) ChangePassword(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Reload user to get password hash
	if err := h.db.First(user, user.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load user"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "current password is incorrect"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), 12)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	if err := h.db.Model(user).Updates(map[string]interface{}{
		"password":     string(hashed),
		"password_set": true,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update password"})
	}

	return c.JSON(fiber.Map{"ok": true})
}

// DeleteAccount handles DELETE /api/auth/account
func (h *AuthHandler) DeleteAccount(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "password is required"})
	}

	// Reload user to get password hash
	if err := h.db.First(user, user.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load user"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid password"})
	}

	// Schedule deletion 7 days from now
	deletionTime := time.Now().Add(7 * 24 * time.Hour)
	if err := h.db.Model(user).Updates(map[string]any{
		"status":                "pending_deletion",
		"deletion_scheduled_at": &deletionTime,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to schedule deletion"})
	}

	return c.JSON(fiber.Map{
		"ok":                    true,
		"message":               "Account scheduled for deletion. Log in within 7 days to cancel.",
		"deletion_scheduled_at": deletionTime.Format(time.RFC3339),
	})
}

// CleanupDeletedAccounts permanently deletes users whose deletion grace period has expired.
// Call this on server start and periodically (e.g. daily).
func CleanupDeletedAccounts(db *gorm.DB) {
	var users []models.User
	db.Where("status = ? AND deletion_scheduled_at < ?", "pending_deletion", time.Now()).Find(&users)

	for _, u := range users {
		uid := u.ID
		// Cascade delete user data
		db.Where("user_id = ?", uid).Delete(&models.Passkey{})
		db.Where("user_id = ?", uid).Delete(&models.OAuthAccount{})
		db.Where("user_id = ?", uid).Delete(&models.MagicLinkToken{})
		db.Where("user_id = ?", uid).Delete(&models.OtpCode{})
		db.Where("user_id = ?", uid).Delete(&models.DeviceToken{})
		db.Where("user_id = ?", uid).Delete(&models.Tunnel{})
		// Hard delete the user (bypasses soft-delete)
		db.Unscoped().Delete(&models.User{}, uid)
	}
}
