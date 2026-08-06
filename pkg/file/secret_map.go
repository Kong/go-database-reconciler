package file

import (
	"regexp"
	"strings"
)

// SecretMap records, per entity instance, which of its own field names are
// backed by a DECK_* environment variable reference. Built once from a
// mock-rendered Content (env var refs left as their bare name instead of
// their real value — see EnvVarsMock) and consulted at diff time via the
// same EntityKey constructors, so masking can target the exact field on the
// exact entity that was actually templated, never a coincidental value match.
type SecretMap map[EntityKey]map[string]bool

// BuildSecretMap walks a mock-rendered Content and records, per entity
// instance, which field names are secret.
func BuildSecretMap(mock *Content) SecretMap {
	sm := make(SecretMap)
	if mock == nil {
		return sm
	}

	for i := range mock.Services {
		svc := &mock.Services[i]
		svcName := deref(svc.Name)
		svcID := deref(svc.ID)

		// The service's OWN fields (any field can be templated, not just
		// ones we'd expect to be secret — a user can template anything).
		// Record on both ID-based and name-based candidates, so diff-time
		// resolveEntityKeys finds the field whichever candidate it tries first.
		if svcID != "" {
			recordSecrets(svc, SimpleKey("service", svcName, svcID), sm)
		}
		recordSecrets(svc, SimpleKey("service", svcName, ""), sm)

		for _, p := range svc.Plugins {
			key := PluginKey(deref(p.Name), deref(p.InstanceName), svcName, "", "", "", deref(p.ID))
			recordSecrets(p, key, sm)
		}
		for _, r := range svc.Routes {
			recordNestedRoute(r, sm)
		}
	}

	// Top-level routes have no structural parent in the file.
	for i := range mock.Routes {
		recordNestedRoute(&mock.Routes[i], sm)
	}

	for i := range mock.Upstreams {
		u := &mock.Upstreams[i]
		upstreamName := deref(u.Name)
		upstreamID := deref(u.ID)
		if upstreamID != "" {
			recordSecrets(u, SimpleKey("upstream", upstreamName, upstreamID), sm)
		}
		recordSecrets(u, SimpleKey("upstream", upstreamName, ""), sm)

		for _, t := range u.Targets {
			targetID := deref(t.ID)
			targetName := deref(t.Target.Target)
			if targetID != "" {
				recordSecrets(t, SimpleKey("target", targetName, targetID), sm)
			}
			recordSecrets(t, SimpleKey("target", targetName, ""), sm)
		}
	}

	for i := range mock.ConsumerGroups {
		cg := &mock.ConsumerGroups[i]
		cgName := deref(cg.Name)
		cgID := deref(cg.ID)
		if cgID != "" {
			recordSecrets(cg, SimpleKey("consumer_group", cgName, cgID), sm)
		}
		recordSecrets(cg, SimpleKey("consumer_group", cgName, ""), sm)
	}

	for i := range mock.Consumers {
		c := &mock.Consumers[i]
		consumerName := deref(c.Username)
		consumerID := deref(c.ID)

		if consumerID != "" {
			recordSecrets(c, SimpleKey("consumer", consumerName, consumerID), sm)
		}
		recordSecrets(c, SimpleKey("consumer", consumerName, ""), sm)

		for _, p := range c.Plugins {
			key := PluginKey(deref(p.Name), deref(p.InstanceName), "", "", consumerName, "", deref(p.ID))
			recordSecrets(p, key, sm)
		}
		for _, cred := range c.BasicAuths {
			key := CredentialKey("basicauth_credential", deref(cred.Username), consumerName, deref(cred.ID))
			recordSecrets(cred, key, sm)
		}
		for _, cred := range c.KeyAuths {
			key := CredentialKey("keyauth_credential", deref(cred.Key), consumerName, deref(cred.ID))
			recordSecrets(cred, key, sm)
		}
		for _, cred := range c.HMACAuths {
			key := CredentialKey("hmacauth_credential", deref(cred.Username), consumerName, deref(cred.ID))
			recordSecrets(cred, key, sm)
		}
		for _, cred := range c.JWTAuths {
			key := CredentialKey("jwt_secret", deref(cred.Key), consumerName, deref(cred.ID))
			recordSecrets(cred, key, sm)
		}
		for _, cred := range c.Oauth2Creds {
			key := CredentialKey("oauth2_credential", deref(cred.ClientID), consumerName, deref(cred.ID))
			recordSecrets(cred, key, sm)
		}
	}

	for i := range mock.Certificates {
		c := &mock.Certificates[i]
		recordSecrets(c, SimpleKey("certificate", "", deref(c.ID)), sm)

		// SNIs are nested under certificates
		for _, sni := range c.SNIs {
			sniName := deref(sni.Name)
			sniID := deref(sni.ID)
			if sniID != "" {
				recordSecrets(sni, SimpleKey("sni", sniName, sniID), sm)
			}
			recordSecrets(sni, SimpleKey("sni", sniName, ""), sm)
		}
	}

	for i := range mock.CACertificates {
		ca := &mock.CACertificates[i]
		recordSecrets(ca, SimpleKey("ca_certificate", "", deref(ca.ID)), sm)
	}

	for i := range mock.Keys {
		k := &mock.Keys[i]
		keyName := deref(k.Name)
		keyID := deref(k.ID)
		if keyID != "" {
			recordSecrets(k, SimpleKey("key", keyName, keyID), sm)
		}
		recordSecrets(k, SimpleKey("key", keyName, ""), sm)
	}

	for i := range mock.Vaults {
		v := &mock.Vaults[i]
		vaultName := deref(v.Name)
		vaultID := deref(v.ID)
		// Vault.Config is freeform, like plugin Config — vault backend
		// credentials can legitimately live there.
		if vaultID != "" {
			recordSecrets(v, SimpleKey("vault", vaultName, vaultID), sm)
		}
		recordSecrets(v, SimpleKey("vault", vaultName, ""), sm)
	}

	for i := range mock.FilterChains {
		fc := &mock.FilterChains[i]
		filterChainName := deref(fc.Name)
		filterChainID := deref(fc.ID)
		if filterChainID != "" {
			recordSecrets(fc, SimpleKey("filter_chain", filterChainName, filterChainID), sm)
		}
		recordSecrets(fc, SimpleKey("filter_chain", filterChainName, ""), sm)
	}

	for i := range mock.Licenses {
		l := &mock.Licenses[i]
		recordSecrets(l, SimpleKey("license", "", deref(l.ID)), sm)
	}

	for i := range mock.Partials {
		p := &mock.Partials[i]
		partialName := deref(p.Name)
		partialID := deref(p.ID)
		if partialID != "" {
			recordSecrets(p, SimpleKey("partial", partialName, partialID), sm)
		}
		recordSecrets(p, SimpleKey("partial", partialName, ""), sm)
	}

	for i := range mock.RBACRoles {
		r := &mock.RBACRoles[i]
		recordSecrets(r, SimpleKey("rbac_role", "", deref(r.ID)), sm)
	}

	for i := range mock.KeySets {
		ks := &mock.KeySets[i]
		keySetName := deref(ks.Name)
		keySetID := deref(ks.ID)
		if keySetID != "" {
			recordSecrets(ks, SimpleKey("key_set", keySetName, keySetID), sm)
		}
		recordSecrets(ks, SimpleKey("key_set", keySetName, ""), sm)
	}

	for i := range mock.AIModels {
		am := &mock.AIModels[i]
		aiModelName := deref(am.Name)
		aiModelID := deref(am.ID)
		if aiModelID != "" {
			recordSecrets(am, SimpleKey("ai_model", aiModelName, aiModelID), sm)
		}
		recordSecrets(am, SimpleKey("ai_model", aiModelName, ""), sm)
	}

	for i := range mock.CustomEntities {
		ce := &mock.CustomEntities[i]
		recordSecrets(ce, SimpleKey("custom_entity", "", deref(ce.ID)), sm)
	}

	for i := range mock.ServicePackages {
		sp := &mock.ServicePackages[i]
		spName := deref(sp.Name)
		spID := deref(sp.ID)
		if spID != "" {
			recordSecrets(sp, SimpleKey("service_package", spName, spID), sm)
		}
		recordSecrets(sp, SimpleKey("service_package", spName, ""), sm)
	}

	// Top-level plugins have no structural parent — their scope comes from
	// their own Service/Route/Consumer/ConsumerGroup reference fields.
	for i := range mock.Plugins {
		p := &mock.Plugins[i]
		var svcName, routeName, consumerName, cgName string
		if p.Service != nil {
			svcName = deref(p.Service.Name)
		}
		if p.Route != nil {
			routeName = deref(p.Route.Name)
		}
		if p.Consumer != nil {
			consumerName = deref(p.Consumer.Username)
		}
		if p.ConsumerGroup != nil {
			cgName = deref(p.ConsumerGroup.Name)
		}
		key := PluginKey(deref(p.Name), deref(p.InstanceName), svcName, routeName, consumerName, cgName, deref(p.ID))
		recordSecrets(p, key, sm)
	}

	return sm
}

