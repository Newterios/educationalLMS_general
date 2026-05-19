package uuid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

type Generator struct{}

func New() *Generator { return &Generator{} }

// Generate returns a UUID v4 hex string as the raw code (sent in email)
// and its SHA-256 hex digest (stored in DB). UUID v4 provides ~122 bits
// of entropy — sufficient without bcrypt-strength hashing.
func (g *Generator) Generate() (raw string, hash string, err error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", "", fmt.Errorf("generate code: %w", err)
	}
	raw = id.String()
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return raw, hash, nil
}
