package diff

import (
	"github.com/kong/go-database-reconciler/pkg/file"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// resolveEntityKeys returns candidate identities for a reconciled entity
// object (as found on crud.Event.Obj), in priority order, using the same
// file.*Key constructors the file-side extraction uses.
//
// Why a list, not a single key: by the time an entity reaches diff time it
// ALWAYS has an ID — either declared in the source file, matched against
// the existing live entity, or freshly generated for a new one. Extraction
// (which reads the raw file) only sees an ID when the file itself declares
// one. So a diff-time object's ID is not a reliable signal of "the file
// declared this" on its own. Trying the ID-based key first handles the case
// where the file DID declare an id (both sides then agree on that exact
// value); falling back to the natural scope/name-based key handles the far
// more common case where it didn't (the ID present at diff time is one the
// builder assigned, which extraction could never have predicted).
//
// Returns false for entity types that don't need per-instance keying.
func resolveEntityKeys(obj any) ([]file.EntityKey, bool) {
	switch e := obj.(type) {
	case *state.Plugin:
		var service, route, consumer, consumerGroup string
		if e.Service != nil {
			service = derefStr(e.Service.Name)
		}
		if e.Route != nil {
			route = derefStr(e.Route.Name)
		}
		if e.Consumer != nil {
			consumer = derefStr(e.Consumer.Username)
		}
		if e.ConsumerGroup != nil {
			consumerGroup = derefStr(e.ConsumerGroup.Name)
		}
		name, instanceName := derefStr(e.Name), derefStr(e.InstanceName)

		var keys []file.EntityKey
		if id := derefStr(e.ID); id != "" {
			keys = append(keys, file.PluginKey(name, instanceName, service, route, consumer, consumerGroup, id))
		}
		keys = append(keys, file.PluginKey(name, instanceName, service, route, consumer, consumerGroup, ""))
		// Fallback: try without route scope. If the route name was templated in
		// the state file, the secretMap won't have an entry with the resolved
		// route name — only with the templated string or without route scope.
		if route != "" {
			if id := derefStr(e.ID); id != "" {
				keys = append(keys, file.PluginKey(name, instanceName, service, "", consumer, consumerGroup, id))
			}
			keys = append(keys, file.PluginKey(name, instanceName, service, "", consumer, consumerGroup, ""))
		}
		// Fallback: try without service scope for templated service names
		if service != "" {
			if id := derefStr(e.ID); id != "" {
				keys = append(keys, file.PluginKey(name, instanceName, "", route, consumer, consumerGroup, id))
			}
			keys = append(keys, file.PluginKey(name, instanceName, "", route, consumer, consumerGroup, ""))
		}
		// Fallback: try without consumer scope for templated consumer names
		if consumer != "" {
			if id := derefStr(e.ID); id != "" {
				keys = append(keys, file.PluginKey(name, instanceName, service, route, "", consumerGroup, id))
			}
			keys = append(keys, file.PluginKey(name, instanceName, service, route, "", consumerGroup, ""))
		}
		// Fallback: try without consumer group scope for templated consumer group names
		if consumerGroup != "" {
			if id := derefStr(e.ID); id != "" {
				keys = append(keys, file.PluginKey(name, instanceName, service, route, consumer, "", id))
			}
			keys = append(keys, file.PluginKey(name, instanceName, service, route, consumer, "", ""))
		}
		// Final fallback: try with just plugin name (no scopes, no instance_name, no ID)
		keys = append(keys, file.PluginKey(name, "", "", "", "", "", ""))
		// Also try with plugin name + ID (no scopes/instance_name) if ID exists
		if id := derefStr(e.ID); id != "" {
			keys = append(keys, file.PluginKey(name, "", "", "", "", "", id))
		}
		return keys, true

	case *state.Certificate:
		// Certificates have no name at all — without an id, both sides
		// fall back to the same shared, type-level key. Safe here because
		// cert/key are fixed schema fields, never repurposed for non-secret
		// data on a different certificate instance.
		return simpleKeys("certificate", "", derefStr(e.ID)), true

	case *state.Key:
		return simpleKeys("key", derefStr(e.Name), derefStr(e.ID)), true

	// Consumer credential types. Credential identifying fields and types must match
	// the consumerCredentials registry in pkg/file/secret_map.go.
	case *state.KeyAuth:
		return credentialKeys("keyauth_credential", derefStr(e.Key), consumerName(e.Consumer), derefStr(e.ID)), true
	case *state.BasicAuth:
		return credentialKeys("basicauth_credential", derefStr(e.Username), consumerName(e.Consumer), derefStr(e.ID)), true
	case *state.HMACAuth:
		return credentialKeys("hmacauth_credential", derefStr(e.Username), consumerName(e.Consumer), derefStr(e.ID)), true
	case *state.JWTAuth:
		return credentialKeys("jwt_secret", derefStr(e.Key), consumerName(e.Consumer), derefStr(e.ID)), true
	case *state.Oauth2Credential:
		return credentialKeys("oauth2_credential", derefStr(e.ClientID), consumerName(e.Consumer), derefStr(e.ID)), true

	case *state.MTLSAuth:
		return credentialKeys("mtls_auth_credential", derefStr(e.SubjectName), consumerName(e.Consumer), derefStr(e.ID)), true

	case *state.Vault:
		// Vault.Config is a freeform map (like plugin Config) — vault
		// backend credentials can legitimately live there.
		return simpleKeys("vault", derefStr(e.Name), derefStr(e.ID)), true

	// Any field on any of these entity types can, in principle, be
	// templated by the user — there's no restriction in decK's templating
	// on which fields may use ${{ env "..." }}. So each is covered by the
	// same generic per-entity field-name recording BuildSecretMap already
	// does for Plugin/Certificate/Key/Vault, rather than being assumed to
	// have no secrets. simpleKeys tries the id-based candidate first (only
	// useful if the file itself declared that id), then the name-based
	// fallback — mirroring the same file-may-not-have-declared-an-id
	// reasoning as the plugin/certificate/key cases above.
	case *state.Service:
		return simpleKeys("service", derefStr(e.Name), derefStr(e.ID)), true
	case *state.Route:
		return simpleKeys("route", derefStr(e.Name), derefStr(e.ID)), true
	case *state.Upstream:
		return simpleKeys("upstream", derefStr(e.Name), derefStr(e.ID)), true
	case *state.Target:
		return simpleKeys("target", derefStr(e.Target.Target), derefStr(e.ID)), true
	case *state.Consumer:
		return simpleKeys("consumer", derefStr(e.Username), derefStr(e.ID)), true
	case *state.ConsumerGroup:
		return simpleKeys("consumer_group", derefStr(e.Name), derefStr(e.ID)), true
	case *state.SNI:
		return simpleKeys("sni", derefStr(e.Name), derefStr(e.ID)), true
	case *state.CACertificate:
		return simpleKeys("ca_certificate", "", derefStr(e.ID)), true
	case *state.FilterChain:
		return simpleKeys("filter_chain", derefStr(e.Name), derefStr(e.ID)), true
	case *state.Partial:
		return simpleKeys("partial", derefStr(e.Name), derefStr(e.ID)), true
	case *state.License:
		return simpleKeys("license", "", derefStr(e.ID)), true
	case *state.AIModel:
		return simpleKeys("ai_model", derefStr(e.Name), derefStr(e.ID)), true
	default:
		return nil, false
	}
}

// credentialKeys builds candidates for a consumer credential: id-based
// first (if the file declared one), then the credential's own key/username
// scoped to its parent consumer — both of which are user-authored values,
// not auto-assigned, so this second form is reliable on its own in practice.
func credentialKeys(kind, ownKey, consumer, id string) []file.EntityKey {
	var keys []file.EntityKey
	if id != "" {
		keys = append(keys, file.CredentialKey(kind, ownKey, consumer, id))
	}
	keys = append(keys, file.CredentialKey(kind, ownKey, consumer, ""))
	return keys
}

// simpleKeys builds candidates for entities keyed by file.SimpleKey: the
// id-based candidate is tried first (only useful if the file itself
// declared that id — see resolveEntityKeys' doc comment for why a
// diff-time id alone can't be trusted), then the name-based fallback that
// extraction computes when the file has no declared id, the common case.
func simpleKeys(kind, name, id string) []file.EntityKey {
	var keys []file.EntityKey
	if id != "" {
		keys = append(keys, file.SimpleKey(kind, name, id))
	}
	keys = append(keys, file.SimpleKey(kind, name, ""))
	return keys
}

func consumerName(c *kong.Consumer) string {
	if c == nil {
		return ""
	}
	return derefStr(c.Username)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