// recordNestedRoute records a route's own fields (any field can be
// templated — e.g. methods, hosts, paths — not just ones we'd expect to be
// secret) plus its nested plugins. A route-scoped plugin's reconciled
// object carries only its Route reference, never also Service — even
// when nested under a service in the file — so no service scope here.
func recordNestedRoute(r *FRoute, sm SecretMap) {
	routeName := deref(r.Name)
	routeID := deref(r.ID)
	// Record on both ID-based and name-based candidates.
	if routeID != "" {
		recordSecrets(r, SimpleKey("route", routeName, routeID), sm)
	}
	recordSecrets(r, SimpleKey("route", routeName, ""), sm)

	for _, p := range r.Plugins {
		key := PluginKey(deref(p.Name), deref(p.InstanceName), "", routeName, "", "", deref(p.ID))
		recordSecrets(p, key, sm)
	}
}

// rawTemplateEnvPattern matches an unrendered `env "DECK_..."` reference
// inside a template expression, e.g. `${{ env "DECK_RATE_LIMIT_0" }}`. This
// lets BuildSecretMap detect secrets when the caller renders with
// EnvVarsSkip (raw, unrendered template text left in place) rather than
// EnvVarsMock (bare env var name) — both are valid inputs to this function.
var rawTemplateEnvPattern = regexp.MustCompile(`env\s+"(DECK_[^"]*)"`)

