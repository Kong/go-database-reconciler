package file

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/dump"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ensureJSON(t *testing.T) {
	type args struct {
		m map[string]any
	}
	tests := []struct {
		name string
		args args
		want map[string]any
	}{
		{
			"empty array is kept as is",
			args{map[string]any{
				"foo": []any{},
			}},
			map[string]any{
				"foo": []any{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ensureJSON(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ensureJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadKongStateFromStdinFailsToParseText(t *testing.T) {
	filenames := []string{"-"}
	assert := assert.New(t)
	assert.Equal("-", filenames[0])

	var content bytes.Buffer
	content.Write([]byte("hunter2\n"))

	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content.Bytes()); err != nil {
		panic(err)
	}

	if _, err := tmpfile.Seek(0, 0); err != nil {
		panic(err)
	}

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }() // Restore original Stdin

	os.Stdin = tmpfile

	c, err := GetContentFromFilesWithEnvVars(filenames, EnvVarsExpand)
	require.Error(t, err)
	assert.Nil(c)
}

func TestTransformNotFalse(t *testing.T) {
	filenames := []string{"-"}
	assert := assert.New(t)

	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString("_transform: false\nservices:\n- host: test.com\n  name: test service\n")
	if err != nil {
		panic(err)
	}

	if _, err := tmpfile.Seek(0, 0); err != nil {
		panic(err)
	}

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }() // Restore original Stdin

	os.Stdin = tmpfile

	c, err := GetContentFromFilesWithEnvVars(filenames, EnvVarsExpand)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	parsed, err := Get(ctx, c, RenderConfig{}, dump.Config{}, nil)
	assert.Equal(err, ErrorTransformFalseNotSupported)
	assert.Nil(parsed)

	parsed, _, err = GetForKonnect(ctx, c, RenderConfig{}, nil)
	assert.Equal(err, ErrorTransformFalseNotSupported)
	assert.Nil(parsed)
}

func TestReadKongStateFromStdin(t *testing.T) {
	filenames := []string{"-"}
	assert := assert.New(t)
	assert.Equal("-", filenames[0])

	var content bytes.Buffer
	content.Write([]byte("services:\n- host: test.com\n  name: test service\n"))

	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content.Bytes()); err != nil {
		panic(err)
	}

	if _, err := tmpfile.Seek(0, 0); err != nil {
		panic(err)
	}

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }() // Restore original Stdin

	os.Stdin = tmpfile

	c, err := GetContentFromFilesWithEnvVars(filenames, EnvVarsExpand)
	assert.NotNil(c)
	require.NoError(t, err)

	assert.Equal(kong.Service{
		Name: new("test service"),
		Host: new("test.com"),
	},
		c.Services[0].Service)
}

func TestReadKongStateFromFile(t *testing.T) {
	filenames := []string{"testdata/config.yaml"}
	require.Equal(t, "testdata/config.yaml", filenames[0])

	c, err := GetContentFromFilesWithEnvVars(filenames, EnvVarsExpand)
	require.NotNil(t, c)
	require.NoError(t, err)

	t.Run("enabled field for service is read", func(t *testing.T) {
		assert.Equal(t, kong.Service{
			Name:    new("svc1"),
			Host:    new("mockbin.org"),
			Enabled: new(true),
		}, c.Services[0].Service)
	})
}

func TestGetContentFromFilesCompatibilityWrapper(t *testing.T) {
	t.Setenv("DECK_SVC2_HOST", "2.example.com")
	t.Setenv("DECK_FILE_LOG_FUNCTION", "return")

	filenames := []string{"testdata/file.yaml"}

	expanded, err := GetContentFromFiles(filenames, false)
	require.NoError(t, err)
	require.NotNil(t, expanded)
	assert.Equal(t, "2.example.com", *expanded.Services[0].Host)

	mocked, err := GetContentFromFiles(filenames, true)
	require.NoError(t, err)
	require.NotNil(t, mocked)
	assert.Equal(t, "DECK_SVC2_HOST", *mocked.Services[0].Host)
}
