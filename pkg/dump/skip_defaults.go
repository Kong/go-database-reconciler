package dump

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/ettle/strcase"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"
)

var (
	defaultFieldsCache = map[string]interface{}{}
	defaultFieldsMu    sync.Mutex
)

func removeDefaultsFromState(ctx context.Context, group *errgroup.Group,
	state *utils.KongRawState, schemaFetcher *SchemaFetcher,
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

			err := processStateEntities([]interface{}{consumerGroup}, schemaFetcher, "consumer_groups")
			if err != nil {
				return fmt.Errorf("error removing defaults from consumer_groups: %w", err)
			}

			err = processStateEntities(consumers, schemaFetcher, "consumers")
			if err != nil {
				return fmt.Errorf("error removing defaults from consumers: %w", err)
			}

			err = processStateEntities(plugins, schemaFetcher, "plugins")
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
		err := processStateEntities(state.Consumers, schemaFetcher, "consumers")
		if err != nil {
			return fmt.Errorf("error removing defaults from consumers: %w", err)
		}
		return nil
	})

	// Key Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.KeyAuths, schemaFetcher, "keyauth_credentials")
		if err != nil {
			return fmt.Errorf("error removing defaults from key auths: %w", err)
		}
		return nil
	})

	// HMAC Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.HMACAuths, schemaFetcher, "hmacauth_credentials")
		if err != nil {
			return fmt.Errorf("error removing defaults from hmac auths: %w", err)
		}
		return nil
	})

	// JWT Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.JWTAuths, schemaFetcher, "jwt_secrets")
		if err != nil {
			return fmt.Errorf("error removing defaults from jwt auths: %w", err)
		}
		return nil
	})

	// Basic Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.BasicAuths, schemaFetcher, "basicauth_credential")
		if err != nil {
			return fmt.Errorf("error removing defaults from basic auths: %w", err)
		}
		return nil
	})

	// OAuth2 credentials
	group.Go(func() error {
		err := processStateEntities(state.Oauth2Creds, schemaFetcher, "oauth2_credentials")
		if err != nil {
			return fmt.Errorf("error removing defaults from oauth2 creds: %w", err)
		}
		return nil
	})

	// ACL Groups
	group.Go(func() error {
		err := processStateEntities(state.ACLGroups, schemaFetcher, "acls")
		if err != nil {
			return fmt.Errorf("error removing defaults from acl groups: %w", err)
		}
		return nil
	})

	// mTLS Auth credentials
	group.Go(func() error {
		err := processStateEntities(state.MTLSAuths, schemaFetcher, "mtls_auth_credentials")
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
		err := processStateEntities(state.Services, schemaFetcher, "services")
		if err != nil {
			return fmt.Errorf("error removing defaults from services: %w", err)
		}
		return nil
	})

	// Routes
	group.Go(func() error {
		err := processStateEntities(state.Routes, schemaFetcher, "routes")
		if err != nil {
			return fmt.Errorf("error removing defaults from routes: %w", err)
		}
		return nil
	})

	// Plugins
	group.Go(func() error {
		err := processStateEntities(state.Plugins, schemaFetcher, "plugins")
		if err != nil {
			return fmt.Errorf("error removing defaults from plugins: %w", err)
		}
		return nil
	})

	// Filter Chains
	group.Go(func() error {
		err := processStateEntities(state.FilterChains, schemaFetcher, "filter_chains")
		if err != nil {
			return fmt.Errorf("error removing defaults from filter chains: %w", err)
		}
		return nil
	})

	// Certificates
	group.Go(func() error {
		err := processStateEntities(state.Certificates, schemaFetcher, "certificates")
		if err != nil {
			return fmt.Errorf("error removing defaults from certificates: %w", err)
		}
		return nil
	})

	// CA Certificates
	group.Go(func() error {
		err := processStateEntities(state.CACertificates, schemaFetcher, "ca_certificates")
		if err != nil {
			return fmt.Errorf("error removing defaults from ca certificates: %w", err)
		}
		return nil
	})

	// SNIs
	group.Go(func() error {
		err := processStateEntities(state.SNIs, schemaFetcher, "snis")
		if err != nil {
			return fmt.Errorf("error removing defaults from snis: %w", err)
		}
		return nil
	})

	// Upstreams
	group.Go(func() error {
		err := processStateEntities(state.Upstreams, schemaFetcher, "upstreams")
		if err != nil {
			return fmt.Errorf("error removing defaults from upstreams: %w", err)
		}
		return nil
	})

	// Targets
	group.Go(func() error {
		err := processStateEntities(state.Targets, schemaFetcher, "targets")
		if err != nil {
			return fmt.Errorf("error removing defaults from targets: %w", err)
		}
		return nil
	})

	// Vaults
	group.Go(func() error {
		err := processStateEntities(state.Vaults, schemaFetcher, "vaults")
		if err != nil {
			return fmt.Errorf("error removing defaults from vaults: %w", err)
		}
		return nil
	})

	// Partials
	group.Go(func() error {
		err := processStateEntities(state.Partials, schemaFetcher, "partials")
		if err != nil {
			return fmt.Errorf("error removing defaults from partials: %w", err)
		}
		return nil
	})

	// Keys
	group.Go(func() error {
		err := processStateEntities(state.Keys, schemaFetcher, "keys")
		if err != nil {
			return fmt.Errorf("error removing defaults from keys: %w", err)
		}
		return nil
	})

	// Key Sets
	group.Go(func() error {
		err := processStateEntities(state.KeySets, schemaFetcher, "key_sets")
		if err != nil {
			return fmt.Errorf("error removing defaults from key sets: %w", err)
		}
		return nil
	})

	// Licenses
	group.Go(func() error {
		err := processStateEntities(state.Licenses, schemaFetcher, "licenses")
		if err != nil {
			return fmt.Errorf("error removing defaults from licenses: %w", err)
		}
		return nil
	})

	// RBAC Roles
	group.Go(func() error {
		err := processStateEntities(state.RBACRoles, schemaFetcher, "rbac_roles")
		if err != nil {
			return fmt.Errorf("error removing defaults from rbac roles: %w", err)
		}
		return nil
	})

	// RBAC Endpoint Permissions
	group.Go(func() error {
		err := processStateEntities(state.RBACEndpointPermissions, schemaFetcher, "rbac_endpoint_permissions")
		if err != nil {
			return fmt.Errorf("error removing defaults from rbac endpoint permissions: %w", err)
		}
		return nil
	})
}

