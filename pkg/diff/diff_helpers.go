package diff

import (
	"encoding/json"
	"fmt"
	"os"
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

func MaskEnvVarValue(diffString string) string {
	for _, envVar := range parseDeckEnvVars() {
		diffString = strings.Replace(diffString, envVar.Value, "[masked]", -1)
	}
	return diffString
}