// isSecretFieldValue reports whether a field's rendered value indicates it
// was templated from a DECK_-prefixed environment variable, under either
// EnvVarsMock rendering (bare name, e.g. "DECK_X") or EnvVarsSkip rendering
// (raw template text, e.g. `${{ env "DECK_X" }}`).
func isSecretFieldValue(s string) bool {
	if strings.HasPrefix(s, "DECK_") {
		return true
	}
	return rawTemplateEnvPattern.MatchString(s)
}

func recordSecrets(entity interface{}, key EntityKey, sm SecretMap) {
	walkObject(entity, "", func(fieldName string, value interface{}) {
		s, ok := value.(string)
		if !ok || !isSecretFieldValue(s) {
			return
		}
		if sm[key] == nil {
			sm[key] = make(map[string]bool)
		}
		sm[key][leafFieldName(fieldName)] = true
	})
}

// leafFieldName extracts the plain field name masking looks up by (the JSON
// key a struct field serializes as), from a full walk path. For a slice
// field like "methods", walkObject produces per-element paths such as
// "methods[0]" — the array index must be stripped so a templated element
// is still attributed to "methods", matching what maskFieldsByName looks up
// via the struct field's JSON tag (which never includes an index).
func leafFieldName(fieldName string) string {
	parts := strings.Split(fieldName, ".")
	last := parts[len(parts)-1]
	if idx := strings.Index(last, "["); idx != -1 {
		last = last[:idx]
	}
	return last
}
