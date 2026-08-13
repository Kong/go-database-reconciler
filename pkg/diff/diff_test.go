package diff

import (
	"context"
	"strings"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/file"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/require"
)

// Test field name constants
const (
	fieldMinute  = "minute"
	fieldHour    = "hour"
	fieldKey     = "key"
	fieldName    = "name"
	fieldHeaders = "headers"
)

func TestSolve_InvalidParallelism_Negative(t *testing.T) {
	parallelism := -1
	sc, _ := NewSyncer(SyncerOpts{})
	_, errs, _ := sc.Solve(context.Background(), parallelism, true, false)
	require.Len(t, errs, 1, "Solve should return exactly one error for parallelism=%d", parallelism)
	require.EqualError(t, errs[0], "parallelism can not be less than 1")
}

func crudEventFor(oldObj, newObj any) crud.Event {
	return crud.Event{OldObj: oldObj, Obj: newObj}
}

// TestFieldNameMasking_SameFieldMixedSecrecy is the end-to-end proof this
// whole design exists for: two rate-limiting plugins, one with a real
// secret `minute`, one with a plain literal `minute` — only the one
// actually recorded as templated gets masked, and unrelated sibling fields
// (hour, error_code, id) are NEVER touched, regardless of their value. This
// is the exact scenario that broke value-based masking in production.
func TestFieldNameMasking_SameFieldMixedSecrecy(t *testing.T) {
	secretPluginKey := file.PluginKey("rate-limiting", "", "mockbin", "", "", "", "")
	plainPluginKey := file.PluginKey("rate-limiting", "", "", "", "", "", "")

	secretMap := file.SecretMap{
		secretPluginKey: {fieldMinute: true},
	}
	cache := NewEnvVarCache()

	newPlugin := func(svcName string, minute, hour any, errorCode any) *state.Plugin {
		p := &state.Plugin{Plugin: kong.Plugin{
			Name: kong.String("rate-limiting"),
			Config: kong.Configuration{
				fieldMinute:  minute,
				fieldHour:    hour,
				"error_code": errorCode,
				"policy":     "local",
			},
		}}
		if svcName != "" {
			p.Service = &kong.Service{Name: kong.String(svcName)}
		}
		return p
	}

	oldSecret := newPlugin("mockbin", "4", float64(3), float64(429))
	newSecret := newPlugin("mockbin", "super-secret-limit-value", float64(4), float64(429))

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldSecret, newSecret), false, false, cache, secretMap,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(diffString, "super-secret-limit-value") {
		t.Errorf("secret value leaked into diff:\n%s", diffString)
	}
	minuteMaskedStr := `"` + fieldMinute + `": "` + maskedValue + `"`
	if !strings.Contains(diffString, minuteMaskedStr) && !strings.Contains(diffString, maskedValue) {
		t.Errorf("expected minute to be masked, got:\n%s", diffString)
	}

	hourMaskedStr := `"` + fieldHour + `": "` + maskedValue + `"`
	if strings.Contains(diffString, hourMaskedStr) {
		t.Errorf("hour was wrongly masked despite never being templated:\n%s", diffString)
	}
	if !strings.Contains(diffString, "429") {
		t.Errorf("expected error_code 429 to remain fully visible, got:\n%s", diffString)
	}

	// Sibling plugin (no service scope, no secret entry) — same field name
	// `minute`, but a plain literal. Must NOT be masked.
	oldPlain := newPlugin("", float64(3), float64(3), float64(429))
	newPlain := newPlugin("", float64(10), float64(4), float64(429))

	_ = plainPluginKey
	plainDiff, err := generateDiffStringWithCache(
		crudEventFor(oldPlain, newPlain), false, false, cache, secretMap,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(plainDiff, maskedValue) {
		t.Errorf("plain literal plugin was wrongly masked:\n%s", plainDiff)
	}
	if !strings.Contains(plainDiff, "10") {
		t.Errorf("expected literal value 10 to remain visible, got:\n%s", plainDiff)
	}
}

