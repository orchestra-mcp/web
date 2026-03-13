package models

import (
	"time"

	"gorm.io/gorm"
)

// Passkey represents a WebAuthn/FIDO2 credential stored for passwordless authentication.
type Passkey struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	UserID         uint           `gorm:"index" json:"user_id"`
	CredentialID   string         `gorm:"uniqueIndex" json:"credential_id"` // base64url-encoded
	PublicKey      []byte         `json:"-"`                                // COSE public key bytes
	Name           string         `json:"name"`                             // user-given label
	AAGUID         string         `json:"aaguid"`                           // authenticator AAGUID
	SignCount      uint32         `json:"sign_count"`
	Transports     string         `json:"transports"`      // comma-separated: usb,nfc,ble,internal
	BackupEligible bool           `json:"backup_eligible"`
	BackupState    bool           `json:"backup_state"`
	LastUsedAt     *time.Time     `json:"last_used_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	User           User           `gorm:"foreignKey:UserID" json:"-"`
}
