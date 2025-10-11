package file

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
)

// default env var prefix, can be set using SetEnvVarPrefix
var envVarPrefix = "DECK_"

// SetEnvVarPrefix sets the prefix for environment variables used in the state file.
// The default prefix is "DECK_". This sets a library global(!!) value.
func SetEnvVarPrefix(prefix string) {
	envVarPrefix = prefix
}

func getPrefixedEnvVar(key string) (string, error) {
	if !strings.HasPrefix(key, envVarPrefix) {
		return "", fmt.Errorf("environment variables in the state file must "+
			"be prefixed with '%s', found: '%s'", envVarPrefix, key)
	}
	value, exists := os.LookupEnv(key)
	if !exists {
		return "", fmt.Errorf("environment variable '%s' present in state file but not set", key)
	}
	return value, nil
}

// getPrefixedEnvVarMocked is used when we mock the env variables while rendering a template.
// It will always return the name of the environment variable in this case.
func getPrefixedEnvVarMocked(key string) (string, error) {
	if !strings.HasPrefix(key, envVarPrefix) {
		return "", fmt.Errorf("environment variables in the state file must "+
			"be prefixed with '%s', found: '%s'", envVarPrefix, key)
	}
	return key, nil
}

func toBool(key string) (bool, error) {
	return strconv.ParseBool(key)
}

// toBoolMocked is used when we mock the env variables while rendering a template.
// It will always return false in this case.
func toBoolMocked(_ string) (bool, error) {
	return false, nil
}

func toInt(key string) (int, error) {
	return strconv.Atoi(key)
}

// toIntMocked is used when we mock the env variables while rendering a template.
// It will always return 42 in this case.
func toIntMocked(_ string) (int, error) {
	return 42, nil
}

func toFloat(key string) (float64, error) {
	return strconv.ParseFloat(key, 64)
}

// toFloatMocked is used when we mock the env variables while rendering a template.
// It will always return 42 in this case.
func toFloatMocked(_ string) (float64, error) {
	return 42, nil
}

func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return strings.Replace(v, "\n", "\n"+pad, -1)
}

func renderTemplate(content string, mockEnvVars bool) (string, error) {
	var templateFuncs template.FuncMap
	if mockEnvVars {
		templateFuncs = template.FuncMap{
			"env":     getPrefixedEnvVarMocked,
			"toBool":  toBoolMocked,
			"toInt":   toIntMocked,
			"toFloat": toFloatMocked,
			"indent":  indent,
		}
	} else {
		templateFuncs = template.FuncMap{
			"env":     getPrefixedEnvVar,
			"toBool":  toBool,
			"toInt":   toInt,
			"toFloat": toFloat,
			"indent":  indent,
		}
	}
	t := template.New("state").Funcs(templateFuncs).Delims("${{", "}}")

	// Parse content line by line
	var allContent bytes.Buffer
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			var buffer bytes.Buffer
			lineTemplate, err := t.Clone()
			if err != nil {
				return "", err
			}

			lineTemplate, err = lineTemplate.Parse(line)
			if err != nil {
				return "", err
			}

			// Clear the buffer before executing the template
			buffer.Reset()

			err = lineTemplate.Execute(&buffer, nil)
			if err != nil {
				return "", err
			}

			line = buffer.String()
		}

		allContent.WriteString(line + "\n")
	}

	result := allContent.String()
	if !strings.HasSuffix(content, "\n") {
		result = strings.TrimSuffix(result, "\n")
	}
	return result, nil
}
