package usecase

import (
	"crypto/sha256"
	"encoding/hex"
)

// sha256Hex returns the hex-encoded SHA-256 digest of s.
// Used to hash refresh tokens and verification codes before DB storage
// so raw values never touch the database.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
