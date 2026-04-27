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
	mode := EnvVarsExpand

	os.Setenv("DECK_MY_VARIABLE", "my_value")

	output, err := renderTemplate(content, mode)
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
	mode := EnvVarsExpand

	os.Setenv("PREFIX_MY_VARIABLE", "my_value")

	output, err := renderTemplate(content, mode)
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

	// Comment lines preserve the original template expressions intact.
	expectedOutput := `Hello, my_value!
  # Also, ${{ env "DECK_NOT_SET_DOESNT_ERROR" }}!`
	mode := EnvVarsExpand

	os.Setenv("DECK_MY_VARIABLE", "my_value")

	output, err := renderTemplate(content, mode)
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
Also, ${{ env "DECK_NOT_SET_ERRORS" }}!`

	mode := EnvVarsExpand

	os.Setenv("DECK_MY_VARIABLE", "my_value")

	_, err := renderTemplate(content, mode)
	if err == nil {
		t.Errorf("expected error but did not receive one")
	}

	os.Unsetenv("DECK_MY_VARIABLE")
}

func Test_renderTemplateMock(t *testing.T) {
	content := `Hello, ${{ env "DECK_MY_VARIABLE" }}!`
	// EnvVarsMock returns the variable name, not the value, and does not
	// require the env var to be set.
	expectedOutput := `Hello, DECK_MY_VARIABLE!`

	output, err := renderTemplate(content, EnvVarsMock)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if output != expectedOutput {
		t.Errorf("Expected output %q, but got %q", expectedOutput, output)
	}
}

func Test_renderTemplateMockDoesNotRequireEnvVars(t *testing.T) {
	// Even with an unset env var, EnvVarsMock should not error.
	content := `Hello, ${{ env "DECK_NOT_SET" }}!`

	_, err := renderTemplate(content, EnvVarsMock)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func Test_renderTemplateSkip(t *testing.T) {
	// EnvVarsSkip returns the content unchanged, including template expressions.
	content := `Hello, ${{ env "DECK_MY_VARIABLE" }}!`

	output, err := renderTemplate(content, EnvVarsSkip)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if output != content {
		t.Errorf("Expected output %q, but got %q", content, output)
	}
}

func Test_renderTemplateSkipDoesNotRequireEnvVars(t *testing.T) {
	// EnvVarsSkip must not error even when referenced env vars are not set.
	content := `Hello, ${{ env "DECK_NOT_SET" }}!`

	_, err := renderTemplate(content, EnvVarsSkip)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// Test_renderTemplatePreservesHTMLWithHashNotComment verifies that HTML content
// with lines starting with # is not treated as comments and is preserved correctly,
// in case of line-wrapping.
func Test_renderTemplatePreservesHTMLWithHashNotComment(t *testing.T) {
	content := `plugins:
- name: request-termination
  config:
    body: '<html>
      <head>
        <style>
          body { background-color:
            #ffffff; color:
            #333333; }
          .header { color:
            #ff5733; }
        </style>
      </head>
      <body style="margin: 0; padding: 10px; background:
        #f0f0f0;">
        <h1>Hello World</h1>
      </body>
    </html>'`

	output, err := renderTemplate(content, EnvVarsExpand)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if output != content {
		t.Errorf("Expected content to be unchanged, but got %q", output)
	}
}
