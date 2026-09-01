package diff

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Kong/gojsondiff"
	"github.com/Kong/gojsondiff/formatter"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/kong/go-database-reconciler/pkg/state"
)

var differ = gojsondiff.New()

// PEM block type constants for certificate and key masking
const (
	pemBlockBegin = "-----BEGIN"
	pemBlockEnd   = "-----END"
)

// JWK (JSON Web Key) field constant
const jwkTypeField = "kty"

func getDocumentDiff(a, b *state.Document) (string, error) {
	aCopy := a.ShallowCopy()
	bCopy := a.ShallowCopy()
	aContent := *a.Content
	bContent := *b.Content
	aCopy.Content = nil
	bCopy.Content = nil
	objDiff, err := getDiff(aCopy, bCopy)
	if err != nil {
		return "", err
	}
	var contentDiff string
	if json.Valid([]byte(aContent)) && json.Valid([]byte(bContent)) {
		aContent, err = prettyPrintJSONString(aContent)
		if err != nil {
			return "", err
		}
		bContent, err = prettyPrintJSONString(bContent)
		if err != nil {
			return "", err
		}
	}
	edits := myers.ComputeEdits(span.URIFromPath("old"), aContent, bContent)
	contentDiff = fmt.Sprint(gotextdiff.ToUnified("old", "new", aContent, edits))

	return objDiff + contentDiff, nil
}

func prettyPrintJSONString(JSONString string) (string, error) {
	jBlob := []byte(JSONString)
	var obj any
	err := json.Unmarshal(jBlob, &obj)
	if err != nil {
		return "", err
	}
	bytes, err := json.MarshalIndent(obj, "", "\t")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func getDiff(a, b any, defaults ...map[string]any) (string, error) {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return "", err
	}

	// remove timestamps from JSON data without modifying the original data
	aJSON = removeTimestamps(aJSON)
	bJSON = removeTimestamps(bJSON)

	// When defaults are provided, fill missing fields in 'a' (old/current) that
	// are present in 'b' (new/target) with their schema default values.
	// This ensures the diff shows modifications (e.g. "-https") instead of
	// additions (e.g. "+protocols [http]") when a user changes a field away
	// from its default value and defaults have been stripped from both states.
	if len(defaults) > 0 && defaults[0] != nil {
		aJSON, bJSON = fillMissingDefaults(aJSON, bJSON, defaults[0])
	}

	d, err := differ.Compare(aJSON, bJSON)
	if err != nil {
		return "", err
	}
	var leftObject map[string]any
	err = json.Unmarshal(aJSON, &leftObject)
	if err != nil {
		return "", err
	}

	formatter := formatter.NewAsciiFormatter(leftObject,
		formatter.AsciiFormatterConfig{})
	diffString, err := formatter.Format(d)
	return diffString, err
}

// fillMissingDefaults injects schema default values into both oldJSON and newJSON
// for fields that are present in one but absent in the other. This produces correct
// modification diffs when both states have had their defaults stripped.
func fillMissingDefaults(oldJSON, newJSON []byte, defaults map[string]any) ([]byte, []byte) {
	var oldMap, newMap map[string]any
	if err := json.Unmarshal(oldJSON, &oldMap); err != nil {
		return oldJSON, newJSON
	}
	if err := json.Unmarshal(newJSON, &newMap); err != nil {
		return oldJSON, newJSON
	}

	oldChanged := false
	newChanged := false

	// Fill missing fields in oldMap when they exist in newMap
	for key, newVal := range newMap {
		if _, existsInOld := oldMap[key]; !existsInOld && newVal != nil {
			if defVal, hasDefault := defaults[key]; hasDefault {
				oldMap[key] = defVal
				oldChanged = true
			}
		}
	}

	// Fill missing fields in newMap when they exist in oldMap
	for key, oldVal := range oldMap {
		if _, existsInNew := newMap[key]; !existsInNew && oldVal != nil {
			if defVal, hasDefault := defaults[key]; hasDefault {
				newMap[key] = defVal
				newChanged = true
			}
		}
	}

	resultOld := oldJSON
	resultNew := newJSON

	if oldChanged {
		if result, err := json.Marshal(oldMap); err == nil {
			resultOld = result
		}
	}

	if newChanged {
		if result, err := json.Marshal(newMap); err == nil {
			resultNew = result
		}
	}

	return resultOld, resultNew
}

func removeTimestamps(jsonData []byte) []byte {
	var dataMap map[string]any
	if err := json.Unmarshal(jsonData, &dataMap); err != nil {
		return jsonData
	}
	delete(dataMap, "created_at")
	delete(dataMap, "updated_at")
	modifiedJSON, err := json.Marshal(dataMap)
	if err != nil {
		return jsonData
	}
	return modifiedJSON
}

