package file

import (
	"reflect"
	"strconv"
	"strings"
)

// ExtractSecretFieldsFromContent scans rendered state for fields containing DECK_* environment variable references.
// Returns a list of field names (paths) that contain secrets.
//
// Example output for nested structure:
//
//	["services[0].routes[0].plugins[0].config.api_key", "consumers[0].password"]
func ExtractSecretFieldsFromContent(content *Content) []string {
	secretFields := make(map[string]bool)

	walkObject(content, "", func(fieldName string, value interface{}) {
		if strVal, ok := value.(string); ok {
			if strings.HasPrefix(strVal, "DECK_") {
				secretFields[fieldName] = true
			}
		}
	})

	result := make([]string, 0, len(secretFields))
	for fieldName := range secretFields {
		result = append(result, fieldName)
	}
	return result
}

// walkObject recursively traverses any Go object (struct, slice, map) and calls the callback for each value.
// It builds full paths like "services[0].routes[0].plugins[0].config.api_key" as it descends.
func walkObject(obj interface{}, prefix string, callback func(string, interface{})) {
	if obj == nil {
		return
	}

	val := reflect.ValueOf(obj)

	// Handle pointers - dereference them
	for val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		// Walk struct fields
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			fieldValue := val.Field(i)

			// Use JSON tag if available, otherwise use field name
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				jsonTag = field.Name
			}
			jsonTag = strings.Split(jsonTag, ",")[0]

			fieldName := jsonTag
			if prefix != "" {
				fieldName = prefix + "." + jsonTag
			}

			callback(fieldName, fieldValue.Interface())
			walkObject(fieldValue.Interface(), fieldName, callback)
		}

	case reflect.Slice:
		// Walk slice elements with array index notation
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			elemName := prefix + "[" + strconv.Itoa(i) + "]"
			walkObject(elem.Interface(), elemName, callback)
		}

	case reflect.Map:
		// Walk map key-value pairs
		for _, key := range val.MapKeys() {
			mapValue := val.MapIndex(key)
			keyName := prefix + "." + key.String()
			callback(keyName, mapValue.Interface())
			walkObject(mapValue.Interface(), keyName, callback)
		}

	case reflect.String:
		// A leaf string value, reached either directly or after dereferencing
		// a *string struct field (e.g. Certificate.Cert). Struct fields call
		// the callback with the un-dereferenced pointer before recursing
		// here (which harmlessly fails a string type-assertion), so this is
		// what actually delivers the real value for *string fields — without
		// it, secrets stored as direct pointer fields (certificates, keys,
		// credentials) are silently never detected, unlike secrets nested
		// inside a map (e.g. plugin Config), which reach the callback via
		// the Map case above instead.
		callback(prefix, val.String())
	}
}
