package diff

import (
	"encoding/json"
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
	var obj interface{}
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

func getDiff(a, b interface{}, defaults ...map[string]interface{}) (string, error) {
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
	var leftObject map[string]interface{}
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
func fillMissingDefaults(oldJSON, newJSON []byte, defaults map[string]interface{}) ([]byte, []byte) {
	var oldMap, newMap map[string]interface{}
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
	var dataMap map[string]interface{}
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

func parseDeckEnvVars() []EnvVar {
	const envVarPrefix = "DECK_"
	var parsedEnvVars []EnvVar

	for _, envVarStr := range os.Environ() {
		envPair := strings.SplitN(envVarStr, "=", 2)
		if strings.HasPrefix(envPair[0], envVarPrefix) {
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

// MaskEnvVarValue masks DECK_ env var values in diff output using position-aware
// regex and word-boundary matching to avoid corrupting unrelated content like UUIDs.
func MaskEnvVarValue(diffString string) string {
	envVars := parseDeckEnvVars()
	if len(envVars) == 0 {
		return diffString
	}

	// Build sorted list of values (longest first) for substring replacement
	var secrets []string
	seen := make(map[string]bool, len(envVars))
	for _, ev := range envVars {
		if ev.Value != "" && !seen[ev.Value] {
			secrets = append(secrets, ev.Value)
			seen[ev.Value] = true
		}
	}
	if len(secrets) == 0 {
		return diffString
	}
	sort.Slice(secrets, func(i, j int) bool {
		return len(secrets[i]) > len(secrets[j])
	})

	// Pre-compile word-boundary patterns once for all secrets
	secretPatterns := make([]*regexp.Regexp, len(secrets))
	for idx, secret := range secrets {
		secretPatterns[idx] = regexp.MustCompile(`\b` + regexp.QuoteMeta(secret) + `\b`)
	}

	maskFn := func(s string) string {
		for _, re := range secretPatterns {
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

		result := kvPattern.ReplaceAllStringFunc(line, func(match string) string {
			sub := kvPattern.FindStringSubmatch(match)
			if sub == nil {
				return match
			}
			switch {
			case sub[1] != "": // quoted string
				masked := maskFn(sub[1])
				if masked != sub[1] {
					return match[:len(match)-len(`"`+sub[1]+`"`)] + `"` + masked + `"`
				}
			case sub[2] != "": // number
				if seen[sub[2]] {
					prefix := match[:len(match)-len(sub[2])]
					if isJSON {
						return prefix + `"` + maskedValue + `"`
					}
					return prefix + maskedValue
				}
			case sub[3] != "": // YAML unquoted
				masked := maskFn(sub[3])
				if masked != sub[3] {
					return match[:len(match)-len(sub[3])] + masked
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
