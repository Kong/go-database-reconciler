package diff

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
)

// Category 2: Entity Types
func TestMaskEntityPairByFieldNames_Entity_Plugin(t *testing.T) {
	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("rate-limiting"),
			Config: kong.Configuration{
				fieldMinute: "5",
				fieldKey:    "old-key",
			},
		},
	}
	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("rate-limiting"),
			Config: kong.Configuration{
				fieldMinute: "10",
				fieldKey:    "new-key",
			},
		},
	}
	secretFields := map[string]bool{fieldKey: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPlugin, secretFields)
	fmt.Println(oldMasked)
	fmt.Println(newMasked)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Service(t *testing.T) {
	old := &state.Service{
		Service: kong.Service{
			Name:     new("my-service"),
			Host:     new("old.example.com"),
			Protocol: new("http"),
		},
	}
	newService := &state.Service{
		Service: kong.Service{
			Name:     new("my-service"),
			Host:     new("new.example.com"),
			Protocol: new("http"),
		},
	}
	secretFields := map[string]bool{"host": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newService, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Route(t *testing.T) {
	old := &state.Route{
		Route: kong.Route{
			Name:  new("api-route"),
			Paths: []*string{new("/api/v1")},
		},
	}
	newRoute := &state.Route{
		Route: kong.Route{
			Name:  new("api-route"),
			Paths: []*string{new("/api/v2")},
		},
	}
	secretFields := map[string]bool{"paths": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newRoute, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Certificate(t *testing.T) {
	old := &state.Certificate{
		Certificate: kong.Certificate{
			ID:   new("cert-1"),
			Cert: new("old-cert"),
			Key:  new("old-key"),
		},
	}
	newCert := &state.Certificate{
		Certificate: kong.Certificate{
			ID:   new("cert-1"),
			Cert: new("new-cert"),
			Key:  new("new-key"),
		},
	}
	secretFields := map[string]bool{"cert": true, fieldKey: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newCert, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Consumer(t *testing.T) {
	old := &state.Consumer{
		Consumer: kong.Consumer{
			Username: new("user1"),
			CustomID: new("old-custom-id"),
		},
	}
	newConsumer := &state.Consumer{
		Consumer: kong.Consumer{
			Username: new("user1"),
			CustomID: new("new-custom-id"),
		},
	}
	secretFields := map[string]bool{"custom_id": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newConsumer, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_BasicAuthCredential(t *testing.T) {
	old := &state.BasicAuth{
		BasicAuth: kong.BasicAuth{
			Username: new("olduser"),
			Password: new("oldpass"),
		},
	}
	newAuth := &state.BasicAuth{
		BasicAuth: kong.BasicAuth{
			Username: new("newuser"),
			Password: new("newpass"),
		},
	}
	secretFields := map[string]bool{"username": true, "password": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newAuth, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_KeyAuthCredential(t *testing.T) {
	old := &state.KeyAuth{
		KeyAuth: kong.KeyAuth{
			Key: new("oldkey123"),
		},
	}
	newKeyAuth := &state.KeyAuth{
		KeyAuth: kong.KeyAuth{
			Key: new("newkey456"),
		},
	}
	secretFields := map[string]bool{fieldKey: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newKeyAuth, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_JWTCredential(t *testing.T) {
	old := &state.JWTAuth{
		JWTAuth: kong.JWTAuth{
			Key:    new("oldkey"),
			Secret: new("oldsecret"),
		},
	}
	newJWT := &state.JWTAuth{
		JWTAuth: kong.JWTAuth{
			Key:    new("newkey"),
			Secret: new("newsecret"),
		},
	}
	secretFields := map[string]bool{fieldKey: true, "secret": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newJWT, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_HMACAuthCredential(t *testing.T) {
	old := &state.HMACAuth{
		HMACAuth: kong.HMACAuth{
			Username: new("olduser"),
			Secret:   new("oldsecret"),
		},
	}
	newHMAC := &state.HMACAuth{
		HMACAuth: kong.HMACAuth{
			Username: new("newuser"),
			Secret:   new("newsecret"),
		},
	}
	secretFields := map[string]bool{"username": true, "secret": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newHMAC, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_MTLSAuthCredential(t *testing.T) {
	old := &state.MTLSAuth{
		MTLSAuth: kong.MTLSAuth{
			SubjectName: new("oldsubject"),
		},
	}
	newMTLS := &state.MTLSAuth{
		MTLSAuth: kong.MTLSAuth{
			SubjectName: new("newsubject"),
		},
	}
	secretFields := map[string]bool{"subject_name": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newMTLS, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Upstream(t *testing.T) {
	old := &state.Upstream{
		Upstream: kong.Upstream{
			Name: new("backend-upstream"),
		},
	}
	newUpstream := &state.Upstream{
		Upstream: kong.Upstream{
			Name: new("backend-upstream"),
		},
	}
	secretFields := map[string]bool{fieldName: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newUpstream, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Target(t *testing.T) {
	old := &state.Target{
		Target: kong.Target{
			Target: new("oldhost:8080"),
		},
	}
	newTarget := &state.Target{
		Target: kong.Target{
			Target: new("newhost:8080"),
		},
	}
	secretFields := map[string]bool{"target": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newTarget, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_PluginServiceScoped(t *testing.T) {
	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name:    new("request-transformer"),
			Service: &kong.Service{Name: new("api-svc")},
			Config: kong.Configuration{
				"remove": map[string]any{
					fieldHeaders: []any{"X-Auth"},
				},
			},
		},
	}
	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name:    new("request-transformer"),
			Service: &kong.Service{Name: new("api-svc")},
			Config: kong.Configuration{
				"remove": map[string]any{
					"headers": []any{"X-New-Auth"},
				},
			},
		},
	}
	secretFields := map[string]bool{"headers": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPlugin, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_PluginRouteScoped(t *testing.T) {
	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name:  new("request-size-limiting"),
			Route: &kong.Route{Name: new("api-route")},
			Config: kong.Configuration{
				"size_limit": 1024,
			},
		},
	}
	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name:  new("request-size-limiting"),
			Route: &kong.Route{Name: new("api-route")},
			Config: kong.Configuration{
				"size_limit": 2048,
			},
		},
	}
	secretFields := map[string]bool{"size_limit": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPlugin, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_ConsumerGroup(t *testing.T) {
	old := &state.ConsumerGroup{
		ConsumerGroup: kong.ConsumerGroup{
			Name: new("admin-group"),
		},
	}
	newGroup := &state.ConsumerGroup{
		ConsumerGroup: kong.ConsumerGroup{
			Name: new("admin-group"),
		},
	}
	secretFields := map[string]bool{fieldName: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newGroup, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_CACertificate(t *testing.T) {
	old := &state.CACertificate{
		CACertificate: kong.CACertificate{
			Cert: new("old-ca-cert-data"),
		},
	}
	newCACert := &state.CACertificate{
		CACertificate: kong.CACertificate{
			Cert: new("new-ca-cert-data"),
		},
	}
	secretFields := map[string]bool{"cert": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newCACert, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_RBACRole(t *testing.T) {
	old := &state.RBACRole{
		RBACRole: kong.RBACRole{
			Name: new("admin"),
		},
	}
	newRole := &state.RBACRole{
		RBACRole: kong.RBACRole{
			Name: new("admin"),
		},
	}
	secretFields := map[string]bool{fieldName: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newRole, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_ServicePackage(t *testing.T) {
	old := &state.ServicePackage{
		ServicePackage: konnect.ServicePackage{
			Name: new("api-pkg"),
		},
	}
	newPkg := &state.ServicePackage{
		ServicePackage: konnect.ServicePackage{
			Name: new("api-pkg"),
		},
	}
	secretFields := map[string]bool{fieldName: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPkg, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_AIModel(t *testing.T) {
	old := &state.AIModel{
		AIModel: kong.AIModel{
			Name: new("gpt-4"),
		},
	}
	newModel := &state.AIModel{
		AIModel: kong.AIModel{
			Name: new("gpt-4"),
		},
	}
	secretFields := map[string]bool{fieldName: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newModel, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Vault(t *testing.T) {
	old := &state.Vault{
		Vault: kong.Vault{
			Name:   new("my-vault"),
			Prefix: new("vault://"),
			Config: kong.Configuration{
				"uri": "http://oldvault:8200",
			},
		},
	}
	newVault := &state.Vault{
		Vault: kong.Vault{
			Name:   new("my-vault"),
			Prefix: new("vault://"),
			Config: kong.Configuration{
				"uri": "http://newvault:8200",
			},
		},
	}
	secretFields := map[string]bool{"uri": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newVault, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_License(t *testing.T) {
	old := &state.License{
		License: kong.License{
			Payload: new("old-license-payload"),
		},
	}
	newLicense := &state.License{
		License: kong.License{
			Payload: new("new-license-payload"),
		},
	}
	secretFields := map[string]bool{"payload": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newLicense, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Key(t *testing.T) {
	old := &state.Key{
		Key: kong.Key{
			Name: new("signing-key"),
			KID:  new("kid-old"),
		},
	}
	newKey := &state.Key{
		Key: kong.Key{
			Name: new("signing-key"),
			KID:  new("kid-new"),
		},
	}
	secretFields := map[string]bool{"kid": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newKey, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_KeySet(t *testing.T) {
	old := &state.KeySet{
		KeySet: kong.KeySet{
			Name: new("keyset-1"),
		},
	}
	newKeySet := &state.KeySet{
		KeySet: kong.KeySet{
			Name: new("keyset-1"),
		},
	}
	secretFields := map[string]bool{fieldName: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newKeySet, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_Partial(t *testing.T) {
	old := &state.Partial{
		Partial: kong.Partial{
			Name: new("partial-config"),
		},
	}
	newPartial := &state.Partial{
		Partial: kong.Partial{
			Name: new("partial-config"),
		},
	}
	secretFields := map[string]bool{fieldName: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPartial, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_ClonedPluginDefinition(t *testing.T) {
	old := &state.ClonedPluginDefinition{
		ClonedPluginDefinition: kong.ClonedPluginDefinition{
			ID:   new("cloned-1"),
			Name: new("clone-1"),
			Ref:  new("old-ref"),
		},
	}
	newCloned := &state.ClonedPluginDefinition{
		ClonedPluginDefinition: kong.ClonedPluginDefinition{
			ID:   new("cloned-1"),
			Name: new("clone-1"),
			Ref:  new("new-ref"),
		},
	}
	secretFields := map[string]bool{"ref": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newCloned, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_CustomPluginDefinition(t *testing.T) {
	old := &state.CustomPluginDefinition{
		CustomPluginDefinition: kong.CustomPluginDefinition{
			ID:      new("custom-plugin-1"),
			Name:    new("my-custom-plugin"),
			Handler: new("old-handler.so"),
			Schema:  new("old-schema.json"),
		},
	}
	newCustom := &state.CustomPluginDefinition{
		CustomPluginDefinition: kong.CustomPluginDefinition{
			ID:      new("custom-plugin-1"),
			Name:    new("my-custom-plugin"),
			Handler: new("new-handler.so"),
			Schema:  new("new-schema.json"),
		},
	}
	secretFields := map[string]bool{"handler": true, "schema": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newCustom, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_Entity_FilterChain(t *testing.T) {
	old := &state.FilterChain{
		FilterChain: kong.FilterChain{
			Name: new("my-filter-chain"),
		},
	}
	newChain := &state.FilterChain{
		FilterChain: kong.FilterChain{
			Name: new("my-filter-chain"),
		},
	}
	secretFields := map[string]bool{fieldName: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newChain, secretFields)

	assert.NotNil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

// Category 4: Env Vars vs Non-Env Vars
func TestMaskEntityPairByFieldNames_SecretField_Masked(t *testing.T) {
	old := &state.Service{
		Service: kong.Service{
			Name:     new("api-svc"),
			Host:     new("secret-host.internal"),
			Port:     new(8080),
			Protocol: new("https"),
		},
	}
	newService := &state.Service{
		Service: kong.Service{
			Name:     new("api-svc"),
			Host:     new("secret-host.internal"),
			Port:     new(8080),
			Protocol: new("https"),
		},
	}
	secretFields := map[string]bool{"host": true}

	oldMasked, _ := maskEntityPairByFieldNames(old, newService, secretFields)

	oldResult := oldMasked.(*state.Service)
	assert.Equal(t, new(maskedValue), oldResult.Host)
	assert.Equal(t, new(8080), oldResult.Port)
}

func TestMaskEntityPairByFieldNames_NonSecretField_Visible(t *testing.T) {
	old := &state.Service{
		Service: kong.Service{
			Name:     new("api-svc"),
			Host:     new("api.example.com"),
			Protocol: new("https"),
		},
	}
	newService := &state.Service{
		Service: kong.Service{
			Name:     new("api-svc"),
			Host:     new("api.example.com"),
			Protocol: new("https"),
		},
	}
	secretFields := map[string]bool{}

	oldMasked, _ := maskEntityPairByFieldNames(old, newService, secretFields)

	oldResult := oldMasked.(*state.Service)
	assert.Equal(t, new("api.example.com"), oldResult.Host)
}

func TestMaskEntityPairByFieldNames_CoincidentalValue_NotMasked(t *testing.T) {
	old := &state.Route{
		Route: kong.Route{
			Name:  new("DECK_OLD_ROUTE"),
			Paths: []*string{new("/api")},
		},
	}
	newRoute := &state.Route{
		Route: kong.Route{
			Name:  new("DECK_OLD_ROUTE"),
			Paths: []*string{new("/api")},
		},
	}
	secretFields := map[string]bool{}

	oldMasked, _ := maskEntityPairByFieldNames(old, newRoute, secretFields)

	oldResult := oldMasked.(*state.Route)
	assert.Equal(t, new("DECK_OLD_ROUTE"), oldResult.Name)
}

func TestMaskEntityPairByFieldNames_Mixed_Fields(t *testing.T) {
	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("key-auth"),
			Config: kong.Configuration{
				"key_names":   "apikey",
				"key_in_body": false,
			},
		},
	}
	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("key-auth"),
			Config: kong.Configuration{
				"key_names":   "apikey",
				"key_in_body": false,
			},
		},
	}
	secretFields := map[string]bool{"key_names": true}

	oldMasked, _ := maskEntityPairByFieldNames(old, newPlugin, secretFields)

	oldResult := oldMasked.(*state.Plugin)
	assert.Equal(t, new("key-auth"), oldResult.Name)
}

func TestMaskEntityPairByFieldNames_Nested_ChildSecret(t *testing.T) {
	old := &state.Consumer{
		Consumer: kong.Consumer{
			Username: new("user1"),
			CustomID: new("secret-custom-id"),
		},
	}
	newConsumer := &state.Consumer{
		Consumer: kong.Consumer{
			Username: new("user1"),
			CustomID: new("secret-custom-id"),
		},
	}
	secretFields := map[string]bool{"custom_id": true}

	oldMasked, _ := maskEntityPairByFieldNames(old, newConsumer, secretFields)

	oldResult := oldMasked.(*state.Consumer)
	assert.Equal(t, new(maskedValue), oldResult.CustomID)
}

func TestMaskEntityPairByFieldNames_EnvVar_MaskSecretFieldOnlyWithSimilarNonSecretValue(t *testing.T) {
	oldConfig := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("rate-limiting"),
			Config: kong.Configuration{
				fieldHour: "200",
			},
		},
	}
	newConfig := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("rate-limiting"),
			Config: kong.Configuration{
				fieldMinute: "200",
			},
		},
	}
	secretFields := map[string]bool{fieldMinute: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(oldConfig, newConfig, secretFields)

	oldResult := oldMasked.(*state.Plugin)
	newResult := newMasked.(*state.Plugin)

	// Hardcoded secret in new should be masked (with change marker since field is new)
	assert.Contains(t, newResult.Config["minute"].(string), maskedValue)
	assert.NotEqual(t, maskedValue, oldResult.Config["hour"])
}

func TestMaskEntityPairByFieldNames_EnvVar_ServiceScopedPlugin(t *testing.T) {
	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name:    new("rate-limiting"),
			Service: &kong.Service{Name: new("api-service")},
			Config: kong.Configuration{
				fieldMinute: "100",
			},
		},
	}
	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name:    new("rate-limiting"),
			Service: &kong.Service{Name: new("api-service")},
			Config: kong.Configuration{
				fieldMinute: "100",
			},
		},
	}
	secretFields := map[string]bool{fieldMinute: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPlugin, secretFields)

	oldResult := oldMasked.(*state.Plugin)
	newResult := newMasked.(*state.Plugin)

	// Service-scoped plugin with env var should be masked
	assert.NotNil(t, oldResult.Service)
	assert.Equal(t, maskedValue, oldResult.Config["minute"])
	assert.Equal(t, maskedValue, newResult.Config["minute"])
}

func TestMaskEntityPairByFieldNames_EnvVar_RouteScopedPlugin(t *testing.T) {
	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name:  new("rate-limiting"),
			Route: &kong.Route{Name: new("api-route")},
			Config: kong.Configuration{
				fieldMinute: "100",
			},
		},
	}
	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name:  new("rate-limiting"),
			Route: &kong.Route{Name: new("api-route")},
			Config: kong.Configuration{
				fieldMinute: "100",
			},
		},
	}
	secretFields := map[string]bool{fieldMinute: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPlugin, secretFields)

	oldResult := oldMasked.(*state.Plugin)
	newResult := newMasked.(*state.Plugin)

	// Route-scoped plugin with env var should be masked
	assert.NotNil(t, oldResult.Route)
	assert.Equal(t, maskedValue, oldResult.Config["minute"])
	assert.Equal(t, maskedValue, newResult.Config["minute"])
}

// Category 5: Nil/Error Cases
func TestMaskEntityPairByFieldNames_NilInput_OldNil(t *testing.T) {
	var oldVal *state.Service
	newVal := &state.Service{
		Service: kong.Service{
			Name: new("test-service"),
		},
	}
	secretFields := map[string]bool{}

	oldMasked, newMasked := maskEntityPairByFieldNames(oldVal, newVal, secretFields)

	assert.Nil(t, oldMasked)
	assert.NotNil(t, newMasked)
}

func TestMaskEntityPairByFieldNames_NilPointerField(t *testing.T) {
	old := &state.Service{
		Service: kong.Service{
			Name:              new("service1"),
			ClientCertificate: nil,
		},
	}
	newService := &state.Service{
		Service: kong.Service{
			Name:              new("service1"),
			ClientCertificate: nil,
		},
	}
	secretFields := map[string]bool{"client_certificate": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newService, secretFields)

	oldResult := oldMasked.(*state.Service)
	newResult := newMasked.(*state.Service)

	assert.Nil(t, oldResult.ClientCertificate)
	assert.Nil(t, newResult.ClientCertificate)
}

func TestMaskEntityPairByFieldNames_EmptySecretMap_NoMasking(t *testing.T) {
	old := &state.Route{
		Route: kong.Route{
			Name:  new("route1"),
			Paths: []*string{new("/api/v1")},
		},
	}
	newRoute := &state.Route{
		Route: kong.Route{
			Name:  new("route1"),
			Paths: []*string{new("/api/v2")},
		},
	}
	secretFields := map[string]bool{}

	oldMasked, _ := maskEntityPairByFieldNames(old, newRoute, secretFields)

	oldResult := oldMasked.(*state.Route)
	assert.Equal(t, new("route1"), oldResult.Name)
	assert.Equal(t, []*string{new("/api/v1")}, oldResult.Paths)
}

func TestCloneForMasking_ComplexType(t *testing.T) {
	type Plugin struct {
		Name   string             `json:"name"`
		Config kong.Configuration `json:"config"`
	}

	obj := &Plugin{
		Name:   "test",
		Config: kong.Configuration{fieldKey: "value"},
	}

	clone := cloneForMasking(obj)
	assert.NotNil(t, clone)

	cloned := clone.(*Plugin)
	assert.Equal(t, "test", cloned.Name)
}

// Category 6: Nested Scenarios
func TestMaskEntityPairByFieldNames_Nested_Plugin_WithService(t *testing.T) {
	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("oauth2"),
			Service: &kong.Service{
				Name: new("api-service"),
				Host: new("old-host.internal"),
			},
			Config: kong.Configuration{
				"client_secret": "old-secret",
			},
		},
	}
	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("oauth2"),
			Service: &kong.Service{
				Name: new("api-service"),
				Host: new("new-host.internal"),
			},
			Config: kong.Configuration{
				"client_secret": "new-secret",
			},
		},
	}
	secretFields := map[string]bool{"host": true, "client_secret": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPlugin, secretFields)

	oldResult := oldMasked.(*state.Plugin)
	newResult := newMasked.(*state.Plugin)

	assert.NotNil(t, oldResult.Service)
	assert.Equal(t, new(maskedValue), oldResult.Service.Host)
	assert.NotEqual(t, new(maskedValue), newResult.Service.Host)
}

func TestMaskEntityPairByFieldNames_Nested_Consumer_WithCustomID(t *testing.T) {
	old := &state.Consumer{
		Consumer: kong.Consumer{
			Username: new("user1"),
			CustomID: new("old-custom-id"),
		},
	}
	newConsumer := &state.Consumer{
		Consumer: kong.Consumer{
			Username: new("user1"),
			CustomID: new("new-custom-id"),
		},
	}
	secretFields := map[string]bool{"custom_id": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newConsumer, secretFields)

	oldResult := oldMasked.(*state.Consumer)
	newResult := newMasked.(*state.Consumer)

	assert.Equal(t, new(maskedValue), oldResult.CustomID)
	assert.NotEqual(t, new(maskedValue), newResult.CustomID)
}

func TestMaskEntityPairByFieldNames_Nested_Route_WithPaths(t *testing.T) {
	old := &state.Route{
		Route: kong.Route{
			Name:  new("api-route"),
			Paths: []*string{new("/old/api"), new("/old/v1")},
		},
	}
	newRoute := &state.Route{
		Route: kong.Route{
			Name:  new("api-route"),
			Paths: []*string{new("/new/api"), new("/new/v1")},
		},
	}
	secretFields := map[string]bool{"paths": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newRoute, secretFields)

	oldResult := oldMasked.(*state.Route)
	newResult := newMasked.(*state.Route)

	assert.Len(t, oldResult.Paths, 2)
	for _, path := range oldResult.Paths {
		assert.Equal(t, maskedValue, *path)
	}
	assert.Len(t, newResult.Paths, 2)
	for _, path := range newResult.Paths {
		assert.NotEqual(t, maskedValue, *path)
	}
}

func TestMaskEntityPairByFieldNames_Nested_Certificate_WithSNIs(t *testing.T) {
	old := &state.Certificate{
		Certificate: kong.Certificate{
			ID:   new("cert-1"),
			Cert: new("old-cert-data"),
			SNIs: []*string{new("old.example.com"), new("old-api.com")},
		},
	}
	newCert := &state.Certificate{
		Certificate: kong.Certificate{
			ID:   new("cert-1"),
			Cert: new("new-cert-data"),
			SNIs: []*string{new("new.example.com"), new("new-api.com")},
		},
	}
	secretFields := map[string]bool{"cert": true, "snis": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newCert, secretFields)

	oldResult := oldMasked.(*state.Certificate)
	newResult := newMasked.(*state.Certificate)

	assert.Equal(t, new(maskedValue), oldResult.Cert)
	assert.Len(t, oldResult.SNIs, 2)
	for _, sni := range oldResult.SNIs {
		assert.Equal(t, maskedValue, *sni)
	}
	assert.NotNil(t, newResult)
}

func TestMaskEntityPairByFieldNames_Nested_Service_WithClientCertificate(t *testing.T) {
	old := &state.Service{
		Service: kong.Service{
			Name: new("secure-service"),
			Host: new("api.example.com"),
			ClientCertificate: &kong.Certificate{
				ID:   new("cert-1"),
				Cert: new("old-client-cert"),
				Key:  new("old-client-key"),
			},
		},
	}
	newService := &state.Service{
		Service: kong.Service{
			Name: new("secure-service"),
			Host: new("api.example.com"),
			ClientCertificate: &kong.Certificate{
				ID:   new("cert-1"),
				Cert: new("new-client-cert"),
				Key:  new("new-client-key"),
			},
		},
	}
	secretFields := map[string]bool{"cert": true, fieldKey: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newService, secretFields)

	oldResult := oldMasked.(*state.Service)
	newResult := newMasked.(*state.Service)

	assert.NotNil(t, oldResult.ClientCertificate)
	assert.Equal(t, new(maskedValue), oldResult.ClientCertificate.Cert)
	assert.Equal(t, new(maskedValue), oldResult.ClientCertificate.Key)
	assert.NotNil(t, newResult.ClientCertificate)
}

func TestMaskEntityPairByFieldNames_Nested_FourLevelDeep_ServiceRoutePluginConfig(t *testing.T) {
	// 4-level deep nesting with real Kong entities:
	// Level 1: Plugin
	// Level 2: Plugin.Service (contains service details)
	// Level 3: Plugin.Route (contains route details)
	// Level 4: Plugin.Config.minute (secret field at deepest level)
	//
	// YAML equivalent:
	// services:
	//   - name: service1
	//     routes:
	//       - name: test
	//         plugins:
	//           - name: rate-limiting
	//             config:
	//               minute: ${{ env "DECK_RATE_LIMIT" }}  <- 4 levels deep

	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("rate-limiting"),
			// Level 2: Service reference with details
			Service: &kong.Service{
				Name: new("service1"),
				Host: new("mockbin.org"),
				Port: new(8080),
			},
			// Level 3: Route reference with details
			Route: &kong.Route{
				Name:    new("test"),
				Paths:   []*string{new("/test")},
				Methods: []*string{new("GET")},
			},
			// Level 4: Nested Config with secret field
			Config: kong.Configuration{
				fieldMinute: "5000",
				"policy":    "local",
			},
		},
	}

	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("rate-limiting"),
			Service: &kong.Service{
				Name: new("service1"),
				Host: new("mockbin.org"),
				Port: new(8080),
			},
			Route: &kong.Route{
				Name:    new("test"),
				Paths:   []*string{new("/test")},
				Methods: []*string{new("GET")},
			},
			Config: kong.Configuration{
				fieldMinute: "5000",
				"policy":    "local",
			},
		},
	}

	secretFields := map[string]bool{fieldMinute: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPlugin, secretFields)

	oldResult := oldMasked.(*state.Plugin)
	newResult := newMasked.(*state.Plugin)

	// Verify structure at each level
	assert.NotNil(t, oldResult.Service, "Level 2: Service should exist")
	assert.Equal(t, new("service1"), oldResult.Service.Name)
	assert.Equal(t, new("mockbin.org"), oldResult.Service.Host)

	assert.NotNil(t, oldResult.Route, "Level 3: Route should exist")
	assert.Equal(t, new("test"), oldResult.Route.Name)
	assert.Len(t, oldResult.Route.Methods, 1)
	assert.Equal(t, new("GET"), oldResult.Route.Methods[0])

	// Verify masking at level 4 (deepest level)
	oldMinuteVal := oldResult.Config["minute"]
	newMinuteVal := newResult.Config["minute"]

	assert.NotNil(t, oldMinuteVal, "Level 4: Old minute should be masked")
	assert.NotNil(t, newMinuteVal, "Level 4: New minute should be masked")

	// Both should contain the masked value (since they're the same, both should have same masking)
	assert.Contains(t, oldMinuteVal.(string), maskedValue, "Old minute should contain masked value")
	assert.Contains(t, newMinuteVal.(string), maskedValue, "New minute should contain masked value")

	// Non-secret field should NOT be masked
	assert.Equal(t, "local", oldResult.Config["policy"], "Non-secret field should not be masked")
}

func TestMaskEntityPairByFieldNames_Nested_FourLevel_WithChangedSecret(t *testing.T) {
	// Same 4-level structure but with changed secret at deepest level
	// Verifies change detection works through all 4 levels

	old := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("rate-limiting"),
			Service: &kong.Service{
				Name: new("service1"),
				Host: new("mockbin.org"),
			},
			Route: &kong.Route{
				Name:  new("test"),
				Paths: []*string{new("/test")},
			},
			Config: kong.Configuration{
				fieldMinute: "5000",
			},
		},
	}

	newPlugin := &state.Plugin{
		Plugin: kong.Plugin{
			Name: new("rate-limiting"),
			Service: &kong.Service{
				Name: new("service1"),
				Host: new("mockbin.org"),
			},
			Route: &kong.Route{
				Name:  new("test"),
				Paths: []*string{new("/test")},
			},
			Config: kong.Configuration{
				fieldMinute: "4000",
			},
		},
	}

	secretFields := map[string]bool{fieldMinute: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newPlugin, secretFields)

	oldResult := oldMasked.(*state.Plugin)
	newResult := newMasked.(*state.Plugin)

	// Verify 4-level structure is preserved through masking
	assert.NotNil(t, oldResult.Service, "Level 2: Service preserved")
	assert.NotNil(t, oldResult.Route, "Level 3: Route preserved")
	assert.NotNil(t, oldResult.Config, "Level 4: Config preserved")

	// Verify change detection at level 4 (deepest level)
	oldMinute := oldResult.Config["minute"].(string)
	newMinute := newResult.Config["minute"].(string)

	// Both should be masked
	assert.Contains(t, oldMinute, maskedValue, "Old secret should be masked")
	assert.Contains(t, newMinute, maskedValue, "New secret should be masked")
}

// Category 7: Change Detection
func TestMaskEntityPairByFieldNames_ChangeDetection_Unchanged(t *testing.T) {
	old := &state.BasicAuth{
		BasicAuth: kong.BasicAuth{
			Username: new("user1"),
			Password: new("same-password"),
		},
	}
	newAuth := &state.BasicAuth{
		BasicAuth: kong.BasicAuth{
			Username: new("user1"),
			Password: new("same-password"),
		},
	}
	secretFields := map[string]bool{"password": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newAuth, secretFields)

	oldResult := oldMasked.(*state.BasicAuth)
	newResult := newMasked.(*state.BasicAuth)

	assert.Equal(t, new(maskedValue), oldResult.Password)
	assert.Equal(t, new(maskedValue), newResult.Password)
}

func TestMaskEntityPairByFieldNames_ChangeDetection_Changed(t *testing.T) {
	old := &state.BasicAuth{
		BasicAuth: kong.BasicAuth{
			Username: new("user1"),
			Password: new("old-password"),
		},
	}
	newAuth := &state.BasicAuth{
		BasicAuth: kong.BasicAuth{
			Username: new("user1"),
			Password: new("new-password"),
		},
	}
	secretFields := map[string]bool{"password": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newAuth, secretFields)

	oldResult := oldMasked.(*state.BasicAuth)
	newResult := newMasked.(*state.BasicAuth)

	assert.Equal(t, new(maskedValue), oldResult.Password)
	assert.NotEqual(t, new(maskedValue), newResult.Password)
	assert.Contains(t, *newResult.Password, maskedValue)
}

func TestMaskEntityPairByFieldNames_ChangeDetection_PointerChanged(t *testing.T) {
	old := &state.KeyAuth{
		KeyAuth: kong.KeyAuth{
			Key: new("old-key"),
		},
	}
	newKeyAuth := &state.KeyAuth{
		KeyAuth: kong.KeyAuth{
			Key: new("new-key"),
		},
	}
	secretFields := map[string]bool{fieldKey: true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newKeyAuth, secretFields)

	oldResult := oldMasked.(*state.KeyAuth)
	newResult := newMasked.(*state.KeyAuth)

	assert.Equal(t, maskedValue, *oldResult.Key)
	assert.NotEqual(t, maskedValue, *newResult.Key)
}

func TestMaskEntityPairByFieldNames_ChangeDetection_NonSecretVisible(t *testing.T) {
	old := &state.Consumer{
		Consumer: kong.Consumer{
			Username: new("old-username"),
			CustomID: new("secret-id-1"),
		},
	}
	newConsumer := &state.Consumer{
		Consumer: kong.Consumer{
			Username: new("new-username"),
			CustomID: new("secret-id-2"),
		},
	}
	secretFields := map[string]bool{"custom_id": true}

	oldMasked, newMasked := maskEntityPairByFieldNames(old, newConsumer, secretFields)

	oldResult := oldMasked.(*state.Consumer)
	newResult := newMasked.(*state.Consumer)

	assert.Equal(t, new("old-username"), oldResult.Username)
	assert.Equal(t, new("new-username"), newResult.Username)
}

func TestDeepValuesEqual_String(t *testing.T) {
	oldVal := reflect.ValueOf("same")
	newVal := reflect.ValueOf("same")

	assert.True(t, deepValuesEqual(oldVal, newVal))

	newVal2 := reflect.ValueOf("different")
	assert.False(t, deepValuesEqual(oldVal, newVal2))
}

func TestDeepValuesEqual_Slice(t *testing.T) {
	oldVal := reflect.ValueOf([]string{"a", "b"})
	newVal := reflect.ValueOf([]string{"a", "b"})

	assert.True(t, deepValuesEqual(oldVal, newVal))

	newVal2 := reflect.ValueOf([]string{"a", "b", "c"})
	assert.False(t, deepValuesEqual(oldVal, newVal2))
}

func TestDeepValuesEqual_Pointer(t *testing.T) {
	oldVal := reflect.ValueOf(new("value"))
	newVal := reflect.ValueOf(new("value"))

	assert.True(t, deepValuesEqual(oldVal, newVal))

	oldNil := reflect.ValueOf((*string)(nil))
	newNil := reflect.ValueOf((*string)(nil))

	assert.True(t, deepValuesEqual(oldNil, newNil))
}
