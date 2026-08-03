package file

import "strings"

// EntityKey is a stable, parse-independent identity for a single entity
// instance. It is intentionally minimal: a Kind plus a canonical Name.
// For entities whose identity involves parent scope (plugins, credentials),
// that scope is folded into Name by the constructor helpers below — so the
// same key can be produced from a parsed state file and from a reconciled
// state.* object, and both are guaranteed to agree because they call the
// same constructor.
type EntityKey struct {
	Kind string // "plugin", "certificate", "key", "keyauth_credential", ...
	Name string // canonical identity (parent scope already folded in)
}

func (k EntityKey) String() string {
	return k.Kind + "\x00" + k.Name
}

const keySep = "\x1f" // unit separator

// PluginKey builds a plugin's identity from its type name, instance name, and
// parent scope (all by name). An explicit id wins outright. This mirrors the
// reconciler's plugin "fields" composite index (name + service + route +
// consumer + consumer-group), keyed on names for cross-parse stability.
func PluginKey(name, instanceName, service, route, consumer, consumerGroup, explicitID string) EntityKey {
	if explicitID != "" {
		return EntityKey{Kind: "plugin", Name: "id:" + explicitID}
	}
	return EntityKey{Kind: "plugin", Name: strings.Join(
		[]string{name, instanceName, service, route, consumer, consumerGroup}, keySep,
	)}
}

// CredentialKey builds a consumer-credential identity: its own key
// (username/key/client_id) scoped to its parent consumer.
func CredentialKey(kind, ownKey, consumer, explicitID string) EntityKey {
	if explicitID != "" {
		return EntityKey{Kind: kind, Name: "id:" + explicitID}
	}
	return EntityKey{Kind: kind, Name: consumer + keySep + ownKey}
}

// SimpleKey builds an identity for a top-level entity that needs no parent
// scope (certificate, key, vault, ...): its own natural name, or explicit id.
func SimpleKey(kind, name, explicitID string) EntityKey {
	if explicitID != "" {
		return EntityKey{Kind: kind, Name: "id:" + explicitID}
	}
	return EntityKey{Kind: kind, Name: name}
}

// deref safely dereferences a *string, returning "" for nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
