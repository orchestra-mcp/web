package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
	"time"
)

// generateTestTOTP produces a valid TOTP code for the given base32 secret at the current time.
func generateTestTOTP(secret string) string {
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	counter := uint64(time.Now().Unix() / 30)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0F
	trunc := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7FFFFFFF
	return fmt.Sprintf("%06d", trunc%uint32(math.Pow10(6)))
}

func TestValidateTOTP_ValidCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP" // standard test base32 secret
	code := generateTestTOTP(secret)

	if !validateTOTP(secret, code) {
		t.Errorf("expected valid TOTP code %q to pass validation", code)
	}
}

func TestValidateTOTP_InvalidCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"

	if validateTOTP(secret, "000000") && validateTOTP(secret, "000000") {
		// It's possible (1 in ~333k chance per window) that 000000 is valid.
		// Only fail if it passes on a completely different secret too.
		if validateTOTP("GEZDGNBVGY3TQOJQ", "000000") {
			t.Error("unlikely: 000000 valid for two different secrets")
		}
	}
}

func TestValidateTOTP_WrongLength(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"

	if validateTOTP(secret, "12345") {
		t.Error("5-digit code should not validate")
	}
	if validateTOTP(secret, "1234567") {
		t.Error("7-digit code should not validate")
	}
}

func TestValidateTOTP_InvalidSecret(t *testing.T) {
	// Non-base32 secret should return false, not panic
	if validateTOTP("not-valid-base32!!!", "123456") {
		t.Error("invalid base32 secret should fail validation")
	}
}

func TestValidateTOTP_EmptyInputs(t *testing.T) {
	if validateTOTP("", "123456") {
		t.Error("empty secret should fail")
	}
	secret := "JBSWY3DPEHPK3PXP"
	if validateTOTP(secret, "") {
		t.Error("empty code should fail")
	}
}

func TestValidateTOTP_LowercaseSecret(t *testing.T) {
	secret := "jbswy3dpehpk3pxp" // lowercase base32
	code := generateTestTOTP("JBSWY3DPEHPK3PXP")

	// validateTOTP uppercases the secret, so this should work
	if !validateTOTP(secret, code) {
		t.Error("lowercase secret should be accepted")
	}
}
