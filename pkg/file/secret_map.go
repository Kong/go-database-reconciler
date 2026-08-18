package file

import (
	"reflect"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"
)

// SecretMap records, per entity instance, which of its own field names are
// backed by a DECK_* environment variable reference. Built once from a
// mock-rendered YAML string (env var refs left as their bare name instead of
// their real value — see EnvVarsSkip or EnvVarsMock) and consulted at diff time via the
// same EntityKey constructors, so masking can target the exact field on the
// exact entity that was actually templated, never a coincidental value match.
type SecretMap map[EntityKey]map[string]bool

// BuildSecretMap takes a rendered template string (with DECK_* env var names visible)
// and walks through the structure to record which field names are secret.
// Works with any YAML content, including type mismatches (e.g., string values in int fields),
// by parsing into a generic map and extracting secrets without strict struct unmarshaling.
func BuildSecretMap(mockYAML string) SecretMap {
	sm := make(SecretMap)
	if mockYAML == "" {
		return sm
	}

	// Parse YAML into a generic map/interface{} to detect templated fields
	// without being blocked by type mismatches or schema validation.
	var genericContent map[string]any
	err := yaml.Unmarshal([]byte(mockYAML), &genericContent)
	if err != nil {
		// If YAML parsing fails, return empty map (no secrets found)
		return sm
	}

	// Use generic extraction for all cases - handles all entity types, nested structures,
	// and type mismatches consistently without needing strict struct unmarshalling.
	return buildSecretMapFromGeneric(genericContent)
}

// rawTemplateEnvPattern matches an unrendered `env "DECK_..."` reference
// inside a template expression, e.g. `${{ env "DECK_RATE_LIMIT_0" }}`. This
// lets BuildSecretMap detect secrets when the caller renders with
// EnvVarsSkip (raw, unrendered template text left in place) rather than
// EnvVarsMock (bare env var name) — both are valid inputs to this function.
var rawTemplateEnvPattern = regexp.MustCompile(`env\s+"(DECK_[^"]*)"`)

// isSecretFieldValue reports whether a field's rendered value indicates it
// was templated from a DECK_-prefixed environment variable, under either
// EnvVarsMock rendering (bare name, e.g. "DECK_X") or EnvVarsSkip rendering
// (raw template text, e.g. `${{ env "DECK_X" }}`).
func isSecretFieldValue(s string) bool {
	if strings.HasPrefix(s, "DECK_") {
		return true
	}
	return rawTemplateEnvPattern.MatchString(s)
}

// leafFieldName extracts the plain field name masking looks up by (the JSON
// key a struct field serializes as), from a full walk path. For a slice
// field like "methods", walkObject produces per-element paths such as
// "methods[0]" — the array index must be stripped so a templated element
// is still attributed to "methods", matching what maskFieldsByName looks up
// via the struct field's JSON tag (which never includes an index).
func leafFieldName(fieldName string) string {
	parts := strings.Split(fieldName, ".")
	last := parts[len(parts)-1]
	if idx := strings.Index(last, "["); idx != -1 {
		last = last[:idx]
	}
	return last
}

// discoverConsumerCredentials uses reflection to extract credential metadata from FConsumer.
// It reads the "credential" struct tag (format: "credentialType,identifyingField") and
// returns a map from YAML field name to credential type and identifying field.
// This ensures credentials are always in sync with the FConsumer struct definition.
func discoverConsumerCredentials() map[string]struct {
	credType         string
	identifyingField string
} {
	result := make(map[string]struct {
		credType         string
		identifyingField string
	})

	// Reflect on FConsumer struct to find credential fields
	t := reflect.TypeFor[FConsumer]()

	for field := range t.Fields() {

		// Check for "credential" tag (e.g., "keyauth_credential,key")
		credTag, hasTag := field.Tag.Lookup("credential")
		if !hasTag || credTag == "" {
			continue
		}

		// Parse the tag format: "credentialType,identifyingField"
		credType, identifyingField, ok := strings.Cut(credTag, ",")
		if !ok {
			continue
		}

		credType = strings.TrimSpace(credType)
		identifyingField = strings.TrimSpace(identifyingField)

		// Get the YAML field name from the json tag
		jsonTag, ok := field.Tag.Lookup("json")
		if !ok || jsonTag == "" {
			continue
		}

		// Extract just the field name (before any commas)
		yamlFieldName, _, _ := strings.Cut(jsonTag, ",")
		if yamlFieldName == "" || yamlFieldName == "-" {
			continue
		}

		result[yamlFieldName] = struct {
			credType         string
			identifyingField string
		}{
			credType:         credType,
			identifyingField: identifyingField,
		}
	}

	return result
}

