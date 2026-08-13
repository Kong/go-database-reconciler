package diff

import (
	"encoding/json"
	"reflect"
	"strings"
)

// maskedValueChanged is visually identical to maskedValue in a terminal (uses
// a trailing zero-width joiner character U+200D), but is a different string.
// Used on the "new" side of a masked field when the real, pre-mask value
// differed from the old side — so a value-equality-based diff (gojsondiff)
// still renders the field as changed instead of silently treating two
// masked-to-the-same-literal values as unchanged.
// Note: U+200D (zero-width joiner) is less likely to render as visible space
// than U+200B (zero-width space) in various terminals and tools.
var maskedValueChanged = maskedValue + string(rune(0x200D))

// maskEntityPairByFieldNames masks the old and new sides of an entity
// TOGETHER, field name by field name, so it can tell whether the real
// (pre-mask) value actually changed. Unmodified secret fields are masked
// identically on both sides; changed ones get a visually-identical but
// byte-different placeholder on the new side, preserving the "this changed"
// signal without revealing what changed.
// Neither original object is mutated.
func maskEntityPairByFieldNames(oldEntity, newEntity any, secretFields map[string]bool) (any, any) {
	if len(secretFields) == 0 {
		return oldEntity, newEntity
	}
	oldClone := cloneForMasking(oldEntity)
	newClone := cloneForMasking(newEntity)

	oldVal, newVal := reflect.ValueOf(oldClone), reflect.ValueOf(newClone)

	maskFieldPairByName(oldVal, newVal, secretFields)
	return oldClone, newClone
}

func maskFieldPairByName(oldV, newV reflect.Value, secretFields map[string]bool) {
	if !oldV.IsValid() || !newV.IsValid() || oldV.Kind() != newV.Kind() {
		return
	}

	switch oldV.Kind() { //nolint:exhaustive
	case reflect.Pointer:
		if oldV.IsNil() || newV.IsNil() {
			return
		}
		maskFieldPairByName(oldV.Elem(), newV.Elem(), secretFields)
	case reflect.Struct:
		t := oldV.Type()
		// Cache field name lookups to avoid repeated string parsing.
		fieldNames := make([]string, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			fieldNames[i] = jsonFieldName(t.Field(i))
		}

		for i := 0; i < oldV.NumField(); i++ {
			ofv, nfv := oldV.Field(i), newV.Field(i)
			if !ofv.CanInterface() {
				continue
			}

			if ofv.Kind() == reflect.Map && ofv.Type().Key().Kind() == reflect.String &&
				ofv.Type().Elem().Kind() == reflect.Interface {
				maskMapPair(ofv, nfv, secretFields)
				continue
			}

			name := fieldNames[i]
			if secretFields[name] {
				setMaskedPair(ofv, nfv)
				continue
			}
			maskFieldPairByName(ofv, nfv, secretFields)
		}
	case reflect.Slice, reflect.Array:
		n := oldV.Len()
		if newV.Len() < n {
			n = newV.Len()
		}
		for i := 0; i < n; i++ {
			maskFieldPairByName(oldV.Index(i), newV.Index(i), secretFields)
		}
	case reflect.Interface:
		if oldV.IsNil() || newV.IsNil() {
			return
		}
		maskFieldPairByName(oldV.Elem(), newV.Elem(), secretFields)
	}
}

// cloneForMasking creates a deep copy of an entity via JSON marshal/unmarshal.
// Used to avoid mutating the original object during masking. Maps are
// reference types in Go - a shallow copy would share map/slice pointers
// with the original, causing mutations during masking to affect the original.
// Returns nil if entity can't be cloned.
func cloneForMasking(entity any) any {
	t := reflect.TypeOf(entity)
	if t == nil || t.Kind() != reflect.Pointer {
		return nil
	}

	b, err := json.Marshal(entity)
	if err != nil {
		return nil
	}

	clone := reflect.New(t.Elem()).Interface()
	if err := json.Unmarshal(b, clone); err != nil {
		return nil
	}
	return clone
}

// setMaskedPair masks a matched field pair — a *string/string, or a
// slice/array of them (e.g. Route.Methods []*string) — marking the new
// side as changed if the real values differed anywhere within it.
// Uses fast paths for common string types before falling back to full
// deep comparison for complex types.
func setMaskedPair(oldF, newF reflect.Value) {
	// Fast path: direct string comparison (most common case)
	if oldF.Kind() == reflect.String && newF.Kind() == reflect.String {
		changed := oldF.String() != newF.String()
		setMaskedValue(oldF, false)
		setMaskedValue(newF, changed)
		return
	}

	// Fast path: pointer to string
	if oldF.Kind() == reflect.Pointer && newF.Kind() == reflect.Pointer &&
		oldF.Type().Elem().Kind() == reflect.String {
		var oVal, nVal string
		if !oldF.IsNil() {
			oVal = oldF.Elem().String()
		}
		if !newF.IsNil() {
			nVal = newF.Elem().String()
		}
		changed := oVal != nVal
		setMaskedValue(oldF, false)
		setMaskedValue(newF, changed)
		return
	}

	// Slow path: full deep comparison for complex types (slices, arrays, etc.)
	changed := !deepValuesEqual(oldF, newF)
	setMaskedValue(oldF, false)
	setMaskedValue(newF, changed)
}

