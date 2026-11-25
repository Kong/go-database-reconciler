package file

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetContentFromReader_DuplicateKeys(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantKey string
	}{
		{
			name: "duplicate top-level key",
			content: "_format_version: \"3.0\"\n" +
				"services:\n- name: a\n  host: example.com\n" +
				"services:\n- name: b\n  host: example.com\n",
			wantKey: "services",
		},
		{
			name: "duplicate nested key",
			content: "_format_version: \"3.0\"\n" +
				"services:\n- name: a\n  host: one.com\n  host: two.com\n",
			wantKey: "host",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetContentFromReader(strings.NewReader(tt.content), EnvVarsSkip)
			require.Error(t, err)
			require.Contains(t, err.Error(), "already set in map")
			require.Contains(t, err.Error(), tt.wantKey)
		})
	}
}

func TestGetContentFromReader_NoDuplicateKeys(t *testing.T) {
	content := "_format_version: \"3.0\"\nservices:\n- name: a\n  host: example.com\n"
	_, err := GetContentFromReader(strings.NewReader(content), EnvVarsSkip)
	require.NoError(t, err)
}
