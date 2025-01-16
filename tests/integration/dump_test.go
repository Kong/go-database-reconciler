//go:build integration

package integration

import (
	"context"
	"testing"

	deckDump "github.com/kong/go-database-reconciler/pkg/dump"
	"github.com/kong/go-kong/kong"
	"github.com/kong/go-kong/kong/custom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Dump_SelectTags_30(t *testing.T) {
	tests := []struct {
		name         string
		stateFile    string
		expectedFile string
	}{
		{
			name:         "dump with select-tags",
			stateFile:    "testdata/dump/001-entities-with-tags/kong.yaml",
			expectedFile: "testdata/dump/001-entities-with-tags/expected30.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=3.0.0 <3.1.0")
			setup(t)

			assert.NoError(t, sync(tc.stateFile))

			output, err := dump(
				"--select-tag", "managed-by-deck",
				"--select-tag", "org-unit-42",
				"-o", "-",
			)
			assert.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			assert.NoError(t, err)
			assert.Equal(t, output, expected)
		})
	}
}

func Test_Dump_SelectTags_3x(t *testing.T) {
	tests := []struct {
		name         string
		stateFile    string
		expectedFile string
	}{
		{
			name:         "dump with select-tags",
			stateFile:    "testdata/dump/001-entities-with-tags/kong.yaml",
			expectedFile: "testdata/dump/001-entities-with-tags/expected.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=3.1.0 <3.8.0")
			setup(t)

			assert.NoError(t, sync(tc.stateFile))

			output, err := dump(
				"--select-tag", "managed-by-deck",
				"--select-tag", "org-unit-42",
				"-o", "-",
			)
			assert.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			assert.NoError(t, err)
			assert.Equal(t, output, expected)
		})
	}
}

func Test_Dump_SelectTags_38x(t *testing.T) {
	tests := []struct {
		name         string
		stateFile    string
		expectedFile string
	}{
		{
			name:         "dump with select-tags",
			stateFile:    "testdata/dump/001-entities-with-tags/kong.yaml",
			expectedFile: "testdata/dump/001-entities-with-tags/expected38.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=3.8.0")
			setup(t)

			assert.NoError(t, sync(tc.stateFile))

			output, err := dump(
				"--select-tag", "managed-by-deck",
				"--select-tag", "org-unit-42",
				"-o", "-",
			)
			assert.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			assert.NoError(t, err)
			assert.Equal(t, output, expected)
		})
	}
}

func Test_Dump_SkipConsumers(t *testing.T) {
	tests := []struct {
		name          string
		stateFile     string
		expectedFile  string
		skipConsumers bool
		runWhen       func(t *testing.T)
	}{
		{
			name:          "3.2 & 3.3 dump with skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected.yaml",
			skipConsumers: true,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.2.0 <3.4.0") },
		},
		{
			name:          "3.2 & 3.3 dump with no skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected-no-skip.yaml",
			skipConsumers: false,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.2.0 <3.4.0") },
		},
		{
			name:          "3.4 dump with skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected.yaml",
			skipConsumers: true,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.4.0 <3.5.0") },
		},
		{
			name:          "3.4 dump with no skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected-no-skip-34.yaml",
			skipConsumers: false,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.4.0 <3.5.0") },
		},
		{
			name:          "3.5 dump with skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected.yaml",
			skipConsumers: true,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.5.0") },
		},
		{
			name:          "3.5 dump with no skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected-no-skip-35.yaml",
			skipConsumers: false,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.5.0 <3.8.0") },
		},
		{
			name:          "3.8.1 dump with no skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected-no-skip-38.yaml",
			skipConsumers: false,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.8.0 <3.9.0") },
		},
		{
			name:          "3.9.0 dump with no skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected-no-skip-39.yaml",
			skipConsumers: false,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.9.0") },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.runWhen(t)
			setup(t)

			assert.NoError(t, sync(tc.stateFile))

			var (
				output string
				err    error
			)
			if tc.skipConsumers {
				output, err = dump(
					"--skip-consumers",
					"-o", "-",
				)
			} else {
				output, err = dump(
					"-o", "-",
				)
			}
			assert.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			assert.NoError(t, err)
			assert.Equal(t, expected, output)
		})
	}
}

func Test_Dump_SkipConsumers_Konnect(t *testing.T) {
	tests := []struct {
		name          string
		stateFile     string
		expectedFile  string
		skipConsumers bool
	}{
		{
			name:          "dump with skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected_konnect.yaml",
			skipConsumers: true,
		},
		{
			name:          "dump with no skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected-no-skip_konnect.yaml",
			skipConsumers: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKonnect(t)
			setup(t)

			assert.NoError(t, sync(tc.stateFile))

			var (
				output string
				err    error
			)
			if tc.skipConsumers {
				output, err = dump(
					"--skip-consumers",
					"-o", "-",
				)
			} else {
				output, err = dump(
					"-o", "-",
				)
			}
			assert.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			assert.NoError(t, err)
			assert.Equal(t, expected, output)
		})
	}
}