type EnvVar struct {
	Key   string
	Value string
}

// EnvVarCache holds precomputed environment variable data to avoid redundant
// parsing, sorting, and regex compilation during multiple masking operations.
// It is built once per sync run (env vars are immutable during a run) and then
// only read, so it is safe to share across the parallel Solve goroutines.
type EnvVarCache struct {
	EnvVars []EnvVar
	// Secrets holds the unique, non-empty env var values sorted longest-first
	// so that longer values are masked before their potential substrings.
	Secrets []string
	// SecretsSet is a membership set of Secrets, used for exact-match checks
	// (e.g. numeric values that must match an env var value exactly).
	SecretsSet map[string]bool
	// SecretPatterns are word-boundary regexes for each secret, index-aligned
	// with Secrets. Compiled once here rather than on every masking call.
	SecretPatterns []*regexp.Regexp
	// PEMPatterns are the compiled BEGIN/END block regexes for each sensitive
	// PEM type present in the env vars. Compiled once here.
	PEMPatterns []*regexp.Regexp
	HasJWK      bool
}

// NewEnvVarCache precomputes all environment variable data needed for masking.
// This avoids redundant parsing, sorting, and regex compilation on every
// MaskEnvVarValue call. Environment variables don't change during a sync run,
// so this single precomputation significantly improves performance.
func NewEnvVarCache() *EnvVarCache {
	envVars := parseDeckEnvVars()
	cache := &EnvVarCache{
		EnvVars:        envVars,
		SecretsSet:     make(map[string]bool),
		SecretPatterns: make([]*regexp.Regexp, 0),
		PEMPatterns:    make([]*regexp.Regexp, 0),
	}

	if len(envVars) == 0 {
		return cache
	}

	// Extract unique secrets and detect PEM types / JWK in a single pass.
	pemTypes := make(map[string]bool)
	for _, ev := range envVars {
		if ev.Value != "" && !cache.SecretsSet[ev.Value] {
			cache.Secrets = append(cache.Secrets, ev.Value)
			cache.SecretsSet[ev.Value] = true
		}

		if t := pemTypeFromValue(ev.Value); t != "" {
			pemTypes[t] = true
		}

		if isJWK(ev.Value) {
			cache.HasJWK = true
		}
	}

	// Sort secrets by length (longest first) for proper masking order.
	sort.Slice(cache.Secrets, func(i, j int) bool {
		return len(cache.Secrets[i]) > len(cache.Secrets[j])
	})

	// Precompile word-boundary patterns for each secret.
	for _, secret := range cache.Secrets {
		cache.SecretPatterns = append(cache.SecretPatterns,
			regexp.MustCompile(`\b`+regexp.QuoteMeta(secret)+`\b`))
	}

	// Build one regex per PEM type (CERTIFICATE, PRIVATE KEY, etc.) to find and
	// mask those blocks in the diff. `[ \t]*` grabs any spaces/tabs right
	// before "-----BEGIN" so an indented cert gets replaced cleanly.
	for pemType := range pemTypes {
		cache.PEMPatterns = append(cache.PEMPatterns, regexp.MustCompile(
			`[ \t]*`+regexp.QuoteMeta(pemBlockBegin)+`\s+`+regexp.QuoteMeta(pemType)+
				`(?s)\s*-----.*?`+regexp.QuoteMeta(pemBlockEnd)+`\s+`+regexp.QuoteMeta(pemType)+
				`\s*-----`,
		))
	}

	return cache
}

// nonSecretDeckVars is a whitelist of DECK_ configuration variables that should NOT be masked.
// Only variables in this list are treated as non-secrets; all others are masked.
var nonSecretDeckVars = map[string]bool{
	// Flags
	"DECK_ANALYTICS":            true,
	"DECK_SKIP_DEFAULTS_FILL":   true,
	"DECK_SKIP_CA_VERIFICATION": true,

	// Public URLs
	"DECK_KONNECT_ADDR": true,
	"DECK_KONG_ADDR":    true,

	// Configuration
	"DECK_FORMAT":                     true,
	"DECK_KONNECT_RUNTIME_GROUP_NAME": true,
	"DECK_KONNECT_CONTROL_PLANE_NAME": true,

	// TLS file paths (not the contents)
	"DECK_CA_CERT_FILE":         true,
	"DECK_TLS_CLIENT_CERT_FILE": true,
	"DECK_TLS_CLIENT_KEY_FILE":  true,
	"DECK_TLS_SERVER_NAME":      true,

	// Timing
	"DECK_TIMEOUT": true,
}

