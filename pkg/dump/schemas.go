package dump

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kong/go-database-reconciler/pkg/types"
	"github.com/kong/go-kong/kong"
)

// Some entities in Konnect have different names compared to Kong Gateway
var kongToKonnectEntitiesMap = map[string]string{
	"services":              "service",
	"routes":                "route",
	"upstreams":             "upstream",
	"targets":               "target",
	"jwt_secrets":           "jwt",
	"consumers":             "consumer",
	"consumer_groups":       "consumer_group",
	"certificates":          "certificate",
	"ca_certificates":       "ca_certificate",
	"keys":                  "key",
	"key_sets":              "key-set",
	"hmacauth_credentials":  "hmac-auth",
	"basicauth_credentials": "basic-auth",
	"mtls_auth_credentials": "mtls-auth",
	"snis":                  "sni",
}

type SchemaFetcher struct {
	ctx                 context.Context
	client              *kong.Client
	isKonnect           bool
	entitySchemaCache   *types.SchemaCache
	pluginSchemasCache  *types.SchemaCache
	partialSchemasCache *types.SchemaCache
	vaultSchemaCache    *types.SchemaCache
}

func NewSchemaFetcher(ctx context.Context, client *kong.Client, isKonnect bool) *SchemaFetcher {
	return &SchemaFetcher{
		ctx:       ctx,
		client:    client,
		isKonnect: isKonnect,
		entitySchemaCache: types.NewSchemaCache(func(ctx context.Context,
			entityType string,
		) (map[string]interface{}, error) {
			return getEntitySchema(ctx, client, isKonnect, entityType)
		}),
		pluginSchemasCache: types.NewSchemaCache(func(ctx context.Context,
			pluginName string,
		) (map[string]interface{}, error) {
			return client.Plugins.GetFullSchema(ctx, &pluginName)
		}),
		partialSchemasCache: types.NewSchemaCache(func(ctx context.Context,
			partialType string,
		) (map[string]interface{}, error) {
			return client.Partials.GetFullSchema(ctx, &partialType)
		}),
		vaultSchemaCache: types.NewSchemaCache(func(ctx context.Context,
			vaultType string,
		) (map[string]interface{}, error) {
			// works only for gateway, not konnect
			return getVaultSchema(ctx, client, vaultType)
		}),
	}
}

func (s *SchemaFetcher) getSchema(entityType, entityIdentifier string) (kong.Schema, error) {
	var (
		schema kong.Schema
		err    error
	)

	// for unit tests, we may have an uninitialized client
	if s.client == nil {
		return kong.Schema{}, fmt.Errorf("kong client is not initialized")
	}

	if entityType == "plugins" {
		schema, err = s.pluginSchemasCache.Get(s.ctx, entityIdentifier)
		return schema, err
	}

	if entityType == "partials" {
		schema, err = s.partialSchemasCache.Get(s.ctx, entityIdentifier)
		return schema, err
	}

	if entityType == "vaults" {
		schema, err = s.vaultSchemaCache.Get(s.ctx, entityIdentifier)
		return schema, err
	}

	schema, err = s.entitySchemaCache.Get(s.ctx, entityType)
	return schema, err
}

func getEntitySchema(ctx context.Context, client *kong.Client, isKonnect bool, entityType string) (kong.Schema, error) {
	var (
		schema kong.Schema
		err    error
	)
	if isKonnect {
		schema, err = getKonnectEntitySchema(ctx, client, entityType)
		return schema, err
	}

	schema, err = client.Schemas.Get(ctx, entityType)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, fmt.Errorf("schema for entity type %s not found", entityType)
	}
	return schema, nil
}

func getKonnectEntitySchema(ctx context.Context, client *kong.Client, entityType string) (kong.Schema, error) {
	var (
		schema map[string]interface{}
		ok     bool
	)

	entityType, ok = kongToKonnectEntitiesMap[entityType]
	if !ok {
		return schema, nil
	}

	endpoint := fmt.Sprintf("/v1/schemas/json/%s", entityType)

	req, err := client.NewRequest(http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return schema, err
	}
	resp, err := client.Do(ctx, req, &schema)
	if resp == nil {
		return schema, fmt.Errorf("invalid HTTP response: %w", err)
	}
	if err != nil {
		return schema, fmt.Errorf("failed to fetch schema: %w", err)
	}

	return schema, nil
}

func getVaultSchema(ctx context.Context, client *kong.Client, vaultType string) (kong.Schema, error) {
	var schema map[string]interface{}

	endpoint := fmt.Sprintf("/schemas/vaults/%s", vaultType)
	req, err := client.NewRequest(http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return schema, err
	}
	resp, err := client.Do(ctx, req, &schema)
	if resp == nil {
		return schema, fmt.Errorf("invalid HTTP response: %w", err)
	}
	if err != nil {
		return schema, fmt.Errorf("failed to fetch schema: %w", err)
	}

	return schema, nil
}