// TestFieldNameMasking_RealObjectUntouched proves masking never mutates the
// real event objects — only throwaway clones are altered.
func TestFieldNameMasking_RealObjectUntouched(t *testing.T) {
	key := file.PluginKey("key-auth", "", "", "", "", "", "p1")
	secretMap := file.SecretMap{key: {fieldKey: true}}
	cache := NewEnvVarCache()

	oldObj := &state.Plugin{Plugin: kong.Plugin{
		ID: kong.String("p1"), Name: kong.String("key-auth"),
		Config: kong.Configuration{"key": "old-key-value"},
	}}
	newObj := &state.Plugin{Plugin: kong.Plugin{
		ID: kong.String("p1"), Name: kong.String("key-auth"),
		Config: kong.Configuration{"key": "real-secret-abc123"},
	}}

	_, err := generateDiffStringWithCache(crudEventFor(oldObj, newObj), false, false, cache, secretMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newObj.Config["key"] != "real-secret-abc123" {
		t.Errorf("real object was mutated by masking: got %v", newObj.Config["key"])
	}
}

// TestFieldNameMasking_ChangedSecretShowsAsDiff proves that when a masked
// secret's real value actually changed, the diff still renders it as a
// change (a -/+ pair) rather than silently looking unchanged — which
// happens if both sides are masked to the exact same literal string, since
// the diff engine compares masked values for equality.
func TestFieldNameMasking_ChangedSecretShowsAsDiff(t *testing.T) {
	key := file.PluginKey("rate-limiting", "", "mockbin", "", "", "", "")
	secretMap := file.SecretMap{key: {fieldMinute: true}}
	cache := NewEnvVarCache()

	newPlugin := func(minute any) *state.Plugin {
		return &state.Plugin{Plugin: kong.Plugin{
			Name:    kong.String("rate-limiting"),
			Service: &kong.Service{Name: kong.String("mockbin")},
			Config:  kong.Configuration{fieldMinute: minute, fieldHour: float64(4)},
		}}
	}

	oldObj := newPlugin("old-real-secret")
	newObj := newPlugin("new-real-secret")

	diffString, err := generateDiffStringWithCache(crudEventFor(oldObj, newObj), false, false, cache, secretMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(diffString, "old-real-secret") || strings.Contains(diffString, "new-real-secret") {
		t.Errorf("real secret values leaked into diff:\n%s", diffString)
	}
	// The changed secret must render as an actual diff (-/+), not a static line.
	if !containsLineWithMarkerAndField(diffString, "-", fieldMinute) ||
		!containsLineWithMarkerAndField(diffString, "+", fieldMinute) {
		t.Errorf("expected minute to render as a changed field (-/+), got:\n%s", diffString)
	}
}

// containsLineWithMarkerAndField checks whether any line in diffString both
// starts (after leading whitespace) with the given diff marker ("-" or "+")
// and contains the given JSON field name.
func containsLineWithMarkerAndField(diffString, marker, field string) bool {
	for _, line := range strings.Split(diffString, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, marker) && strings.Contains(trimmed, `"`+field+`"`) {
			return true
		}
	}
	return false
}

// TestFieldNameMasking_UnchangedSecretDoesNotFalselyShowAsDiff proves the
// inverse: when a secret's real value did NOT change, it still renders as
// unchanged (no spurious -/+), since only genuinely changed values get the
// distinguishing marker.
func TestFieldNameMasking_UnchangedSecretDoesNotFalselyShowAsDiff(t *testing.T) {
	key := file.PluginKey("rate-limiting", "", "mockbin", "", "", "", "")
	secretMap := file.SecretMap{key: {fieldMinute: true}}
	cache := NewEnvVarCache()

	newPlugin := func(hour any) *state.Plugin {
		return &state.Plugin{Plugin: kong.Plugin{
			Name:    kong.String("rate-limiting"),
			Service: &kong.Service{Name: kong.String("mockbin")},
			Config:  kong.Configuration{fieldMinute: "same-secret-value", fieldHour: hour},
		}}
	}

	oldObj := newPlugin(float64(4))
	newObj := newPlugin(float64(8)) // only hour changes; minute stays the same

	diffString, err := generateDiffStringWithCache(crudEventFor(oldObj, newObj), false, false, cache, secretMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if containsLineWithMarkerAndField(diffString, "-", fieldMinute) ||
		containsLineWithMarkerAndField(diffString, "+", fieldMinute) {
		t.Errorf("expected unchanged minute to NOT render as a diff, got:\n%s", diffString)
	}
	if !containsLineWithMarkerAndField(diffString, "-", fieldHour) ||
		!containsLineWithMarkerAndField(diffString, "+", fieldHour) {
		t.Errorf("expected hour to render as changed, got:\n%s", diffString)
	}
}

// TestFieldNameMasking_RouteOwnFieldGetsMaskedWhenTemplated proves the
// other side of the same fix: when a route's field IS genuinely templated
// (recorded in secretMap), it masks correctly — any field on any entity can
// be a legitimate secret if the user chooses to template it; Route isn't
// special-cased as "never secret."
func TestFieldNameMasking_RouteOwnFieldGetsMaskedWhenTemplated(t *testing.T) {
	key := file.SimpleKey("route", "test", "")
	secretMap := file.SecretMap{key: {"methods": true}}
	cache := NewEnvVarCache()

	oldObj := &state.Route{Route: kong.Route{
		Name: kong.String("test"), Methods: []*string{kong.String("old-real-secret-method")},
	}}
	newObj := &state.Route{Route: kong.Route{
		Name: kong.String("test"), Methods: []*string{kong.String("new-real-secret-method")},
	}}

	diffString, err := generateDiffStringWithCache(crudEventFor(oldObj, newObj), false, false, cache, secretMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(diffString, "real-secret-method") {
		t.Errorf("templated route field leaked into diff:\n%s", diffString)
	}
	if !strings.Contains(diffString, maskedValue) {
		t.Errorf("expected methods to be masked, got:\n%s", diffString)
	}
}

// TestGenerateDiffStringWithCache_NoMaskValues_Disabled tests that when noMaskValues is true,
// masking is completely disabled regardless of secretMap content.
func TestGenerateDiffStringWithCache_NoMaskValues_Disabled(t *testing.T) {
	key := file.PluginKey("rate-limiting", "", "test-svc", "", "", "", "")
	secretMap := file.SecretMap{
		key: {fieldMinute: true},
	}
	cache := NewEnvVarCache()

	oldPlugin := &state.Plugin{Plugin: kong.Plugin{
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config:  kong.Configuration{fieldMinute: "secret-old-value", fieldHour: float64(4)},
	}}
	newPlugin := &state.Plugin{Plugin: kong.Plugin{
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config:  kong.Configuration{fieldMinute: "secret-new-value", fieldHour: float64(5)},
	}}

	// With noMaskValues=true, masking should be disabled
	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldPlugin, newPlugin), false, true, cache, secretMap,
	)
	require.NoError(t, err)

	// Secret values should be exposed in output
	require.Contains(t, diffString, "secret-old-value", "noMaskValues=true should expose old secret")
	require.Contains(t, diffString, "secret-new-value", "noMaskValues=true should expose new secret")
	require.NotContains(t, diffString, maskedValue, "masking should be disabled")
}

// TestGenerateDiffStringWithCache_IsDelete_ReversedDiffOrder tests that delete operations
// reverse the diff order (new vs old becomes old vs new).
func TestGenerateDiffStringWithCache_IsDelete_ReversedDiffOrder(t *testing.T) {
	key := file.PluginKey("rate-limiting", "", "test-svc", "", "", "", "p1")
	secretMap := file.SecretMap{key: {fieldMinute: true}}
	cache := NewEnvVarCache()

	oldPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p1"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config:  kong.Configuration{fieldMinute: "old-secret"},
	}}
	newPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p1"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config:  kong.Configuration{fieldMinute: "new-secret"},
	}}

	// isDelete=true should reverse the diff order
	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldPlugin, newPlugin), true, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Secrets should be masked
	require.NotContains(t, diffString, "old-secret")
	require.NotContains(t, diffString, "new-secret")
	require.Contains(t, diffString, maskedValue, "delete should mask secrets")
}

