package file

import (
	"strings"
	"testing"
)

// ============================================================================
// Test Helpers
// ============================================================================

// assertSecretField verifies that a field is marked as secret in SecretMap
func assertSecretField(t *testing.T, sm SecretMap, key EntityKey, fieldName string) {
	t.Helper()
	if sm[key] == nil {
		t.Errorf("key %v not found in SecretMap", key)
		return
	}
	if !sm[key][fieldName] {
		t.Errorf("field %q not marked as secret for key %v, got: %v", fieldName, key, sm[key])
	}
}

// assertNoSecretField verifies that a field is NOT marked as secret
func assertNoSecretField(t *testing.T, sm SecretMap, key EntityKey, fieldName string) {
	t.Helper()
	if sm[key] != nil && sm[key][fieldName] {
		t.Errorf("field %q should not be marked as secret for key %v, but was", fieldName, key)
	}
}

// assertKeyExists verifies that an EntityKey exists in SecretMap
func assertKeyExists(t *testing.T, sm SecretMap, key EntityKey) {
	t.Helper()
	if sm[key] == nil {
		t.Errorf("key %v not found in SecretMap", key)
	}
}

// ============================================================================
// Service Tests
// ============================================================================

func TestBuildSecretMap_Service_SimpleFieldTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    host: ${{ env "DECK_HOST" }}
    port: 8080
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertSecretField(t, sm, key, "host")
	assertNoSecretField(t, sm, key, "port")
}

func TestBuildSecretMap_Service_MultipleTemplatedFields(t *testing.T) {
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    host: ${{ env "DECK_HOST" }}
    protocol: ${{ env "DECK_PROTOCOL" }}
    port: 8080
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertSecretField(t, sm, key, "host")
	assertSecretField(t, sm, key, "protocol")
	assertNoSecretField(t, sm, key, "port")
}

func TestBuildSecretMap_Service_WithID(t *testing.T) {
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    id: 12345678-1234-1234-1234-123456789012
    host: ${{ env "DECK_HOST" }}
`
	sm := BuildSecretMap(yaml)

	// Both ID-based and name-based keys should exist
	keyWithID := SimpleKey("service", "test-svc", "12345678-1234-1234-1234-123456789012")
	keyWithoutID := SimpleKey("service", "test-svc", "")

	assertSecretField(t, sm, keyWithID, "host")
	assertSecretField(t, sm, keyWithoutID, "host")
}

// ============================================================================
// Route Tests
// ============================================================================

func TestBuildSecretMap_Route_SimpleFieldTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
routes:
  - name: test-route
    paths:
      - ${{ env "DECK_PATH" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("route", "test-route", "")
	assertSecretField(t, sm, key, "paths")
}

// ============================================================================
// Plugin Tests
// ============================================================================

func TestBuildSecretMap_Plugin_TopLevel_ConfigTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
plugins:
  - name: rate-limiting
    config:
      minute: ${{ env "DECK_RATE_LIMIT" }}
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("rate-limiting", "", "", "", "", "", "")
	assertSecretField(t, sm, key, "minute")
}

func TestBuildSecretMap_Plugin_ServiceScoped(t *testing.T) {
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    plugins:
      - name: rate-limiting
        config:
          minute: ${{ env "DECK_RATE_LIMIT" }}
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("rate-limiting", "", "test-svc", "", "", "", "")
	assertSecretField(t, sm, key, "minute")
}

func TestBuildSecretMap_Plugin_RouteScoped(t *testing.T) {
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_RATE_LIMIT" }}
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("rate-limiting", "", "", "test-route", "", "", "")
	assertSecretField(t, sm, key, "minute")
}

func TestBuildSecretMap_Plugin_WithTemplatedParentScope(t *testing.T) {
	yaml := `
_format_version: "3.0"
services:
  - name: ${{ env "DECK_SERVICE_NAME" }}
    plugins:
      - name: rate-limiting
        config:
          minute: ${{ env "DECK_RATE_LIMIT" }}
`
	sm := BuildSecretMap(yaml)

	// When parent service name is templated, fallback keys without service scope should exist
	keyWithService := PluginKey("rate-limiting", "", "${{ env \"DECK_SERVICE_NAME\" }}", "", "", "", "")
	keyWithoutService := PluginKey("rate-limiting", "", "", "", "", "", "")

	// At least one of these should have the secret recorded
	found := sm[keyWithService] != nil && sm[keyWithService]["minute"]

	if sm[keyWithoutService] != nil && sm[keyWithoutService]["minute"] {
		found = true
	}
	if !found {
		t.Errorf("expected to find 'minute' secret for plugin with templated parent scope")
	}
}

// ============================================================================
// Consumer Tests
// ============================================================================

func TestBuildSecretMap_Consumer_SimpleFieldTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    custom_id: ${{ env "DECK_CUSTOM_ID" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("consumer", "test-consumer", "")
	assertSecretField(t, sm, key, "custom_id")
}

func TestBuildSecretMap_Consumer_WithScopedPlugin(t *testing.T) {
	// Consumer-scoped plugins should also record secret fields
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    plugins:
      - name: request-transformer
        config:
          add:
            headers:
              X-API-Key: ${{ env "DECK_API_KEY" }}
`
	sm := BuildSecretMap(yaml)
	// Plugin scoped to consumer should be keyed with consumer name
	key := PluginKey("request-transformer", "", "", "", "test-consumer", "", "")
	assertSecretField(t, sm, key, "X-API-Key")
}

