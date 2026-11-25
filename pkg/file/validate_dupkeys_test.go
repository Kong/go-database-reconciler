package file

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetContentFromReader_DuplicateKeys(t *testing.T) {
	const header = "_format_version: \"3.0\"\n"

	tests := []struct {
		name          string
		content       string
		wantError     bool
		expectedError []string
	}{
		{
			name: "duplicate top-level key",
			content: header +
				"services:\n- name: a\n  host: example.com\n" +
				"services:\n- name: b\n  host: example.com\n",
			wantError:     true,
			expectedError: []string{"already set in map", "services"},
		},
		{
			name: "duplicate nested key",
			content: header +
				"services:\n- name: a\n  host: one.com\n  host: two.com\n",
			wantError:     true,
			expectedError: []string{"already set in map", "host"},
		},
		{
			name: "duplicate _info key",
			content: header +
				"_info:\n  select_tags:\n  - a\n" +
				"_info:\n  select_tags:\n  - b\n",
			wantError:     true,
			expectedError: []string{"already set in map", "_info"},
		},
		{
			name: "valid file with _info",
			content: header +
				"_info:\n  select_tags:\n  - a\n" +
				"services:\n- name: a\n  host: example.com\n",
			wantError: false,
		},
		{
			name: "valid file without duplicates",
			content: header +
				"services:\n- name: a\n  host: example.com\n",
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetContentFromReader(strings.NewReader(tt.content), EnvVarsSkip)
			if !tt.wantError {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, want := range tt.expectedError {
				require.Contains(t, err.Error(), want)
			}
		})
	}
}