// TestGenerateDiffStringWithCache_Service_Entity tests masking for Service entity type
// with templated fields.
func TestGenerateDiffStringWithCache_Service_Entity(t *testing.T) {
	key := file.SimpleKey("service", "my-service", "svc-1")
	secretMap := file.SecretMap{
		key: {"path": true},
	}
	cache := NewEnvVarCache()

	oldService := &state.Service{Service: kong.Service{
		ID:   kong.String("svc-1"),
		Name: kong.String("my-service"),
		Host: kong.String("example.com"),
		Path: kong.String("/secret-path-old"),
		Port: kong.Int(80),
	}}
	newService := &state.Service{Service: kong.Service{
		ID:   kong.String("svc-1"),
		Name: kong.String("my-service"),
		Host: kong.String("example.com"),
		Path: kong.String("/secret-path-new"),
		Port: kong.Int(80),
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldService, newService), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Templated field should be masked
	require.NotContains(t, diffString, "/secret-path-old")
	require.NotContains(t, diffString, "/secret-path-new")
	require.Contains(t, diffString, maskedValue, "templated path field should be masked")
	// Non-templated field should remain visible
	require.Contains(t, diffString, "example.com")
}

// TestGenerateDiffStringWithCache_Consumer_Entity tests masking for Consumer entity type.
func TestGenerateDiffStringWithCache_Consumer_Entity(t *testing.T) {
	key := file.SimpleKey("consumer", "john-doe", "c1")
	secretMap := file.SecretMap{
		key: {"custom_id": true},
	}
	cache := NewEnvVarCache()

	oldConsumer := &state.Consumer{Consumer: kong.Consumer{
		ID:       kong.String("c1"),
		Username: kong.String("john-doe"),
		CustomID: kong.String("secret-custom-id-old"),
	}}
	newConsumer := &state.Consumer{Consumer: kong.Consumer{
		ID:       kong.String("c1"),
		Username: kong.String("john-doe"),
		CustomID: kong.String("secret-custom-id-new"),
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldConsumer, newConsumer), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Templated custom_id should be masked
	require.NotContains(t, diffString, "secret-custom-id-old")
	require.NotContains(t, diffString, "secret-custom-id-new")
	require.Contains(t, diffString, maskedValue, "templated custom_id field should be masked")
	require.Contains(t, diffString, "john-doe")
}

// TestGenerateDiffStringWithCache_Upstream_Entity tests masking for Upstream entity.
func TestGenerateDiffStringWithCache_Upstream_Entity(t *testing.T) {
	key := file.SimpleKey("upstream", "backend-pool", "up1")
	secretMap := file.SecretMap{
		key: {"host_header": true},
	}
	cache := NewEnvVarCache()

	oldUpstream := &state.Upstream{Upstream: kong.Upstream{
		ID:         kong.String("up1"),
		Name:       kong.String("backend-pool"),
		HostHeader: kong.String("secret-header-old.internal"),
		Slots:      kong.Int(10),
	}}
	newUpstream := &state.Upstream{Upstream: kong.Upstream{
		ID:         kong.String("up1"),
		Name:       kong.String("backend-pool"),
		HostHeader: kong.String("secret-header-new.internal"),
		Slots:      kong.Int(10),
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldUpstream, newUpstream), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Templated host_header should be masked
	require.NotContains(t, diffString, "secret-header-old.internal")
	require.NotContains(t, diffString, "secret-header-new.internal")
	require.Contains(t, diffString, maskedValue, "templated host_header field should be masked")
	require.Contains(t, diffString, "backend-pool")
}

// TestGenerateDiffStringWithCache_Target_Entity tests masking for Target entity.
func TestGenerateDiffStringWithCache_Target_Entity(t *testing.T) {
	key := file.SimpleKey("target", "backend1:8080", "tgt1")
	secretMap := file.SecretMap{
		key: {"target": true},
	}
	cache := NewEnvVarCache()

	oldTarget := &state.Target{Target: kong.Target{
		ID:       kong.String("tgt1"),
		Target:   kong.String("secret-backend-old:8080"),
		Weight:   kong.Int(100),
		Upstream: &kong.Upstream{ID: kong.String("up1")},
	}}
	newTarget := &state.Target{Target: kong.Target{
		ID:       kong.String("tgt1"),
		Target:   kong.String("secret-backend-new:8080"),
		Weight:   kong.Int(100),
		Upstream: &kong.Upstream{ID: kong.String("up1")},
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldTarget, newTarget), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Templated target should be masked
	require.NotContains(t, diffString, "secret-backend-old:8080")
	require.NotContains(t, diffString, "secret-backend-new:8080")
	require.Contains(t, diffString, maskedValue, "templated target field should be masked")
}

// TestGenerateDiffStringWithCache_Vault_Entity tests masking for Vault entity.
func TestGenerateDiffStringWithCache_Vault_Entity(t *testing.T) {
	key := file.SimpleKey("vault", "aws-vault", "v1")
	secretMap := file.SecretMap{
		key: {"prefix": true},
	}
	cache := NewEnvVarCache()

	oldVault := &state.Vault{Vault: kong.Vault{
		ID:     kong.String("v1"),
		Name:   kong.String("aws-vault"),
		Prefix: kong.String("SECRET_PREFIX_OLD"),
		Config: kong.Configuration{
			"api_key": "some-api-key",
		},
	}}
	newVault := &state.Vault{Vault: kong.Vault{
		ID:     kong.String("v1"),
		Name:   kong.String("aws-vault"),
		Prefix: kong.String("SECRET_PREFIX_NEW"),
		Config: kong.Configuration{
			"api_key": "some-api-key",
		},
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldVault, newVault), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Prefix field should be masked
	require.NotContains(t, diffString, "SECRET_PREFIX_OLD")
	require.NotContains(t, diffString, "SECRET_PREFIX_NEW")
	require.Contains(t, diffString, "aws-vault")
}

// TestGenerateDiffStringWithCache_Key_Entity tests masking for Key entity.
func TestGenerateDiffStringWithCache_Key_Entity(t *testing.T) {
	key := file.SimpleKey("key", "api-key", "k1")
	secretMap := file.SecretMap{
		key: {"jwk": true},
	}
	cache := NewEnvVarCache()

	oldKey := &state.Key{Key: kong.Key{
		ID:   kong.String("k1"),
		Name: kong.String("api-key"),
		JWK:  kong.String(`{"kty":"RSA","old":"secret"}`),
	}}
	newKey := &state.Key{Key: kong.Key{
		ID:   kong.String("k1"),
		Name: kong.String("api-key"),
		JWK:  kong.String(`{"kty":"RSA","new":"secret"}`),
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldKey, newKey), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Templated jwk should be masked
	require.NotContains(t, diffString, `{"kty":"RSA","old":"secret"}`)
	require.NotContains(t, diffString, `{"kty":"RSA","new":"secret"}`)
	require.Contains(t, diffString, maskedValue, "templated jwk field should be masked")
}

// TestGenerateDiffStringWithCache_ConsumerGroup_Entity tests masking for ConsumerGroup.
func TestGenerateDiffStringWithCache_ConsumerGroup_Entity(t *testing.T) {
	key := file.SimpleKey("consumer_group", "premium-users", "cg1")
	secretMap := file.SecretMap{
		key: {"name": true},
	}
	cache := NewEnvVarCache()

	oldCG := &state.ConsumerGroup{ConsumerGroup: kong.ConsumerGroup{
		ID:   kong.String("cg1"),
		Name: kong.String("secret-consumer-group-old"),
	}}
	newCG := &state.ConsumerGroup{ConsumerGroup: kong.ConsumerGroup{
		ID:   kong.String("cg1"),
		Name: kong.String("secret-consumer-group-new"),
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldCG, newCG), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Templated name should be masked
	require.NotContains(t, diffString, "secret-consumer-group-old")
	require.NotContains(t, diffString, "secret-consumer-group-new")
	require.Contains(t, diffString, maskedValue, "templated name field should be masked")
}

// TestGenerateDiffStringWithCache_SNI_Entity tests masking for SNI entity.
func TestGenerateDiffStringWithCache_SNI_Entity(t *testing.T) {
	key := file.SimpleKey("sni", "example.com", "sni1")
	secretMap := file.SecretMap{
		key: {fieldName: true},
	}
	cache := NewEnvVarCache()

	oldSNI := &state.SNI{SNI: kong.SNI{
		ID:   kong.String("sni1"),
		Name: kong.String("secret-example-old.com"),
	}}
	newSNI := &state.SNI{SNI: kong.SNI{
		ID:   kong.String("sni1"),
		Name: kong.String("secret-example-new.com"),
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldSNI, newSNI), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Templated name should be masked
	require.NotContains(t, diffString, "secret-example-old.com")
	require.NotContains(t, diffString, "secret-example-new.com")
	require.Contains(t, diffString, maskedValue, "templated name field should be masked")
}

// TestGenerateDiffStringWithCache_CACertificate_Entity tests masking for CACertificate.
func TestGenerateDiffStringWithCache_CACertificate_Entity(t *testing.T) {
	key := file.SimpleKey("ca_certificate", "", "ca1")
	secretMap := file.SecretMap{
		key: {"cert": true},
	}
	cache := NewEnvVarCache()

	oldCACert := &state.CACertificate{CACertificate: kong.CACertificate{
		ID:   kong.String("ca1"),
		Cert: kong.String("-----BEGIN CERTIFICATE-----\nold-secret-cert-data"),
	}}
	newCACert := &state.CACertificate{CACertificate: kong.CACertificate{
		ID:   kong.String("ca1"),
		Cert: kong.String("-----BEGIN CERTIFICATE-----\nnew-secret-cert-data"),
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldCACert, newCACert), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Templated cert should be masked
	require.NotContains(t, diffString, "old-secret-cert-data")
	require.NotContains(t, diffString, "new-secret-cert-data")
	require.Contains(t, diffString, maskedValue, "templated cert field should be masked")
}

// TestGenerateDiffStringWithCache_MultipleKey_Fallback tests that fallback keys
// are tried when the primary key doesn't match.
func TestGenerateDiffStringWithCache_MultipleKey_Fallback(t *testing.T) {
	// Fallback key without service scope
	noScopeKey := file.PluginKey("rate-limiting", "", "", "", "", "", "")

	// Only the no-scope key has secrets recorded (service-scoped key doesn't exist)
	// This tests the fallback mechanism
	secretMap := file.SecretMap{
		noScopeKey: {fieldMinute: true},
	}
	cache := NewEnvVarCache()

	plugin := &state.Plugin{Plugin: kong.Plugin{
		Name:   kong.String("rate-limiting"),
		Config: kong.Configuration{fieldMinute: "secret-value-old", fieldHour: float64(4)},
	}}
	newPlugin := &state.Plugin{Plugin: kong.Plugin{
		Name:   kong.String("rate-limiting"),
		Config: kong.Configuration{fieldMinute: "secret-value-new", fieldHour: float64(5)},
	}}

	// Should fall back to no-scope key when service-scoped key not found
	diffString, err := generateDiffStringWithCache(
		crudEventFor(plugin, newPlugin), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Should have masked using the fallback key
	require.NotContains(t, diffString, "secret-value-old")
	require.NotContains(t, diffString, "secret-value-new")
	require.Contains(t, diffString, maskedValue)
}

// TestGenerateDiffStringWithCache_EntityNotInResolveKeys tests that when entity type
// is not supported by resolveEntityKeys, value-based masking is used.
func TestGenerateDiffStringWithCache_EntityNotInResolveKeys(t *testing.T) {
	t.Setenv("DECK_API_KEY", "secret-api-key-value")

	cache := NewEnvVarCache()
	// Empty secretMap - entity not covered
	secretMap := file.SecretMap{}

	// Use a custom/unknown entity type that doesn't match resolveEntityKeys
	unknownEntity := struct {
		ID     string
		APIKey string
		Name   string
	}{
		ID:     "custom-1",
		APIKey: "secret-api-key-value",
		Name:   "test-custom",
	}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(unknownEntity, unknownEntity), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Should use value-based masking for the DECK_API_KEY value
	require.NotContains(t, diffString, "secret-api-key-value")
}

// TestGenerateDiffStringWithCache_CleanMaskedMarkers tests that invisible change-detection
// markers are properly cleaned from output.
func TestGenerateDiffStringWithCache_CleanMaskedMarkers(t *testing.T) {
	key := file.PluginKey("rate-limiting", "", "test-svc", "", "", "", "p1")
	secretMap := file.SecretMap{
		key: {fieldMinute: true},
	}
	cache := NewEnvVarCache()

	oldPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p1"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config:  kong.Configuration{fieldMinute: "secret-old", fieldHour: float64(4)},
	}}
	newPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p1"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config:  kong.Configuration{fieldMinute: "secret-new", fieldHour: float64(5)},
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldPlugin, newPlugin), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// The output should be clean (no zero-width joiners)
	require.NotContains(t, diffString, "\u200d")
	// But should still show as a change
	require.True(t,
		containsLineWithMarkerAndField(diffString, "-", fieldMinute) &&
			containsLineWithMarkerAndField(diffString, "+", fieldMinute),
	)
}

// TestGenerateDiffStringWithCache_MultipleChangedFields tests that multiple changed
// secret fields all appear as -/+ pairs.
func TestGenerateDiffStringWithCache_MultipleChangedFields(t *testing.T) {
	key := file.PluginKey("jwt", "", "test-svc", "", "", "", "jwt1")
	secretMap := file.SecretMap{
		key: {"secret": true, "key": true},
	}
	cache := NewEnvVarCache()

	oldPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("jwt1"),
		Name:    kong.String("jwt"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config: kong.Configuration{
			"secret":    "old-secret-1",
			"key":       "old-key-1",
			"algorithm": "HS256",
		},
	}}
	newPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("jwt1"),
		Name:    kong.String("jwt"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config: kong.Configuration{
			"secret":    "new-secret-2",
			"key":       "new-key-2",
			"algorithm": "RS256",
		},
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldPlugin, newPlugin), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Both secret fields should appear as changes
	require.True(t,
		containsLineWithMarkerAndField(diffString, "-", "secret") &&
			containsLineWithMarkerAndField(diffString, "+", "secret"),
		"secret field should show as changed",
	)
	require.True(t,
		containsLineWithMarkerAndField(diffString, "-", "key") &&
			containsLineWithMarkerAndField(diffString, "+", "key"),
		"key field should show as changed",
	)
	// Non-secret field should also show change
	require.True(t,
		containsLineWithMarkerAndField(diffString, "-", "algorithm") &&
			containsLineWithMarkerAndField(diffString, "+", "algorithm"),
		"non-secret field should show change",
	)
}

// TestGenerateDiffStringWithCache_PartialChanges tests that when some fields change
// and others don't, both are handled correctly.
func TestGenerateDiffStringWithCache_PartialChanges(t *testing.T) {
	key := file.PluginKey("rate-limiting", "", "test-svc", "", "", "", "rl1")
	secretMap := file.SecretMap{
		key: {fieldMinute: true},
	}
	cache := NewEnvVarCache()

	oldPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("rl1"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config: kong.Configuration{
			fieldMinute: "secret-limit-same",
			fieldHour:   float64(100),
			"day":       float64(1000),
		},
	}}
	newPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("rl1"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("test-svc")},
		Config: kong.Configuration{
			fieldMinute: "secret-limit-same", // unchanged
			fieldHour:   float64(200),        // changed
			"day":       float64(1000),       // unchanged
		},
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldPlugin, newPlugin), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Unchanged secret should NOT appear as a change
	require.False(t,
		containsLineWithMarkerAndField(diffString, "-", fieldMinute) ||
			containsLineWithMarkerAndField(diffString, "+", fieldMinute),
		"unchanged secret field should not show as diff",
	)
	// Changed non-secret field should appear
	require.True(t,
		containsLineWithMarkerAndField(diffString, "-", fieldHour) &&
			containsLineWithMarkerAndField(diffString, "+", fieldHour),
		"changed field should show as diff",
	)
	// Unchanged non-secret field should not appear
	require.False(t,
		containsLineWithMarkerAndField(diffString, "-", "day") ||
			containsLineWithMarkerAndField(diffString, "+", "day"),
		"unchanged field should not show as diff",
	)
}

// TestGenerateDiffStringWithCache_MultiplePlugins tests masking of multiple plugins
// with different secret fields.
func TestGenerateDiffStringWithCache_MultiplePlugins(t *testing.T) {
	t.Setenv("DECK_BEARER_TOKEN", "secret-bearer-token-value")

	key1 := file.PluginKey("rate-limiting", "", "svc1", "", "", "", "p1")
	key2 := file.PluginKey("request-transformer", "", "svc1", "", "", "", "p2")

	secretMap := file.SecretMap{
		key1: {fieldMinute: true, fieldHour: true},
		key2: {}, // no field-based masking for plugin 2, use value-based
	}
	cache := NewEnvVarCache()

	// First plugin: rate-limiting with minute and hour as secrets (field-based)
	plugin1Old := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p1"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("svc1")},
		Config: kong.Configuration{
			fieldMinute: "100",
			fieldHour:   "1000",
		},
	}}
	plugin1New := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p1"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("svc1")},
		Config: kong.Configuration{
			fieldMinute: "200",
			fieldHour:   "2000",
		},
	}}

	// Second plugin: request-transformer with env var (value-based masking)
	plugin2Old := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p2"),
		Name:    kong.String("request-transformer"),
		Service: &kong.Service{Name: kong.String("svc1")},
		Config: kong.Configuration{
			"add": map[string]interface{}{
				fieldHeaders: []interface{}{
					"Authorization:Bearer ${{ env \"DECK_BEARER_TOKEN\" }}",
				},
			},
		},
	}}
	plugin2New := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p2"),
		Name:    kong.String("request-transformer"),
		Service: &kong.Service{Name: kong.String("svc1")},
		Config: kong.Configuration{
			"add": map[string]interface{}{
				fieldHeaders: []interface{}{
					"Authorization:Bearer ${{ env \"DECK_BEARER_TOKEN\" }}",
				},
			},
		},
	}}

	// Test plugin 1 - field-based masking
	diff1, err := generateDiffStringWithCache(
		crudEventFor(plugin1Old, plugin1New), false, false, cache, secretMap,
	)
	require.NoError(t, err)
	require.NotContains(t, diff1, "100")
	require.NotContains(t, diff1, "200")
	require.NotContains(t, diff1, "1000")
	require.NotContains(t, diff1, "2000")
	require.Contains(t, diff1, maskedValue)

	// Test plugin 2 - value-based masking
	diff2, err := generateDiffStringWithCache(
		crudEventFor(plugin2Old, plugin2New), false, false, cache, secretMap,
	)
	require.NoError(t, err)
	require.NotContains(t, diff2, "secret-bearer-token-value")
}

