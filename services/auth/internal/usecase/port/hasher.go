package port

// PasswordHasher abstracts bcrypt.
// Production impl uses cost ≥ 12 (ARCHITECTURE.md §11).
type PasswordHasher interface {
	Hash(plain string) (string, error)
	// Compare returns model.ErrUnauthenticated when plain does not match hash,
	// and nil on success.
	Compare(hash, plain string) error
}
