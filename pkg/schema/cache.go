package schema

import (
	"context"
	"sync"
)

// Fetcher is a function that retrieves a schema by identifier from an external source.
type Fetcher func(ctx context.Context, identifier string) (map[string]interface{}, error)

// Cache provides thread-safe caching of schemas keyed by a string identifier.
// It lazily fetches schemas on first access and returns cached results thereafter.
type Cache struct {
	fetcher Fetcher
	cache   map[string]map[string]interface{}
	mu      sync.RWMutex
}

// NewCache creates a new schema Cache backed by the given fetcher function.
func NewCache(fetcher Fetcher) *Cache {
	return &Cache{
		fetcher: fetcher,
		cache:   make(map[string]map[string]interface{}),
	}
}

// Get returns the cached schema for the given identifier, fetching it on first access.
func (c *Cache) Get(ctx context.Context, identifier string) (map[string]interface{}, error) {
	c.mu.RLock()
	if s, ok := c.cache[identifier]; ok {
		c.mu.RUnlock()
		return s, nil
	}
	c.mu.RUnlock()

	s, err := c.fetcher(ctx, identifier)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[identifier] = s
	c.mu.Unlock()
	return s, nil
}
