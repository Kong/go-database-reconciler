package diff

import (
	"testing"

	"github.com/kong/go-database-reconciler/pkg/file"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// lastKey returns the fallback (non-id) candidate — what extraction would
// have computed for a plugin declared in the file with no explicit id.
func lastKey(t *testing.T, keys []file.EntityKey, ok bool) file.EntityKey {
	t.Helper()
	if !ok || len(keys) == 0 {
		t.Fatal("expected at least one candidate key")
	}
	return keys[len(keys)-1]
}

// The three rate-limiting plugins from the canonical example must resolve to
// three DISTINCT scope-based keys, purely from name + parent scope — no ids
// needed. Critically, these plugins ALSO carry a real ID here (as they would
// once reconciled against an existing Kong entity) to prove the fallback
// candidate still matches what extraction computes from a file with no
// declared id — this is the exact bug that broke production masking.
func TestResolveEntityKeys_PluginsDistinctByScope(t *testing.T) {
	svcScoped := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("11111111-1111-1111-1111-111111111111"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("mockbin")},
	}}
	// A real route-scoped plugin carries only its Route reference, never
	// also Service — confirmed empirically against a live Kong instance.
	routeScoped := &state.Plugin{Plugin: kong.Plugin{
		ID:    kong.String("22222222-2222-2222-2222-222222222222"),
		Name:  kong.String("rate-limiting"),
		Route: &kong.Route{Name: kong.String("test")},
	}}
	global := &state.Plugin{Plugin: kong.Plugin{
		ID:   kong.String("33333333-3333-3333-3333-333333333333"),
		Name: kong.String("rate-limiting"),
	}}

	seen := map[string]bool{}
	for _, p := range []*state.Plugin{svcScoped, routeScoped, global} {
		keys, ok := resolveEntityKeys(p)
		fallback := lastKey(t, keys, ok)
		if seen[fallback.String()] {
			t.Errorf("duplicate fallback key for a distinct plugin: %q", fallback.String())
		}
		seen[fallback.String()] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 distinct fallback plugin keys, got %d", len(seen))
	}
}