// deepValuesEqual compares two values for equality after dereferencing
// pointers, recursing into slices/arrays element-wise. Used only to decide
// whether a matched secret field changed — never to reveal real values.
func deepValuesEqual(a, b reflect.Value) bool {
	for a.Kind() == reflect.Pointer && !a.IsNil() {
		a = a.Elem()
	}
	for b.Kind() == reflect.Pointer && !b.IsNil() {
		b = b.Elem()
	}
	if a.Kind() == reflect.Pointer || b.Kind() == reflect.Pointer {
		return a.Kind() == b.Kind() // one or both nil pointers
	}
	if a.Kind() != b.Kind() {
		return false
	}
	switch a.Kind() { //nolint:exhaustive
	case reflect.String:
		return a.String() == b.String()
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !deepValuesEqual(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	default:
		if !a.CanInterface() || !b.CanInterface() {
			return true // conservative: treat as unchanged rather than panic
		}
		return reflect.DeepEqual(a.Interface(), b.Interface())
	}
}

// setMaskedValue masks a value in place: a *string/string directly, or
// every element of a slice/array of them (e.g. Route.Methods []*string).
func setMaskedValue(v reflect.Value, changed bool) {
	switch v.Kind() { //nolint:exhaustive
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		setMaskedValue(v.Elem(), changed)
	case reflect.String:
		if !v.CanSet() {
			return
		}
		if changed {
			v.SetString(maskedValueChanged)
		} else {
			v.SetString(maskedValue)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			setMaskedValue(v.Index(i), changed)
		}
	}
}

// maskMapPair masks a matched pair of freeform map[string]any fields
// (e.g. plugin Config) key by key, marking changed secret values distinctly.
// Creates new masked maps to avoid mutating originals (shallow copy shares refs).
func maskMapPair(oldF, newF reflect.Value, secretFields map[string]bool) {
	oldMap, _ := toStringAnyMap(oldF.Interface())
	newMap, _ := toStringAnyMap(newF.Interface())

	maskedOld := make(map[string]any, len(oldMap))
	maskedNew := make(map[string]any, len(newMap))

	keys := make(map[string]bool, len(oldMap)+len(newMap))
	for k := range oldMap {
		keys[k] = true
	}
	for k := range newMap {
		keys[k] = true
	}

	for k := range keys {
		ov, oOK := oldMap[k]
		nv, nOK := newMap[k]

		if secretFields[k] {
			if oOK {
				maskedOld[k] = maskedValue
			}
			if nOK {
				if !oOK || !reflect.DeepEqual(ov, nv) {
					maskedNew[k] = maskedValueChanged
				} else {
					maskedNew[k] = maskedValue
				}
			}
			continue
		}

		if oOK {
			maskedOld[k] = maskInterfaceValue(ov, secretFields)
		}
		if nOK {
			maskedNew[k] = maskInterfaceValue(nv, secretFields)
		}
	}

	if oldF.CanSet() {
		oldF.Set(reflect.ValueOf(maskedOld))
	}
	if newF.CanSet() {
		newF.Set(reflect.ValueOf(maskedNew))
	}
}

func toStringAnyMap(m any) (map[string]any, bool) {
	if orig, ok := m.(map[string]any); ok {
		return orig, true
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, false
	}
	var orig map[string]any
	if err := json.Unmarshal(b, &orig); err != nil {
		return nil, false
	}
	return orig, true
}

// jsonFieldName returns the JSON key a struct field serializes as.
func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return field.Name
	}
	return strings.Split(tag, ",")[0]
}

// maskMapValue masks a map[string]any value key-by-key: a key that matches
// a secret field name is masked outright (any value type); otherwise its
// value is recursed into, since plugin config commonly nests further maps
// (e.g. redis.password) and slices.
func maskMapValue(m any, secretFields map[string]bool) (map[string]any, bool) {
	orig, ok := toStringAnyMap(m)
	if !ok {
		return nil, false
	}

	result := make(map[string]any, len(orig))
	for k, val := range orig {
		if secretFields[k] {
			result[k] = maskedValue
			continue
		}
		result[k] = maskInterfaceValue(val, secretFields)
	}
	return result, true
}

func maskInterfaceValue(val any, secretFields map[string]bool) any {
	switch v := val.(type) {
	case map[string]any:
		masked, _ := maskMapValue(v, secretFields)
		return masked
	case []any:
		out := make([]any, len(v))
		for i, elem := range v {
			out[i] = maskInterfaceValue(elem, secretFields)
		}
		return out
	default:
		return val
	}
}
