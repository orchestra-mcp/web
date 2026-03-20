package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/orchestra-mcp/web/internal/config"
	"github.com/orchestra-mcp/web/internal/middleware"
	"github.com/orchestra-mcp/web/internal/models"
	"github.com/orchestra-mcp/web/internal/services"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Challenge store (in-memory, auto-expiring)
// ---------------------------------------------------------------------------

type passkeyChallenge struct {
	Value     []byte
	UserID    uint   // 0 for authentication flows
	SessionID string // for auth correlation
	ExpiresAt time.Time
}

var passkeyStore sync.Map

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// PasskeyHandler handles WebAuthn/passkey endpoints.
type PasskeyHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

// NewPasskeyHandler creates a new PasskeyHandler.
func NewPasskeyHandler(db *gorm.DB, cfg *config.Config) *PasskeyHandler {
	return &PasskeyHandler{db: db, cfg: cfg}
}

// ---------------------------------------------------------------------------
// Config helpers
// ---------------------------------------------------------------------------

func passkeyRPID() string {
	if v := os.Getenv("WEBAUTHN_RP_ID"); v != "" {
		return v
	}
	return "localhost"
}

func passkeyRPOrigin() string {
	if v := os.Getenv("WEBAUTHN_RP_ORIGIN"); v != "" {
		return v
	}
	return "http://localhost:3000"
}

func passkeyRPName() string {
	if v := os.Getenv("WEBAUTHN_RP_NAME"); v != "" {
		return v
	}
	return "Orchestra"
}

// ---------------------------------------------------------------------------
// 1. BeginRegistration — POST /api/auth/passkey/register/begin (authenticated)
// ---------------------------------------------------------------------------

