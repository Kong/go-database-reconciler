package schema

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kong/go-kong/kong"
)

// KongToKonnectEntitiesMap maps Kong Gateway entity type names to Konnect entity type names.
// Some entities in Konnect have different names compared to Kong Gateway.
var KongToKonnectEntitiesMap = map[string]string{
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
	"vaults":                "vault",
}

// FetchEntitySchema fetches the schema for a given entity type from either
// the Kong Gateway admin API or the Konnect API.
// If the schema is not found (e.g. EE-only entities), it returns (nil, nil).
func FetchEntitySchema(
	ctx context.Context, client *kong.Client, isKonnect bool, entityType string,
) (kong.Schema, error) {
	if isKonnect {
		return fetchKonnectEntitySchema(ctx, client, entityType)
	}

	var schema map[string]interface{}
	endpoint := fmt.Sprintf("/schemas/%s", entityType)
	req, err := client.NewRequest(http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(ctx, req, &schema)
	if resp == nil {
		return nil, fmt.Errorf("invalid HTTP response: %w", err)
	}
	// If the schema is not found (e.g. EE features), return nil without error.
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return schema, nil
}

// fetchKonnectEntitySchema fetches the entity schema from the Konnect /v1/schemas/json/{type} endpoint.
func fetchKonnectEntitySchema(
	ctx context.Context, client *kong.Client, entityType string,
) (kong.Schema, error) {
	var schema map[string]interface{}

	konnectType, ok := KongToKonnectEntitiesMap[entityType]
	if !ok {
		return schema, nil
	}

	endpoint := fmt.Sprintf("/v1/schemas/json/%s", konnectType)
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

// FetchPluginSchema fetches the full schema for a plugin by name from the Kong admin API.
func FetchPluginSchema(
	ctx context.Context, client *kong.Client, pluginName string,
) (map[string]interface{}, error) {
	return client.Plugins.GetFullSchema(ctx, &pluginName)
}

// FetchPartialSchema fetches the full schema for a partial by type from the Kong admin API.
func FetchPartialSchema(
	ctx context.Context, client *kong.Client, partialType string,
) (map[string]interface{}, error) {
	return client.Partials.GetFullSchema(ctx, &partialType)
}

// FetchVaultSchema fetches the schema for a vault by type from either
// the Kong Gateway admin API or the Konnect API.
func FetchVaultSchema(
	ctx context.Context, client *kong.Client, vaultType string, isKonnect bool,
) (kong.Schema, error) {
	if isKonnect {
		return fetchKonnectVaultSchema(ctx, client, vaultType)
	}

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

// fetchKonnectVaultSchema fetches a vault-type-specific schema from the Konnect API
// by parsing the conditional allOf structure in the full vault schema.
func fetchKonnectVaultSchema(
	ctx context.Context, client *kong.Client, vaultType string,
) (kong.Schema, error) {
	var schema map[string]interface{}

	fullSchema, err := fetchKonnectEntitySchema(ctx, client, "vaults")
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

	// Extract the specific vault type schema from the conditional allOf structure.
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

	for _, condition := range allOf {
		conditionMap, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		if ifClause, exists := conditionMap[ifKey]; exists {
			if ifMap, ok := ifClause.(map[string]interface{}); ok {
				if properties, exists := ifMap[propertiesKey]; exists {
					if propsMap, ok := properties.(map[string]interface{}); ok {
						if nameClause, exists := propsMap[nameKey]; exists {
							if nameMap, ok := nameClause.(map[string]interface{}); ok {
								if constValue, exists := nameMap[constKey]; exists {
									if constStr, ok := constValue.(string); ok && constStr == vaultType {
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