func TestBuildSecretMap_Consumer_WithMultipleScopedPlugins(t *testing.T) {
	// Multiple consumer-scoped plugins
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    plugins:
      - name: key-auth
        config:
          key_in_header: ${{ env "DECK_KEY_HEADER" }}
      - name: rate-limiting
        config:
          minute: ${{ env "DECK_RATE_LIMIT" }}
`
	sm := BuildSecretMap(yaml)
	keyAuth := PluginKey("key-auth", "", "", "", "test-consumer", "", "")
	rateLim := PluginKey("rate-limiting", "", "", "", "test-consumer", "", "")
	assertSecretField(t, sm, keyAuth, "key_in_header")
	assertSecretField(t, sm, rateLim, "minute")
}

func TestBuildSecretMap_Consumer_ACLGroups_NoSecretHandling(t *testing.T) {
	// ACLGroups are references only, not secrets
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    acls:
      - group: admin-group
      - group: readonly-group
`
	sm := BuildSecretMap(yaml)
	// ACLGroups contain no secret data - they're just group names
	// The consumer itself should exist but with no secret fields from ACLs
	key := SimpleKey("consumer", "test-consumer", "")
	// group field should not be marked as secret (it's just a group reference)
	assertNoSecretField(t, sm, key, "group")
}

func TestBuildSecretMap_Consumer_Groups_NoSecretHandling(t *testing.T) {
	// Groups field references consumer groups, not secrets
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    groups:
      - name: group1
      - name: group2
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("consumer", "test-consumer", "")
	// group name references are not secrets
	assertNoSecretField(t, sm, key, "name")
}

// ============================================================================
// Credential Tests (Consumer Nested)
// ============================================================================

func TestBuildSecretMap_Consumer_BasicAuthCredential(t *testing.T) {
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    basicauth_credentials:
      - username: ${{ env "DECK_BASIC_USER" }}
        password: ${{ env "DECK_BASIC_PASS" }}
`
	sm := BuildSecretMap(yaml)
	key := CredentialKey("basicauth_credential", "${{ env \"DECK_BASIC_USER\" }}", "test-consumer", "")
	assertSecretField(t, sm, key, "username")
	assertSecretField(t, sm, key, "password")
}

func TestBuildSecretMap_Consumer_KeyAuthCredential(t *testing.T) {
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    keyauth_credentials:
      - key: ${{ env "DECK_KEY_AUTH" }}
`
	sm := BuildSecretMap(yaml)
	key := CredentialKey("keyauth_credential", "${{ env \"DECK_KEY_AUTH\" }}", "test-consumer", "")
	assertSecretField(t, sm, key, "key")
}

func TestBuildSecretMap_Consumer_HMACAuthCredential(t *testing.T) {
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    hmacauth_credentials:
      - username: ${{ env "DECK_HMAC_USER" }}
        secret: ${{ env "DECK_HMAC_SECRET" }}
`
	sm := BuildSecretMap(yaml)
	key := CredentialKey("hmacauth_credential", "${{ env \"DECK_HMAC_USER\" }}", "test-consumer", "")
	assertSecretField(t, sm, key, "username")
	assertSecretField(t, sm, key, "secret")
}

func TestBuildSecretMap_Consumer_JWTAuthCredential(t *testing.T) {
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    jwt_secrets:
      - key: ${{ env "DECK_JWT_KEY" }}
        secret: ${{ env "DECK_JWT_SECRET" }}
`
	sm := BuildSecretMap(yaml)
	key := CredentialKey("jwt_secret", "${{ env \"DECK_JWT_KEY\" }}", "test-consumer", "")
	assertSecretField(t, sm, key, "key")
	assertSecretField(t, sm, key, "secret")
}

func TestBuildSecretMap_Consumer_Oauth2Credential(t *testing.T) {
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    oauth2_credentials:
      - client_id: ${{ env "DECK_OAUTH_CLIENT_ID" }}
        client_secret: ${{ env "DECK_OAUTH_SECRET" }}
`
	sm := BuildSecretMap(yaml)
	key := CredentialKey("oauth2_credential", "${{ env \"DECK_OAUTH_CLIENT_ID\" }}", "test-consumer", "")
	assertSecretField(t, sm, key, "client_id")
	assertSecretField(t, sm, key, "client_secret")
}

func TestBuildSecretMap_Consumer_MTLSAuthCredential(t *testing.T) {
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    mtls_auth_credentials:
      - subject_name: ${{ env "DECK_MTLS_SUBJECT" }}
`
	sm := BuildSecretMap(yaml)
	key := CredentialKey("mtls_auth_credential", "${{ env \"DECK_MTLS_SUBJECT\" }}", "test-consumer", "")
	assertSecretField(t, sm, key, "subject_name")
}