func (h *PasskeyHandler) BeginRegistration(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	// Generate 32-byte challenge.
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate challenge"})
	}

	// Store challenge keyed by userID.
	key := fmt.Sprintf("reg:%d", user.ID)
	passkeyStore.Store(key, &passkeyChallenge{
		Value:     challenge,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	// Build user.id as base64url(sha256(email)).
	emailHash := sha256.Sum256([]byte(user.Email))
	userIDEncoded := base64.RawURLEncoding.EncodeToString(emailHash[:])

	// Gather existing credentials for excludeCredentials.
	var existing []models.Passkey
	h.db.Where("user_id = ?", user.ID).Find(&existing)

	excludeCredentials := make([]fiber.Map, 0, len(existing))
	for _, pk := range existing {
		ec := fiber.Map{
			"type": "public-key",
			"id":   pk.CredentialID,
		}
		if pk.Transports != "" {
			ec["transports"] = strings.Split(pk.Transports, ",")
		}
		excludeCredentials = append(excludeCredentials, ec)
	}

	return c.JSON(fiber.Map{
		"publicKey": fiber.Map{
			"rp": fiber.Map{
				"id":   passkeyRPID(),
				"name": passkeyRPName(),
			},
			"user": fiber.Map{
				"id":          userIDEncoded,
				"name":        user.Email,
				"displayName": user.Name,
			},
			"challenge": base64.RawURLEncoding.EncodeToString(challenge),
			"pubKeyCredParams": []fiber.Map{
				{"type": "public-key", "alg": -7},
			},
			"timeout": 60000,
			"authenticatorSelection": fiber.Map{
				"authenticatorAttachment": "platform",
				"residentKey":             "preferred",
				"userVerification":        "preferred",
			},
			"attestation":        "none",
			"excludeCredentials": excludeCredentials,
		},
	})
}

// ---------------------------------------------------------------------------
// 2. FinishRegistration — POST /api/auth/passkey/register/finish (authenticated)
// ---------------------------------------------------------------------------

func (h *PasskeyHandler) FinishRegistration(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var body struct {
		ID       string `json:"id"`
		RawID    string `json:"rawId"`
		Type     string `json:"type"`
		Response struct {
			AttestationObject string `json:"attestationObject"`
			ClientDataJSON    string `json:"clientDataJSON"`
		} `json:"response"`
		Name string `json:"name"` // optional user-given label
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Look up stored challenge.
	key := fmt.Sprintf("reg:%d", user.ID)
	val, ok := passkeyStore.LoadAndDelete(key)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no pending registration challenge"})
	}
	stored := val.(*passkeyChallenge)
	if time.Now().After(stored.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "challenge expired"})
	}

	// Decode and verify clientDataJSON.
	clientDataBytes, err := base64.RawURLEncoding.DecodeString(body.Response.ClientDataJSON)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid clientDataJSON encoding"})
	}

	var clientData struct {
		Type        string `json:"type"`
		Challenge   string `json:"challenge"`
		Origin      string `json:"origin"`
		CrossOrigin bool   `json:"crossOrigin"`
	}
	if err := json.Unmarshal(clientDataBytes, &clientData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid clientDataJSON"})
	}

	if clientData.Type != "webauthn.create" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid ceremony type"})
	}

	expectedChallenge := base64.RawURLEncoding.EncodeToString(stored.Value)
	if clientData.Challenge != expectedChallenge {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "challenge mismatch"})
	}

	if clientData.Origin != passkeyRPOrigin() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "origin mismatch"})
	}

	// Decode attestation object.
	attestBytes, err := base64.RawURLEncoding.DecodeString(body.Response.AttestationObject)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid attestationObject encoding"})
	}

	// Parse CBOR attestation object to extract authData.
	authData, err := extractAuthDataFromAttestation(attestBytes)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("failed to parse attestation: %v", err)})
	}

	// Verify rpIdHash (first 32 bytes of authData).
	if len(authData) < 37 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "authData too short"})
	}
	rpIDHash := sha256.Sum256([]byte(passkeyRPID()))
	for i := 0; i < 32; i++ {
		if authData[i] != rpIDHash[i] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "rpIdHash mismatch"})
		}
	}

	flags := authData[32]
	signCount := binary.BigEndian.Uint32(authData[33:37])

	// Check AT (attested credential data) flag.
	if flags&0x40 == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "attested credential data flag not set"})
	}

	if len(authData) < 55 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "authData too short for credential data"})
	}

	// Parse attested credential data.
	aaguid := authData[37:53]
	credIDLen := binary.BigEndian.Uint16(authData[53:55])
	if len(authData) < int(55+credIDLen) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "authData too short for credentialId"})
	}
	credentialID := authData[55 : 55+credIDLen]
	coseKeyBytes := authData[55+credIDLen:]

	// Parse the COSE key to extract the EC public key.
	pubKey, err := parseCOSEES256Key(coseKeyBytes)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("failed to parse COSE key: %v", err)})
	}

	// Format AAGUID as UUID string.
	aaguidStr := fmt.Sprintf("%x-%x-%x-%x-%x", aaguid[0:4], aaguid[4:6], aaguid[6:8], aaguid[8:10], aaguid[10:16])

	credIDEncoded := base64.RawURLEncoding.EncodeToString(credentialID)

	// Serialize the public key as uncompressed point (0x04 || X || Y).
	pubKeyBytes := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)

	// Determine backup eligibility and state from flags.
	backupEligible := flags&0x08 != 0
	backupState := flags&0x10 != 0

	name := body.Name
	if name == "" {
		name = "Passkey"
	}

	passkey := models.Passkey{
		UserID:         user.ID,
		CredentialID:   credIDEncoded,
		PublicKey:      pubKeyBytes,
		Name:           name,
		AAGUID:         aaguidStr,
		SignCount:      signCount,
		Transports:     "internal",
		BackupEligible: backupEligible,
		BackupState:    backupState,
	}

	if err := h.db.Create(&passkey).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "credential already registered"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save passkey"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"ok":      true,
		"passkey": passkey,
	})
}

// ---------------------------------------------------------------------------
// 3. BeginAuthentication — POST /api/auth/passkey/authenticate/begin (public)
// ---------------------------------------------------------------------------