func processStateEntities[T any](entities []T, schemaFetcher *SchemaFetcher, entityType string) error {
	if len(entities) == 0 {
		return nil
	}

	for _, e := range entities {
		err := removeDefaultsFromEntity(e, entityType, schemaFetcher)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeDefaultsFromEntity(entity interface{}, entityType string, schemaFetcher *SchemaFetcher) error {
	ptr := reflect.ValueOf(entity)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("entity is not a pointer")
	}

	v := reflect.Indirect(ptr)

	entityIdentifier, err := getEntityIdentifier(v, entityType)
	if err != nil {
		return fmt.Errorf("error getting entity identifier for schema fetching: %w", err)
	}

	schema, err := schemaFetcher.getSchema(entityType, entityIdentifier)
	if err != nil {
		return fmt.Errorf("error fetching schema for entity %s of type %s: %w", entityIdentifier, entityType, err)
	}

	defaultFields := make(map[string]interface{})
	jsonb, err := json.Marshal(&schema)
	if err != nil {
		return err
	}
	gjsonSchema := gjson.ParseBytes((jsonb))

	defaultFieldsMu.Lock()
	defer defaultFieldsMu.Unlock()

	cacheKey := entityType + "::" + entityIdentifier

	if defaultFieldsCached, exists := defaultFieldsCache[cacheKey]; exists {
		defaultFields = defaultFieldsCached.(map[string]interface{})
	} else {
		defaultFields = parseSchemaForDefaults(gjsonSchema, defaultFields)
		if defaultFields == nil {
			return fmt.Errorf("error parsing schema for defaults")
		}
		defaultFieldsCache[cacheKey] = defaultFields
	}

	// no processing needed if no default fields found
	if len(defaultFields) == 0 {
		return nil
	}

	err = stripDefaultValuesFromEntity(v, defaultFields)
	if err != nil {
		return fmt.Errorf("error parsing entity with defaults: %w", err)
	}

	return nil
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

func parseSchemaForDefaults(schema gjson.Result, defaultFields map[string]interface{}) map[string]interface{} {
	schemaFields := schema.Get("fields")
	if schemaFields.Type == gjson.Null {
		schemaFields = schema.Get("properties")
	}
	defaultRecordValue := schema.Get("default")

	isObject := false
	if schemaFields.IsObject() {
		isObject = true
	}

	schemaFields.ForEach(func(key, value gjson.Result) bool {
		fname := ""

		var fieldValue gjson.Result
		var fieldSchema gjson.Result

		if isObject && key.Type != gjson.Null {
			fname = key.String()
			fieldSchema = value
		} else {
			ms := value.Map()
			for k := range ms {
				fname = k
				break
			}
			fieldSchema = value.Get(fname)
		}

		if fieldSchema.Get("fields").Exists() || fieldSchema.Get("properties").Exists() {
			nestedMap := parseSchemaForDefaults(fieldSchema, make(map[string]interface{}))
			if nestedMap == nil {
				return false
			}
			defaultFields[fname] = nestedMap
		}

		if isObject {
			fieldValue = value.Get("default")
		} else if defaultRecordValue.Exists() && defaultRecordValue.Get(fname).Exists() {
			fieldValue = defaultRecordValue.Get(fname)
		} else {
			fieldValue = value.Get(fname + ".default")
		}

		if fieldValue.Exists() {
			defaultFields[fname] = fieldValue.Value()
		}

		return true
	})

	// All credentials' schemas in konnect are embedded under "value" field
	// which doesn't match gateway schema or internal go-kong representation
	// Thus, merging values from "value" field to the defaultFields map directly
	// ------------------------------------------------------------------------
	// @TODO: Uncomment the code below if such handling is needed in future.
	// At the moment, only jwt_secrets have a default for "algorithm" which is a required
	// field. Hence, not skipping it.
	// if valueMap, ok := defaultFields["value"]; ok {
	// 	for k, v := range valueMap.(map[string]interface{}) {
	// 		defaultFields[k] = v
	// 	}
	// 	delete(defaultFields, "value")
	// }

	return defaultFields
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
