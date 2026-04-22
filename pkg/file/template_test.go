package file

import (
	"fmt"
	"os"
	"strings"
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

	expectedOutput := `Hello, my_value!`
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

// Benchmark_renderTemplate benchmarks the original implementation
// that removes comment lines entirely.
func Benchmark_renderTemplate(b *testing.B) {
	var content strings.Builder
	blockCount := 10000
	for i := 0; i < blockCount; i++ {
		os.Setenv(fmt.Sprintf("DECK_MY_VARIABLE%d", i), "my_value")

		var block strings.Builder
		for j := 0; j < 9; j++ { // 9 lines of content
			line := strings.Repeat("x", 50) // 50 chars per line
			block.WriteString(line + "\n")
		}
		block.WriteString(fmt.Sprintf("Value: ${{ env \"DECK_MY_VARIABLE%d\" }}\n", i))
		content.WriteString(block.String())
	}
	defer func() {
		for i := 0; i < blockCount; i++ {
			os.Unsetenv(fmt.Sprintf("DECK_MY_VARIABLE%d", i))
		}
	}()

	contentStr := content.String()

	b.ResetTimer()

	for b.Loop() {
		_, _ = renderTemplate(contentStr, EnvVarsExpand)
	}
}

// Benchmark_renderTemplateWithPreservingComment benchmarks the new implementation
// that preserves comment lines but strips template expressions from them.
func Benchmark_renderTemplateWithPreservingComment(b *testing.B) {
	var content strings.Builder
	blockCount := 10000
	for i := 0; i < blockCount; i++ {
		os.Setenv(fmt.Sprintf("DECK_MY_VARIABLE%d", i), "my_value")

		var block strings.Builder
		for j := 0; j < 9; j++ { // 9 lines of content
			line := strings.Repeat("x", 50) // 50 chars per line
			block.WriteString(line + "\n")
		}
		block.WriteString(fmt.Sprintf("Value: ${{ env \"DECK_MY_VARIABLE%d\" }}\n", i))
		content.WriteString(block.String())
	}
	defer func() {
		for i := 0; i < blockCount; i++ {
			os.Unsetenv(fmt.Sprintf("DECK_MY_VARIABLE%d", i))
		}
	}()

	contentStr := content.String()

	b.ResetTimer()

	for b.Loop() {
		_, _ = renderTemplateWithPreservingComment(contentStr, EnvVarsExpand)
	}
}

// Benchmark_renderTemplateWithComments benchmarks the original implementation
// with content that includes comment lines containing template expressions.
func Benchmark_renderTemplateWithComments(b *testing.B) {
	var content strings.Builder
	blockCount := 10000
	for i := 0; i < blockCount; i++ {
		os.Setenv(fmt.Sprintf("DECK_MY_VARIABLE%d", i), "my_value")

		var block strings.Builder
		for j := 0; j < 8; j++ { // 8 lines of content
			line := strings.Repeat("x", 50) // 50 chars per line
			block.WriteString(line + "\n")
		}
		// Add a comment line with a template expression
		block.WriteString(fmt.Sprintf("# Comment with ${{ env \"DECK_UNSET_VAR%d\" }}\n", i))
		block.WriteString(fmt.Sprintf("Value: ${{ env \"DECK_MY_VARIABLE%d\" }}\n", i))
		content.WriteString(block.String())
	}
	defer func() {
		for i := 0; i < blockCount; i++ {
			os.Unsetenv(fmt.Sprintf("DECK_MY_VARIABLE%d", i))
		}
	}()

	contentStr := content.String()

	b.ResetTimer()

	for b.Loop() {
		_, _ = renderTemplate(contentStr, EnvVarsExpand)
	}
}

// Benchmark_renderTemplateWithPreservingCommentWithComments benchmarks the new
// implementation with content that includes comment lines containing template expressions.
func Benchmark_renderTemplateWithPreservingCommentWithComments(b *testing.B) {
	var content strings.Builder
	blockCount := 10000
	for i := 0; i < blockCount; i++ {
		os.Setenv(fmt.Sprintf("DECK_MY_VARIABLE%d", i), "my_value")

		var block strings.Builder
		for j := 0; j < 8; j++ { // 8 lines of content
			line := strings.Repeat("x", 50) // 50 chars per line
			block.WriteString(line + "\n")
		}
		// Add a comment line with a template expression
		block.WriteString(fmt.Sprintf("# Comment with ${{ env \"DECK_UNSET_VAR%d\" }}\n", i))
		block.WriteString(fmt.Sprintf("Value: ${{ env \"DECK_MY_VARIABLE%d\" }}\n", i))
		content.WriteString(block.String())
	}
	defer func() {
		for i := 0; i < blockCount; i++ {
			os.Unsetenv(fmt.Sprintf("DECK_MY_VARIABLE%d", i))
		}
	}()

	contentStr := content.String()

	b.ResetTimer()

	for b.Loop() {
		_, _ = renderTemplateWithPreservingComment(contentStr, EnvVarsExpand)
	}
}

