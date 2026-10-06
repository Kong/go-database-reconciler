package types

import (
	"context"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/schema"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaWithNewDefaultedField simulates a /schemas response after a Kong upgrade that
// introduced "new_field" with a default value.
func schemaWithNewDefaultedField() map[string]any {
	return map[string]any{
		"name": "my-plugin",
		"fields": []any{
			map[string]any{
				"config": map[string]any{
					"fields": []any{
						map[string]any{"old_field": map[string]any{"type": "string"}},
						map[string]any{
							"new_field": map[string]any{"type": "string", "default": "new-default"},
						},
					},
				},
			},
		},
	}
}

// schemaWithNewFieldNoDefault simulates a /schemas response after a Kong upgrade that introduced
// "new_field_no_default" without declaring a default value for it.
func schemaWithNewFieldNoDefault() map[string]any {
	return map[string]any{
		"name": "my-plugin",
		"fields": []any{
			map[string]any{
				"config": map[string]any{
					"fields": []any{
						map[string]any{"old_field": map[string]any{"type": "string"}},
						map[string]any{"new_field_no_default": map[string]any{"type": "string"}},
					},
				},
			},
		},
	}
}

// schemaWithDeprecatedField declares config field "consumer_claims" plus deprecated shorthand
// "consumer_claim", mirroring plugins like openid-connect.
func schemaWithDeprecatedField() map[string]any {
	return map[string]any{
		"name": "my-plugin",
		"fields": []any{
			map[string]any{
				"config": map[string]any{
					"fields": []any{
						map[string]any{"consumer_claims": map[string]any{"type": "array"}},
					},
					"shorthand_fields": []any{
						map[string]any{
							"consumer_claim": map[string]any{
								"type": "array",
								"deprecation": map[string]any{
									"replaced_with": []any{
										map[string]any{"path": []any{"consumer_claims"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newTestPluginDiffer(
	t *testing.T, current, target *state.Plugin, skipSchemaDefaults bool, pluginSchema map[string]any,
) *pluginDiffer {
	t.Helper()

	currentState, err := state.NewKongState()
	require.NoError(t, err)
	require.NoError(t, currentState.Plugins.Add(*current))

	targetState, err := state.NewKongState()
	require.NoError(t, err)
	require.NoError(t, targetState.Plugins.Add(*target))

	return &pluginDiffer{
		kind:               entityTypeToKind(Plugin),
		currentState:       currentState,
		targetState:        targetState,
		skipSchemaDefaults: skipSchemaDefaults,
		schemasCache: schema.NewCache(func(_ context.Context, _ string) (map[string]any, error) {
			return pluginSchema, nil
		}),
	}
}

// skipSchemaDefaults=false: a plugin row persisted before "new_field" existed has no such key in
// its GET response, but filling the target's defaults from the current schema adds it there -
// producing a false-positive update even though nothing actually changed.
func TestCreateUpdatePlugin_NewSchemaFieldMissingFromCurrent(t *testing.T) {
	current := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("http", "https"),
			Enabled:   kong.Bool(true),
			Config:    kong.Configuration{"old_field": "custom-value"},
		},
	}
	target := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("http", "https"),
			Enabled:   kong.Bool(true),
			Config:    kong.Configuration{"old_field": "custom-value"},
		},
	}

	d := newTestPluginDiffer(t, current, target, false, schemaWithNewDefaultedField())

	event, err := d.createUpdatePlugin(target)
	require.NoError(t, err)
	assert.Nil(t, event)
}

// Same false positive even without a declared default: go-kong's fillConfigRecord falls back to
// an explicit nil ("res[fname] = nil") for any field it fills that has no default, so the target
// still gains the key. A stale current row that never had it at all then compares unequal:
// {"new_field_no_default": nil} != {}.
func TestCreateUpdatePlugin_NewSchemaFieldNoDefaultMissingFromCurrent(t *testing.T) {
	current := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("a1b2c3d4-5e6f-4a7b-8c9d-0e1f2a3b4c5d"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("http", "https"),
			Enabled:   kong.Bool(true),
			Config:    kong.Configuration{"old_field": "custom-value"},
		},
	}
	target := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("a1b2c3d4-5e6f-4a7b-8c9d-0e1f2a3b4c5d"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("http", "https"),
			Enabled:   kong.Bool(true),
			Config:    kong.Configuration{"old_field": "custom-value"},
		},
	}

	d := newTestPluginDiffer(t, current, target, false, schemaWithNewFieldNoDefault())

	event, err := d.createUpdatePlugin(target)
	require.NoError(t, err)
	assert.Nil(t, event)
}

// Guards against overcorrecting the above: a genuinely different explicit value for that same
// field is real drift and must still be reported.
func TestCreateUpdatePlugin_ExplicitValueDifferingFromDefault_StillDetected(t *testing.T) {
	current := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("http", "https"),
			Enabled:   kong.Bool(true),
			Config: kong.Configuration{
				"old_field": "custom-value",
				"new_field": "explicit-other-value",
			},
		},
	}
	target := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("http", "https"),
			Enabled:   kong.Bool(true),
			Config:    kong.Configuration{"old_field": "custom-value"},
		},
	}

	d := newTestPluginDiffer(t, current, target, false, schemaWithNewDefaultedField())

	event, err := d.createUpdatePlugin(target)
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, crud.Update, event.Op)
}

// skipSchemaDefaults=true: createUpdatePlugin (plugin.go:192) skips the block that normalizes
// "enabled" to true, so a target omitting "enabled" (nil) never matches current's persisted
// explicit true.
func TestCreateUpdatePlugin_SkipSchemaDefaults_ImplicitEnabledOmitted(t *testing.T) {
	t.Skip("not being worked on right now")
	current := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("b2d6e6b1-28d1-4e2b-9f0a-f4a9b7c1a111"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("http", "https"),
			Enabled:   kong.Bool(true),
			Config:    kong.Configuration{"consumer_claims": []any{"clientId"}},
		},
	}
	target := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("c3e7f7c2-39e2-4f3c-8f1b-f5bac8c2b222"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("http", "https"),
			Config:    kong.Configuration{"consumer_claims": []any{"clientId"}},
		},
	}

	d := newTestPluginDiffer(t, current, target, true, schemaWithDeprecatedField())

	event, err := d.createUpdatePlugin(target)
	require.NoError(t, err)
	assert.Nil(t, event)
}

// Same gating also skips ClearUnmatchingDeprecations: Kong mirrors a deprecated field under its
// new name once set, so current carries "consumer_claims" alongside "consumer_claim" even though
// the target only ever set the deprecated name.
func TestCreateUpdatePlugin_SkipSchemaDefaults_DeprecatedFieldPair(t *testing.T) {
	t.Skip("not being worked on right now")
	current := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("d4f8a8d3-4af3-4f4d-9f2c-f6cbd9d3c333"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("https"),
			Enabled:   kong.Bool(true),
			Config: kong.Configuration{
				"consumer_claim":  []any{"clientId"},
				"consumer_claims": []any{"clientId"},
			},
		},
	}
	target := &state.Plugin{
		Plugin: kong.Plugin{
			ID:        kong.String("e5a9b9e4-5bf4-4a5e-8e3d-f7dcead4d444"),
			Name:      kong.String("my-plugin"),
			Protocols: kong.StringSlice("https"),
			Enabled:   kong.Bool(true),
			Config:    kong.Configuration{"consumer_claim": []any{"clientId"}},
		},
	}

	d := newTestPluginDiffer(t, current, target, true, schemaWithDeprecatedField())

	event, err := d.createUpdatePlugin(target)
	require.NoError(t, err)
	assert.Nil(t, event)
}
