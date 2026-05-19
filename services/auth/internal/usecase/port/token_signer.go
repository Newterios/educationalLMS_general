package port

import "time"

// AccessClaims holds the decoded payload of an access JWT.
type AccessClaims struct {
	UserID        string
	SessionID     string
	Role          string
	EmailVerified bool
}

// RefreshClaims holds the decoded payload of a refresh JWT.
type RefreshClaims struct {
	UserID    string
	SessionID string
}

// TokenSigner mints and verifies JWTs.
// Production impl uses golang-jwt/jwt/v5 HS256.
// Access claims: sub=userID, jti=sessionID, role, email_verified, iat, exp.
// Refresh claims: sub=userID, jti=sessionID, iat, exp.
type TokenSigner interface {
	SignAccess(userID, sessionID, role string, emailVerified bool) (token string, expiresAt time.Time, err error)
	SignRefresh(userID, sessionID string) (token string, expiresAt time.Time, err error)
	ParseAccess(token string) (AccessClaims, error)
	ParseRefresh(token string) (RefreshClaims, error)
}