// buildSecretMapFromGeneric extracts secret field information from a generic
// map/interface{} structure parsed from YAML. Used as a fallback when strict
// struct unmarshaling fails due to type mismatches (e.g., string values in
// int fields from EnvVarsExpand-rendered templates).
// This walks the generic structure for ALL entity types and records any field
// containing a DECK_* prefix or template expression as a secret field.
func buildSecretMapFromGeneric(content map[string]any) SecretMap {
	sm := make(SecretMap)

	// Helper to recursively detect and record secrets in any value
	var walkForSecrets func(path string, val any, baseKey EntityKey)
	walkForSecrets = func(path string, val any, baseKey EntityKey) {
		switch v := val.(type) {
		case map[string]any:
			for k, child := range v {
				childPath := k
				if path != "" {
					childPath = path + "." + k
				}
				walkForSecrets(childPath, child, baseKey)
			}
		case []any:
			for _, child := range v {
				walkForSecrets(path, child, baseKey)
			}
		case string:
			// Check if this value is a secret (DECK_ prefix or template pattern)
			if isSecretFieldValue(v) {
				fieldName := leafFieldName(path)
				if sm[baseKey] == nil {
					sm[baseKey] = make(map[string]bool)
				}
				sm[baseKey][fieldName] = true
			}
		}
	}

	// Process services and their nested entities
	if services, ok := content["services"].([]any); ok {
		for _, svcVal := range services {
			if svcMap, ok := svcVal.(map[string]any); ok {
				svcName := getStringField(svcMap, "name")
				svcID := getStringField(svcMap, "id")

				// Process service's own fields (excluding nested routes/plugins)
				svcMapCopy := excludeNestedEntities(svcMap, "routes", "plugins")
				if svcID != "" {
					walkForSecrets("", svcMapCopy, SimpleKey("service", svcName, svcID))
				}
				walkForSecrets("", svcMapCopy, SimpleKey("service", svcName, ""))

				// Process nested routes within this service
				if routes, ok := svcMap["routes"].([]any); ok {
					for _, routeVal := range routes {
						if routeMap, ok := routeVal.(map[string]any); ok {
							routeName := getStringField(routeMap, "name")
							routeID := getStringField(routeMap, "id")

							// Process route's own fields (excluding nested plugins)
							routeMapCopy := excludeNestedEntities(routeMap, "plugins")
							if routeID != "" {
								walkForSecrets("", routeMapCopy, SimpleKey("route", routeName, routeID))
							}
							walkForSecrets("", routeMapCopy, SimpleKey("route", routeName, ""))

							// Process nested plugins within this route
							if plugins, ok := routeMap["plugins"].([]any); ok {
								for _, pluginVal := range plugins {
									if pluginMap, ok := pluginVal.(map[string]any); ok {
										processGenericPlugin(pluginMap, "", routeName, "", "", "", walkForSecrets)
									}
								}
							}
						}
					}
				}

				// Process nested plugins within this service
				if plugins, ok := svcMap["plugins"].([]any); ok {
					for _, pluginVal := range plugins {
						if pluginMap, ok := pluginVal.(map[string]any); ok {
							processGenericPlugin(pluginMap, svcName, "", "", "", "", walkForSecrets)
						}
					}
				}
			}
		}
	}

	// Process routes
	if routes, ok := content["routes"].([]any); ok {
		for _, routeVal := range routes {
			if routeMap, ok := routeVal.(map[string]any); ok {
				routeName := getStringField(routeMap, "name")
				routeID := getStringField(routeMap, "id")

				// Process route's own fields (excluding nested plugins)
				routeMapCopy := excludeNestedEntities(routeMap, "plugins")
				if routeID != "" {
					walkForSecrets("", routeMapCopy, SimpleKey("route", routeName, routeID))
				}
				walkForSecrets("", routeMapCopy, SimpleKey("route", routeName, ""))

				// Process nested plugins within this route
				if plugins, ok := routeMap["plugins"].([]any); ok {
					for _, pluginVal := range plugins {
						if pluginMap, ok := pluginVal.(map[string]any); ok {
							processGenericPlugin(pluginMap, "", routeName, "", "", "", walkForSecrets)
						}
					}
				}
			}
		}
	}

	// Process consumers and their nested entities
	if consumers, ok := content["consumers"].([]any); ok {
		for _, consumerVal := range consumers {
			if consumerMap, ok := consumerVal.(map[string]any); ok {
				consumerName := getStringField(consumerMap, "username")
				consumerID := getStringField(consumerMap, "id")

				// Process consumer's own fields (excluding nested credentials/plugins)
				consumerMapCopy := excludeNestedEntities(consumerMap, "basicauth_credentials", "keyauth_credentials",
					"hmacauth_credentials", "jwt_secrets", "oauth2_credentials", "plugins")
				if consumerID != "" {
					walkForSecrets("", consumerMapCopy, SimpleKey("consumer", consumerName, consumerID))
				}
				walkForSecrets("", consumerMapCopy, SimpleKey("consumer", consumerName, ""))

				// Process nested credentials discovered via reflection from FConsumer struct tags
				consumerCredentialTypes := discoverConsumerCredentials()
				for yamlFieldName, credInfo := range consumerCredentialTypes {
					if creds, ok := consumerMap[yamlFieldName].([]any); ok {
						for _, credVal := range creds {
							if credMap, ok := credVal.(map[string]any); ok {
								// Extract the credential's identifying field (discovered from struct tag)
								credKey := getStringField(credMap, credInfo.identifyingField)
								credID := getStringField(credMap, "id")

								// Record with consumer scope using the credential type from struct tag
								if credID != "" {
									walkForSecrets("", credMap, CredentialKey(credInfo.credType, credKey, consumerName, credID))
								}
								walkForSecrets("", credMap, CredentialKey(credInfo.credType, credKey, consumerName, ""))
							}
						}
					}
				}

				// Process nested plugins within this consumer
				if plugins, ok := consumerMap["plugins"].([]any); ok {
					for _, pluginVal := range plugins {
						if pluginMap, ok := pluginVal.(map[string]any); ok {
							processGenericPlugin(pluginMap, "", "", "", consumerName, "", walkForSecrets)
						}
					}
				}
			}
		}
	}

	// Process consumer_groups
	if consumerGroups, ok := content["consumer_groups"].([]any); ok {
		for _, cgVal := range consumerGroups {
			if cgMap, ok := cgVal.(map[string]any); ok {
				cgName := getStringField(cgMap, "name")
				cgID := getStringField(cgMap, "id")
				if cgID != "" {
					walkForSecrets("", cgMap, SimpleKey("consumer_group", cgName, cgID))
				}
				walkForSecrets("", cgMap, SimpleKey("consumer_group", cgName, ""))
			}
		}
	}

	// Process certificates
	if certificates, ok := content["certificates"].([]any); ok {
		for _, certVal := range certificates {
			if certMap, ok := certVal.(map[string]any); ok {
				certID := getStringField(certMap, "id")
				walkForSecrets("", certMap, SimpleKey("certificate", "", certID))
			}
		}
	}

	// Process ca_certificates
	if caCerts, ok := content["ca_certificates"].([]any); ok {
		for _, caCertVal := range caCerts {
			if caCertMap, ok := caCertVal.(map[string]any); ok {
				caCertID := getStringField(caCertMap, "id")
				walkForSecrets("", caCertMap, SimpleKey("ca_certificate", "", caCertID))
			}
		}
	}

	// Process keys
	if keys, ok := content["keys"].([]any); ok {
		for _, keyVal := range keys {
			if keyMap, ok := keyVal.(map[string]any); ok {
				keyName := getStringField(keyMap, "name")
				keyID := getStringField(keyMap, "id")
				if keyID != "" {
					walkForSecrets("", keyMap, SimpleKey("key", keyName, keyID))
				}
				walkForSecrets("", keyMap, SimpleKey("key", keyName, ""))
			}
		}
	}

	// Process key_sets
	if keySets, ok := content["key_sets"].([]any); ok {
		for _, ksVal := range keySets {
			if ksMap, ok := ksVal.(map[string]any); ok {
				ksName := getStringField(ksMap, "name")
				ksID := getStringField(ksMap, "id")
				if ksID != "" {
					walkForSecrets("", ksMap, SimpleKey("key_set", ksName, ksID))
				}
				walkForSecrets("", ksMap, SimpleKey("key_set", ksName, ""))
			}
		}
	}

	// Process upstreams
	if upstreams, ok := content["upstreams"].([]any); ok {
		for _, upstreamVal := range upstreams {
			if upstreamMap, ok := upstreamVal.(map[string]any); ok {
				upstreamName := getStringField(upstreamMap, "name")
				upstreamID := getStringField(upstreamMap, "id")
				if upstreamID != "" {
					walkForSecrets("", upstreamMap, SimpleKey("upstream", upstreamName, upstreamID))
				}
				walkForSecrets("", upstreamMap, SimpleKey("upstream", upstreamName, ""))
			}
		}
	}

	// Process vaults
	if vaults, ok := content["vaults"].([]any); ok {
		for _, vaultVal := range vaults {
			if vaultMap, ok := vaultVal.(map[string]any); ok {
				vaultName := getStringField(vaultMap, "name")
				vaultID := getStringField(vaultMap, "id")
				if vaultID != "" {
					walkForSecrets("", vaultMap, SimpleKey("vault", vaultName, vaultID))
				}
				walkForSecrets("", vaultMap, SimpleKey("vault", vaultName, ""))
			}
		}
	}

	// Process filter_chains
	if filterChains, ok := content["filter_chains"].([]any); ok {
		for _, fcVal := range filterChains {
			if fcMap, ok := fcVal.(map[string]any); ok {
				fcName := getStringField(fcMap, "name")
				fcID := getStringField(fcMap, "id")
				if fcID != "" {
					walkForSecrets("", fcMap, SimpleKey("filter_chain", fcName, fcID))
				}
				walkForSecrets("", fcMap, SimpleKey("filter_chain", fcName, ""))
			}
		}
	}

	// Process licenses
	if licenses, ok := content["licenses"].([]any); ok {
		for _, licenseVal := range licenses {
			if licenseMap, ok := licenseVal.(map[string]any); ok {
				licenseID := getStringField(licenseMap, "id")
				walkForSecrets("", licenseMap, SimpleKey("license", "", licenseID))
			}
		}
	}

	// Process rbac_roles
	if rbacRoles, ok := content["rbac_roles"].([]any); ok {
		for _, roleVal := range rbacRoles {
			if roleMap, ok := roleVal.(map[string]any); ok {
				roleID := getStringField(roleMap, "id")
				walkForSecrets("", roleMap, SimpleKey("rbac_role", "", roleID))
			}
		}
	}

	// Process ai_models
	if aiModels, ok := content["ai_models"].([]any); ok {
		for _, modelVal := range aiModels {
			if modelMap, ok := modelVal.(map[string]any); ok {
				modelName := getStringField(modelMap, "name")
				modelID := getStringField(modelMap, "id")
				if modelID != "" {
					walkForSecrets("", modelMap, SimpleKey("ai_model", modelName, modelID))
				}
				walkForSecrets("", modelMap, SimpleKey("ai_model", modelName, ""))
			}
		}
	}

	// Process custom_entities
	if customEntities, ok := content["custom_entities"].([]any); ok {
		for _, ceVal := range customEntities {
			if ceMap, ok := ceVal.(map[string]any); ok {
				ceID := getStringField(ceMap, "id")
				walkForSecrets("", ceMap, SimpleKey("custom_entity", "", ceID))
			}
		}
	}

	// Process service_packages
	if servicePackages, ok := content["service_packages"].([]any); ok {
		for _, spVal := range servicePackages {
			if spMap, ok := spVal.(map[string]any); ok {
				spName := getStringField(spMap, "name")
				spID := getStringField(spMap, "id")
				if spID != "" {
					walkForSecrets("", spMap, SimpleKey("service_package", spName, spID))
				}
				walkForSecrets("", spMap, SimpleKey("service_package", spName, ""))
			}
		}
	}

	// Process partials
	if partials, ok := content["partials"].([]any); ok {
		for _, partialVal := range partials {
			if partialMap, ok := partialVal.(map[string]any); ok {
				partialName := getStringField(partialMap, "name")
				partialID := getStringField(partialMap, "id")
				if partialID != "" {
					walkForSecrets("", partialMap, SimpleKey("partial", partialName, partialID))
				}
				walkForSecrets("", partialMap, SimpleKey("partial", partialName, ""))
			}
		}
	}

	// Process plugins (top-level)
	if plugins, ok := content["plugins"].([]any); ok {
		for _, pluginVal := range plugins {
			if pluginMap, ok := pluginVal.(map[string]any); ok {
				pluginName := getStringField(pluginMap, "name")
				pluginID := getStringField(pluginMap, "id")
				baseKey := PluginKey(pluginName, "", "", "", "", "", pluginID)
				walkForSecrets("", pluginMap, baseKey)
			}
		}
	}

	// Process cloned_plugins
	if clonedPlugins, ok := content["cloned_plugins"].([]any); ok {
		for _, cpVal := range clonedPlugins {
			if cpMap, ok := cpVal.(map[string]any); ok {
				cpID := getStringField(cpMap, "id")
				walkForSecrets("", cpMap, SimpleKey("cloned_plugin", "", cpID))
			}
		}
	}

	// Process custom_plugins
	if customPlugins, ok := content["custom_plugins"].([]any); ok {
		for _, customVal := range customPlugins {
			if customMap, ok := customVal.(map[string]any); ok {
				customID := getStringField(customMap, "id")
				walkForSecrets("", customMap, SimpleKey("custom_plugin", "", customID))
			}
		}
	}

	return sm
}