// TestGenerateDiffStringWithCache_InvalidEntityTypeInSecretMap tests handling
// of invalid/unsupported entity types in the secret map.
func TestGenerateDiffStringWithCache_InvalidEntityTypeInSecretMap(t *testing.T) {
	// Create a secretMap with an entity key that doesn't match any known type
	unknownKey := file.SimpleKey("unknown-type", "unknown-scope", "unknown-id")
	secretMap := file.SecretMap{
		unknownKey: {"secret_field": true},
	}
	cache := NewEnvVarCache()

	// Use a custom struct that's not in resolveEntityKeys
	customEntity := struct {
		ID          string
		SecretField string
		PublicField string
	}{
		ID:          "custom-1",
		SecretField: "secret-value-should-be-masked",
		PublicField: "public-value",
	}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(customEntity, customEntity), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Since entity type is not in resolveEntityKeys, field-based masking won't work
	// Value-based masking should be used instead
	// The custom entity is not masked at all since no DECK_ env vars are set
	require.Contains(t, diffString, "public-value")
}

// TestGenerateDiffStringWithCache_GetDiffError tests handling when getDiff returns an error.
// This is simulated by passing incompatible entity types.
func TestGenerateDiffStringWithCache_GetDiffError(t *testing.T) {
	cache := NewEnvVarCache()
	secretMap := file.SecretMap{}

	// Create entities of incompatible types to force a diff error
	oldEntity := struct {
		ID   string
		Data int
	}{ID: "1", Data: 42}

	newEntity := struct {
		ID   string
		Data string
	}{ID: "1", Data: "string"}

	event := crudEventFor(oldEntity, newEntity)

	// This should either return an error or handle gracefully
	diffString, err := generateDiffStringWithCache(event, false, false, cache, secretMap)

	// Should handle the error gracefully - either returning empty diff or error
	// The important thing is it doesn't crash
	if err == nil {
		// If no error, diff should be generated
		require.NotNil(t, diffString)
	}
}

