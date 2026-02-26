package schema

import (
	"context"
	"fmt"

	"github.com/kong/go-kong/kong"
)

// Registry is the central schema manager. It holds all schema caches for different
// entity categories (generic entities, plugins, partials, vaults) and provides a
// single GetSchema method that routes to the correct cache based on entity type.
//
// Create a Registry via NewRegistry and share it across all subsystems that need
// schema access (dump, diff, defaulter, etc.) to avoid duplicate caches and
// redundant API calls.
type Registry struct {
	ctx       context.Context
	client    *kong.Client
	isKonnect bool

	entityCache  *Cache
	pluginCache  *Cache
	partialCache *Cache
	vaultCache   *Cache
}

// NewRegistry creates a new Registry backed by the given Kong client.
// All four caches are initialised with the appropriate fetcher functions for
// either Gateway or Konnect, determined by isKonnect.
func NewRegistry(ctx context.Context, client *kong.Client, isKonnect bool) *Registry {
	r := &Registry{
		ctx:       ctx,
		client:    client,
		isKonnect: isKonnect,
	}

	r.entityCache = NewCache(func(ctx context.Context, entityType string) (map[string]interface{}, error) {
		return FetchEntitySchema(ctx, client, isKonnect, entityType)
	})
	r.pluginCache = NewCache(func(ctx context.Context, pluginName string) (map[string]interface{}, error) {
		return FetchPluginSchema(ctx, client, pluginName)
	})
	r.partialCache = NewCache(func(ctx context.Context, partialType string) (map[string]interface{}, error) {
		return FetchPartialSchema(ctx, client, partialType)
	})
	r.vaultCache = NewCache(func(ctx context.Context, vaultType string) (map[string]interface{}, error) {
		return FetchVaultSchema(ctx, client, vaultType, isKonnect)
	})

	return r
}

// GetSchema returns the schema for a given entity type and identifier.
// For most entities, entityType doubles as the cache key (e.g. "services").
// For plugins, partials, and vaults the identifier is the specific name/type
// (e.g. "rate-limiting", "aws") because each has its own schema.
func (r *Registry) GetSchema(entityType, identifier string) (kong.Schema, error) {
	if r.client == nil {
		return kong.Schema{}, fmt.Errorf("kong client is not initialized")
	}

	switch entityType {
	case "plugins":
		return r.pluginCache.Get(r.ctx, identifier)
	case "partials":
		return r.partialCache.Get(r.ctx, identifier)
	case "vaults":
		return r.vaultCache.Get(r.ctx, identifier)
	default:
		return r.entityCache.Get(r.ctx, entityType)
	}
}

// GetEntitySchema is a convenience method that fetches the schema for a generic
// entity type (services, routes, upstreams, etc.). It is equivalent to
// GetSchema(entityType, entityType).
func (r *Registry) GetEntitySchema(entityType string) (kong.Schema, error) {
	return r.GetSchema(entityType, entityType)
}

// GetPluginSchema is a convenience method that fetches the schema for a plugin
// by its name (e.g. "rate-limiting").
func (r *Registry) GetPluginSchema(pluginName string) (map[string]interface{}, error) {
	return r.pluginCache.Get(r.ctx, pluginName)
}

// GetPartialSchema is a convenience method that fetches the schema for a partial
// by its type.
func (r *Registry) GetPartialSchema(partialType string) (map[string]interface{}, error) {
	return r.partialCache.Get(r.ctx, partialType)
}

// GetVaultSchema is a convenience method that fetches the schema for a vault
// by its type (e.g. "aws", "hcv").
func (r *Registry) GetVaultSchema(vaultType string) (map[string]interface{}, error) {
	return r.vaultCache.Get(r.ctx, vaultType)
}