// getStringField safely extracts a string field from a generic map.
func getStringField(m map[string]any, fieldName string) string {
	if v, ok := m[fieldName]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// excludeNestedEntities returns a copy of the map without specified nested entity fields.
// Used to process an entity's own secrets without recursing into nested entities.
func excludeNestedEntities(m map[string]any, excludeFields ...string) map[string]any {
	excludeSet := make(map[string]bool)
	for _, field := range excludeFields {
		excludeSet[field] = true
	}

	result := make(map[string]any)
	for k, v := range m {
		if !excludeSet[k] {
			result[k] = v
		}
	}
	return result
}

// processGenericPlugin processes a plugin from a generic map with proper scope.
// This is used by buildSecretMapFromGeneric to handle nested plugins with the correct keys.
func processGenericPlugin(pluginMap map[string]any, svcName, routeName, cgName, consumerName, instanceName string,
	walkFunc func(string, any, EntityKey),
) {
	pluginName := getStringField(pluginMap, "name")
	pluginID := getStringField(pluginMap, "id")

	// Create plugin key with proper scope
	key := PluginKey(pluginName, instanceName, svcName, routeName, consumerName, cgName, pluginID)

	// Process plugin's own fields
	pluginMapCopy := excludeNestedEntities(pluginMap)
	walkFunc("", pluginMapCopy, key)

	// Record fallback keys for templated scopes
	isTemplatedSvc := strings.HasPrefix(svcName, "${{") || rawTemplateEnvPattern.MatchString(svcName)
	isTemplatedRoute := strings.HasPrefix(routeName, "${{") || rawTemplateEnvPattern.MatchString(routeName)
	isTemplatedConsumer := strings.HasPrefix(consumerName, "${{") || rawTemplateEnvPattern.MatchString(consumerName)
	isTemplatedCG := strings.HasPrefix(cgName, "${{") || rawTemplateEnvPattern.MatchString(cgName)

	if isTemplatedSvc || isTemplatedRoute || isTemplatedConsumer || isTemplatedCG {
		// Try without each templated scope
		if isTemplatedSvc {
			keyNoSvc := PluginKey(pluginName, instanceName, "", routeName, consumerName, cgName, pluginID)
			walkFunc("", pluginMapCopy, keyNoSvc)
		}
		if isTemplatedRoute {
			keyNoRoute := PluginKey(pluginName, instanceName, svcName, "", consumerName, cgName, pluginID)
			walkFunc("", pluginMapCopy, keyNoRoute)
		}
		if isTemplatedConsumer {
			keyNoConsumer := PluginKey(pluginName, instanceName, svcName, routeName, "", cgName, pluginID)
			walkFunc("", pluginMapCopy, keyNoConsumer)
		}
		if isTemplatedCG {
			keyNoCG := PluginKey(pluginName, instanceName, svcName, routeName, consumerName, "", pluginID)
			walkFunc("", pluginMapCopy, keyNoCG)
		}
	}
}
