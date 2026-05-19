package interceptors

import "context"

type ctxKey int

const (
	ctxKeyUserID ctxKey = iota
	ctxKeySessionID
	ctxKeyRole
	ctxKeyEmailVerified
)

// withClaims stores validated JWT claims into the context. Called by Auth interceptor only.
func withClaims(ctx context.Context, userID, sessionID, role string, emailVerified bool) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	ctx = context.WithValue(ctx, ctxKeySessionID, sessionID)
	ctx = context.WithValue(ctx, ctxKeyRole, role)
	ctx = context.WithValue(ctx, ctxKeyEmailVerified, emailVerified)
	return ctx
}

// UserIDFrom returns the user ID stored by the Auth interceptor.
// Returns "" if the method is unauthenticated (caller should not reach this point).
func UserIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

// SessionIDFrom returns the session ID stored by the Auth interceptor.
func SessionIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeySessionID).(string)
	return v
}

// RoleFrom returns the role stored by the Auth interceptor.
func RoleFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRole).(string)
	return v
}

// EmailVerifiedFrom returns the email_verified flag stored by the Auth interceptor.
func EmailVerifiedFrom(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyEmailVerified).(bool)
	return v
}
