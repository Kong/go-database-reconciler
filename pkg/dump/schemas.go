package dump

import (
	"context"

	"github.com/kong/go-database-reconciler/pkg/schema"
	"github.com/kong/go-kong/kong"
)

// SchemaFetcher wraps a schema.Registry and provides schema access for the
// dump package. It delegates all fetching and caching to the central schema
// package.
type SchemaFetcher struct {
	registry *schema.Registry
}

// NewSchemaFetcher creates a SchemaFetcher backed by a schema.Registry.
func NewSchemaFetcher(ctx context.Context, client *kong.Client, isKonnect bool) *SchemaFetcher {
	return &SchemaFetcher{
		registry: schema.NewRegistry(ctx, client, isKonnect),
	}
}

// getSchema returns the schema for the given entity type and identifier,
// delegating to the underlying schema.Registry.
func (s *SchemaFetcher) getSchema(entityType, entityIdentifier string) (kong.Schema, error) {
	return s.registry.GetSchema(entityType, entityIdentifier)
}

// GetEntitySchema fetches the schema for a given entity type from either
// the Kong Gateway admin API or the Konnect API.
// Deprecated: Use schema.FetchEntitySchema or schema.Registry directly.
func GetEntitySchema(
	ctx context.Context, client *kong.Client, isKonnect bool, entityType string,
) (kong.Schema, error) {
	return schema.FetchEntitySchema(ctx, client, isKonnect, entityType)
}
