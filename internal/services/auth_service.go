package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"gorm.io/gorm"
)

// AuthService handles authentication business logic.
type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewAuthService creates a new AuthService.
func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

// GenerateJWT creates a signed JWT token for the given user.
func (s *AuthService) GenerateJWT(user *models.User) (string, error) {
	claims := middleware.Claims{
		UserID:    user.ID,
		Email:     user.Email,
		Abilities: []string{"ide:access"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// GenerateOTP creates a 6-digit OTP code and stores it in the database.
func (s *AuthService) GenerateOTP(user *models.User, otpType string) (string, error) {
	// Generate 6-digit numeric code
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", n.Int64())

	otp := models.OtpCode{
		UserID:    user.ID,
		Email:     user.Email,
		Code:      code,
		Type:      otpType,
		ExpiresAt: time.Now().Add(s.cfg.OTPExpiry),
	}

	if err := s.db.Create(&otp).Error; err != nil {
		return "", err
	}

	log.Printf("OTP for %s: %s", user.Email, code)
	return code, nil
}

// GenerateMagicLink creates a random 64-char token and stores it.
func (s *AuthService) GenerateMagicLink(user *models.User) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	ml := models.MagicLinkToken{
		UserID:    user.ID,
		Email:     user.Email,
		Token:     token,
		ExpiresAt: time.Now().Add(s.cfg.MagicLinkExpiry),
	}

	if err := s.db.Create(&ml).Error; err != nil {
		return "", err
	}

	log.Printf("Magic link token for %s: %s", user.Email, token)
	return token, nil
}
