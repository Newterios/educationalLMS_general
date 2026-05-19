package port

// AccessClaims holds the decoded payload of an access JWT.
type AccessClaims struct {
	UserID        string
	SessionID     string
	Role          string
	EmailVerified bool
}

// TokenSigner parses access JWTs.  Course service only needs ParseAccess.
type TokenSigner interface {
	ParseAccess(token string) (AccessClaims, error)
}