// ============================================================================
// Certificate Tests
// ============================================================================

func TestBuildSecretMap_Certificate_CertAndKeyTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
certificates:
  - id: cert-123
    cert: ${{ env "DECK_CERT" }}
    key: ${{ env "DECK_KEY" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("certificate", "", "cert-123")
	assertSecretField(t, sm, key, "cert")
	assertSecretField(t, sm, key, "key")
}

// ============================================================================
// Upstream Tests
// ============================================================================

func TestBuildSecretMap_Upstream_SimpleField(t *testing.T) {
	yaml := `
_format_version: "3.0"
upstreams:
  - name: test-upstream
    slots: ${{ env "DECK_SLOTS" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("upstream", "test-upstream", "")
	assertSecretField(t, sm, key, "slots")
}

// ============================================================================
// SNI Tests (nested within Certificates)
// ============================================================================

func TestBuildSecretMap_Certificate_WithTemplatedSNI(t *testing.T) {
	// SNIs are nested within certificates in the state file
	yaml := `
_format_version: "3.0"
certificates:
  - id: cert-123
    cert: ${{ env "DECK_CERT" }}
    key: ${{ env "DECK_KEY" }}
    snis:
      - name: ${{ env "DECK_SNI_NAME" }}
`
	sm := BuildSecretMap(yaml)
	// The certificate key should record the templated SNI name
	certKey := SimpleKey("certificate", "", "cert-123")
	assertSecretField(t, sm, certKey, "cert")
	assertSecretField(t, sm, certKey, "key")
	// The SNI name field within the nested array is also discovered
	assertSecretField(t, sm, certKey, "name")
}

func TestBuildSecretMap_Certificate_WithMultipleSNIs(t *testing.T) {
	// Certificate with multiple SNI names, some templated
	yaml := `
_format_version: "3.0"
certificates:
  - id: cert-123
    cert: ${{ env "DECK_CERT" }}
    snis:
      - name1: example.com
      - name2: ${{ env "DECK_SNI_ALT_NAME" }}
      - name3: api.example.com
`
	sm := BuildSecretMap(yaml)
	certKey := SimpleKey("certificate", "", "cert-123")
	assertSecretField(t, sm, certKey, "cert")
	// The name field should be marked as secret (contains at least one templated name)
	assertSecretField(t, sm, certKey, "name2")
}

// ============================================================================
// Target Tests (nested within Upstreams)
// ============================================================================

func TestBuildSecretMap_Upstream_TargetWithTemplatedField(t *testing.T) {
	// Targets are nested within upstreams, so they're discovered as part of
	// the upstream's nested structure through recursive traversal.
	yaml := `
_format_version: "3.0"
upstreams:
  - name: test-upstream
    targets:
      - target: example.com:8080
        weight: ${{ env "DECK_WEIGHT" }}
`
	sm := BuildSecretMap(yaml)
	// The upstream key should record the nested target field as secret
	upstreamKey := SimpleKey("upstream", "test-upstream", "")
	// The weight field within the nested target array is discovered
	assertSecretField(t, sm, upstreamKey, "weight")
}

func TestBuildSecretMap_Upstream_TargetHostTemplated(t *testing.T) {
	// Test: target field itself (the host:port) is templated
	yaml := `
_format_version: "3.0"
upstreams:
  - name: test-upstream
    targets:
      - target: ${{ env "DECK_TARGET_HOST" }}:8080
        weight: 100
`
	sm := BuildSecretMap(yaml)
	upstreamKey := SimpleKey("upstream", "test-upstream", "")
	// The target field itself is templated
	assertSecretField(t, sm, upstreamKey, "target")
}

func TestBuildSecretMap_Upstream_MultipleTargetsTemplated(t *testing.T) {
	// Test: multiple targets with templated fields
	yaml := `
_format_version: "3.0"
upstreams:
  - name: test-upstream
    targets:
      - target: host1.com:8080
        weight: ${{ env "DECK_WEIGHT_1" }}
        tags:
          - ${{ env "DECK_TAG_1" }}
      - target: host2.com:8080
        weight: ${{ env "DECK_WEIGHT_2" }}
`
	sm := BuildSecretMap(yaml)
	upstreamKey := SimpleKey("upstream", "test-upstream", "")
	// Both weight and tags fields should be marked as secret
	assertSecretField(t, sm, upstreamKey, "weight")
	assertSecretField(t, sm, upstreamKey, "tags")
}

// ============================================================================
// Key Tests
// ============================================================================

func TestBuildSecretMap_Key_NameTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
keys:
  - name: ${{ env "DECK_KEY_NAME" }}
    pem:
      public_key: ${{ env "DECK_PUBLIC_KEY" }}
`
	sm := BuildSecretMap(yaml)
	// Key should be recorded even with templated name
	found := false
	for key := range sm {
		if strings.Contains(key.String(), "key") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find key entry for templated key name")
	}
}

// ============================================================================
// Vault Tests
// ============================================================================

func TestBuildSecretMap_Vault_ConfigTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
vaults:
  - name: test-vault
    prefix: env
    config:
      prefix: ${{ env "DECK_VAULT_PREFIX" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("vault", "test-vault", "")
	assertSecretField(t, sm, key, "prefix")
}

// ============================================================================
// Type Mismatch Tests
// ============================================================================

func TestBuildSecretMap_TypeMismatch_StringInIntField(t *testing.T) {
	// This tests that templated values in int fields are still detected as secrets
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    retries: ${{ env "DECK_RETRIES" }}
    port: 8080
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertSecretField(t, sm, key, "retries")
	assertNoSecretField(t, sm, key, "port")
}

// ============================================================================
// Non-Templated Values Tests
// ============================================================================

func TestBuildSecretMap_NonTemplatedValuesNotMarked(t *testing.T) {
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    host: example.com
    port: 8080
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertNoSecretField(t, sm, key, "host")
	assertNoSecretField(t, sm, key, "port")
}

// ============================================================================
// CACertificate Tests
// ============================================================================

func TestBuildSecretMap_CACertificate_CertTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
ca_certificates:
  - id: ca-cert-123
    cert: ${{ env "DECK_CA_CERT" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("ca_certificate", "", "ca-cert-123")
	assertSecretField(t, sm, key, "cert")
}

// ============================================================================
// ConsumerGroup Tests
// ============================================================================

func TestBuildSecretMap_ConsumerGroup_NameTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
consumer_groups:
  - name: ${{ env "DECK_GROUP_NAME" }}
    description: test group
`
	sm := BuildSecretMap(yaml)
	// ConsumerGroup with templated name should still be recorded
	found := false
	for key := range sm {
		if strings.Contains(key.String(), "consumer_group") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find consumer_group key for templated name")
	}
}

// ============================================================================
// FilterChain Tests
// ============================================================================

func TestBuildSecretMap_FilterChain_NameTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
filter_chains:
  - name: ${{ env "DECK_FILTER_CHAIN_NAME" }}
`
	sm := BuildSecretMap(yaml)
	found := false
	for key := range sm {
		if strings.Contains(key.String(), "filter_chain") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find filter_chain key for templated name")
	}
}

// ============================================================================
// RBACRole Tests
// ============================================================================

func TestBuildSecretMap_RBACRole_IDTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
rbac_roles:
  - id: ${{ env "DECK_ROLE_ID" }}
`
	sm := BuildSecretMap(yaml)
	found := false
	for key := range sm {
		if strings.Contains(key.String(), "rbac_role") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find rbac_role key")
	}
}

// ============================================================================
// ServicePackage Tests
// ============================================================================

func TestBuildSecretMap_ServicePackage_NameTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
service_packages:
  - name: ${{ env "DECK_SERVICE_PKG" }}
    description: test package
`
	sm := BuildSecretMap(yaml)
	found := false
	for key := range sm {
		if strings.Contains(key.String(), "service_package") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find service_package key for templated name")
	}
}

// ============================================================================
// License Tests
// ============================================================================

func TestBuildSecretMap_License_PayloadTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
licenses:
  - id: lic-123
    payload: ${{ env "DECK_LICENSE_PAYLOAD" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("license", "", "lic-123")
	assertSecretField(t, sm, key, "payload")
}

// ============================================================================
// AIModel Tests
// ============================================================================

func TestBuildSecretMap_AIModel_NameTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
ai_models:
  - name: ${{ env "DECK_AI_MODEL_NAME" }}
    provider: openai
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("ai_model", "${{ env \"DECK_AI_MODEL_NAME\" }}", "")
	assertKeyExists(t, sm, key)
}

// ============================================================================
// CustomEntity Tests
// ============================================================================

func TestBuildSecretMap_CustomEntity_IDTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
custom_entities:
  - id: ${{ env "DECK_CUSTOM_ENTITY_ID" }}
    type: custom_type
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("custom_entity", "", "${{ env \"DECK_CUSTOM_ENTITY_ID\" }}")
	assertKeyExists(t, sm, key)
}

