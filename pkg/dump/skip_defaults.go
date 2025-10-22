package dump

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/tidwall/gjson"
)

func removeDefaultsFromState[T any](entities []T, schemaFetcher *SchemaFetcher, entityType string) error {
	if schemaFetcher == nil {
		return fmt.Errorf("schemaFetcher is nil")
	}

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

	fmt.Println(entityIdentifier)

	defaultFields := make(map[string]interface{})
	jsonb, err := json.Marshal(&schema)
	if err != nil {
		return err
	}
	gjsonSchema := gjson.ParseBytes((jsonb))

	defaultFields = parseSchemaForDefaults(gjsonSchema, defaultFields)
	if defaultFields == nil {
		return fmt.Errorf("error parsing schema for defaults: %w", err)
	}

	err = parseEntityWithDefaults(v, defaultFields)
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
	defaultRecordValue := schema.Get("default")

	schemaFields.ForEach(func(_, value gjson.Result) bool {
		ms := value.Map()
		fname := ""
		for k := range ms {
			fname = k
			break
		}

		fieldSchema := value.Get(fname)
		if fieldSchema.Get("fields").Exists() {
			nestedMap := parseSchemaForDefaults(fieldSchema, make(map[string]interface{}))
			if nestedMap == nil {
				return false
			}
			defaultFields[fname] = nestedMap
		}

		if defaultRecordValue.Exists() && defaultRecordValue.Get(fname).Exists() {
			value = defaultRecordValue.Get(fname)
		} else {
			value = value.Get(fname + ".default")
		}

		if value.Exists() {
			defaultFields[fname] = value.Value()
		}

		return true
	})

	return defaultFields
}

func parseEntityWithDefaults(entity reflect.Value, defaultFields map[string]interface{}) error {
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
		snakeCaseFieldName := toSnakeCase(fieldName)
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

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteByte('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
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
		fieldElem := fieldSlice.Index(i)
		defaultElem := defaultSlice.Index(i).Interface()

		if compareValues(fieldElem, defaultElem) {
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

		fieldVal = reflect.ValueOf(fieldVal.Interface())
		defaultVal = reflect.ValueOf(defaultVal.Interface())

		if fieldVal.Kind() == reflect.Map && defaultVal.Kind() == reflect.Map {
			nestedResult := compareMaps(fieldVal, defaultVal)
			if nestedMap, ok := nestedResult.(map[string]interface{}); ok && len(nestedMap) > 0 {
				newMap[key.String()] = nestedResult
			}
		} else if !compareValues(fieldVal.Interface(), defaultVal.Interface()) {
			newMap[key.String()] = fieldVal.Interface()
		}
	}

	return newMap
}