func parseDeckEnvVars() []EnvVar {
	const envVarPrefix = "DECK_"
	var parsedEnvVars []EnvVar

	for _, envVarStr := range os.Environ() {
		envPair := strings.SplitN(envVarStr, "=", 2)
		if strings.HasPrefix(envPair[0], envVarPrefix) {
			// Skip non-secret configuration variables
			if nonSecretDeckVars[envPair[0]] {
				continue
			}

			envVar := EnvVar{}
			envVar.Key = envPair[0]
			envVar.Value = envPair[1]
			parsedEnvVars = append(parsedEnvVars, envVar)
		}
	}

	sort.Slice(parsedEnvVars, func(i, j int) bool {
		return len(parsedEnvVars[i].Value) > len(parsedEnvVars[j].Value)
	})
	return parsedEnvVars
}

const maskedValue = "[masked]"

func isJWK(v string) bool {
	trimmed := strings.TrimSpace(v)
	// Strip a single layer of surrounding quotes (e.g. env vars set as
	// DECK_JWK_KEY='{"kty":"RSA",...}') so the plain-JSON check below can
	// see the leading '{'.
	if len(trimmed) >= 2 {
		if (trimmed[0] == '\'' && trimmed[len(trimmed)-1] == '\'') ||
			(trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"') {
			trimmed = trimmed[1 : len(trimmed)-1]
		}
	}

	if !strings.HasPrefix(trimmed, "{") {
		return false
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return false
	}
	_, hasKty := obj[jwkTypeField]
	return hasKty
}

// normalizeForPEMDetection cleans up a cert/key value so Go's PEM parser can
// recognize it. The parser is strict: it only works if each line starts at
// the very left edge, with no quotes or leading spaces. So this removes
// wrapping quotes and any indentation .
func normalizeForPEMDetection(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = v[1 : len(v)-1]
	}
	v = strings.ReplaceAll(v, `\n`, "\n")

	lines := strings.Split(v, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, " \t")
	}
	return strings.Join(lines, "\n")
}

// pemTypeFromValue extracts the PEM block type (e.g. "CERTIFICATE", "PRIVATE KEY")
func pemTypeFromValue(v string) string {
	normalized := normalizeForPEMDetection(v)
	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		return ""
	}
	return block.Type
}

// jwkPattern matches JWK on both single and multiple lines.
// Allows whitespace (including newlines) between elements, and handles escaped
// quotes and nested objects properly.
var jwkPattern = regexp.MustCompile(
	`\{(?:[^"{}]|\s|"(?:\\.|[^"\\])*"|(?:\{[^{}]*\}))*` +
		`"` + jwkTypeField + `"\s*:\s*"[^"]*"` +
		`(?:[^"{}]|\s|"(?:\\.|[^"\\])*"|(?:\{[^{}]*\}))*\}`,
)

// maskPEMBlocksWithCache finds PEM blocks in the diff output and replaces them
// with [masked], using the precompiled patterns from the cache. Text before the
// block is preserved automatically by ReplaceAll; the trailing quotes/commas
// captured in group 1 are re-appended after the mask.
func maskPEMBlocksWithCache(diffString string, cache *EnvVarCache) string {
	for _, pattern := range cache.PEMPatterns {
		diffString = pattern.ReplaceAllString(diffString, maskedValue+`$1`)
	}
	return diffString
}

// maskJWKBlocks masks JWK objects in the diff, handling both single-line and
// multi-line formats. This also handles JSON-encoded JWK strings (where JWK
// is stored as an escaped JSON string).
func maskJWKBlocks(diffString string) string {
	return jwkPattern.ReplaceAllString(diffString, maskedValue)
}

// Compiled patterns for identifying values in diff output.
var (
	// jsonKeyPattern detects JSON-formatted output by matching a quoted key
	// followed by a colon (e.g., "name":). This is how gojsondiff's ASCII
	// formatter renders keys — YAML/plain text uses unquoted keys.
	jsonKeyPattern = regexp.MustCompile(`"[^"]+"\s*:`)

	// kvPattern matches values after a colon separator:
	//   Group 1: quoted string
	//   Group 2: numeric
	//   Group 3: YAML unquoted
	kvPattern = regexp.MustCompile(
		`:\s*"((?:[^"\\]|\\.)*)"|` +
			`:\s+(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)\b|` +
			`:\s+([^\s"\d{\[\]}\-][^\n,}\]]*)`,
	)

	// arrayElemPattern matches standalone quoted strings in JSON arrays
	// (lines with only whitespace/diff markers before the value).
	arrayElemPattern = regexp.MustCompile(`^([+\- ]*\s+)"((?:[^"\\]|\\.)*)"`)
)

