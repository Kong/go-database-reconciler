package types

import (
	"context"
	"sync"

	"github.com/kong/go-kong/kong"
)

type PartialSchemaCache struct {
	KongClient *kong.Client
	cache      map[string]map[string]interface{}
	cacheMutex sync.RWMutex
}

func NewPartialSchemaCache(client *kong.Client) *PartialSchemaCache {
	return &PartialSchemaCache{
		KongClient: client,
		cache:      make(map[string]map[string]interface{}),
	}
}

func (psc *PartialSchemaCache) Get(ctx context.Context, partialName string) (map[string]interface{}, error) {
	psc.cacheMutex.RLock()
	if schema, ok := psc.cache[partialName]; ok {
		psc.cacheMutex.RUnlock()
		return schema, nil
	}
	psc.cacheMutex.RUnlock()

	schema, err := psc.KongClient.Partials.GetFullSchema(ctx, &partialName)
	if err != nil {
		return nil, err
	}

	psc.cacheMutex.Lock()
	psc.cache[partialName] = schema
	psc.cacheMutex.Unlock()
	return schema, nil
}
