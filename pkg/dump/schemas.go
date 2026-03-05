package dump

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kong/go-database-reconciler/pkg/types"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
)

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
			return getVaultSchema(ctx, client, vaultType, isKonnect)
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

	entityType, ok = utils.KongToKonnectEntitiesMap[entityType]
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

func getVaultSchema(ctx context.Context, client *kong.Client, vaultType string, isKonnect bool) (kong.Schema, error) {
	var schema map[string]interface{}

	if isKonnect {
		return getKonnectVaultSchema(ctx, client, vaultType)
	}

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

func getKonnectVaultSchema(ctx context.Context, client *kong.Client, vaultType string) (kong.Schema, error) {
	var schema map[string]interface{}

	fullSchema, err := getKonnectEntitySchema(ctx, client, "vaults")
	if err != nil {
		return schema, fmt.Errorf("failed to fetch schema: %w", err)
	}

	// Start with the base schema from fullSchema
	schema = make(map[string]interface{})
	for key, value := range fullSchema {
		if key != "allOf" {
			schema[key] = value
		}
	}

	// Extract the specific vault type schema from the full schema
	// The full schema contains conditional logic based on vault name
	// We need to find the matching condition for the given vaultType
	allOf, ok := fullSchema["allOf"].([]interface{})
	if !ok {
		return schema, fmt.Errorf("invalid schema format: allOf not found or not an array")
	}

	const (
		ifKey         = "if"
		thenKey       = "then"
		propertiesKey = "properties"
		nameKey       = "name"
		constKey      = "const"
		configKey     = "config"
	)

	// Look for the matching vault type in the conditional schemas
	for _, condition := range allOf {
		conditionMap, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this condition matches our vault type
		if ifClause, exists := conditionMap[ifKey]; exists {
			if ifMap, ok := ifClause.(map[string]interface{}); ok {
				if properties, exists := ifMap[propertiesKey]; exists {
					if propsMap, ok := properties.(map[string]interface{}); ok {
						if nameClause, exists := propsMap[nameKey]; exists {
							if nameMap, ok := nameClause.(map[string]interface{}); ok {
								if constValue, exists := nameMap[constKey]; exists {
									if constStr, ok := constValue.(string); ok && constStr == vaultType {
										// Found the matching condition, extract the config from "then" clause
										if thenClause, exists := conditionMap[thenKey]; exists {
											if thenMap, ok := thenClause.(map[string]interface{}); ok {
												if thenProps, exists := thenMap[propertiesKey]; exists {
													if thenPropsMap, ok := thenProps.(map[string]interface{}); ok {
														if configSchema, exists := thenPropsMap[configKey]; exists {
															if configSchemaMap, ok := configSchema.(map[string]interface{}); ok {
																if configProps, exists := configSchemaMap[propertiesKey]; exists {
																	if configPropsMap, ok := configProps.(map[string]interface{}); ok {
																		vaultConfigSchema := configPropsMap[vaultType]
																		if schemaProps, exists := schema[propertiesKey]; exists {
																			if schemaPropsMap, ok := schemaProps.(map[string]interface{}); ok {
																				schemaPropsMap[configKey] = vaultConfigSchema
																			}
																		}
																		return schema, nil
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return schema, nil
}