// ============================================================================
// Partial Tests
// ============================================================================

func TestBuildSecretMap_Partial_NameTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
partials:
  - name: ${{ env "DECK_PARTIAL_NAME" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("partial", "${{ env \"DECK_PARTIAL_NAME\" }}", "")
	assertKeyExists(t, sm, key)
}

// ============================================================================
// KeySet Tests
// ============================================================================

func TestBuildSecretMap_KeySet_NameTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
key_sets:
  - name: ${{ env "DECK_KEYSET_NAME" }}
`
	sm := BuildSecretMap(yaml)
	found := false
	for key := range sm {
		if strings.Contains(key.String(), "key_set") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find key_set key for templated name")
	}
}

// ============================================================================
// ClonedPlugin Tests
// ============================================================================

func TestBuildSecretMap_ClonedPlugin_IDTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
cloned_plugins:
  - id: ${{ env "DECK_CLONED_PLUGIN_ID" }}
`
	sm := BuildSecretMap(yaml)
	found := false
	for key := range sm {
		if strings.Contains(key.String(), "cloned_plugin") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find cloned_plugin key")
	}
}

// ============================================================================
// CustomPlugin Tests
// ============================================================================

func TestBuildSecretMap_CustomPlugin_IDTemplated(t *testing.T) {
	yaml := `
_format_version: "3.0"
custom_plugins:
  - id: ${{ env "DECK_CUSTOM_PLUGIN_ID" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("custom_plugin", "", "${{ env \"DECK_CUSTOM_PLUGIN_ID\" }}")
	assertKeyExists(t, sm, key)
}

// ============================================================================
// Empty/Nil Tests
// ============================================================================

func TestBuildSecretMap_EmptyYAML(t *testing.T) {
	yaml := `_format_version: "3.0"`
	sm := BuildSecretMap(yaml)
	if len(sm) != 0 {
		t.Errorf("expected empty SecretMap for empty YAML, got %d entries", len(sm))
	}
}

// ============================================================================
// WalkForSecrets Internal Logic Tests
// ============================================================================

func TestWalkForSecrets_SimpleStringAtRoot(t *testing.T) {
	// Test: Simple string value at root level (path="")
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    host: ${{ env "DECK_HOST" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertSecretField(t, sm, key, "host")
}

func TestWalkForSecrets_NestedMapValue(t *testing.T) {
	// Test: String value nested in a map structure (path="config.api_key")
	yaml := `
_format_version: "3.0"
plugins:
  - name: my-plugin
    config:
      api_key: ${{ env "DECK_API_KEY" }}
      timeout: 30
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("my-plugin", "", "", "", "", "", "")
	assertSecretField(t, sm, key, "api_key")
	assertNoSecretField(t, sm, key, "timeout")
}

