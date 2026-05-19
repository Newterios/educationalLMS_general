package port

import (
	"context"
	"time"
)

// Cache is a generic key-value store for cache-aside patterns.
// Get returns (nil, nil) on miss; callers fall through to the DB silently.
// DeleteByPrefix removes all keys whose names start with prefix (used for list invalidation).
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) error
}
