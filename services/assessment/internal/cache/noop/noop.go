package noop

import (
	"context"
	"time"
)

// Cache is a no-op cache implementation used until Redis is wired in Phase 2.
type Cache struct{}

func New() *Cache { return &Cache{} }

func (c *Cache) Get(_ context.Context, _ string) ([]byte, error)                  { return nil, nil }
func (c *Cache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error { return nil }
func (c *Cache) Delete(_ context.Context, _ string) error                         { return nil }
