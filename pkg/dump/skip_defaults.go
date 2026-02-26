package dump

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ettle/strcase"
	schema_pkg "github.com/kong/go-database-reconciler/pkg/schema"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"golang.org/x/sync/errgroup"
)

func removeDefaultsFromState(ctx context.Context, group *errgroup.Group,
	state *utils.KongRawState, registry *schema_pkg.Registry,
) {
	// Consumer Groups
	group.Go(func() error {
		for i, cg := range state.ConsumerGroups {
			if err := ctx.Err(); err != nil {
				return err
			}

			consumerGroup := cg.ConsumerGroup
			consumers := cg.Consumers
			plugins := cg.Plugins

			err := processStateEntities([]interface{}{consumerGroup}, registry, "consumer_groups")
			if err != nil {
				return fmt.Errorf("error removing defaults from consumer_groups: %w", err)
			}

			err = processStateEntities(consumers, registry, "consumers")
			if err != nil {
				return fmt.Errorf("error removing defaults from consumers: %w", err)
			}

			err = processStateEntities(plugins, registry, "plugins")
			if err != nil {
				return fmt.Errorf("error removing defaults from plugins: %w", err)
			}

			state.ConsumerGroups[i] = cg
		}
		return nil
	})

	// Consumers
	group.Go(func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := processStateEntities(state.Consumers, registry, "consumers")
		if err != nil {
			return fmt.Errorf("error removing defaults from consumers: %w", err)
		}
		return nil
	})

	// Key Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.KeyAuths, registry, "keyauth_credentials")
		if err != nil {
			return fmt.Errorf("error removing defaults from key auths: %w", err)
		}
		return nil
	})

	// HMAC Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.HMACAuths, registry, "hmacauth_credentials")
		if err != nil {
			return fmt.Errorf("error removing defaults from hmac auths: %w", err)
		}
		return nil
	})

	// JWT Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.JWTAuths, registry, "jwt_secrets")
		if err != nil {
			return fmt.Errorf("error removing defaults from jwt auths: %w", err)
		}
		return nil
	})

	// Basic Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.BasicAuths, registry, "basicauth_credentials")
		if err != nil {
			return fmt.Errorf("error removing defaults from basic auths: %w", err)
		}
		return nil
	})

	// OAuth2 credentials
	group.Go(func() error {
		err := processStateEntities(state.Oauth2Creds, registry, "oauth2_credentials")
		if err != nil {
			return fmt.Errorf("error removing defaults from oauth2 creds: %w", err)
		}
		return nil
	})

	// ACL Groups
	group.Go(func() error {
		err := processStateEntities(state.ACLGroups, registry, "acls")
		if err != nil {
			return fmt.Errorf("error removing defaults from acl groups: %w", err)
		}
		return nil
	})

	// mTLS Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.MTLSAuths, registry, "mtls_auth_credentials")
		if err != nil {
			return fmt.Errorf("error removing defaults from mtls auths: %w", err)
		}
		return nil
	})

	// Services
	group.Go(func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := processStateEntities(state.Services, registry, "services")
		if err != nil {
			return fmt.Errorf("error removing defaults from services: %w", err)
		}
		return nil
	})

	// Routes
	group.Go(func() error {
		err := processStateEntities(state.Routes, registry, "routes")
		if err != nil {
			return fmt.Errorf("error removing defaults from routes: %w", err)
		}
		return nil
	})

	// Plugins
	group.Go(func() error {
		err := processStateEntities(state.Plugins, registry, "plugins")
		if err != nil {
			return fmt.Errorf("error removing defaults from plugins: %w", err)
		}
		return nil
	})

	// Filter Chains
	group.Go(func() error {
		err := processStateEntities(state.FilterChains, registry, "filter_chains")
		if err != nil {
			return fmt.Errorf("error removing defaults from filter chains: %w", err)
		}
		return nil
	})

	// Certificates
	group.Go(func() error {
		err := processStateEntities(state.Certificates, registry, "certificates")
		if err != nil {
			return fmt.Errorf("error removing defaults from certificates: %w", err)
		}
		return nil
	})

	// CA Certificates
	group.Go(func() error {
		err := processStateEntities(state.CACertificates, registry, "ca_certificates")
		if err != nil {
			return fmt.Errorf("error removing defaults from ca certificates: %w", err)
		}
		return nil
	})

	// SNIs
	group.Go(func() error {
		err := processStateEntities(state.SNIs, registry, "snis")
		if err != nil {
			return fmt.Errorf("error removing defaults from snis: %w", err)
		}
		return nil
	})

	// Upstreams
	group.Go(func() error {
		err := processStateEntities(state.Upstreams, registry, "upstreams")
		if err != nil {
			return fmt.Errorf("error removing defaults from upstreams: %w", err)
		}
		return nil
	})

	// Targets
	group.Go(func() error {
		err := processStateEntities(state.Targets, registry, "targets")
		if err != nil {
			return fmt.Errorf("error removing defaults from targets: %w", err)
		}
		return nil
	})

	// Vaults
	group.Go(func() error {
		err := processStateEntities(state.Vaults, registry, "vaults")
		if err != nil {
			return fmt.Errorf("error removing defaults from vaults: %w", err)
		}
		return nil
	})

	// Partials
	group.Go(func() error {
		err := processStateEntities(state.Partials, registry, "partials")
		if err != nil {
			return fmt.Errorf("error removing defaults from partials: %w", err)
		}
		return nil
	})

	// Keys
	group.Go(func() error {
		err := processStateEntities(state.Keys, registry, "keys")
		if err != nil {
			return fmt.Errorf("error removing defaults from keys: %w", err)
		}
		return nil
	})

	// Key Sets
	group.Go(func() error {
		err := processStateEntities(state.KeySets, registry, "key_sets")
		if err != nil {
			return fmt.Errorf("error removing defaults from key sets: %w", err)
		}
		return nil
	})

	// Licenses
	group.Go(func() error {
		err := processStateEntities(state.Licenses, registry, "licenses")
		if err != nil {
			return fmt.Errorf("error removing defaults from licenses: %w", err)
		}
		return nil
	})

	// RBAC Roles
	group.Go(func() error {
		err := processStateEntities(state.RBACRoles, registry, "rbac_roles")
		if err != nil {
			return fmt.Errorf("error removing defaults from rbac roles: %w", err)
		}
		return nil
	})

	// RBAC Endpoint Permissions
	group.Go(func() error {
		err := processStateEntities(state.RBACEndpointPermissions, registry, "rbac_endpoint_permissions")
		if err != nil {
			return fmt.Errorf("error removing defaults from rbac endpoint permissions: %w", err)
		}
		return nil
	})
}

