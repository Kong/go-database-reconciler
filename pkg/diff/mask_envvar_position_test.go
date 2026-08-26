package diff

import (
	"strings"
	"testing"
)

// Regression tests for https://github.com/Kong/deck/issues/1379
//
// MaskEnvVarValue previously only masked DECK_ env var values when they
// appeared in specific positions of a diff line (JSON quoted value, bare
// number, YAML "key: value", or standalone array element). A secret that
// appeared anywhere else — after an "=", inside a compound value such as
// "Bearer <secret>", or as free text — was printed verbatim.

func TestMaskEnvVarValueMasksSecretInAnyPosition(t *testing.T) {
	secret := "supersecretotelapmtoken12345"
	t.Setenv("DECK_OTEL_APM_TOKEN", secret)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "json_quoted_value",
			input: `"headers": {"Authorization": "Bearer ` + secret + `"}`,
		},
		{
			name:  "yaml_unquoted_value",
			input: `  headers.Authorization: Bearer ` + secret,
		},
		{
			name:  "equals_sign_separator",
			input: `token=` + secret,
		},
		{
			name:  "free_text",
			input: secret + ` is the token`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MaskEnvVarValue(tc.input)
			if strings.Contains(got, secret) {
				t.Fatalf("secret leaked in diff output:\ninput:  %s\noutput: %s", tc.input, got)
			}
			if !strings.Contains(got, maskedValue) {
				t.Fatalf("expected masked marker %q in output, got: %s", maskedValue, got)
			}
		})
	}
}
