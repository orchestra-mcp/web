package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/orchestra-mcp/web/internal/middleware"
	"gorm.io/gorm"
)

// PowerSyncHandler provides endpoints for PowerSync client authentication:
//   - GET  /api/powersync/keys  — JWKS public key (PowerSync verifies client tokens)
//   - POST /api/powersync/token — Issue a PowerSync JWT for the authenticated user

type PowerSyncHandler struct {
	db      *gorm.DB
	keyOnce sync.Once
	privKey *rsa.PrivateKey
	kid     string
}

func NewPowerSyncHandler(db *gorm.DB) *PowerSyncHandler {
	return &PowerSyncHandler{db: db}
}

// ensureKey loads the persistent RSA key from disk, or generates and saves one.
// The key is stored at $HOME/.orchestra/powersync-key.pem so it survives restarts.
func (h *PowerSyncHandler) ensureKey() {
	h.keyOnce.Do(func() {
		keyPath := persistentKeyPath()
		if key, err := loadKey(keyPath); err == nil {
			h.privKey = key
			h.kid = "powersync-1"
			return
		}
		// Generate a new key and persist it.
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic("failed to generate RSA key for PowerSync: " + err.Error())
		}
		if err := saveKey(keyPath, key); err != nil {
			// Non-fatal — key won't survive restarts but will work for this session.
			fmt.Printf("[PowerSync] Warning: could not persist key to %s: %v\n", keyPath, err)
		}
		h.privKey = key
		h.kid = "powersync-1"
	})
}

func persistentKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	return filepath.Join(home, ".orchestra", "powersync-key.pem")
}

func loadKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func saveKey(path string, key *rsa.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// Keys returns the JWKS (JSON Web Key Set) containing the RSA public key.
// PowerSync service calls this to verify client JWTs.
// GET /api/powersync/keys
func (h *PowerSyncHandler) Keys(c fiber.Ctx) error {
	h.ensureKey()
	pub := &h.privKey.PublicKey

	jwks := fiber.Map{
		"keys": []fiber.Map{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": h.kid,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
	return c.JSON(jwks)
}

// Token issues a short-lived PowerSync JWT for the authenticated user.
// The client passes this token to PowerSync to authenticate the sync connection.
// POST /api/powersync/token
func (h *PowerSyncHandler) Token(c fiber.Ctx) error {
	h.ensureKey()

	// Get authenticated user from middleware.
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	now := time.Now()
	psClaims := jwt.MapClaims{
		"sub":     fmt.Sprintf("%d", user.ID),
		"iat":     now.Unix(),
		"exp":     now.Add(1 * time.Hour).Unix(),
		"aud":     "powersync",
		"user_id": fmt.Sprintf("%d", user.ID),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, psClaims)
	token.Header["kid"] = h.kid

	signed, err := token.SignedString(h.privKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to sign token",
		})
	}

	return c.JSON(fiber.Map{
		"token":      signed,
		"expires_at": now.Add(1 * time.Hour).Unix(),
	})
}
