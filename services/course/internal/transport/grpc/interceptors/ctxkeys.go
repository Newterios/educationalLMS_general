package interceptors

import "context"

type ctxKey int

const (
	ctxKeyUserID ctxKey = iota
	ctxKeySessionID
	ctxKeyRole
	ctxKeyEmailVerified
)

func withClaims(ctx context.Context, userID, sessionID, role string, emailVerified bool) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	ctx = context.WithValue(ctx, ctxKeySessionID, sessionID)
	ctx = context.WithValue(ctx, ctxKeyRole, role)
	ctx = context.WithValue(ctx, ctxKeyEmailVerified, emailVerified)
	return ctx
}

func UserIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

func SessionIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeySessionID).(string)
	return v
}

func RoleFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRole).(string)
	return v
}

func EmailVerifiedFrom(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyEmailVerified).(bool)
	return v
}
