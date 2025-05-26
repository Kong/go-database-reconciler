package types

import (
	"context"
	"sync"

	"github.com/kong/go-kong/kong"
)

type PluginSchemaCache struct {
	KongClient *kong.Client
	cache      map[string]map[string]interface{}
	cacheMutex sync.RWMutex
}

func NewPluginSchemaCache(client *kong.Client) *PluginSchemaCache {
	return &PluginSchemaCache{
		KongClient: client,
		cache:      make(map[string]map[string]interface{}),
	}
}

func (psc *PluginSchemaCache) Get(ctx context.Context, pluginName string) (map[string]interface{}, error) {
	psc.cacheMutex.RLock()
	if schema, ok := psc.cache[pluginName]; ok {
		psc.cacheMutex.RUnlock()
		return schema, nil
	}
	psc.cacheMutex.RUnlock()

	schema, err := psc.KongClient.Plugins.GetFullSchema(ctx, &pluginName)
	if err != nil {
		return nil, err
	}

	psc.cacheMutex.Lock()
	psc.cache[pluginName] = schema
	psc.cacheMutex.Unlock()
	return schema, nil
}