func Test_Dump_KonnectRename(t *testing.T) {
	tests := []struct {
		name         string
		stateFile    string
		expectedFile string
		flags        []string
	}{
		{
			name:         "dump with konnect-control-plane-name",
			stateFile:    "testdata/sync/026-konnect-rename/konnect_test_cp.yaml",
			expectedFile: "testdata/sync/026-konnect-rename/konnect_test_cp.yaml",
			flags:        []string{"--konnect-control-plane-name", "test"},
		},
		{
			name:         "dump with konnect-runtime-group-name",
			stateFile:    "testdata/sync/026-konnect-rename/konnect_test_rg.yaml",
			expectedFile: "testdata/sync/026-konnect-rename/konnect_test_cp.yaml",
			flags:        []string{"--konnect-runtime-group-name", "test"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				reset(t, tc.flags...)
			})
			runWhenKonnect(t)
			setup(t)

			assert.NoError(t, sync(tc.stateFile))

			var (
				output string
				err    error
			)
			flags := []string{"-o", "-", "--with-id"}
			flags = append(flags, tc.flags...)
			output, err = dump(flags...)

			assert.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			assert.NoError(t, err)
			assert.Equal(t, expected, output)
		})
	}
}

func Test_Dump_ConsumerGroupConsumersWithCustomID(t *testing.T) {
	runWhen(t, "enterprise", ">=3.0.0")
	setup(t)

	require.NoError(t, sync("testdata/sync/028-consumer-group-consumers-custom_id/kong.yaml"))

	var output string
	flags := []string{"-o", "-", "--with-id"}
	output, err := dump(flags...)
	assert.NoError(t, err)

	expected, err := readFile("testdata/sync/028-consumer-group-consumers-custom_id/kong.yaml")
	assert.NoError(t, err)
	assert.Equal(t, expected, output)
}

func Test_Dump_ConsumerGroupConsumersWithCustomID_Konnect(t *testing.T) {
	runWhen(t, "konnect", "")
	setup(t)

	require.NoError(t, sync("testdata/sync/028-consumer-group-consumers-custom_id/kong.yaml"))

	var output string
	flags := []string{"-o", "-", "--with-id"}
	output, err := dump(flags...)
	assert.NoError(t, err)

	expected, err := readFile("testdata/dump/003-consumer-group-consumers-custom_id/konnect.yaml")
	assert.NoError(t, err)
	assert.Equal(t, expected, output)
}

func Test_Dump_CustomEntities(t *testing.T) {
	kong.RunWhenEnterprise(t, ">=3.0.0", kong.RequiredFeatures{})
	setup(t)

	require.NoError(t, sync("testdata/sync/001-create-a-service/kong3x.yaml"))
	// Create a degraphql_route attached to the service after created a service by dedicated client
	// because deck sync does not support custom entities.
	const serviceID = "58076db2-28b6-423b-ba39-a797193017f7" // ID of the service in the config file
	client, err := getTestClient()
	require.NoError(t, err)
	r, err := client.DegraphqlRoutes.Create(context.Background(), &kong.DegraphqlRoute{
		Service: &kong.Service{
			ID: kong.String(serviceID),
		},
		URI:   kong.String("/graphql"),
		Query: kong.String("query{ name }"),
	})
	require.NoError(t, err, "Should create degraphql_routes sucessfully")
	t.Logf("Created degraphql_routes %s attached to service %s", *r.ID, serviceID)

	// Since degraphql_route does not run cascade delete on services, we need to clean it up after the test.
	t.Cleanup(func() {
		err := client.DegraphqlRoutes.Delete(context.Background(), kong.String(serviceID), r.ID)
		require.NoError(t, err, "should delete degraphql_routes in cleanup")
	})

	// Call dump.Get with custom entities because deck's `dump` command does not support custom entities.
	rawState, err := deckDump.Get(context.Background(), client, deckDump.Config{
		CustomEntityTypes: []string{"degraphql_routes"},
	})
	require.NoError(t, err, "Should dump from Kong successfully")
	require.Len(t, rawState.CustomEntities, 1, "Dumped raw state should contain 1 custom entity")
	// check entity type
	typ := rawState.CustomEntities[0].Type()
	require.Equal(t, custom.Type("degraphql_routes"), typ, "Entity should have type degraphql_routes")
	// check fields of the entity
	obj := rawState.CustomEntities[0].Object()
	uri, ok := obj["uri"].(string)
	require.Truef(t, ok, "'uri' field should have type 'string' but actual '%T'", obj["uri"])
	require.Equal(t, "/graphql", uri)
	query, ok := obj["query"].(string)
	require.Truef(t, ok, "'query' field should have type 'string' but actual '%T'", obj["query"])
	require.Equal(t, "query{ name }", query)
}
