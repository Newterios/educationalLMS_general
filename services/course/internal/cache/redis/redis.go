package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Cache implements port.Cache using Redis (sliding-window TTL).
type Cache struct {
	client *goredis.Client
}

func New(redisURL string) (*Cache, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Cache{client: goredis.NewClient(opts)}, nil
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == goredis.Nil {
		return nil, nil
	}
	return val, err
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// DeleteByPrefix removes all keys matching prefix* using SCAN+DEL (non-atomic, best-effort).
func (c *Cache) DeleteByPrefix(ctx context.Context, prefix string) error {
	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

// Incr increments a counter used by the rate-limit interceptor.
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Expire sets TTL on an existing key (used for rate-limit window).
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *Cache) Close() error {
	return c.client.Close()
}
