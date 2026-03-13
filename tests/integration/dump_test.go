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

	client, err := getTestClient()
	require.NoError(t, err)

	// Clean up any existing decorations before starting the test
	existingDecorations, err := client.GraphqlRateLimitingCostDecorations.ListAll(context.Background())
	require.NoError(t, err)
	for _, d := range existingDecorations {
		_ = client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), d.ID)
	}

	// Create a graphql_ratelimiting_cost_decoration using the dedicated client
	decoration, err := client.GraphqlRateLimitingCostDecorations.CreateWithID(context.Background(), &kong.GraphqlRateLimitingCostDecoration{
		ID:           kong.String("d5308258-3c34-4f28-94f9-52e3a8a6c4b1"),
		TypePath:     kong.String("Query.users"),
		AddConstant:  kong.Float64(1.5),
		MulConstant:  kong.Float64(2.0),
		AddArguments: kong.StringSlice("limit"),
		MulArguments: kong.StringSlice("first", "last"),
	})
	require.NoError(t, err, "Should create graphql_ratelimiting_cost_decoration successfully")
	t.Logf("Created graphql_ratelimiting_cost_decoration %s with type_path %s", *decoration.ID, *decoration.TypePath)

	// Clean up after the test
	t.Cleanup(func() {
		err := client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), decoration.ID)
		require.NoError(t, err, "should delete graphql_ratelimiting_cost_decoration in cleanup")
	})

	// Call dump.Get with custom entities
	rawState, err := deckDump.Get(context.Background(), client, deckDump.Config{
		CustomEntityTypes: []string{"graphql_ratelimiting_cost_decorations"},
	})
	require.NoError(t, err, "Should dump from Kong successfully")
	require.Len(t, rawState.CustomEntities, 1, "Dumped raw state should contain 1 custom entity")

	// Check entity type
	typ := rawState.CustomEntities[0].Type()
	require.Equal(t, custom.Type("graphql_ratelimiting_cost_decorations"), typ,
		"Entity should have type graphql_ratelimiting_cost_decorations")

	// Check fields of the entity
	obj := rawState.CustomEntities[0].Object()

	typePath, ok := obj["type_path"].(string)
	require.Truef(t, ok, "'type_path' field should have type 'string' but actual '%T'", obj["type_path"])
	require.Equal(t, "Query.users", typePath)

	addConstant, ok := obj["add_constant"].(float64)
	require.Truef(t, ok, "'add_constant' field should have type 'float64' but actual '%T'", obj["add_constant"])
	require.Equal(t, 1.5, addConstant)

	mulConstant, ok := obj["mul_constant"].(float64)
	require.Truef(t, ok, "'mul_constant' field should have type 'float64' but actual '%T'", obj["mul_constant"])
	require.Equal(t, 2.0, mulConstant)
}