func processStateEntities[T any](entities []T, registry *schema_pkg.Registry, entityType string) error {
	if len(entities) == 0 {
		return nil
	}

	for _, e := range entities {
		err := removeDefaultsFromEntity(e, entityType, registry)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeDefaultsFromEntity(entity interface{}, entityType string, registry *schema_pkg.Registry) error {
	ptr := reflect.ValueOf(entity)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("entity is not a pointer")
	}

	v := reflect.Indirect(ptr)

	entityIdentifier, err := getEntityIdentifier(v, entityType)
	if err != nil {
		return fmt.Errorf("error getting entity identifier for schema fetching: %w", err)
	}

	schema, err := registry.GetSchema(entityType, entityIdentifier)
	if err != nil {
		return fmt.Errorf("error fetching schema for entity %s of type %s: %w", entityIdentifier, entityType, err)
	}

	cacheKey := entityType + "::" + entityIdentifier
	defaultFields, err := schema_pkg.GetDefaultsFromSchema(schema, cacheKey)
	if err != nil {
		return err
	}

	// no processing needed if no default fields found
	if len(defaultFields) == 0 {
		return nil
	}

	defaultFields = handleExceptions(entityType, defaultFields)

	err = stripDefaultValuesFromEntity(v, defaultFields)
	if err != nil {
		return fmt.Errorf("error parsing entity with defaults: %w", err)
	}

	return nil
}

func handleExceptions(entityType string, defaultFields map[string]interface{}) map[string]interface{} {
	// Don't skip default for "algorithm" field in jwt_secrets
	if entityType == "jwt_secrets" {
		delete(defaultFields, "algorithm")
	}

	return defaultFields
}

func getEntityIdentifier(v reflect.Value, entityType string) (string, error) {
	var zero reflect.Value
	entityIdentifier := entityType

	switch entityType {
	case "plugins", "vaults":
		name := v.FieldByName("Name")
		if name == zero {
			return "", fmt.Errorf("entity %s has no Name field for schema fetching", entityType)
		}

		entityIdentifier = *name.Interface().(*string)
	case "partials":
		partialType := v.FieldByName("Type")
		if partialType == zero {
			return "", fmt.Errorf("entity partial has no Type field for schema fetching")
		}

		entityIdentifier = *partialType.Interface().(*string)
	}

	return entityIdentifier, nil
}

func stripDefaultValuesFromEntity(entity reflect.Value, defaultFields map[string]interface{}) error {
	if entity.Kind() != reflect.Struct {
		return fmt.Errorf("entity is not a struct")
	}

	entityType := entity.Type()
	for i := 0; i < entity.NumField(); i++ {
		field := entity.Field(i)
		fieldType := entityType.Field(i)
		fieldName := fieldType.Name

		if !field.CanInterface() {
			continue
		}
		fieldValue := field.Interface()
		snakeCaseFieldName := strcase.ToSnake(fieldName)
		if defaultValue, hasDefault := defaultFields[snakeCaseFieldName]; hasDefault {
			if compareValues(fieldValue, defaultValue) {
				if field.CanSet() {
					field.SetZero()
				}
			}

			fieldVal := reflect.ValueOf(fieldValue)
			defaultVal := reflect.ValueOf(defaultValue)

			if fieldVal.Kind() == reflect.Map && defaultVal.Kind() == reflect.Map {
				newMap := compareMaps(fieldVal, defaultVal)
				if field.CanSet() {
					field.Set(reflect.ValueOf(newMap))
				} else {
					return fmt.Errorf("cannot set field value for key: %s", fieldName)
				}
			}
		}
	}

	return nil
}

func isNumericKind(kind reflect.Kind) bool {
	//nolint:exhaustive
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func compareNumeric(fieldVal, defaultVal reflect.Value) bool {
	// Convert both values to float64 for comparison
	var fieldFloat, defaultFloat float64

	// Convert field value to float64
	//nolint:exhaustive
	switch fieldVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldFloat = float64(fieldVal.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fieldFloat = float64(fieldVal.Uint())
	case reflect.Float32, reflect.Float64:
		fieldFloat = fieldVal.Float()
	default:
		return false
	}

	// Convert default value to float64
	//nolint:exhaustive
	switch defaultVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		defaultFloat = float64(defaultVal.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		defaultFloat = float64(defaultVal.Uint())
	case reflect.Float32, reflect.Float64:
		defaultFloat = defaultVal.Float()
	default:
		return false
	}

	return fieldFloat == defaultFloat
}

func compareValues(fieldValue interface{}, defaultValue interface{}) bool {
	if fieldValue == nil && defaultValue == nil {
		return true
	}
	if fieldValue == nil || defaultValue == nil {
		return false
	}

	fieldVal := reflect.ValueOf(fieldValue)
	defaultVal := reflect.ValueOf(defaultValue)

	if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() {
		fieldVal = fieldVal.Elem()
		fieldValue = fieldVal.Interface()
	}

	if reflect.DeepEqual(fieldValue, defaultValue) {
		return true
	}

	if isNumericKind(fieldVal.Kind()) && isNumericKind(defaultVal.Kind()) {
		return compareNumeric(fieldVal, defaultVal)
	}

	if fieldVal.Kind() == reflect.String && defaultVal.Kind() == reflect.String {
		return fieldVal.String() == defaultVal.String()
	}

	if (fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array) &&
		(defaultVal.Kind() == reflect.Slice || defaultVal.Kind() == reflect.Array) {
		return compareSlices(fieldVal, defaultVal)
	}

	return false
}

func compareSlices(fieldSlice, defaultSlice reflect.Value) bool {
	if fieldSlice.Len() != defaultSlice.Len() {
		return false
	}

	for i := 0; i < fieldSlice.Len(); i++ {
		fieldElem := fieldSlice.Index(i).Interface()
		defaultElem := defaultSlice.Index(i).Interface()

		if !compareValues(fieldElem, defaultElem) {
			return false
		}
	}

	return true
}

func compareMaps(fieldMap, defaultMap reflect.Value) interface{} {
	newMap := make(map[string]interface{})
	for _, key := range fieldMap.MapKeys() {
		fieldVal := fieldMap.MapIndex(key)
		defaultVal := defaultMap.MapIndex(key)

		if !defaultVal.IsValid() {
			value := fieldVal.Interface()
			if value != nil {
				newMap[key.String()] = value
			}
			continue
		}

		if !fieldVal.CanInterface() || !defaultVal.CanInterface() {
			continue // Skip unexported fields
		}

		fieldVal = reflect.ValueOf(fieldVal.Interface())
		defaultVal = reflect.ValueOf(defaultVal.Interface())

		if !fieldVal.IsValid() || !defaultVal.IsValid() {
			continue
		}

		if fieldVal.Kind() == reflect.Map && defaultVal.Kind() == reflect.Map {
			nestedResult := compareMaps(fieldVal, defaultVal)
			if nestedMap, ok := nestedResult.(map[string]interface{}); ok && len(nestedMap) > 0 {
				newMap[key.String()] = nestedResult
			}
		} else if fieldVal.CanInterface() && defaultVal.CanInterface() &&
			!compareValues(fieldVal.Interface(), defaultVal.Interface()) {
			newMap[key.String()] = fieldVal.Interface()
		}
	}

	return newMap
}