func TestWalkForSecrets_DeepNestedMap(t *testing.T) {
	// Test: String value in deeply nested map structure (path="config.auth.token")
	yaml := `
_format_version: "3.0"
plugins:
  - name: oauth-plugin
    config:
      auth:
        token: ${{ env "DECK_OAUTH_TOKEN" }}
        refresh_interval: 3600
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("oauth-plugin", "", "", "", "", "", "")
	assertSecretField(t, sm, key, "token")
}

func TestWalkForSecrets_ArrayOfStrings(t *testing.T) {
	// Test: Array of string values (path="paths[0]" -> leafFieldName="paths")
	yaml := `
_format_version: "3.0"
routes:
  - name: test-route
    paths:
      - DECK_PATH_1
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("route", "test-route", "")
	// All paths array is marked as secret because at least one element is a secret
	assertSecretField(t, sm, key, "paths")
}

func TestWalkForSecrets_MapInArray(t *testing.T) {
	// Test: Map structure within an array element
	// leafFieldName extracts the innermost field, so "X-API-Key" not "headers"
	yaml := `
_format_version: "3.0"
plugins:
  - name: request-transformer
    config:
      add:
        headers:
          X-API-Key: ${{ env "DECK_API_KEY" }}
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("request-transformer", "", "", "", "", "", "")
	assertSecretField(t, sm, key, "X-API-Key")
}

func TestWalkForSecrets_ArrayInMap(t *testing.T) {
	// Test: Array within a map structure
	yaml := `
_format_version: "3.0"
plugins:
  - name: cors
    config:
      allowed_origins:
        - https://example.com
        - DECK_ALLOWED_ORIGIN
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("cors", "", "", "", "", "", "")
	assertSecretField(t, sm, key, "allowed_origins")
}