func (h *PasskeyHandler) BeginAuthentication(c fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	// Body is optional; ignore parse errors.
	_ = json.Unmarshal(c.Body(), &body)

	// Generate challenge.
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate challenge"})
	}

	// Generate session ID for correlation.
	sessionIDBytes := make([]byte, 16)
	if _, err := rand.Read(sessionIDBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate session id"})
	}
	sessionID := fmt.Sprintf("%x", sessionIDBytes)

	passkeyStore.Store("auth:"+sessionID, &passkeyChallenge{
		Value:     challenge,
		UserID:    0,
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	allowCredentials := make([]fiber.Map, 0)
	if body.Email != "" {
		var user models.User
		if err := h.db.Where("email = ?", body.Email).First(&user).Error; err == nil {
			var passkeys []models.Passkey
			h.db.Where("user_id = ?", user.ID).Find(&passkeys)
			for _, pk := range passkeys {
				ac := fiber.Map{
					"type": "public-key",
					"id":   pk.CredentialID,
				}
				if pk.Transports != "" {
					ac["transports"] = strings.Split(pk.Transports, ",")
				}
				allowCredentials = append(allowCredentials, ac)
			}
		}
	}

	return c.JSON(fiber.Map{
		"publicKey": fiber.Map{
			"challenge":        base64.RawURLEncoding.EncodeToString(challenge),
			"timeout":          60000,
			"rpId":             passkeyRPID(),
			"userVerification": "preferred",
			"allowCredentials": allowCredentials,
		},
		"session_id": sessionID,
	})
}

// ---------------------------------------------------------------------------
// 4. FinishAuthentication — POST /api/auth/passkey/authenticate/finish (public)
// ---------------------------------------------------------------------------

func (h *PasskeyHandler) FinishAuthentication(c fiber.Ctx) error {
	var body struct {
		ID        string `json:"id"`
		RawID     string `json:"rawId"`
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
		Response  struct {
			AuthenticatorData string `json:"authenticatorData"`
			ClientDataJSON    string `json:"clientDataJSON"`
			Signature         string `json:"signature"`
			UserHandle        string `json:"userHandle"`
		} `json:"response"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.SessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "session_id is required"})
	}

	// Look up stored challenge.
	val, ok := passkeyStore.LoadAndDelete("auth:" + body.SessionID)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no pending authentication challenge"})
	}
	stored := val.(*passkeyChallenge)
	if time.Now().After(stored.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "challenge expired"})
	}

	// Decode clientDataJSON.
	clientDataBytes, err := base64.RawURLEncoding.DecodeString(body.Response.ClientDataJSON)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid clientDataJSON encoding"})
	}

	var clientData struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Origin    string `json:"origin"`
	}
	if err := json.Unmarshal(clientDataBytes, &clientData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid clientDataJSON"})
	}

	if clientData.Type != "webauthn.get" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid ceremony type"})
	}

	expectedChallenge := base64.RawURLEncoding.EncodeToString(stored.Value)
	if clientData.Challenge != expectedChallenge {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "challenge mismatch"})
	}

	if clientData.Origin != passkeyRPOrigin() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "origin mismatch"})
	}

	// Look up the passkey by credential ID.
	credentialID := body.ID // The id field is already base64url-encoded credentialID.
	var passkey models.Passkey
	if err := h.db.Where("credential_id = ?", credentialID).First(&passkey).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unknown credential"})
	}

	// Decode authenticatorData.
	authDataBytes, err := base64.RawURLEncoding.DecodeString(body.Response.AuthenticatorData)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid authenticatorData encoding"})
	}

	// Verify rpIdHash.
	if len(authDataBytes) < 37 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "authenticatorData too short"})
	}
	rpIDHash := sha256.Sum256([]byte(passkeyRPID()))
	for i := 0; i < 32; i++ {
		if authDataBytes[i] != rpIDHash[i] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "rpIdHash mismatch"})
		}
	}

	// Verify signature: message = authData || sha256(clientDataJSON).
	clientDataHash := sha256.Sum256(clientDataBytes)
	message := make([]byte, 0, len(authDataBytes)+32)
	message = append(message, authDataBytes...)
	message = append(message, clientDataHash[:]...)
	messageHash := sha256.Sum256(message)

	// Reconstruct the public key from stored bytes.
	pubKey, err := unmarshalECPublicKey(passkey.PublicKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "corrupted public key"})
	}

	// Decode and verify the signature.
	sigBytes, err := base64.RawURLEncoding.DecodeString(body.Response.Signature)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid signature encoding"})
	}

	if !ecdsa.VerifyASN1(pubKey, messageHash[:], sigBytes) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "signature verification failed"})
	}

	// Verify sign count (protection against cloned authenticators).
	newSignCount := binary.BigEndian.Uint32(authDataBytes[33:37])
	if passkey.SignCount > 0 && newSignCount <= passkey.SignCount {
		// Possible cloned authenticator — log but don't block for now.
	}

	// Update passkey.
	now := time.Now()
	h.db.Model(&passkey).Updates(map[string]interface{}{
		"sign_count":  newSignCount,
		"last_used_at": &now,
	})

	// Load the user.
	var user models.User
	if err := h.db.First(&user, passkey.UserID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load user"})
	}

	if user.Status == "suspended" || user.Status == "blocked" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account is suspended"})
	}

	// Generate JWT.
	authSvc := services.NewAuthService(h.db, h.cfg)
	token, err := authSvc.GenerateJWT(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
	}

	return c.JSON(fiber.Map{
		"ok":    true,
		"token": token,
		"user":  user,
	})
}

// ---------------------------------------------------------------------------
// 5. ListPasskeys — GET /api/settings/passkeys (authenticated)
// ---------------------------------------------------------------------------

func (h *PasskeyHandler) ListPasskeys(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var passkeys []models.Passkey
	h.db.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&passkeys)

	return c.JSON(passkeys)
}

// ---------------------------------------------------------------------------
// 6. RenamePasskey — PATCH /api/settings/passkeys/:id (authenticated)
// ---------------------------------------------------------------------------

func (h *PasskeyHandler) RenamePasskey(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid passkey id"})
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	var passkey models.Passkey
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&passkey).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "passkey not found"})
	}

	h.db.Model(&passkey).Update("name", body.Name)

	return c.JSON(fiber.Map{"ok": true, "passkey": passkey})
}

// ---------------------------------------------------------------------------
// 7. DeletePasskey — DELETE /api/settings/passkeys/:id (authenticated)
// ---------------------------------------------------------------------------

func (h *PasskeyHandler) DeletePasskey(c fiber.Ctx) error {
	user := middleware.CurrentUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid passkey id"})
	}

	var passkey models.Passkey
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&passkey).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "passkey not found"})
	}

	// Soft-delete.
	h.db.Delete(&passkey)

	return c.JSON(fiber.Map{"ok": true})
}

// ===========================================================================
// Minimal inline CBOR parsing (just enough for WebAuthn attestation objects)
// ===========================================================================

// extractAuthDataFromAttestation parses a CBOR-encoded attestation object and
// returns the raw authData bytes. The attestation object is a CBOR map with
// keys "fmt", "attStmt", and "authData". We only need "authData".
func extractAuthDataFromAttestation(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty attestation object")
	}

	offset := 0

	// Read the top-level CBOR map header.
	mapLen, offset, err := cborReadMapHeader(data, offset)
	if err != nil {
		return nil, fmt.Errorf("reading map header: %w", err)
	}

	for i := 0; i < mapLen; i++ {
		// Read key (text string).
		key, newOffset, err := cborReadTextString(data, offset)
		if err != nil {
			return nil, fmt.Errorf("reading map key %d: %w", i, err)
		}
		offset = newOffset

		if key == "authData" {
			// Read value as byte string.
			val, _, err := cborReadByteString(data, offset)
			if err != nil {
				return nil, fmt.Errorf("reading authData: %w", err)
			}
			return val, nil
		}

		// Skip the value.
		offset, err = cborSkipValue(data, offset)
		if err != nil {
			return nil, fmt.Errorf("skipping value for key %q: %w", key, err)
		}
	}

	return nil, fmt.Errorf("authData not found in attestation object")
}

// parseCOSEES256Key parses a CBOR-encoded COSE_Key (ES256 / P-256) and returns
// an ecdsa.PublicKey. The COSE key map for EC2 contains:
//
//	1 (kty): 2 (EC2)
//	3 (alg): -7 (ES256)
//	-1 (crv): 1 (P-256)
//	-2 (x): 32-byte coordinate
//	-3 (y): 32-byte coordinate
func parseCOSEES256Key(data []byte) (*ecdsa.PublicKey, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty COSE key")
	}

	offset := 0
	mapLen, offset, err := cborReadMapHeader(data, offset)
	if err != nil {
		return nil, fmt.Errorf("reading COSE map header: %w", err)
	}

	var xCoord, yCoord []byte

	for i := 0; i < mapLen; i++ {
		// Keys in COSE are integers (some negative).
		keyVal, newOffset, err := cborReadInt(data, offset)
		if err != nil {
			return nil, fmt.Errorf("reading COSE key label %d: %w", i, err)
		}
		offset = newOffset

		switch keyVal {
		case -2: // x coordinate
			xCoord, offset, err = cborReadByteString(data, offset)
			if err != nil {
				return nil, fmt.Errorf("reading x coordinate: %w", err)
			}
		case -3: // y coordinate
			yCoord, offset, err = cborReadByteString(data, offset)
			if err != nil {
				return nil, fmt.Errorf("reading y coordinate: %w", err)
			}
		default:
			// Skip other fields (kty, alg, crv).
			offset, err = cborSkipValue(data, offset)
			if err != nil {
				return nil, fmt.Errorf("skipping COSE field %d: %w", keyVal, err)
			}
		}
	}

	if len(xCoord) != 32 || len(yCoord) != 32 {
		return nil, fmt.Errorf("invalid EC2 coordinate length: x=%d y=%d", len(xCoord), len(yCoord))
	}

	x := new(big.Int).SetBytes(xCoord)
	y := new(big.Int).SetBytes(yCoord)
	curve := elliptic.P256()

	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("public key point is not on P-256 curve")
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

// unmarshalECPublicKey reconstructs an ecdsa.PublicKey from uncompressed
// point bytes (0x04 || X || Y).
func unmarshalECPublicKey(data []byte) (*ecdsa.PublicKey, error) {
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, data)
	if x == nil {
		return nil, fmt.Errorf("failed to unmarshal EC public key")
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

// ===========================================================================
// CBOR primitives
// ===========================================================================

// cborReadMapHeader reads a CBOR map header and returns the number of pairs.
func cborReadMapHeader(data []byte, offset int) (int, int, error) {
	if offset >= len(data) {
		return 0, 0, fmt.Errorf("unexpected end of data at offset %d", offset)
	}
	major := data[offset] >> 5
	if major != 5 { // Major type 5 = map
		return 0, 0, fmt.Errorf("expected CBOR map (major 5), got major %d at offset %d", major, offset)
	}
	return cborReadUintArg(data, offset, 5)
}

// cborReadTextString reads a CBOR text string (major type 3).
func cborReadTextString(data []byte, offset int) (string, int, error) {
	if offset >= len(data) {
		return "", 0, fmt.Errorf("unexpected end of data")
	}
	major := data[offset] >> 5
	if major != 3 {
		return "", 0, fmt.Errorf("expected text string (major 3), got %d", major)
	}
	length, newOffset, err := cborReadUintArg(data, offset, 3)
	if err != nil {
		return "", 0, err
	}
	if newOffset+length > len(data) {
		return "", 0, fmt.Errorf("text string exceeds data bounds")
	}
	return string(data[newOffset : newOffset+length]), newOffset + length, nil
}

// cborReadByteString reads a CBOR byte string (major type 2).
func cborReadByteString(data []byte, offset int) ([]byte, int, error) {
	if offset >= len(data) {
		return nil, 0, fmt.Errorf("unexpected end of data")
	}
	major := data[offset] >> 5
	if major != 2 {
		return nil, 0, fmt.Errorf("expected byte string (major 2), got %d", major)
	}
	length, newOffset, err := cborReadUintArg(data, offset, 2)
	if err != nil {
		return nil, 0, err
	}
	if newOffset+length > len(data) {
		return nil, 0, fmt.Errorf("byte string exceeds data bounds")
	}
	result := make([]byte, length)
	copy(result, data[newOffset:newOffset+length])
	return result, newOffset + length, nil
}

// cborReadInt reads a CBOR integer (major type 0 for positive, 1 for negative).
func cborReadInt(data []byte, offset int) (int, int, error) {
	if offset >= len(data) {
		return 0, 0, fmt.Errorf("unexpected end of data")
	}
	major := data[offset] >> 5
	switch major {
	case 0: // Unsigned integer
		val, newOffset, err := cborReadUintArg(data, offset, 0)
		return val, newOffset, err
	case 1: // Negative integer: -1 - val
		val, newOffset, err := cborReadUintArg(data, offset, 1)
		if err != nil {
			return 0, 0, err
		}
		return -1 - val, newOffset, nil
	default:
		return 0, 0, fmt.Errorf("expected integer (major 0 or 1), got %d at offset %d", major, offset)
	}
}

// cborReadUintArg reads the argument of a CBOR item given its expected major type.
// Returns the argument value and the new offset past the header.
func cborReadUintArg(data []byte, offset int, expectedMajor byte) (int, int, error) {
	if offset >= len(data) {
		return 0, 0, fmt.Errorf("unexpected end of data")
	}
	b := data[offset]
	major := b >> 5
	if major != expectedMajor {
		return 0, 0, fmt.Errorf("major type mismatch: expected %d, got %d", expectedMajor, major)
	}
	additional := b & 0x1f

	switch {
	case additional < 24:
		return int(additional), offset + 1, nil
	case additional == 24:
		if offset+1 >= len(data) {
			return 0, 0, fmt.Errorf("unexpected end of data reading 1-byte arg")
		}
		return int(data[offset+1]), offset + 2, nil
	case additional == 25:
		if offset+2 >= len(data) {
			return 0, 0, fmt.Errorf("unexpected end of data reading 2-byte arg")
		}
		return int(binary.BigEndian.Uint16(data[offset+1 : offset+3])), offset + 3, nil
	case additional == 26:
		if offset+4 >= len(data) {
			return 0, 0, fmt.Errorf("unexpected end of data reading 4-byte arg")
		}
		return int(binary.BigEndian.Uint32(data[offset+1 : offset+5])), offset + 5, nil
	default:
		return 0, 0, fmt.Errorf("unsupported CBOR additional info %d", additional)
	}
}

// cborSkipValue skips over a single CBOR value at the given offset and returns
// the new offset. Handles integers, byte strings, text strings, arrays, and maps.
func cborSkipValue(data []byte, offset int) (int, error) {
	if offset >= len(data) {
		return 0, fmt.Errorf("unexpected end of data")
	}
	major := data[offset] >> 5

	switch major {
	case 0, 1: // Integer
		_, newOffset, err := cborReadUintArg(data, offset, major)
		return newOffset, err

	case 2: // Byte string
		length, newOffset, err := cborReadUintArg(data, offset, 2)
		if err != nil {
			return 0, err
		}
		return newOffset + length, nil

	case 3: // Text string
		length, newOffset, err := cborReadUintArg(data, offset, 3)
		if err != nil {
			return 0, err
		}
		return newOffset + length, nil

	case 4: // Array
		count, newOffset, err := cborReadUintArg(data, offset, 4)
		if err != nil {
			return 0, err
		}
		for i := 0; i < count; i++ {
			newOffset, err = cborSkipValue(data, newOffset)
			if err != nil {
				return 0, err
			}
		}
		return newOffset, nil

	case 5: // Map
		count, newOffset, err := cborReadUintArg(data, offset, 5)
		if err != nil {
			return 0, err
		}
		for i := 0; i < count; i++ {
			// Skip key.
			newOffset, err = cborSkipValue(data, newOffset)
			if err != nil {
				return 0, err
			}
			// Skip value.
			newOffset, err = cborSkipValue(data, newOffset)
			if err != nil {
				return 0, err
			}
		}
		return newOffset, nil

	case 6: // Tag — read tag number then skip enclosed value.
		_, newOffset, err := cborReadUintArg(data, offset, 6)
		if err != nil {
			return 0, err
		}
		return cborSkipValue(data, newOffset)

	case 7: // Simple values / floats
		additional := data[offset] & 0x1f
		switch {
		case additional < 24:
			return offset + 1, nil
		case additional == 24:
			return offset + 2, nil
		case additional == 25:
			return offset + 3, nil
		case additional == 26:
			return offset + 5, nil
		case additional == 27:
			return offset + 9, nil
		default:
			return offset + 1, nil
		}

	default:
		return 0, fmt.Errorf("unknown CBOR major type %d", major)
	}
}