// This is the exact production bug: a plugin already exists in Kong (so it
// carries a real, reconciler-assigned ID at diff time) but the SOURCE FILE
// never declared an id for it. Extraction (reading the raw file) can only
// ever compute the scope-based key. resolveEntityKeys must offer that same
// scope-based key as a fallback candidate, not just the id-based one.
func TestResolveEntityKeys_ExistingPluginWithoutFileDeclaredID(t *testing.T) {
	// Simulates the reconciled object: has a real ID (matched against Kong),
	// but that ID was never in the file — extraction has no way to know it.
	reconciled := &state.Plugin{Plugin: kong.Plugin{
		ID:      kong.String("5421eb40-5ca7-45fc-a116-e4e9960865e7"),
		Name:    kong.String("rate-limiting"),
		Service: &kong.Service{Name: kong.String("mockbin")},
	}}

	// What extraction would have computed from the raw file (no id declared).
	extractionKey := file.PluginKey("rate-limiting", "", "mockbin", "", "", "", "")

	keys, ok := resolveEntityKeys(reconciled)
	if !ok {
		t.Fatal("plugin should resolve candidate keys")
	}

	found := false
	for _, k := range keys {
		if k.String() == extractionKey.String() {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("extraction's scope-based key %q was not among the candidates: %+v", extractionKey.String(), keys)
	}
}

// If the file DOES declare an explicit id, and the entity is matched
// against that exact id, the id-based candidate (tried first) must match
// directly — no scope information required.
func TestResolveEntityKeys_ExplicitFileDeclaredID(t *testing.T) {
	reconciled := &state.Plugin{Plugin: kong.Plugin{
		ID:   kong.String("declared-in-file-id"),
		Name: kong.String("rate-limiting"),
	}}
	extractionKey := file.PluginKey("rate-limiting", "", "", "", "", "", "declared-in-file-id")

	keys, ok := resolveEntityKeys(reconciled)
	if !ok || keys[0].String() != extractionKey.String() {
		t.Errorf("expected id-based key to be the first candidate and match extraction, got: %+v", keys)
	}
}

func TestResolveEntityKeys_Certificate(t *testing.T) {
	c := &state.Certificate{Certificate: kong.Certificate{ID: kong.String("cert-1")}}
	keys, ok := resolveEntityKeys(c)
	if !ok || keys[0].Kind != "certificate" || keys[0].String() != file.SimpleKey("certificate", "", "cert-1").String() {
		t.Errorf("unexpected certificate keys: %+v (ok=%v)", keys, ok)
	}
}

// A certificate with no file-declared id (common — certs are often created
// without one) must still resolve to the shared type-level fallback key
// used by extraction, so cert/key masking still applies.
func TestResolveEntityKeys_CertificateWithoutFileDeclaredID(t *testing.T) {
	reconciled := &state.Certificate{Certificate: kong.Certificate{ID: kong.String("auto-assigned-id")}}
	extractionKey := file.SimpleKey("certificate", "", "")

	keys, ok := resolveEntityKeys(reconciled)
	if !ok {
		t.Fatal("certificate should resolve candidate keys")
	}
	if keys[len(keys)-1].String() != extractionKey.String() {
		t.Errorf("expected shared fallback key to match extraction, got: %+v", keys)
	}
}

func TestResolveEntityKeys_Key(t *testing.T) {
	key := &state.Key{Key: kong.Key{ID: kong.String("k-1")}}
	keys, ok := resolveEntityKeys(key)
	if !ok || keys[0].Kind != fieldKey || keys[0].String() != file.SimpleKey(fieldKey, "", "k-1").String() {
		t.Errorf("unexpected key: %+v (ok=%v)", keys, ok)
	}
}

// Two basic-auth credentials with the same username but on DIFFERENT
// consumers must resolve to distinct fallback keys (parent-scoped identity).
func TestResolveEntityKeys_CredentialScopedToConsumer(t *testing.T) {
	c1 := &state.BasicAuth{BasicAuth: kong.BasicAuth{
		ID:       kong.String("id-1"), // simulates a reconciled, already-existing credential
		Username: kong.String("admin"),
		Consumer: &kong.Consumer{Username: kong.String("alice")},
	}}
	c2 := &state.BasicAuth{BasicAuth: kong.BasicAuth{
		ID:       kong.String("id-2"),
		Username: kong.String("admin"),
		Consumer: &kong.Consumer{Username: kong.String("bob")},
	}}
	k1, ok1 := resolveEntityKeys(c1)
	k2, ok2 := resolveEntityKeys(c2)
	if !ok1 || !ok2 {
		t.Fatal("credentials should resolve keys")
	}
	if lastKey(t, k1, ok1).String() == lastKey(t, k2, ok2).String() {
		t.Errorf("same username on different consumers should be distinct")
	}
}

func TestResolveEntityKeys_UnknownEntityReturnsFalse(t *testing.T) {
	// Not a recognized entity type at all — no coverage exists for it.
	if _, ok := resolveEntityKeys("not an entity"); ok {
		t.Errorf("non-entity should return ok=false")
	}
}

// Service (and similarly Route, Upstream, Target, Consumer, ConsumerGroup,
// SNI, CACertificate, FilterChain) have no legitimate secret fields in
// Kong's schema, but MUST still return ok=true here — otherwise they'd fall
// through to the value-based fallback, which scans every string field for a
// coincidental match against ANY DECK_-prefixed env var value, however
// unrelated. This is a regression test for exactly that: a route's
// `methods` field getting masked because its value happened to equal an
// unrelated DECK_HOST env var.
func TestResolveEntityKeys_NoSecretEntityTypesAreCovered(t *testing.T) {
	entities := []any{
		&state.Service{Service: kong.Service{Name: kong.String("svc")}},
		&state.Route{Route: kong.Route{Name: kong.String("route")}},
		&state.Upstream{Upstream: kong.Upstream{Name: kong.String("up")}},
		&state.Target{Target: kong.Target{Target: kong.String("1.2.3.4:80")}},
		&state.Consumer{Consumer: kong.Consumer{Username: kong.String("alice")}},
		&state.ConsumerGroup{ConsumerGroup: kong.ConsumerGroup{Name: kong.String("cg")}},
		&state.SNI{SNI: kong.SNI{Name: kong.String("sni")}},
		&state.CACertificate{CACertificate: kong.CACertificate{ID: kong.String("ca-1")}},
		&state.FilterChain{FilterChain: kong.FilterChain{Name: kong.String("fc")}},
		&state.Vault{Vault: kong.Vault{Name: kong.String("vault")}},
		&state.License{License: kong.License{ID: kong.String("id")}},
	}
	for _, e := range entities {
		if _, ok := resolveEntityKeys(e); !ok {
			t.Errorf("%T should return ok=true so it never hits the value-based fallback", e)
		}
	}
}