func TestWalkForSecrets_TemplateExpressionFormat(t *testing.T) {
	// Test: Template expression format (${{ env "DECK_..." }})
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    host: ${{ env "DECK_HOST" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertSecretField(t, sm, key, "host")
}

func TestWalkForSecrets_MixedSecretFormats(t *testing.T) {
	// Test: Both bare DECK_ and template expression formats in same structure
	yaml := `
_format_version: "3.0"
plugins:
  - name: test-plugin
    config:
      key1: ${{ env "DECK_KEY_1" }}
      key2: ${{ env "DECK_KEY_2" }}
      key3: regular-value
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("test-plugin", "", "", "", "", "", "")
	assertSecretField(t, sm, key, "key1")
	assertSecretField(t, sm, key, "key2")
	assertNoSecretField(t, sm, key, "key3")
}

func TestWalkForSecrets_NonStringValuesIgnored(t *testing.T) {
	// Test: Non-string values (numbers, booleans, null) are ignored
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    port: 8080
    enabled: true
    retries: null
    host: ${{ env "DECK_HOST" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertSecretField(t, sm, key, "host")
	assertNoSecretField(t, sm, key, "port")
	assertNoSecretField(t, sm, key, "enabled")
	assertNoSecretField(t, sm, key, "retries")
}

func TestWalkForSecrets_NumericStringInTypedField(t *testing.T) {
	// Test: String value "123" doesn't match DECK_ or template pattern, not marked secret
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    port: "123"
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertNoSecretField(t, sm, key, "port")
}

func TestWalkForSecrets_MultipleSecretsSameEntity(t *testing.T) {
	// Test: Multiple secret fields in the same entity are all recorded
	yaml := `
_format_version: "3.0"
consumers:
  - username: test-consumer
    basicauth_credentials:
      - username: ${{ env "DECK_BASIC_USER" }}
        password: ${{ env "DECK_BASIC_PASS" }}
`
	sm := BuildSecretMap(yaml)
	key := CredentialKey("basicauth_credential", "${{ env \"DECK_BASIC_USER\" }}", "test-consumer", "")
	assertSecretField(t, sm, key, "username")
	assertSecretField(t, sm, key, "password")
}

func TestWalkForSecrets_ArrayIndexPathHandling(t *testing.T) {
	// Test: Array indices are properly stripped from field names
	yaml := `
_format_version: "3.0"
certificates:
  - id: cert-123
    cert: ${{ env "DECK_CERT_1" }}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("certificate", "", "cert-123")
	assertSecretField(t, sm, key, "cert")
}

func TestWalkForSecrets_ComplexNestedStructure(t *testing.T) {
	// Test: Complex nested structure with maps and arrays at multiple levels
	// leafFieldName extracts innermost fields: "key" and "value"
	yaml := `
_format_version: "3.0"
plugins:
  - name: complex-plugin
    config:
      level1:
        level2:
          - key: ${{ env "DECK_SECRET_1" }}
            value: public-value
          - key: normal-key
            value: ${{ env "DECK_SECRET_2" }}
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("complex-plugin", "", "", "", "", "", "")
	assertSecretField(t, sm, key, "key")
	assertSecretField(t, sm, key, "value")
}

func TestWalkForSecrets_EmptyCollections(t *testing.T) {
	// Test: Empty maps and arrays don't cause errors
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    host: example.com
    tags: []
    metadata: {}
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertNoSecretField(t, sm, key, "tags")
	assertNoSecretField(t, sm, key, "metadata")
}

func TestWalkForSecrets_PartialSecretArray(t *testing.T) {
	// Test: Array with mix of secret and non-secret values
	yaml := `
_format_version: "3.0"
routes:
  - name: test-route
    paths:
      - /public
      - DECK_PRIVATE_PATH
      - /admin
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("route", "test-route", "")
	// Array containing any secret value marks the whole array field as secret
	assertSecretField(t, sm, key, "paths")
}

func TestWalkForSecrets_DecimalNumberAsString(t *testing.T) {
	// Test: Decimal number as string doesn't match secret pattern
	yaml := `
_format_version: "3.0"
services:
  - name: test-svc
    port: "8080"
    timeout: "30.5"
`
	sm := BuildSecretMap(yaml)
	key := SimpleKey("service", "test-svc", "")
	assertNoSecretField(t, sm, key, "port")
	assertNoSecretField(t, sm, key, "timeout")
}

func TestWalkForSecrets_TemplateSyntaxVariations(t *testing.T) {
	// Test: Only lowercase 'env' is recognized - per deck template standard
	// The regex pattern is: env\s+"(DECK_[^"]*)"
	// This means ENV (uppercase) is NOT valid deck syntax and won't be detected
	yaml := `
_format_version: "3.0"
plugins:
  - name: test-plugin
    config:
      key1: ${{ env "DECK_KEY_1" }}
      key2: ${{ env   "DECK_KEY_2" }}
      key3: ${{ ENV "DECK_KEY_3" }}
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("test-plugin", "", "", "", "", "", "")
	// key1: standard format - DETECTED ✅
	assertSecretField(t, sm, key, "key1")
	// key2: extra spaces - DETECTED ✅ (regex allows \s+)
	assertSecretField(t, sm, key, "key2")
	// key3: uppercase ENV - NOT DETECTED ❌ (regex requires lowercase 'env')
	// This is correct behavior - uppercase ENV is not valid deck syntax
	assertNoSecretField(t, sm, key, "key3")
}

func TestWalkForSecrets_NestedArrayOfObjects(t *testing.T) {
	// Test: Array of objects with secret fields at different nesting levels
	// leafFieldName extracts "secret" field which contains the DECK_ values
	yaml := `
_format_version: "3.0"
plugins:
  - name: test-plugin
    config:
      items:
        - name: item1
          secret: ${{ env "DECK_SECRET_1" }}
        - name: item2
          secret: ${{ env "DECK_SECRET_2" }}
`
	sm := BuildSecretMap(yaml)
	key := PluginKey("test-plugin", "", "", "", "", "", "")
	assertSecretField(t, sm, key, "secret")
}

// ============================================================================
// 4-Level Deep Nesting Tests (Service -> Routes -> Plugins -> Config)
// ============================================================================

func TestBuildSecretMap_FourLevelDeep_ServiceRoutePluginConfig(t *testing.T) {
	// Test: 4-level deep nesting detection
	// Level 1: Service
	// Level 2: Route inside Service
	// Level 3: Plugin inside Route
	// Level 4: Config.minute inside Plugin
	//
	// YAML structure:
	// services:
	//   - name: service1
	//     routes:
	//       - name: test
	//         plugins:
	//           - name: rate-limiting
	//             config:
	//               minute: ${{ env "DECK_RATE_LIMIT" }}  <- 4 levels deep

	yaml := `
_format_version: "3.0"
services:
  - name: service1
    host: mockbin.org
    port: 8080
    routes:
      - name: test
        methods:
          - GET
        paths:
          - /test
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_RATE_LIMIT" }}
              policy: local
`
	sm := BuildSecretMap(yaml)

	// Plugin key: For route-scoped plugins, include route name
	// PluginKey(name, instanceName, service, route, consumer, consumerGroup, explicitID)
	key := PluginKey("rate-limiting", "", "", "test", "", "", "")

	// Verify the deep secret field is detected
	assertSecretField(t, sm, key, "minute")

	// Verify non-secret field is not marked as secret
	assertNoSecretField(t, sm, key, "policy")
}

func TestBuildSecretMap_FourLevelDeep_MultipleRoutesMultiplePlugins(t *testing.T) {
	// Test: Multiple routes with multiple plugins - verify all deep secrets detected
	//
	// Scenario:
	// - Service has 2 routes
	// - Route 1 has rate-limiting plugin with minute: ${{ env "X" }}
	// - Route 2 has oauth2 plugin with client_secret: ${{ env "Y" }}
	// Both should be detected at level 4

	yaml := `
_format_version: "3.0"
services:
  - name: api-service
    host: api.example.com
    routes:
      - name: route1
        paths:
          - /api/v1
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_RATE_LIMIT_1" }}
      - name: route2
        paths:
          - /api/v2
        plugins:
          - name: oauth2
            config:
              client_secret: ${{ env "DECK_OAUTH_SECRET" }}
              client_id: "public-id"
`
	sm := BuildSecretMap(yaml)

	// Check route1's rate-limiting plugin (route-scoped)
	key1 := PluginKey("rate-limiting", "", "", "route1", "", "", "")
	assertSecretField(t, sm, key1, "minute")
	assertNoSecretField(t, sm, key1, "client_secret") // Not in this plugin

	// Check route2's oauth2 plugin (route-scoped)
	key2 := PluginKey("oauth2", "", "", "route2", "", "", "")
	assertSecretField(t, sm, key2, "client_secret")
	assertNoSecretField(t, sm, key2, "minute")    // Not in this plugin
	assertNoSecretField(t, sm, key2, "client_id") // Non-secret field
}

func TestBuildSecretMap_FourLevelDeep_ServiceAndRoutePlugins(t *testing.T) {
	// Test: Mix of service-scoped and route-scoped plugins
	// Both should detect secrets at their respective nesting levels
	//
	// Scenario:
	// - Service has a top-level plugin: service.plugins[].config.key
	// - Route has a plugin: service.routes[].plugins[].config.minute
	// Both are marked as secrets

	yaml := `
_format_version: "3.0"
services:
  - name: secure-service
    host: api.example.com
    plugins:
      - name: key-auth
        config:
          key_names:
            - ${{ env "DECK_KEY_NAME" }}
    routes:
      - name: protected-route
        paths:
          - /protected
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_MINUTE_LIMIT" }}
`
	sm := BuildSecretMap(yaml)

	// Service-level plugin (level 2: service -> plugin.config.key_names)
	servicePluginKey := PluginKey("key-auth", "", "secure-service", "", "", "", "")
	assertSecretField(t, sm, servicePluginKey, "key_names")

	// Route-level plugin (level 4: service -> route -> plugin.config.minute)
	routePluginKey := PluginKey("rate-limiting", "", "", "protected-route", "", "", "")
	assertSecretField(t, sm, routePluginKey, "minute")
}

func TestBuildSecretMap_FourLevelDeep_NestedConfigFields(t *testing.T) {
	// Test: Deeply nested fields within plugin config
	// Plugin config can have nested objects with secrets
	//
	// Scenario:
	// service -> route -> plugin -> config -> redis (object) -> password (secret)

	yaml := `
_format_version: "3.0"
services:
  - name: cache-service
    host: api.example.com
    routes:
      - name: cache-route
        paths:
          - /cache
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_RATE_LIMIT" }}
              redis:
                host: redis.example.com
                port: 6379
                password: ${{ env "DECK_REDIS_PASSWORD" }}
`
	sm := BuildSecretMap(yaml)

	// Both minute and nested redis.password should be detected
	key := PluginKey("rate-limiting", "", "", "cache-route", "", "", "")
	assertSecretField(t, sm, key, "minute")
	// Note: nested config fields like redis.password would be detected
	// as the entire "password" field if it contains env vars
	assertSecretField(t, sm, key, "password")
}

// ============================================================================
// Secret Field Lifecycle Tests (Env Var -> Literal / Removed)
// ============================================================================

func TestBuildSecretMap_SecretFieldChange_EnvVarToLiteral(t *testing.T) {
	// Test: Secret field lifecycle - changes from env var to literal value
	//
	// Scenario:
	// Config 1 (old): minute: ${{ env "DECK_RATE_LIMIT" }}     <- DETECTED as secret
	// Config 2 (new): minute: 100                              <- NOT detected as secret

	oldYaml := `
_format_version: "3.0"
services:
  - name: api-service
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_RATE_LIMIT" }}
              policy: local
`

	newYaml := `
_format_version: "3.0"
services:
  - name: api-service
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              minute: 100
              policy: local
`

	oldSM := BuildSecretMap(oldYaml)
	newSM := BuildSecretMap(newYaml)

	key := PluginKey("rate-limiting", "", "", "test-route", "", "", "")

	// In old config: minute should be detected as secret (has env var)
	assertSecretField(t, oldSM, key, "minute")

	// In new config: minute should NOT be detected as secret (plain literal)
	assertNoSecretField(t, newSM, key, "minute")

	// Policy field should remain non-secret in both
	assertNoSecretField(t, oldSM, key, "policy")
	assertNoSecretField(t, newSM, key, "policy")
}

func TestBuildSecretMap_SecretFieldChange_EnvVarRemoved(t *testing.T) {
	// Test: Secret field is completely removed from config
	//
	// Scenario:
	// Config 1 (old): minute: ${{ env "DECK_RATE_LIMIT" }}  <- DETECTED
	// Config 2 (new): minute field removed completely        <- NOT in config

	oldYaml := `
_format_version: "3.0"
services:
  - name: api-service
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_RATE_LIMIT" }}
              policy: local
`

	newYaml := `
_format_version: "3.0"
services:
  - name: api-service
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              policy: local
			  minute: 20
`

	oldSM := BuildSecretMap(oldYaml)
	newSM := BuildSecretMap(newYaml)

	key := PluginKey("rate-limiting", "", "", "test-route", "", "", "")

	// In old config: minute should be detected as secret
	assertSecretField(t, oldSM, key, "minute")

	// In new config: minute should NOT be detected (field removed)
	assertNoSecretField(t, newSM, key, "minute")

	// Policy field should exist in both
	assertNoSecretField(t, oldSM, key, "policy")
	assertNoSecretField(t, newSM, key, "policy")
}

func TestBuildSecretMap_SecretFieldChange_AddingNewSecret(t *testing.T) {
	// Test: New secret field added to config
	//
	// Scenario:
	// Config 1 (old): Only has policy field
	// Config 2 (new): Adds minute: ${{ env "DECK_RATE_LIMIT" }}  <- NEW SECRET ADDED

	oldYaml := `
_format_version: "3.0"
services:
  - name: api-service
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              policy: local
`

	newYaml := `
_format_version: "3.0"
services:
  - name: api-service
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_RATE_LIMIT" }}
              policy: local
`

	oldSM := BuildSecretMap(oldYaml)
	newSM := BuildSecretMap(newYaml)

	key := PluginKey("rate-limiting", "", "", "test-route", "", "", "")

	// In old config: minute should NOT be detected (not present)
	assertNoSecretField(t, oldSM, key, "minute")

	// In new config: minute should be detected (has env var)
	assertSecretField(t, newSM, key, "minute")

	// Policy field should exist in both (non-secret)
	assertNoSecretField(t, oldSM, key, "policy")
	assertNoSecretField(t, newSM, key, "policy")
}

func TestBuildSecretMap_SecretFieldChange_MultipleSecrets_PartialRemoval(t *testing.T) {
	// Test: Complex scenario with multiple secret fields - some removed, some kept
	//
	// Scenario:
	// Config 1 (old): minute (secret), hour (secret), policy (non-secret)
	// Config 2 (new): hour (secret), policy (non-secret) - minute removed

	oldYaml := `
_format_version: "3.0"
services:
  - name: api-service
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              minute: ${{ env "DECK_RATE_LIMIT" }}
              hour: ${{ env "DECK_HOUR_LIMIT" }}
              policy: local
`

	newYaml := `
_format_version: "3.0"
services:
  - name: api-service
    routes:
      - name: test-route
        plugins:
          - name: rate-limiting
            config:
              hour: ${{ env "DECK_HOUR_LIMIT" }}
              policy: local
`

	oldSM := BuildSecretMap(oldYaml)
	newSM := BuildSecretMap(newYaml)

	key := PluginKey("rate-limiting", "", "", "test-route", "", "", "")

	// Old config: both minute and hour are secrets
	assertSecretField(t, oldSM, key, "minute")
	assertSecretField(t, oldSM, key, "hour")

	// New config: only hour is secret (minute was removed)
	assertNoSecretField(t, newSM, key, "minute")
	assertSecretField(t, newSM, key, "hour")

	// Policy should never be secret in both
	assertNoSecretField(t, oldSM, key, "policy")
	assertNoSecretField(t, newSM, key, "policy")
}
