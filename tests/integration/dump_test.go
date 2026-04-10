//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	deckDump "github.com/kong/go-database-reconciler/pkg/dump"
	"github.com/kong/go-kong/kong"
	"github.com/kong/go-kong/kong/custom"
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

			require.NoError(t, sync(tc.stateFile))

			output, err := dump(
				"--select-tag", "managed-by-deck",
				"--select-tag", "org-unit-42",
				"-o", "-",
			)
			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
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

			require.NoError(t, sync(tc.stateFile))

			output, err := dump(
				"--select-tag", "managed-by-deck",
				"--select-tag", "org-unit-42",
				"-o", "-",
			)
			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
			assert.Equal(t, output, expected)
		})
	}
}

func Test_Dump_SelectTags_Post_38x(t *testing.T) {
	tests := []struct {
		name           string
		stateFile      string
		expectedFile   string
		runWhenVersion string
	}{
		{
			name:           "dump with select-tags >=3.8.0 <3.10.0",
			stateFile:      "testdata/dump/001-entities-with-tags/kong.yaml",
			expectedFile:   "testdata/dump/001-entities-with-tags/expected38.yaml",
			runWhenVersion: ">=3.8.0 <3.10.0",
		},
		{
			name:           "dump with select-tags >=3.10.0 < 3.11.0",
			stateFile:      "testdata/dump/001-entities-with-tags/kong.yaml",
			expectedFile:   "testdata/dump/001-entities-with-tags/expected310.yaml",
			runWhenVersion: ">=3.10.0 <3.11.0",
		},
		{
			name:           "dump with select-tags >=3.11.0",
			stateFile:      "testdata/dump/001-entities-with-tags/kong.yaml",
			expectedFile:   "testdata/dump/001-entities-with-tags/expected311.yaml",
			runWhenVersion: ">=3.11.0 <3.12.0",
		},
		{
			name:           "dump with select-tags >=3.12.0",
			stateFile:      "testdata/dump/001-entities-with-tags/kong.yaml",
			expectedFile:   "testdata/dump/001-entities-with-tags/expected312.yaml",
			runWhenVersion: ">=3.12.0",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", tc.runWhenVersion)
			setup(t)

			require.NoError(t, sync(tc.stateFile))

			output, err := dump(
				"--select-tag", "managed-by-deck",
				"--select-tag", "org-unit-42",
				"-o", "-",
			)
			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
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
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.9.0 <3.10.0") },
		},
		{
			name:          "3.10.0 dump with no skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected-no-skip-310.yaml",
			skipConsumers: false,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.10.0 <3.12.0") },
		},
		{
			name:          "3.12.0 dump with no skip-consumers",
			stateFile:     "testdata/dump/002-skip-consumers/kong34.yaml",
			expectedFile:  "testdata/dump/002-skip-consumers/expected-no-skip-312.yaml",
			skipConsumers: false,
			runWhen:       func(t *testing.T) { runWhen(t, "enterprise", ">=3.12.0") },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.runWhen(t)
			setup(t)

			require.NoError(t, sync(tc.stateFile))

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
			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
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

			require.NoError(t, sync(tc.stateFile))

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
			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
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

			require.NoError(t, sync(tc.stateFile))

			var (
				output string
				err    error
			)
			flags := []string{"-o", "-", "--with-id"}
			flags = append(flags, tc.flags...)
			output, err = dump(flags...)

			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
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
	require.NoError(t, err)

	expected, err := readFile("testdata/sync/028-consumer-group-consumers-custom_id/kong.yaml")
	require.NoError(t, err)
	assert.Equal(t, expected, output)
}

func Test_Dump_ConsumerGroupConsumersWithCustomID_Konnect(t *testing.T) {
	runWhen(t, "konnect", "")
	setup(t)

	require.NoError(t, sync("testdata/sync/028-consumer-group-consumers-custom_id/kong.yaml"))

	var output string
	flags := []string{"-o", "-", "--with-id"}
	output, err := dump(flags...)
	require.NoError(t, err)

	expected, err := readFile("testdata/dump/003-consumer-group-consumers-custom_id/konnect.yaml")
	require.NoError(t, err)
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

func Test_Dump_GraphqlRateLimitingCostDecorations(t *testing.T) {
	kong.RunWhenEnterprise(t, ">=3.0.0", kong.RequiredFeatures{})
	setup(t)

	// Sync a service first so we can attach cost decorations to it
	require.NoError(t, sync("testdata/sync/001-create-a-service/kong3x.yaml"))
	const serviceID = "58076db2-28b6-423b-ba39-a797193017f7"

	client, err := getTestClient()
	require.NoError(t, err)

	tests := []struct {
		name              string
		decorations       []*kong.GraphqlRateLimitingCostDecoration
		customEntityTypes []string
		expectedCount     int
		expectedTypePaths []string
	}{
		{
			name: "dump single decoration with all fields",
			decorations: []*kong.GraphqlRateLimitingCostDecoration{
				{
					ID:           kong.String("d5308258-3c34-4f28-94f9-52e3a8a6c4b1"),
					Service:      &kong.Service{ID: kong.String(serviceID)},
					TypePath:     kong.String("Query.users"),
					AddConstant:  kong.Float64(1.5),
					MulConstant:  kong.Float64(2.0),
					AddArguments: kong.StringSlice("limit"),
					MulArguments: kong.StringSlice("first", "last"),
				},
			},
			customEntityTypes: []string{"graphql_ratelimiting_cost_decorations"},
			expectedCount:     1,
			expectedTypePaths: []string{"Query.users"},
		},
		{
			name: "dump multiple decorations",
			decorations: []*kong.GraphqlRateLimitingCostDecoration{
				{
					ID:          kong.String("a1b2c3d4-1111-2222-3333-444455556666"),
					Service:     &kong.Service{ID: kong.String(serviceID)},
					TypePath:    kong.String("Query.users"),
					AddConstant: kong.Float64(1.0),
				},
				{
					ID:          kong.String("a1b2c3d4-1111-2222-3333-444422229999"),
					Service:     &kong.Service{ID: kong.String(serviceID)},
					TypePath:    kong.String("Query.posts"),
					AddConstant: kong.Float64(2.0),
				},
				{
					ID:           kong.String("a1b2c3d4-3333-4444-5555-666677778888"),
					Service:      &kong.Service{ID: kong.String(serviceID)},
					TypePath:     kong.String("Mutation.createUser"),
					MulConstant:  kong.Float64(3.0),
					MulArguments: kong.StringSlice("count"),
				},
			},
			customEntityTypes: []string{"graphql_ratelimiting_cost_decorations"},
			expectedCount:     3,
			expectedTypePaths: []string{"Query.users", "Query.posts", "Mutation.createUser"},
		},
		{
			name:              "dump empty when none exist",
			decorations:       nil,
			customEntityTypes: []string{"graphql_ratelimiting_cost_decorations"},
			expectedCount:     0,
			expectedTypePaths: nil,
		},
		{
			name: "dump mixed with other custom entity types",
			decorations: []*kong.GraphqlRateLimitingCostDecoration{
				{
					ID:          kong.String("b2c3d4e5-4444-5555-6666-777788889999"),
					Service:     &kong.Service{ID: kong.String(serviceID)},
					TypePath:    kong.String("Query.mixed"),
					AddConstant: kong.Float64(1.0),
				},
			},
			customEntityTypes: []string{"graphql_ratelimiting_cost_decorations", "degraphql_routes"},
			expectedCount:     1,
			expectedTypePaths: []string{"Query.mixed"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset Kong state and re-sync the service for each sub-test
			// to ensure no stale decorations from previous sub-tests.
			reset(t)
			require.NoError(t, sync("testdata/sync/001-create-a-service/kong3x.yaml"))

			// Create decorations for this test case
			for _, deco := range tc.decorations {
				_, err := client.GraphqlRateLimitingCostDecorations.CreateForService(
					context.Background(), deco)
				require.NoError(t, err, "Should create decoration successfully")
			}

			// Call dump.Get with custom entities
			rawState, err := deckDump.Get(context.Background(), client, deckDump.Config{
				CustomEntityTypes: tc.customEntityTypes,
			})
			require.NoError(t, err, "Should dump from Kong successfully")

			// Filter only graphql_ratelimiting_cost_decorations from the result
			var costDecoEntities []custom.Entity
			for _, entity := range rawState.CustomEntities {
				if entity.Type() == "graphql_ratelimiting_cost_decorations" {
					costDecoEntities = append(costDecoEntities, entity)
				}
			}
			require.Len(t, costDecoEntities, tc.expectedCount)

			// Verify expected type_paths
			typePaths := make(map[string]bool)
			for _, entity := range costDecoEntities {
				obj := entity.Object()
				typePath, ok := obj["type_path"].(string)
				require.True(t, ok, "type_path should be a string")
				typePaths[typePath] = true
			}
			for _, expected := range tc.expectedTypePaths {
				require.True(t, typePaths[expected], "Should contain %s", expected)
			}
		})
	}
}

func Test_Dump_KeysAndKeySets(t *testing.T) {
	runWhen(t, "kong", ">=3.1.0")
	setup(t)

	tests := []struct {
		name         string
		stateFile    string
		expectedFile string
	}{
		{
			name:         "dump keys and key-sets - jwk keys",
			stateFile:    "testdata/dump/005-keys-and-key_sets/kong.yaml",
			expectedFile: "testdata/dump/005-keys-and-key_sets/kong.yaml",
		},
		{
			name:         "dump keys and key-sets - pem keys",
			stateFile:    "testdata/dump/005-keys-and-key_sets/kong-pem.yaml",
			expectedFile: "testdata/dump/005-keys-and-key_sets/kong-pem.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, sync(tc.stateFile))

			output, err := dump("-o", "-", "--with-id")
			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
			assert.Equal(t, expected, output)
		})
	}
}

func Test_Dump_PluginWithPartials_Select_Tags(t *testing.T) {
	runWhen(t, "kong", ">=3.10.0")
	setup(t)

	tests := []struct {
		name            string
		stateFile       string
		expectedFile    string
		expectedWarning string
	}{
		{
			name:         "dump with select-tags: global plugins with partials",
			stateFile:    "testdata/dump/006-plugin-partials/plugin-partials-different-tags-global.yaml",
			expectedFile: "testdata/dump/006-plugin-partials/select-tagged-dump-global.yaml",
			expectedWarning: "Warning: partial my-redis-config referenced in plugin rate-limiting not found in state.\n" +
				"Ensure valid `default_lookup_tags` are set before syncing.\n",
		},
		{
			name:         "dump with select-tags: nested plugins with partials",
			stateFile:    "testdata/dump/006-plugin-partials/plugin-partials-different-tags-nested.yaml",
			expectedFile: "testdata/dump/006-plugin-partials/select-tagged-dump-nested.yaml",
			expectedWarning: "Warning: partial redis-svc referenced in plugin rate-limiting not found in state.\n" +
				"Ensure valid `default_lookup_tags` are set before syncing.\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, sync(tc.stateFile))

			flags := []string{"-o", "-", "--select-tag", "example"}
			stdout, stderr, err := dumpWithStdErrCheck(flags...)
			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
			assert.Equal(t, expected, stdout)
			assert.Equal(t, tc.expectedWarning, stderr)
		})
	}
}

func Test_Dump_Services_TLS_Sans(t *testing.T) {
	runWhen(t, "enterprise", ">=3.10.0")

	tests := []struct {
		name         string
		stateFile    string
		expectedFile string
	}{
		{
			name:         "dump services with TLS SANs",
			stateFile:    "testdata/sync/046-service-tls-sans/kong.yaml",
			expectedFile: "testdata/dump/007-services-tls-sans/kong.yaml",
		},
		{
			name:         "dump services with https but no TLS SANs",
			stateFile:    "testdata/sync/046-service-tls-sans/no-tls-https.yaml",
			expectedFile: "testdata/dump/007-services-tls-sans/no-tls-https.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setup(t)
			require.NoError(t, sync(tc.stateFile))

			output, err := dump("-o", "-", "--with-id")
			require.NoError(t, err)

			expected, err := readFile(tc.expectedFile)
			require.NoError(t, err)
			assert.Equal(t, expected, output)
		})
	}
}

func Test_Dump_Plugin_Conditional(t *testing.T) {
	runWhen(t, "enterprise", ">=3.14.0")
	client, err := getTestClient()
	require.NoError(t, err)

	ctx := context.Background()
	kongFile := "testdata/sync/003-create-a-plugin/kong-conditional.yaml"

	mustResetKongState(ctx, t, client, deckDump.Config{})
	require.NoError(t, sync(kongFile))

	// resync with no error
	output, err := dump("-o", "-", "--with-id")
	require.NoError(t, err)

	expectedFile := "testdata/dump/008-plugin-conditional/expected.yaml"
	expected, err := readFile(expectedFile)
	require.NoError(t, err)
	assert.Equal(t, expected, output)
}
