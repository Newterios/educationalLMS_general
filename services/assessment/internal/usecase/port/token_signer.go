package port

// AccessClaims holds the decoded JWT access token fields used by interceptors.
type AccessClaims struct {
	UserID        string
	SessionID     string
	Role          string
	EmailVerified bool
}

// TokenSigner can sign and verify JWT tokens.
type TokenSigner interface {
	ParseAccess(token string) (AccessClaims, error)
}
