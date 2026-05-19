package port

import (
	"context"
	"time"
)

// Cache is a generic key-value store used for cache-aside patterns.
// Get returns (nil, nil) on a cache miss — callers fall through to the DB.
// Write failures are non-fatal; implementations must log and return nil.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
