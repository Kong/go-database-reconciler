//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderDeterministicSort_ConsumerCredentials tests that consumer credentials
// are sorted deterministically in the rendered output.
func TestRenderDeterministicSort_ConsumerCredentials(t *testing.T) {
	testDir := filepath.Join("testdata", "render", "deterministic-sort", "001-consumer-credentials")
	inputPath := filepath.Join(testDir, "input.yaml")
	expectedPath := filepath.Join(testDir, "expected.yaml")

	compareRenderOutput(t, inputPath, expectedPath)
}

// TestRenderDeterministicSort_ConsumerGroupPlugins tests that consumer group plugins
// are sorted deterministically in the rendered output.
func TestRenderDeterministicSort_ConsumerGroupPlugins(t *testing.T) {
	testDir := filepath.Join("testdata", "render", "deterministic-sort", "002-consumer-group-plugins")
	inputPath := filepath.Join(testDir, "input.yaml")
	expectedPath := filepath.Join(testDir, "expected.yaml")

	compareRenderOutput(t, inputPath, expectedPath)
}

// TestRenderDeterministicSort_ConsumerGroupsMembership tests that consumer group
// memberships are sorted deterministically in the rendered output.
func TestRenderDeterministicSort_ConsumerGroupsMembership(t *testing.T) {
	testDir := filepath.Join("testdata", "render", "deterministic-sort", "003-consumer-groups-membership")
	inputPath := filepath.Join(testDir, "input.yaml")
	expectedPath := filepath.Join(testDir, "expected.yaml")

	compareRenderOutput(t, inputPath, expectedPath)
}

// TestRenderDeterministicSort_ComplexState tests that a complex state with multiple
// entity types is sorted deterministically in the rendered output.
func TestRenderDeterministicSort_ComplexState(t *testing.T) {
	testDir := filepath.Join("testdata", "render", "deterministic-sort", "004-complex-state")
	inputPath := filepath.Join(testDir, "input.yaml")
	expectedPath := filepath.Join(testDir, "expected.yaml")

	compareRenderOutput(t, inputPath, expectedPath)
}

// TestRenderDeterministicSort_NestedEntities tests that nested entities
// (routes under services, plugins under routes/services/consumers, targets under upstreams,
// credentials under consumers, SNIs under certificates, etc.) are sorted deterministically.
func TestRenderDeterministicSort_NestedEntities(t *testing.T) {
	testDir := filepath.Join("testdata", "render", "deterministic-sort", "005-nested-entities")
	inputPath := filepath.Join(testDir, "input.yaml")
	expectedPath := filepath.Join(testDir, "expected.yaml")

	compareRenderOutput(t, inputPath, expectedPath)
}

// TestRenderDeterministicSort_ComplexNestedStructure tests a complex state with
// multiple services, routes with plugins, consumers with all credential types,
// upstreams with targets, and certificates with SNIs - all sorted deterministically.
func TestRenderDeterministicSort_ComplexNestedStructure(t *testing.T) {
	testDir := filepath.Join("testdata", "render", "deterministic-sort", "006-complex-nested-structure")
	inputPath := filepath.Join(testDir, "input.yaml")
	expectedPath := filepath.Join(testDir, "expected.yaml")

	compareRenderOutput(t, inputPath, expectedPath)
}

// TestRenderDeterministicSort_MultipleRuns verifies that multiple render calls
// with the same input produce identical output.
func TestRenderDeterministicSort_MultipleRuns(t *testing.T) {
	inputPath := filepath.Join("testdata", "render", "deterministic-sort", "004-complex-state", "input.yaml")

	var outputs []string
	for i := 0; i < 10; i++ {
		output := renderYAMLFile(t, inputPath)
		outputs = append(outputs, output)
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		assert.Equal(t, outputs[0], outputs[i],
			"Render run %d produced different output than run 0", i)
	}
}

// compareRenderOutput renders the input file and compares against expected output.
func compareRenderOutput(t *testing.T, inputPath, expectedPath string) {
	t.Helper()

	// Render the input
	actual := renderYAMLFile(t, inputPath)

	// Read expected output
	expected, err := os.ReadFile(expectedPath)
	require.NoError(t, err, "Failed to read expected file: %s", expectedPath)

	// Compare
	assert.Equal(t, string(expected), actual,
		"Rendered output does not match expected output.\nInput: %s\nExpected: %s",
		inputPath, expectedPath)
}

// TestGenerateExpectedFiles is a helper test to generate expected.yaml files.
// Run with: go test -tags integration -run TestGenerateExpectedFiles -v
// After running, review the generated files and commit them.
func TestGenerateExpectedFiles(t *testing.T) {
	t.Skip("Skipping: only run manually to regenerate expected files")

	testCases := []string{
		"001-consumer-credentials",
		"002-consumer-group-plugins",
		"003-consumer-groups-membership",
		"004-complex-state",
		"005-nested-entities",
		"006-complex-nested-structure",
	}

	for _, tc := range testCases {
		testDir := filepath.Join("testdata", "render", "deterministic-sort", tc)
		inputPath := filepath.Join(testDir, "input.yaml")
		expectedPath := filepath.Join(testDir, "expected.yaml")

		output := renderYAMLFile(t, inputPath)

		err := os.WriteFile(expectedPath, []byte(output), 0644)
		require.NoError(t, err, "Failed to write expected file: %s", expectedPath)

		t.Logf("Generated: %s", expectedPath)
	}
}