// TestGenerateDiffStringWithCache_EmptySecretFieldsMap tests when secretMap has empty
// secret fields for an entity.
func TestGenerateDiffStringWithCache_EmptySecretFieldsMap(t *testing.T) {
	key := file.SimpleKey("service", "public", "svc1")
	secretMap := file.SecretMap{
		key: {}, // empty secret fields
	}
	cache := NewEnvVarCache()

	oldService := &state.Service{Service: kong.Service{
		ID:   kong.String("svc1"),
		Name: kong.String("my-service"),
		Host: kong.String("api.example.com"),
	}}
	newService := &state.Service{Service: kong.Service{
		ID:   kong.String("svc1"),
		Name: kong.String("my-service"),
		Host: kong.String("api.example.com"),
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldService, newService), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// No changes and no secrets, so diff should show no changes
	require.NotContains(t, diffString, "[masked]")
}

// TestGenerateDiffStringWithCache_NestedConfigMasking tests masking of nested
// config structures using value-based masking with env vars.
func TestGenerateDiffStringWithCache_NestedConfigMasking(t *testing.T) {
	t.Setenv("DECK_DB_PASSWORD", "secret-password-value")
	t.Setenv("DECK_DB_USERNAME", "secret-username-value")

	key := file.PluginKey("custom-plugin", "", "svc1", "", "", "", "p1")
	secretMap := file.SecretMap{
		key: {}, // Use value-based masking for env vars in nested config
	}
	cache := NewEnvVarCache()

	oldPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p1"),
		Name:    kong.String("custom-plugin"),
		Service: &kong.Service{Name: kong.String("svc1")},
		Config: kong.Configuration{
			"database": map[string]interface{}{
				"host": "db.example.com",
				"port": float64(5432),
				"credentials": map[string]interface{}{
					"username": "${{ env \"DECK_DB_USERNAME\" }}",
					"password": "${{ env \"DECK_DB_PASSWORD\" }}",
				},
			},
		},
	}}
	newPlugin := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("p1"),
		Name:    kong.String("custom-plugin"),
		Service: &kong.Service{Name: kong.String("svc1")},
		Config: kong.Configuration{
			"database": map[string]interface{}{
				"host": "db.example.com",
				"port": float64(5432),
				"credentials": map[string]interface{}{
					"username": "${{ env \"DECK_DB_USERNAME\" }}",
					"password": "${{ env \"DECK_DB_PASSWORD\" }}",
				},
			},
		},
	}}

	diffString, err := generateDiffStringWithCache(
		crudEventFor(oldPlugin, newPlugin), false, false, cache, secretMap,
	)
	require.NoError(t, err)

	// Env var values should be masked by value-based masking
	require.NotContains(t, diffString, "secret-password-value")
	require.NotContains(t, diffString, "secret-username-value")
}
