package schema

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tidwall/gjson"
)

var (
	defaultsCache   = map[string]interface{}{}
	defaultsCacheMu sync.Mutex
)

// GetDefaultsFromSchema parses a Kong entity schema and returns a map of
// field names to their default values.
//
// If cacheKey is non-empty, results are cached and subsequent calls with the
// same key return the cached value. Use a key like "entityType::identifier"
// (e.g. "plugins::rate-limiting") to avoid collisions.
func GetDefaultsFromSchema(schema map[string]interface{}, cacheKey string) (map[string]interface{}, error) {
	if cacheKey != "" {
		defaultsCacheMu.Lock()
		defer defaultsCacheMu.Unlock()

		if cached, exists := defaultsCache[cacheKey]; exists {
			return cached.(map[string]interface{}), nil
		}
	}

	jsonb, err := json.Marshal(&schema)
	if err != nil {
		return nil, err
	}
	gjsonSchema := gjson.ParseBytes(jsonb)
	defaults := ParseSchemaForDefaults(gjsonSchema, make(map[string]interface{}))
	if defaults == nil {
		return nil, fmt.Errorf("error parsing schema for defaults")
	}

	if cacheKey != "" {
		defaultsCache[cacheKey] = defaults
	}

	return defaults, nil
}

// ResetDefaultsCache clears the defaults cache. This is primarily useful in tests.
func ResetDefaultsCache() {
	defaultsCacheMu.Lock()
	defer defaultsCacheMu.Unlock()
	defaultsCache = map[string]interface{}{}
}

// ParseSchemaForDefaults walks a gjson schema result and extracts all fields
// that have default values, returning a nested map of field name → default value.
//
// It handles:
//   - Simple "default" fields
//   - Nested "record" type fields (recursive)
//   - "shorthand_fields" with "translate_backwards" or "deprecation.replaced_with" paths
//   - Konnect's "value" wrapper for credentials
func ParseSchemaForDefaults(schema gjson.Result, defaultFields map[string]interface{}) map[string]interface{} {
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
			nestedMap := ParseSchemaForDefaults(fieldSchema, make(map[string]interface{}))
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

	// Handle shorthand_fields by finding defaults from their "replaced_with" or "translate_backwards" paths
	shorthandFields := schema.Get("shorthand_fields")
	if shorthandFields.Exists() && shorthandFields.IsArray() {
		shorthandFields.ForEach(func(_, value gjson.Result) bool {
			ms := value.Map()
			for fieldName := range ms {
				fieldSchema := value.Get(fieldName)
				replacements := fieldSchema.Get("deprecation.replaced_with.#.path").Array()
				var replacedPaths []string
				var pathArray []gjson.Result

				if len(replacements) > 0 {
					pathArray = replacements[0].Array()
				} else {
					backwardTranslation := fieldSchema.Get("translate_backwards")
					if backwardTranslation.Exists() {
						pathArray = backwardTranslation.Array()
					}
				}

				replacedPaths = make([]string, len(pathArray))
				for i, segment := range pathArray {
					replacedPaths[i] = segment.String()
				}

				if len(replacedPaths) > 0 {
					defaultValue := FindDefaultInReplacementPath(schema, replacedPaths)
					if defaultValue.Exists() {
						defaultFields[fieldName] = defaultValue.Value()
					}
				}
			}
			return true
		})
	}

	// All credentials' schemas in Konnect are embedded under "value" field
	// which doesn't match gateway schema or internal go-kong representation.
	// Merge values from "value" field to the defaultFields map directly.
	if valueMap, ok := defaultFields["value"]; ok {
		for k, v := range valueMap.(map[string]interface{}) {
			defaultFields[k] = v
		}
		delete(defaultFields, "value")
	}

	return defaultFields
}

// FindDefaultInReplacementPath traverses the schema to find the default value at the specified path.
func FindDefaultInReplacementPath(schema gjson.Result, pathSegments []string) gjson.Result {
	schemaFields := schema.Get("fields")
	if schemaFields.Type == gjson.Null {
		schemaFields = schema.Get("properties")
	}

	current := schemaFields

	for i, segment := range pathSegments {
		var next gjson.Result
		isLastSegment := i == len(pathSegments)-1
		isArray := current.IsArray()

		current.ForEach(func(key, value gjson.Result) bool {
			var fieldExists bool
			var fieldValue gjson.Result

			if isArray {
				fieldExists = value.Get(segment).Exists()
				fieldValue = value
			} else {
				fieldExists = key.String() == segment
				fieldValue = current
			}

			if fieldExists {
				if isLastSegment {
					next = fieldValue.Get(segment + ".default")
				} else {
					next = fieldValue.Get(segment + ".fields")
					if next.Type == gjson.Null {
						next = fieldValue.Get(segment + ".properties")
					}
				}
				return false
			}
			return true
		})

		if !next.Exists() {
			return gjson.Result{}
		}

		current = next
	}

	return current
}
