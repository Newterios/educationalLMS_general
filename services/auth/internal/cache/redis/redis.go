package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Cache implements port.Cache using Redis.
type Cache struct {
	client *goredis.Client
}

// New creates a Cache from the given Redis URL.
func New(redisURL string) (*Cache, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := goredis.NewClient(opts)
	return &Cache{client: client}, nil
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == goredis.Nil {
		return nil, nil // cache miss
	}
	return val, err
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// LPush is the DLQWriter interface for notification-svc dead-letter.
func (c *Cache) LPush(ctx context.Context, key, value string) error {
	return c.client.LPush(ctx, key, value).Err()
}

// Incr satisfies interceptors.RateLimiter (INCR backing the sliding window).
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Expire sets the TTL for a rate-limit window key on the first request.
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// Close releases the Redis connection.
func (c *Cache) Close() error {
	return c.client.Close()
}
