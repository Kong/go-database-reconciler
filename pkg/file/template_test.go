package file

import (
	"os"
	"testing"
)

func Test_SetEnvVarPrefix(t *testing.T) {
	oldPrefix := envVarPrefix
	prefix := "NEW_PREFIX_"
	SetEnvVarPrefix(prefix)
	if envVarPrefix != prefix {
		envVarPrefix = oldPrefix
		t.Errorf("Expected prefix %q, but got %q", prefix, envVarPrefix)
	}
	envVarPrefix = oldPrefix
}

func Test_getPrefixedEnvVar(t *testing.T) {
	key := "DECK_MY_VARIABLE"
	expectedValue := "my_value"
	os.Setenv(key, expectedValue)

	value, err := getPrefixedEnvVar(key)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if value != expectedValue {
		t.Errorf("Expected value %q, but got %q", expectedValue, value)
	}

	// Clean up
	os.Unsetenv(key)
}

func Test_renderTemplate(t *testing.T) {
	content := "Hello, ${{ env \"DECK_MY_VARIABLE\" }}!"
	expectedOutput := "Hello, my_value!"
	mockEnvVars := false

	os.Setenv("DECK_MY_VARIABLE", "my_value")

	output, err := renderTemplate(content, mockEnvVars)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if output != expectedOutput {
		t.Errorf("Expected output %q, but got %q", expectedOutput, output)
	}
	os.Unsetenv("DECK_MY_VARIABLE")
}

func Test_renderTemplateCustomPrefix(t *testing.T) {
	oldPrefix := envVarPrefix
	SetEnvVarPrefix("PREFIX_")
	content := "Hello, ${{ env \"PREFIX_MY_VARIABLE\" }}!"
	expectedOutput := "Hello, my_value!"
	mockEnvVars := false

	os.Setenv("PREFIX_MY_VARIABLE", "my_value")

	output, err := renderTemplate(content, mockEnvVars)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if output != expectedOutput {
		t.Errorf("Expected output %q, but got %q", expectedOutput, output)
	}
	os.Unsetenv("PREFIX_MY_VARIABLE")
	SetEnvVarPrefix(oldPrefix)
}

func Test_renderTemplateIgnoresComments(t *testing.T) {
	content := `Hello, ${{ env "DECK_MY_VARIABLE" }}!
  # Also, ${{ env "DECK_NOT_SET_DOESNT_ERROR" }}!`

	expectedOutput := `Hello, my_value!
  # Also, ${{ env "DECK_NOT_SET_DOESNT_ERROR" }}!`
	mockEnvVars := false

	os.Setenv("DECK_MY_VARIABLE", "my_value")

	output, err := renderTemplate(content, mockEnvVars)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if output != expectedOutput {
		t.Errorf("Expected output %q, but got %q", expectedOutput, output)
	}
	os.Unsetenv("DECK_MY_VARIABLE")
}

func Test_renderTemplateErrorWhenNotSet(t *testing.T) {
	content := `
Hello, ${{ env "DECK_MY_VARIABLE" }}!
Also, ${{ env "DECK_NOT_SET_DOESNT_ERROR" }}!`

	mockEnvVars := false

	os.Setenv("DECK_MY_VARIABLE", "my_value")

	_, err := renderTemplate(content, mockEnvVars)
	if err == nil {
		t.Errorf("expected error but did not receive one")
	}

	os.Unsetenv("DECK_MY_VARIABLE")
}