// MaskEnvVarValue masks DECK_ env var values in diff output using a precomputed cache.
// Phase 1: masks multiline PEM blocks (certs/keys) using BEGIN/END markers.
// Phase 2: masks JSON Web Key (JWK) values using regex pattern matching.
// Phase 3: masks single-line values using position-aware regex matching.
func MaskEnvVarValue(diffString string) string {
	cache := NewEnvVarCache()
	return maskEnvVarValueWithCache(diffString, cache)
}

// maskEnvVarValueWithCache masks DECK_ env var values using a precomputed cache.
// This is the core masking function used internally and during Solve to avoid
// recomputing environment variable data.
// OPTIMIZED: Uses early termination and lazy pattern matching to reduce regex overhead.
func maskEnvVarValueWithCache(diffString string, cache *EnvVarCache) string {
	if len(cache.EnvVars) == 0 {
		return diffString
	}
	// Early exit: check if diff contains any markers for secrets/PEM/JWK
	hasSecrets := len(cache.Secrets) > 0
	hasPEM := len(cache.PEMPatterns) > 0
	hasPEMMarker := hasPEM && (strings.Contains(diffString, pemBlockBegin) || strings.Contains(diffString, pemBlockEnd))
	// Check for both plain JSON JWK ("kty") and JSON-encoded JWK string (\"kty\")
	hasJWKMarker := cache.HasJWK && strings.Contains(diffString, `"`+jwkTypeField+`"`)

	// Only apply PEM masking if both patterns are present AND the diff actually contains PEM markers
	if hasPEMMarker {
		diffString = maskPEMBlocksWithCache(diffString, cache)
	}

	if hasJWKMarker {
		diffString = maskJWKBlocks(diffString)
	}

	// Early exit if there is nothing left to mask (no JWK in this diff and no secrets).
	if !hasSecrets {
		return diffString
	}

	maskFn := func(s string) string {
		for _, re := range cache.SecretPatterns {
			s = re.ReplaceAllString(s, maskedValue)
		}
		return s
	}

	// Detect format once: the diff engine (gojsondiff) always produces JSON-like
	// output with quoted keys. Unified text diffs (from getDocumentDiff) never have
	// quoted keys. We check the entire string rather than per-line to avoid false
	// positives from YAML values that happen to contain `":`.
	isJSON := jsonKeyPattern.MatchString(diffString)

	lines := strings.Split(diffString, "\n")
	for i, line := range lines {

		// Skip lines that don't contain any secret - fast path for the majority.
		// This optimization avoids expensive regex matching on lines unlikely to contain secrets.
		if !containsAnySecret(line, cache.Secrets) {
			lines[i] = line
			continue
		}

		result := kvPattern.ReplaceAllStringFunc(line, func(match string) string {
			sub := kvPattern.FindStringSubmatch(match)
			if sub == nil {
				return match
			}
			switch {
			case sub[1] != "":
				// Can't hardcode the prefix since `:\s*` matches both compact and formatted JSON,
				// so the original spacing must be preserved.
				masked := maskFn(sub[1])
				if masked != sub[1] {
					return match[:len(match)-len(`"`+sub[1]+`"`)] + `"` + masked + `"`
				}
			case sub[2] != "": // number
				if cache.SecretsSet[sub[2]] {
					if isJSON {
						return `: "` + maskedValue + `"`
					}
					return ": " + maskedValue
				}
			case sub[3] != "": // YAML unquoted
				masked := maskFn(sub[3])
				if masked != sub[3] {
					return ": " + masked
				}
			}
			return match
		})

		// Fall back to array element masking if no kv match was made.
		if result == line {
			result = arrayElemPattern.ReplaceAllStringFunc(line, func(match string) string {
				sub := arrayElemPattern.FindStringSubmatch(match)
				if sub == nil {
					return match
				}
				masked := maskFn(sub[2])
				if masked == sub[2] {
					return match
				}
				quoted := `"` + sub[2] + `"`
				suffix := match[len(sub[1])+len(quoted):]
				return sub[1] + `"` + masked + `"` + suffix
			})
		}

		lines[i] = result
	}
	return strings.Join(lines, "\n")
}

func containsAnySecret(line string, secrets []string) bool {
	for _, secret := range secrets {
		if strings.Contains(line, secret) {
			return true
		}
	}
	return false
}

// cleanMaskedValueMarkers removes the invisible change-detection marker (U+200D)
// from the diff output. The marker is needed for gojsondiff to detect that a
// masked value changed (byte-different), but it should not appear in the final
// rendered output to avoid rendering as visible space in terminals.
func cleanMaskedValueMarkers(diffString string) string {
	return strings.ReplaceAll(diffString, string(rune(0x200D)), "")
}
