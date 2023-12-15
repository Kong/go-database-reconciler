//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/stretchr/testify/assert"
)

var (
	expectedOutputMasked = ` {
   "connect_timeout": 60000,
   "enabled": true,
   "host": "[masked]",
   "id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
   "name": "svc1",
   "port": 80,
   "protocol": "http",
   "read_timeout": 60000,
   "retries": 5,
   "write_timeout": 60000
+  "tags": [
+    "[masked] is an external host. I like [masked]!",
+    "foo:foo",
+    "baz:[masked]",
+    "another:[masked]",
+    "bar:[masked]"
+  ]
 }
`

	expectedOutputUnMasked = ` {
   "connect_timeout": 60000,
   "enabled": true,
   "host": "mockbin.org",
   "id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
   "name": "svc1",
   "port": 80,
   "protocol": "http",
   "read_timeout": 60000,
   "retries": 5,
   "write_timeout": 60000
+  "tags": [
+    "test"
+  ]
 }
`

	diffEnvVars = map[string]string{
		"DECK_SVC1_HOSTNAME": "mockbin.org",
		"DECK_BARR":          "barbar",
		"DECK_BAZZ":          "bazbaz",   // used more than once
		"DECK_FUB":           "fubfub",   // unused
		"DECK_FOO":           "foo_test", // unused, partial match
	}
)

// test scope:
//   - 2.8.0
func Test_Diff_Masked_OlderThan3x(t *testing.T) {
	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
		envVars          map[string]string
	}{
		{
			name:             "env variable are masked",
			initialStateFile: "testdata/diff/002-mask/initial.yaml",
			stateFile:        "testdata/diff/002-mask/kong.yaml",
			envVars:          diffEnvVars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", "==2.8.0")
			setup(t)

			// initialize state
			assert.NoError(t, sync(tc.initialStateFile))

			out, err := testSync(context.Background(), []string{tc.stateFile}, false, true)
			assert.NoError(t, err)
			assert.NoError(t, err)
			assert.Equal(t, int32(1), out.Stats.CreateOps.Count())
			assert.Equal(t, int32(1), out.Stats.UpdateOps.Count())
			assert.Equal(t, "rate-limiting (global)", out.Changes.Creating[0].Name)
			assert.Equal(t, "plugin", out.Changes.Creating[0].Kind)
			assert.Equal(t, "svc1", out.Changes.Updating[0].Name)
			assert.Equal(t, "service", out.Changes.Updating[0].Kind)
			assert.NotEmpty(t, out.Changes.Updating[0].Diff)
			assert.Equal(t, expectedOutputMasked, out.Changes.Updating[0].Diff)
		})
	}
}

// test scope:
//   - 3.x
func Test_Diff_Masked_NewerThan3x(t *testing.T) {
	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
		envVars          map[string]string
	}{
		{
			name:             "env variable are masked",
			initialStateFile: "testdata/diff/002-mask/initial3x.yaml",
			stateFile:        "testdata/diff/002-mask/kong3x.yaml",
			envVars:          diffEnvVars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.0.0")
			setup(t)

			// initialize state
			assert.NoError(t, sync(tc.initialStateFile))

			out, err := testSync(context.Background(), []string{tc.stateFile}, false, true)
			assert.NoError(t, err)
			assert.NoError(t, err)
			assert.Equal(t, int32(1), out.Stats.CreateOps.Count())
			assert.Equal(t, int32(1), out.Stats.UpdateOps.Count())
			assert.Equal(t, "rate-limiting (global)", out.Changes.Creating[0].Name)
			assert.Equal(t, "plugin", out.Changes.Creating[0].Kind)
			assert.Equal(t, "svc1", out.Changes.Updating[0].Name)
			assert.Equal(t, "service", out.Changes.Updating[0].Kind)
			assert.NotEmpty(t, out.Changes.Updating[0].Diff)
			assert.Equal(t, expectedOutputMasked, out.Changes.Updating[0].Diff)
		})
	}
}

// test scope:
//   - 2.8.0
func Test_Diff_Unmasked_OlderThan3x(t *testing.T) {
	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
		envVars          map[string]string
	}{
		{
			name:             "env variable are unmasked",
			initialStateFile: "testdata/diff/003-unmask/initial.yaml",
			stateFile:        "testdata/diff/003-unmask/kong.yaml",
			envVars:          diffEnvVars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", "==2.8.0")
			setup(t)

			// initialize state
			assert.NoError(t, sync(tc.initialStateFile))

			out, err := testSync(context.Background(), []string{tc.stateFile}, true, true)
			assert.NoError(t, err)
			assert.NoError(t, err)
			assert.Equal(t, int32(1), out.Stats.CreateOps.Count())
			assert.Equal(t, int32(1), out.Stats.UpdateOps.Count())
			assert.Equal(t, "rate-limiting (global)", out.Changes.Creating[0].Name)
			assert.Equal(t, "plugin", out.Changes.Creating[0].Kind)
			assert.Equal(t, "svc1", out.Changes.Updating[0].Name)
			assert.Equal(t, "service", out.Changes.Updating[0].Kind)
			assert.NotEmpty(t, out.Changes.Updating[0].Diff)
			assert.Equal(t, expectedOutputUnMasked, out.Changes.Updating[0].Diff)
		})
	}
}

// test scope:
//   - 3.x
func Test_Diff_Unmasked_NewerThan3x(t *testing.T) {
	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
		envVars          map[string]string
	}{
		{
			name:             "env variable are unmasked",
			initialStateFile: "testdata/diff/003-unmask/initial3x.yaml",
			stateFile:        "testdata/diff/003-unmask/kong3x.yaml",
			envVars:          diffEnvVars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.0.0")
			setup(t)

			// initialize state
			assert.NoError(t, sync(tc.initialStateFile))

			out, err := testSync(context.Background(), []string{tc.stateFile}, true, true)
			assert.NoError(t, err)
			assert.Equal(t, int32(1), out.Stats.CreateOps.Count())
			assert.Equal(t, int32(1), out.Stats.UpdateOps.Count())
			assert.Equal(t, "rate-limiting (global)", out.Changes.Creating[0].Name)
			assert.Equal(t, "plugin", out.Changes.Creating[0].Kind)
			assert.Equal(t, "svc1", out.Changes.Updating[0].Name)
			assert.Equal(t, "service", out.Changes.Updating[0].Kind)
			assert.NotEmpty(t, out.Changes.Updating[0].Diff)
			assert.Equal(t, expectedOutputUnMasked, out.Changes.Updating[0].Diff)
		})
	}
}
