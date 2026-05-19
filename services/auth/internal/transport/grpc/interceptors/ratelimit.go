package interceptors

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RateLimiter provides sliding-window rate limiting backed by Redis INCR+EXPIRE.
type RateLimiter interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// RateLimitConfig maps gRPC full method names to per-minute limits.
// Methods not listed use the global default.
type RateLimitConfig struct {
	GlobalRPM   int64
	MethodLimits map[string]int64 // full method → RPM
}

// RateLimit returns a sliding-window rate-limit interceptor.
// Window = 1 minute; key = "rl:<method>:<userID_or_ip>".
// Per PLAN.md: default=100RPM, Login=10RPM, SubmitAttempt=5RPM.
func RateLimit(limiter RateLimiter, cfg RateLimitConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		limit := cfg.GlobalRPM
		if methodLimit, ok := cfg.MethodLimits[info.FullMethod]; ok {
			limit = methodLimit
		}

		identity := callerIdentity(ctx)
		key := fmt.Sprintf("rl:%s:%s", info.FullMethod, identity)

		count, err := limiter.Incr(ctx, key)
		if err != nil {
			// Redis down → allow request (fail-open strategy)
			return handler(ctx, req)
		}
		// Set TTL on first request in window (idempotent on subsequent calls)
		if count == 1 {
			_ = limiter.Expire(ctx, key, time.Minute)
		}

		if count > limit {
			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded: %d/%d rpm for %s", count, limit, info.FullMethod)
		}

		return handler(ctx, req)
	}
}

// callerIdentity returns the user ID from JWT claims if present, otherwise
// falls back to the x-forwarded-for header, then "anonymous".
func callerIdentity(ctx context.Context) string {
	if uid := UserIDFrom(ctx); uid != "" {
		return uid
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-forwarded-for"); len(vals) > 0 {
			return vals[0]
		}
	}
	return "anon"
}
