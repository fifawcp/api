package totp

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"time"
)

const totpWindow = 10 * time.Minute

func Generate(identifier, secret string) string {
	// Divide current Unix time into fixed windows (e.g. one window per 10 minutes).
	// All calls within the same window produce the same OTP for a given identifier.
	window := time.Now().Unix() / int64(totpWindow.Seconds())

	// HMAC-SHA256 keyed with the JWT secret.
	// Input is "identifier:window" so different identifiers and different time
	// windows always produce a different hash.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%s:%d", identifier, window)))
	h := mac.Sum(nil)

	// Dynamic truncation (same technique used by RFC 6238 TOTP):
	// Use the low 4 bits of the last byte as an offset into the hash.
	offset := h[len(h)-1] & 0x0f

	// Read 4 bytes at that offset and mask the sign bit to get a
	// 31-bit positive integer.
	code := (int(h[offset]&0x7f) << 24) |
		(int(h[offset+1]) << 16) |
		(int(h[offset+2]) << 8) |
		int(h[offset+3])

	// Reduce to 6 digits with zero-padding (e.g. "048291").
	return fmt.Sprintf("%06d", code%1_000_000)
}

func WindowExpiresIn() int64 {
	return int64(totpWindow.Seconds()) - (time.Now().Unix() % int64(totpWindow.Seconds()))
}
