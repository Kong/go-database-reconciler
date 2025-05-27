package types

import (
	"context"
	"sync"
)

type SchemaFetcher func(ctx context.Context, identifier string) (map[string]interface{}, error)

type SchemaCache struct {
	schemaFetcher SchemaFetcher
	cache         map[string]map[string]interface{}
	cacheMutex    sync.RWMutex
}

func NewSchemaCache(fetcher SchemaFetcher) *SchemaCache {
	return &SchemaCache{
		schemaFetcher: fetcher,
		cache:         make(map[string]map[string]interface{}),
	}
}

func (sc *SchemaCache) Get(ctx context.Context, identifier string) (map[string]interface{}, error) {
	sc.cacheMutex.RLock()
	if schema, ok := sc.cache[identifier]; ok {
		sc.cacheMutex.RUnlock()
		return schema, nil
	}
	sc.cacheMutex.RUnlock()

	schema, err := sc.schemaFetcher(ctx, identifier)
	if err != nil {
		return nil, err
	}

	sc.cacheMutex.Lock()
	sc.cache[identifier] = schema
	sc.cacheMutex.Unlock()
	return schema, nil
}
