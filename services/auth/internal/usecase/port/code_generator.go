package port

// CodeGenerator produces one-time verification codes.
// Raw is the value sent in the email (UUID v4 hex, ~122 bits entropy).
// Hash is SHA-256(raw) — the value stored in the DB.
type CodeGenerator interface {
	Generate() (raw string, hash string, err error)
}