func Test_Dump_GraphqlRateLimitingCostDecorations_Multiple(t *testing.T) {
	kong.RunWhenEnterprise(t, ">=3.4.0", kong.RequiredFeatures{})
	setup(t)

	client, err := getTestClient()
	require.NoError(t, err)

	// Clean up any existing decorations before starting the test
	existingDecorations, err := client.GraphqlRateLimitingCostDecorations.ListAll(context.Background())
	require.NoError(t, err)
	for _, d := range existingDecorations {
		_ = client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), d.ID)
	}

	// Create multiple graphql_ratelimiting_cost_decorations
	decoration1, err := client.GraphqlRateLimitingCostDecorations.CreateWithID(context.Background(), &kong.GraphqlRateLimitingCostDecoration{
		ID:          kong.String("a1b2c3d4-1111-2222-3333-444455556666"),
		TypePath:    kong.String("Query.users"),
		AddConstant: kong.Float64(1.0),
	})
	require.NoError(t, err, "Should create first decoration successfully")

	decoration2, err := client.GraphqlRateLimitingCostDecorations.CreateWithID(context.Background(), &kong.GraphqlRateLimitingCostDecoration{
		ID:          kong.String("a1b2c3d4-2222-3333-4444-555566667777"),
		TypePath:    kong.String("Query.posts"),
		AddConstant: kong.Float64(2.0),
	})
	require.NoError(t, err, "Should create second decoration successfully")

	decoration3, err := client.GraphqlRateLimitingCostDecorations.CreateWithID(context.Background(), &kong.GraphqlRateLimitingCostDecoration{
		ID:           kong.String("a1b2c3d4-3333-4444-5555-666677778888"),
		TypePath:     kong.String("Mutation.createUser"),
		MulConstant:  kong.Float64(3.0),
		MulArguments: kong.StringSlice("count"),
	})
	require.NoError(t, err, "Should create third decoration successfully")

	// Clean up after the test
	t.Cleanup(func() {
		_ = client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), decoration1.ID)
		_ = client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), decoration2.ID)
		_ = client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), decoration3.ID)
	})

	// Call dump.Get with custom entities
	rawState, err := deckDump.Get(context.Background(), client, deckDump.Config{
		CustomEntityTypes: []string{"graphql_ratelimiting_cost_decorations"},
	})
	require.NoError(t, err, "Should dump from Kong successfully")
	require.Len(t, rawState.CustomEntities, 3, "Dumped raw state should contain 3 custom entities")

	// Verify all entities have the correct type
	for _, entity := range rawState.CustomEntities {
		require.Equal(t, custom.Type("graphql_ratelimiting_cost_decorations"), entity.Type())
	}

	// Collect all type_paths from dumped entities
	typePaths := make(map[string]bool)
	for _, entity := range rawState.CustomEntities {
		obj := entity.Object()
		typePath, ok := obj["type_path"].(string)
		require.True(t, ok, "type_path should be a string")
		typePaths[typePath] = true
	}

	// Verify all expected type_paths are present
	require.True(t, typePaths["Query.users"], "Should contain Query.users")
	require.True(t, typePaths["Query.posts"], "Should contain Query.posts")
	require.True(t, typePaths["Mutation.createUser"], "Should contain Mutation.createUser")
}

func Test_Dump_GraphqlRateLimitingCostDecorations_EmptyWhenNoneExist(t *testing.T) {
	kong.RunWhenEnterprise(t, ">=3.4.0", kong.RequiredFeatures{})
	setup(t)

	client, err := getTestClient()
	require.NoError(t, err)

	// Clean up ALL existing decorations
	existingDecorations, err := client.GraphqlRateLimitingCostDecorations.ListAll(context.Background())
	require.NoError(t, err)
	for _, d := range existingDecorations {
		_ = client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), d.ID)
	}

	// Dump when no decorations exist
	rawState, err := deckDump.Get(context.Background(), client, deckDump.Config{
		CustomEntityTypes: []string{"graphql_ratelimiting_cost_decorations"},
	})
	require.NoError(t, err, "Should dump successfully even when no decorations exist")
	require.Len(t, rawState.CustomEntities, 0, "Should have no custom entities when none exist")
}

func Test_Dump_GraphqlRateLimitingCostDecorations_MixedWithOtherCustomEntities(t *testing.T) {
	kong.RunWhenEnterprise(t, ">=3.4.0", kong.RequiredFeatures{})
	setup(t)

	client, err := getTestClient()
	require.NoError(t, err)

	// Clean up any existing decorations
	existingDecorations, err := client.GraphqlRateLimitingCostDecorations.ListAll(context.Background())
	require.NoError(t, err)
	for _, d := range existingDecorations {
		_ = client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), d.ID)
	}

	// Create a decoration
	decoration, err := client.GraphqlRateLimitingCostDecorations.CreateWithID(context.Background(), &kong.GraphqlRateLimitingCostDecoration{
		ID:          kong.String("b2c3d4e5-4444-5555-6666-777788889999"),
		TypePath:    kong.String("Query.mixed"),
		AddConstant: kong.Float64(1.0),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.GraphqlRateLimitingCostDecorations.Delete(context.Background(), decoration.ID)
	})

	// Dump with multiple custom entity types (both valid)
	rawState, err := deckDump.Get(context.Background(), client, deckDump.Config{
		CustomEntityTypes: []string{"graphql_ratelimiting_cost_decorations", "degraphql_routes"},
	})
	require.NoError(t, err, "Should dump successfully with multiple custom entity types")

	// Should have at least our decoration
	found := false
	for _, entity := range rawState.CustomEntities {
		if entity.Type() == "graphql_ratelimiting_cost_decorations" {
			obj := entity.Object()
			if obj["type_path"] == "Query.mixed" {
				found = true
				break
			}
		}
	}
	require.True(t, found, "Should find our graphql_ratelimiting_cost_decoration in mixed dump")
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
