package file

import (
	"context"
	"encoding/hex"
	"math/rand"
	"os"
	"reflect"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultTimeout     = 60000
	defaultSlots       = 10000
	defaultWeight      = 100
	defaultConcurrency = 10
	testLimit          = "limit"
	testServiceID      = "fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd"
)

var (
	kong130Version  = semver.MustParse("1.3.0")
	kong340Version  = semver.MustParse("3.4.0")
	kong360Version  = semver.MustParse("3.6.0")
	kong370Version  = semver.MustParse("3.7.0")
	kong380Version  = semver.MustParse("3.8.0")
	kong390Version  = semver.MustParse("3.9.0")
	kong3100Version = semver.MustParse("3.10.0")
	kong3110Version = semver.MustParse("3.11.0")
)

var kongDefaults = KongDefaults{
	Service: &kong.Service{
		Protocol:       new("http"),
		ConnectTimeout: new(defaultTimeout),
		WriteTimeout:   new(defaultTimeout),
		ReadTimeout:    new(defaultTimeout),
	},
	Route: &kong.Route{
		PreserveHost:  new(false),
		RegexPriority: new(0),
		StripPath:     new(false),
		Protocols:     kong.StringSlice("http", "https"),
	},
	Upstream: &kong.Upstream{
		Slots: new(defaultSlots),
		Healthchecks: &kong.Healthcheck{
			Active: &kong.ActiveHealthcheck{
				Concurrency: new(defaultConcurrency),
				Healthy: &kong.Healthy{
					HTTPStatuses: []int{200, 302},
					Interval:     new(0),
					Successes:    new(0),
				},
				HTTPPath: new("/"),
				Type:     new("http"),
				Timeout:  new(1),
				Unhealthy: &kong.Unhealthy{
					HTTPFailures: new(0),
					TCPFailures:  new(0),
					Timeouts:     new(0),
					Interval:     new(0),
					HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
				},
			},
			Passive: &kong.PassiveHealthcheck{
				Healthy: &kong.Healthy{
					HTTPStatuses: []int{
						200, 201, 202, 203, 204, 205,
						206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
						306, 307, 308,
					},
					Successes: new(0),
				},
				Unhealthy: &kong.Unhealthy{
					HTTPFailures: new(0),
					TCPFailures:  new(0),
					Timeouts:     new(0),
					HTTPStatuses: []int{429, 500, 503},
				},
			},
		},
		HashOn:           new("none"),
		HashFallback:     new("none"),
		HashOnCookiePath: new("/"),
	},
	Target: &kong.Target{
		Weight: new(defaultWeight),
	},
}

var defaulterTestOpts = utils.DefaulterOpts{
	KongDefaults:           kongDefaults,
	DisableDynamicDefaults: false,
}

func emptyState() *state.KongState {
	s, _ := state.NewKongState()
	return s
}

func existingRouteState() *state.KongState {
	s, _ := state.NewKongState()
	s.Routes.Add(state.Route{
		Route: kong.Route{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: new("foo"),
		},
	})
	return s
}

func existingServiceState() *state.KongState {
	s, _ := state.NewKongState()
	s.Services.Add(state.Service{
		Service: kong.Service{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: new("foo"),
		},
	})
	return s
}

func existingConsumerCredState() *state.KongState {
	s, _ := state.NewKongState()
	s.Consumers.Add(state.Consumer{
		Consumer: kong.Consumer{
			ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Username: new("foo"),
		},
	})
	s.KeyAuths.Add(state.KeyAuth{
		KeyAuth: kong.KeyAuth{
			ID:  new("5f1ef1ea-a2a5-4a1b-adbb-b0d3434013e5"),
			Key: new("foo-apikey"),
			Consumer: &kong.Consumer{
				ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: new("foo"),
			},
		},
	})
	s.BasicAuths.Add(state.BasicAuth{
		BasicAuth: kong.BasicAuth{
			ID:       new("92f4c849-960b-43af-aad3-f307051408d3"),
			Username: new("basic-username"),
			Password: new("basic-password"),
			Consumer: &kong.Consumer{
				ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: new("foo"),
			},
		},
	})
	s.JWTAuths.Add(state.JWTAuth{
		JWTAuth: kong.JWTAuth{
			ID:     new("917b9402-1be0-49d2-b482-ca4dccc2054e"),
			Key:    new("jwt-key"),
			Secret: new("jwt-secret"),
			Consumer: &kong.Consumer{
				ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: new("foo"),
			},
		},
	})
	s.HMACAuths.Add(state.HMACAuth{
		HMACAuth: kong.HMACAuth{
			ID:       new("e5d81b73-bf9e-42b0-9d68-30a1d791b9c9"),
			Username: new("hmac-username"),
			Secret:   new("hmac-secret"),
			Consumer: &kong.Consumer{
				ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: new("foo"),
			},
		},
	})
	s.ACLGroups.Add(state.ACLGroup{
		ACLGroup: kong.ACLGroup{
			ID:    new("b7c9352a-775a-4ba5-9869-98e926a3e6cb"),
			Group: new("foo-group"),
			Consumer: &kong.Consumer{
				ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: new("foo"),
			},
		},
	})
	s.Oauth2Creds.Add(state.Oauth2Credential{
		Oauth2Credential: kong.Oauth2Credential{
			ID:       new("4eef5285-3d6a-4f6b-b659-8957a940e2ca"),
			ClientID: new("oauth2-clientid"),
			Name:     new("oauth2-name"),
			Consumer: &kong.Consumer{
				ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: new("foo"),
			},
		},
	})
	s.MTLSAuths.Add(state.MTLSAuth{
		MTLSAuth: kong.MTLSAuth{
			ID:          new("92f4c829-968b-42af-afd3-f337051508d3"),
			SubjectName: new("test@example.com"),
			Consumer: &kong.Consumer{
				ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: new("foo"),
			},
		},
	})
	return s
}

func existingUpstreamState() *state.KongState {
	s, _ := state.NewKongState()
	s.Upstreams.Add(state.Upstream{
		Upstream: kong.Upstream{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: new("foo"),
		},
	})
	return s
}

func existingCertificateState() *state.KongState {
	s, _ := state.NewKongState()
	s.Certificates.Add(state.Certificate{
		Certificate: kong.Certificate{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Cert: new("foo"),
			Key:  new("bar"),
		},
	})
	return s
}

func existingCertificateAndSNIState() *state.KongState {
	s, _ := state.NewKongState()
	s.Certificates.Add(state.Certificate{
		Certificate: kong.Certificate{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Cert: new("foo"),
			Key:  new("bar"),
		},
	})
	s.SNIs.Add(state.SNI{
		SNI: kong.SNI{
			ID:   new("a53e9598-3a5e-4c12-a672-71a4cdcf7a47"),
			Name: new("foo.example.com"),
			Certificate: &kong.Certificate{
				ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			},
		},
	})
	s.SNIs.Add(state.SNI{
		SNI: kong.SNI{
			ID:   new("5f8e6848-4cb9-479a-a27e-860e1a77f875"),
			Name: new("bar.example.com"),
			Certificate: &kong.Certificate{
				ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			},
		},
	})
	return s
}

func existingCACertificateState() *state.KongState {
	s, _ := state.NewKongState()
	s.CACertificates.Add(state.CACertificate{
		CACertificate: kong.CACertificate{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Cert: new("foo"),
		},
	})
	return s
}

func existingPluginState() *state.KongState {
	s, _ := state.NewKongState()
	s.Consumers.Add(state.Consumer{
		Consumer: kong.Consumer{
			ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
		},
	})
	s.Routes.Add(state.Route{
		Route: kong.Route{
			ID: new("700bc504-b2b1-4abd-bd38-cec92779659e"),
		},
	})
	s.ConsumerGroups.Add(state.ConsumerGroup{
		ConsumerGroup: kong.ConsumerGroup{
			ID:   new("69ed4618-a653-4b54-8bb6-dc33bd6fe048"),
			Name: new("test-group"),
		},
	})
	s.Plugins.Add(state.Plugin{
		Plugin: kong.Plugin{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: new("foo"),
		},
	})
	s.Plugins.Add(state.Plugin{
		Plugin: kong.Plugin{
			ID:   new("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
			Name: new("bar"),
			Consumer: &kong.Consumer{
				ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
			},
		},
	})
	s.Plugins.Add(state.Plugin{
		Plugin: kong.Plugin{
			ID:   new("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
			Name: new("foo"),
			Consumer: &kong.Consumer{
				ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
			},
			Route: &kong.Route{
				ID: new("700bc504-b2b1-4abd-bd38-cec92779659e"),
			},
			ConsumerGroup: &kong.ConsumerGroup{
				ID: new("69ed4618-a653-4b54-8bb6-dc33bd6fe048"),
			},
		},
	})
	return s
}

func existingScopedPluginState() *state.KongState {
	s, _ := state.NewKongState()

	s.Consumers.Add(state.Consumer{
		Consumer: kong.Consumer{
			ID: new("cID"),
		},
	})

	s.Services.Add(state.Service{
		Service: kong.Service{
			ID: new("sID"),
		},
	})

	s.Routes.Add(state.Route{
		Route: kong.Route{
			ID: new("rID"),
		},
	})

	s.ConsumerGroups.Add(state.ConsumerGroup{
		ConsumerGroup: kong.ConsumerGroup{
			ID:   new("cgID"),
			Name: new("foo"),
		},
	})

	s.Plugins.Add(state.Plugin{
		Plugin: kong.Plugin{
			ID:   new("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
			Name: new("foo"),
			Consumer: &kong.Consumer{
				ID: new("cID"),
			},
			Route: &kong.Route{
				ID: new("rID"),
			},
			ConsumerGroup: &kong.ConsumerGroup{
				ID: new("cgID"),
			},
			Service: &kong.Service{
				ID: new("sID"),
			},
		},
	})

	return s
}

func existingTargetsState() *state.KongState {
	s, _ := state.NewKongState()
	s.Targets.Add(state.Target{
		Target: kong.Target{
			ID:     new("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
			Target: new("bar"),
			Upstream: &kong.Upstream{
				ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
			},
		},
	})
	s.Targets.Add(state.Target{
		Target: kong.Target{
			ID:     new("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
			Target: new("foo"),
			Upstream: &kong.Upstream{
				ID: new("700bc504-b2b1-4abd-bd38-cec92779659e"),
			},
		},
	})
	return s
}

func existingDocumentState() *state.KongState {
	s, _ := state.NewKongState()
	s.ServicePackages.Add(state.ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: new("foo"),
		},
	})
	parent, _ := s.ServicePackages.Get("4bfcb11f-c962-4817-83e5-9433cf20b663")
	s.Documents.Add(state.Document{
		Document: konnect.Document{
			ID:        new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Path:      new("/foo.md"),
			Published: new(true),
			Content:   new("foo"),
			Parent:    parent,
		},
	})
	return s
}

func existingFilterChainState() *state.KongState {
	s, _ := state.NewKongState()
	s.FilterChains.Add(state.FilterChain{
		FilterChain: kong.FilterChain{
			Name:    new("my-service-chain"),
			ID:      new("fa7bd007-e0c6-4ef2-b254-e60d3a341b0c"),
			Enabled: new(true),
			Service: &kong.Service{
				ID: new("ba54b737-38aa-49d1-87c4-64e756b0c6f9"),
			},
			Filters: []*kong.Filter{
				{
					Name:    new("my-filter"),
					Config:  jsonRawMessage(`"config!"`),
					Enabled: new(false),
				},
			},
		},
	})
	s.FilterChains.Add(state.FilterChain{
		FilterChain: kong.FilterChain{
			Name:    new("my-route-chain"),
			ID:      new("ac6758a5-41d4-4493-827f-de9df5b75859"),
			Enabled: new(true),
			Route: &kong.Route{
				ID: new("ec9b7c35-8e95-4a7c-b0da-4fba8986d1cd"),
			},
			Filters: []*kong.Filter{
				{
					Name:    new("my-filter"),
					Config:  jsonRawMessage(`"config!"`),
					Enabled: new(false),
				},
			},
		},
	})

	return s
}

func existingDegraphqlRouteState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.DegraphqlRoutes.Add(
		state.DegraphqlRoute{
			DegraphqlRoute: kong.DegraphqlRoute{
				ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Service: &kong.Service{
					ID: new(testServiceID),
				},
				Methods: kong.StringSlice("GET"),
				URI:     new("/example"),
				Query:   new("query{ example { foo } }"),
			},
		})
	return s
}

func existingGqlCostDecorationState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.GraphqlRateLimitingCostDecorations.Add(
		state.GraphqlRateLimitingCostDecoration{
			GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
				ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				TypePath: new("Query.users"),
				Service: &kong.Service{
					ID: new(testServiceID),
				},
				AddConstant: kong.Float64(1),
				MulConstant: kong.Float64(1),
			},
		})
	return s
}

func existingPartialState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.Partials.Add(
		state.Partial{
			Partial: kong.Partial{
				ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Name: new("existing-partial"),
				Type: new("foo"),
				Config: kong.Configuration{
					"key1": "value1",
					"key2": []any{"a", "b", "c"},
					"key3": map[string]any{
						"k1": "v1",
						"k2": "v2",
						"k3": []any{"a1", "b1"},
					},
				},
			},
		})

	return s
}

func existingConsumerState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.Consumers.Add(state.Consumer{
		Consumer: kong.Consumer{
			ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Username: new("foo"),
		},
	})

	return s
}

func existingConsumerGroupState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.ConsumerGroups.Add(state.ConsumerGroup{
		ConsumerGroup: kong.ConsumerGroup{
			ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: new("foo"),
		},
	})

	return s
}

func existingKeyState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.Keys.Add(
		state.Key{
			Key: kong.Key{
				ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
				Name: new("foo"),
				KID:  new("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
				JWK:  new("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
			},
		})
	return s
}

func existingKeySetState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.KeySets.Add(
		state.KeySet{
			KeySet: kong.KeySet{
				ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
				Name: new("existing-set"),
			},
		})
	return s
}

func existingCustomPluginDefinitionState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.CustomPluginDefinitions.Add(
		state.CustomPluginDefinition{
			CustomPluginDefinition: kong.CustomPluginDefinition{
				ID:      new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
				Name:    new("my-plugin"),
				Schema:  new("return {}"),
				Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
			},
		})
	return s
}

var testRand *rand.Rand

var deterministicUUID = func() *string {
	version := byte(4)
	uuid := make([]byte, 16)
	testRand.Read(uuid)

	// Set version
	uuid[6] = (uuid[6] & 0x0f) | (version << 4)

	// Set variant
	uuid[8] = (uuid[8] & 0xbf) | 0x80

	buf := make([]byte, 36)
	var dash byte = '-'
	hex.Encode(buf[0:8], uuid[0:4])
	buf[8] = dash
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = dash
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = dash
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = dash
	hex.Encode(buf[24:], uuid[10:])
	s := string(buf)
	return &s
}

func TestMain(m *testing.M) {
	uuid = deterministicUUID
	os.Exit(m.Run())
}

func Test_stateBuilder_services(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		targetContent *Content
		currentState  *state.KongState
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "matches ID of an existing service",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("foo"),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:           new("foo"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
						Tags:           kong.StringSlice("tag1"),
					},
				},
			},
		},
		{
			name: "process a non-existent service",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("foo"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           new("foo"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
						Tags:           kong.StringSlice("tag1"),
					},
				},
			},
		},
		{
			name: "process a service with tls_sans",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name:     new("foo"),
								Protocol: new("https"),
								TLSSANs: &kong.SANs{
									DNSNames: kong.StringSlice("example.com"),
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:           new("foo"),
						Protocol:       new("https"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
						Tags:           kong.StringSlice("tag1"),
						TLSSANs: &kong.SANs{
							DNSNames: kong.StringSlice("example.com"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
				selectTags:    []string{"tag1"},
			}
			b.build()
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_ingestRoute(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState *state.KongState
	}
	type args struct {
		route FRoute
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantState *utils.KongRawState
	}{
		{
			name: "generates ID for a non-existing route",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				route: FRoute{
					Route: kong.Route{
						Name: new("foo"),
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						ID:            new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:          new("foo"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
					},
				},
			},
		},
		{
			name: "matches up IDs of routes correctly",
			fields: fields{
				currentState: existingRouteState(),
			},
			args: args{
				route: FRoute{
					Route: kong.Route{
						Name: new("foo"),
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						ID:            new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:          new("foo"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
					},
				},
			},
		},
		{
			name: "grpc route has strip_path=false",
			fields: fields{
				currentState: existingRouteState(),
			},
			args: args{
				route: FRoute{
					Route: kong.Route{
						Name:      new("foo"),
						Protocols: kong.StringSlice("grpc"),
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						ID:            new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:          new("foo"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("grpc"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				currentState: tt.fields.currentState,
			}
			b.rawState = &utils.KongRawState{}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.intermediate, _ = state.NewKongState()
			if err := b.ingestRoute(tt.args.route); (err != nil) != tt.wantErr {
				t.Errorf("stateBuilder.ingestPlugins() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantState, b.rawState)
		})
	}
}

func Test_stateBuilder_ingestTargets(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState *state.KongState
	}
	type args struct {
		targets []kong.Target
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantState *utils.KongRawState
	}{
		{
			name: "generates ID for a non-existing target",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				targets: []kong.Target{
					{
						Target: new("foo"),
						Upstream: &kong.Upstream{
							ID: new("952ddf37-e815-40b6-b119-5379a3b1f7be"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Target: new("foo"),
						Weight: new(100),
						Upstream: &kong.Upstream{
							ID: new("952ddf37-e815-40b6-b119-5379a3b1f7be"),
						},
					},
				},
			},
		},
		{
			name: "matches up IDs of Targets correctly",
			fields: fields{
				currentState: existingTargetsState(),
			},
			args: args{
				targets: []kong.Target{
					{
						Target: new("bar"),
						Upstream: &kong.Upstream{
							ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
						},
					},
					{
						Target: new("foo"),
						Upstream: &kong.Upstream{
							ID: new("700bc504-b2b1-4abd-bd38-cec92779659e"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     new("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
						Target: new("bar"),
						Weight: new(100),
						Upstream: &kong.Upstream{
							ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
						},
					},
					{
						ID:     new("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
						Target: new("foo"),
						Weight: new(100),
						Upstream: &kong.Upstream{
							ID: new("700bc504-b2b1-4abd-bd38-cec92779659e"),
						},
					},
				},
			},
		},
		{
			name: "expands IPv6 address and port correctly",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				targets: []kong.Target{
					{
						ID:     new("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: new("[2001:db8:fd73::e]:1326"),
						Upstream: &kong.Upstream{
							ID: new("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     new("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: new("[2001:0db8:fd73:0000:0000:0000:0000:000e]:1326"),
						Weight: new(100),
						Upstream: &kong.Upstream{
							ID: new("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
		},
		{
			name: "expands IPv6 address correctly",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				targets: []kong.Target{
					{
						ID:     new("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: new("::1"),
						Upstream: &kong.Upstream{
							ID: new("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     new("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: new("[0000:0000:0000:0000:0000:0000:0000:0001]:8000"),
						Weight: new(100),
						Upstream: &kong.Upstream{
							ID: new("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
		},
		{
			name: "expands IPv6 address with square brackets correctly",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				targets: []kong.Target{
					{
						ID:     new("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: new("[::1]"),
						Upstream: &kong.Upstream{
							ID: new("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     new("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: new("[0000:0000:0000:0000:0000:0000:0000:0001]:8000"),
						Weight: new(100),
						Upstream: &kong.Upstream{
							ID: new("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
		},
		{
			name: "handles invalid IPv6 address correctly",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				targets: []kong.Target{
					{
						Target: new("[invalid:ipv6::address]:1326"),
						Upstream: &kong.Upstream{
							ID: new("b1c2d3e4-f5a6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr:   true,
			wantState: &utils.KongRawState{},
		},
		{
			name: "handles invalid IPv6 address correctly",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				targets: []kong.Target{
					{
						Target: new("1:2:3:4"),
						Upstream: &kong.Upstream{
							ID: new("b1c2d3e4-f5a6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr:   true,
			wantState: &utils.KongRawState{},
		},
		{
			name: "handles invalid IPv6 address correctly",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				targets: []kong.Target{
					{
						Target: new("this:is:nuts:!"),
						Upstream: &kong.Upstream{
							ID: new("b1c2d3e4-f5a6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr:   true,
			wantState: &utils.KongRawState{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				currentState: tt.fields.currentState,
			}
			b.rawState = &utils.KongRawState{}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			if err := b.ingestTargets(tt.args.targets); (err != nil) != tt.wantErr {
				t.Errorf("stateBuilder.ingestTargets() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantState, b.rawState)
		})
	}
}

func Test_stateBuilder_ingestPlugins(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState *state.KongState
	}
	type args struct {
		plugins []FPlugin
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantState *utils.KongRawState
	}{
		{
			name: "generates ID for a non-existing plugin",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				plugins: []FPlugin{
					{
						Plugin: kong.Plugin{
							Name: new("foo"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Plugins: []*kong.Plugin{
					{
						ID:     new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:   new("foo"),
						Config: kong.Configuration{},
					},
				},
			},
		},
		{
			name: "matches up IDs of plugins correctly",
			fields: fields{
				currentState: existingPluginState(),
			},
			args: args{
				plugins: []FPlugin{
					{
						Plugin: kong.Plugin{
							Name: new("foo"),
						},
					},
					{
						Plugin: kong.Plugin{
							Name: new("bar"),
							Consumer: &kong.Consumer{
								ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
							},
						},
					},
					{
						Plugin: kong.Plugin{
							Name: new("foo"),
							Consumer: &kong.Consumer{
								ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
							},
							Route: &kong.Route{
								ID: new("700bc504-b2b1-4abd-bd38-cec92779659e"),
							},
							ConsumerGroup: &kong.ConsumerGroup{
								ID: new("69ed4618-a653-4b54-8bb6-dc33bd6fe048"),
							},
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Plugins: []*kong.Plugin{
					{
						ID:     new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:   new("foo"),
						Config: kong.Configuration{},
					},
					{
						ID:   new("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
						Name: new("bar"),
						Consumer: &kong.Consumer{
							ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
						},
						Config: kong.Configuration{},
					},
					{
						ID:   new("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
						Name: new("foo"),
						Consumer: &kong.Consumer{
							ID: new("f77ca8c7-581d-45a4-a42c-c003234228e1"),
						},
						Route: &kong.Route{
							ID: new("700bc504-b2b1-4abd-bd38-cec92779659e"),
						},
						ConsumerGroup: &kong.ConsumerGroup{
							ID: new("69ed4618-a653-4b54-8bb6-dc33bd6fe048"),
						},
						Config: kong.Configuration{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				currentState: tt.fields.currentState,
			}
			b.rawState = &utils.KongRawState{}
			if err := b.ingestPlugins(tt.args.plugins); (err != nil) != tt.wantErr {
				t.Errorf("stateBuilder.ingestPlugins() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantState, b.rawState)
		})
	}
}

func Test_pluginRelations(t *testing.T) {
	type args struct {
		plugin *kong.Plugin
	}
	tests := []struct {
		name         string
		args         args
		wantCID      string
		wantRID      string
		wantSID      string
		wantCGID     string
		currentState *state.KongState
	}{
		{
			name: "empty plugin - no relations",
			args: args{
				plugin: &kong.Plugin{
					Name: new("foo"),
				},
			},
			wantCID:      "",
			wantRID:      "",
			wantSID:      "",
			wantCGID:     "",
			currentState: emptyState(),
		},
		{
			name: "entities exist in state - returns IDs",
			args: args{
				plugin: &kong.Plugin{
					Name: new("foo"),
					Consumer: &kong.Consumer{
						ID: new("cID"),
					},
					Route: &kong.Route{
						ID: new("rID"),
					},
					Service: &kong.Service{
						ID: new("sID"),
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: new("cgID"),
					},
				},
			},
			wantCID:      "cID",
			wantRID:      "rID",
			wantSID:      "sID",
			wantCGID:     "cgID",
			currentState: existingScopedPluginState(),
		},
		{
			name: "invalid UUID for consumer - returns empty",
			args: args{
				plugin: &kong.Plugin{
					Name: new("foo"),
					Consumer: &kong.Consumer{
						ID: new("invalid-not-a-uuid"),
					},
				},
			},
			wantCID:      "",
			wantRID:      "",
			wantSID:      "",
			wantCGID:     "",
			currentState: emptyState(),
		},
		{
			name: "invalid UUID for route ",
			args: args{
				plugin: &kong.Plugin{
					Name: new("foo"),
					Route: &kong.Route{
						ID: new("not-a-valid-uuid-route"),
					},
				},
			},
			wantCID:      "",
			wantRID:      "",
			wantSID:      "",
			wantCGID:     "",
			currentState: emptyState(),
		},
		{
			name: "invalid UUID for service",
			args: args{
				plugin: &kong.Plugin{
					Name: new("foo"),
					Service: &kong.Service{
						ID: new("invalid-service-id"),
					},
				},
			},
			wantCID:      "",
			wantRID:      "",
			wantSID:      "",
			wantCGID:     "",
			currentState: emptyState(),
		},
		{
			name: "invalid UUID for consumer group ",
			args: args{
				plugin: &kong.Plugin{
					Name: new("foo"),
					ConsumerGroup: &kong.ConsumerGroup{
						ID: new("not-valid-cg-uuid"),
					},
				},
			},
			wantCID:      "",
			wantRID:      "",
			wantSID:      "",
			wantCGID:     "",
			currentState: emptyState(),
		},
		{
			name: "all invalid UUIDs ",
			args: args{
				plugin: &kong.Plugin{
					Name: new("foo"),
					Consumer: &kong.Consumer{
						ID: new("bad-consumer-id"),
					},
					Route: &kong.Route{
						ID: new("bad-route-id"),
					},
					Service: &kong.Service{
						ID: new("bad-service-id"),
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: new("bad-cg-id"),
					},
				},
			},
			wantCID:      "",
			wantRID:      "",
			wantSID:      "",
			wantCGID:     "",
			currentState: emptyState(),
		},
		{
			name: "valid UUID for external entity (not in state) - returns UUID (fallback)",
			args: args{
				plugin: &kong.Plugin{
					Name: new("foo"),
					Consumer: &kong.Consumer{
						ID: new("8ca63651-4068-4baa-b2b9-08dc99c29666"),
					},
					Route: &kong.Route{
						ID: new("9ca63651-4068-4baa-b2b9-08dc99c29777"),
					},
					Service: &kong.Service{
						ID: new("aca63651-4068-4baa-b2b9-08dc99c29888"),
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: new("bca63651-4068-4baa-b2b9-08dc99c29999"),
					},
				},
			},
			wantCID:      "8ca63651-4068-4baa-b2b9-08dc99c29666",
			wantRID:      "9ca63651-4068-4baa-b2b9-08dc99c29777",
			wantSID:      "aca63651-4068-4baa-b2b9-08dc99c29888",
			wantCGID:     "bca63651-4068-4baa-b2b9-08dc99c29999",
			currentState: emptyState(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intermediate, _ := state.NewKongState()
			b := &stateBuilder{
				currentState: tt.currentState,
				intermediate: intermediate,
			}
			gotCID, gotRID, gotSID, gotCGID := b.pluginRelations(tt.args.plugin)
			if gotCID != tt.wantCID {
				t.Errorf("pluginRelations() gotCID = %v, want %v", gotCID, tt.wantCID)
			}
			if gotRID != tt.wantRID {
				t.Errorf("pluginRelations() gotRID = %v, want %v", gotRID, tt.wantRID)
			}
			if gotSID != tt.wantSID {
				t.Errorf("pluginRelations() gotSID = %v, want %v", gotSID, tt.wantSID)
			}
			if gotCGID != tt.wantCGID {
				t.Errorf("pluginRelations() gotCGID = %v, want %v", gotCGID, tt.wantCGID)
			}
		})
	}
}

func Test_stateBuilder_ingestFilterChains(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState *state.KongState
	}
	type args struct {
		filterChains []FFilterChain
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantState *utils.KongRawState
	}{
		{
			name: "generates ID for a non-existing filter chain",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				filterChains: []FFilterChain{
					{
						FilterChain: kong.FilterChain{
							Name: new("my-filter-chain"),
							Service: &kong.Service{
								ID: new(testServiceID),
							},
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				FilterChains: []*kong.FilterChain{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("my-filter-chain"),
						Service: &kong.Service{
							ID: new(testServiceID),
						},
					},
				},
			},
		},
		{
			name: "matches up IDs of filter chains correctly",
			fields: fields{
				currentState: existingFilterChainState(),
			},
			args: args{
				filterChains: []FFilterChain{
					{
						FilterChain: kong.FilterChain{
							Service: &kong.Service{
								ID: new("ba54b737-38aa-49d1-87c4-64e756b0c6f9"),
							},
						},
					},
					{
						FilterChain: kong.FilterChain{
							Route: &kong.Route{
								ID: new("ec9b7c35-8e95-4a7c-b0da-4fba8986d1cd"),
							},
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				FilterChains: []*kong.FilterChain{
					{
						ID: new("fa7bd007-e0c6-4ef2-b254-e60d3a341b0c"),
						Service: &kong.Service{
							ID: new("ba54b737-38aa-49d1-87c4-64e756b0c6f9"),
						},
					},
					{
						ID: new("ac6758a5-41d4-4493-827f-de9df5b75859"),
						Route: &kong.Route{
							ID: new("ec9b7c35-8e95-4a7c-b0da-4fba8986d1cd"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				currentState: tt.fields.currentState,
			}
			b.rawState = &utils.KongRawState{}
			if err := b.ingestFilterChains(tt.args.filterChains); (err != nil) != tt.wantErr {
				t.Errorf("stateBuilder.ingestFilterChains() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantState, b.rawState)
		})
	}
}

func Test_stateBuilder_consumers(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
		kongVersion   *semver.Version
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "generates ID for a non-existing consumer",
			fields: fields{
				targetContent: &Content{
					Consumers: []FConsumer{
						{
							Consumer: kong.Consumer{
								Username: new("foo"),
							},
						},
					},
					Info: &Info{},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						ID:       new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Username: new("foo"),
					},
				},
			},
		},
		{
			name: "generates ID for a non-existing credential",
			fields: fields{
				targetContent: &Content{
					Consumers: []FConsumer{
						{
							Consumer: kong.Consumer{
								Username: new("foo"),
							},
							KeyAuths: []*kong.KeyAuth{
								{
									Key: new("foo-key"),
								},
							},
							BasicAuths: []*kong.BasicAuth{
								{
									Username: new("basic-username"),
									Password: new("basic-password"),
								},
							},
							HMACAuths: []*kong.HMACAuth{
								{
									Username: new("hmac-username"),
									Secret:   new("hmac-secret"),
								},
							},
							JWTAuths: []*kong.JWTAuth{
								{
									Key:    new("jwt-key"),
									Secret: new("jwt-secret"),
								},
							},
							Oauth2Creds: []*kong.Oauth2Credential{
								{
									ClientID: new("oauth2-clientid"),
									Name:     new("oauth2-name"),
								},
							},
							ACLGroups: []*kong.ACLGroup{
								{
									Group: new("foo-group"),
								},
							},
						},
					},
					Info: &Info{},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						ID:       new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Username: new("foo"),
					},
				},
				KeyAuths: []*kong.KeyAuth{
					{
						ID:  new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Key: new("foo-key"),
						Consumer: &kong.Consumer{
							ID:       new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: new("foo"),
						},
					},
				},
				BasicAuths: []*kong.BasicAuthOptions{
					{
						BasicAuth: kong.BasicAuth{
							ID:       new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
							Username: new("basic-username"),
							Password: new("basic-password"),
							Consumer: &kong.Consumer{
								ID:       new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
								Username: new("foo"),
							},
						},
					},
				},
				HMACAuths: []*kong.HMACAuth{
					{
						ID:       new("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Username: new("hmac-username"),
						Secret:   new("hmac-secret"),
						Consumer: &kong.Consumer{
							ID:       new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: new("foo"),
						},
					},
				},
				JWTAuths: []*kong.JWTAuth{
					{
						ID:     new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Key:    new("jwt-key"),
						Secret: new("jwt-secret"),
						Consumer: &kong.Consumer{
							ID:       new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: new("foo"),
						},
					},
				},
				Oauth2Creds: []*kong.Oauth2Credential{
					{
						ID:       new("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						ClientID: new("oauth2-clientid"),
						Name:     new("oauth2-name"),
						Consumer: &kong.Consumer{
							ID:       new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: new("foo"),
						},
					},
				},
				ACLGroups: []*kong.ACLGroup{
					{
						ID:    new("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Group: new("foo-group"),
						Consumer: &kong.Consumer{
							ID:       new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: new("foo"),
						},
					},
				},
				MTLSAuths: nil,
			},
		},
		{
			name: "matches ID of an existing consumer",
			fields: fields{
				targetContent: &Content{
					Consumers: []FConsumer{
						{
							Consumer: kong.Consumer{
								Username: new("foo"),
							},
						},
					},
				},
				currentState: existingConsumerCredState(),
			},
			want: &utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Username: new("foo"),
					},
				},
			},
		},
		{
			name: "matches ID of an existing credential",
			fields: fields{
				targetContent: &Content{
					Consumers: []FConsumer{
						{
							Consumer: kong.Consumer{
								Username: new("foo"),
							},
							KeyAuths: []*kong.KeyAuth{
								{
									Key: new("foo-apikey"),
								},
							},
							BasicAuths: []*kong.BasicAuth{
								{
									Username: new("basic-username"),
									Password: new("basic-password"),
								},
							},
							HMACAuths: []*kong.HMACAuth{
								{
									Username: new("hmac-username"),
									Secret:   new("hmac-secret"),
								},
							},
							JWTAuths: []*kong.JWTAuth{
								{
									Key:    new("jwt-key"),
									Secret: new("jwt-secret"),
								},
							},
							Oauth2Creds: []*kong.Oauth2Credential{
								{
									ClientID: new("oauth2-clientid"),
									Name:     new("oauth2-name"),
								},
							},
							ACLGroups: []*kong.ACLGroup{
								{
									Group: new("foo-group"),
								},
							},
							MTLSAuths: []*kong.MTLSAuth{
								{
									ID:          new("533c259e-bf71-4d77-99d2-97944c70a6a4"),
									SubjectName: new("test@example.com"),
								},
							},
						},
					},
					Info: &Info{},
				},
				currentState: existingConsumerCredState(),
			},
			want: &utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Username: new("foo"),
					},
				},
				KeyAuths: []*kong.KeyAuth{
					{
						ID:  new("5f1ef1ea-a2a5-4a1b-adbb-b0d3434013e5"),
						Key: new("foo-apikey"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				BasicAuths: []*kong.BasicAuthOptions{
					{
						BasicAuth: kong.BasicAuth{
							ID:       new("92f4c849-960b-43af-aad3-f307051408d3"),
							Username: new("basic-username"),
							Password: new("basic-password"),
							Consumer: &kong.Consumer{
								ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
								Username: new("foo"),
							},
						},
					},
				},
				HMACAuths: []*kong.HMACAuth{
					{
						ID:       new("e5d81b73-bf9e-42b0-9d68-30a1d791b9c9"),
						Username: new("hmac-username"),
						Secret:   new("hmac-secret"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				JWTAuths: []*kong.JWTAuth{
					{
						ID:     new("917b9402-1be0-49d2-b482-ca4dccc2054e"),
						Key:    new("jwt-key"),
						Secret: new("jwt-secret"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				Oauth2Creds: []*kong.Oauth2Credential{
					{
						ID:       new("4eef5285-3d6a-4f6b-b659-8957a940e2ca"),
						ClientID: new("oauth2-clientid"),
						Name:     new("oauth2-name"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				ACLGroups: []*kong.ACLGroup{
					{
						ID:    new("b7c9352a-775a-4ba5-9869-98e926a3e6cb"),
						Group: new("foo-group"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				MTLSAuths: []*kong.MTLSAuth{
					{
						ID:          new("533c259e-bf71-4d77-99d2-97944c70a6a4"),
						SubjectName: new("test@example.com"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
			},
		},
		{
			name: "does not inject tags if Kong version is older than 1.4",
			fields: fields{
				targetContent: &Content{
					Consumers: []FConsumer{
						{
							Consumer: kong.Consumer{
								Username: new("foo"),
							},
							KeyAuths: []*kong.KeyAuth{
								{
									Key: new("foo-apikey"),
								},
							},
							BasicAuths: []*kong.BasicAuth{
								{
									Username: new("basic-username"),
									Password: new("basic-password"),
								},
							},
							HMACAuths: []*kong.HMACAuth{
								{
									Username: new("hmac-username"),
									Secret:   new("hmac-secret"),
								},
							},
							JWTAuths: []*kong.JWTAuth{
								{
									Key:    new("jwt-key"),
									Secret: new("jwt-secret"),
								},
							},
							Oauth2Creds: []*kong.Oauth2Credential{
								{
									ClientID: new("oauth2-clientid"),
									Name:     new("oauth2-name"),
								},
							},
							ACLGroups: []*kong.ACLGroup{
								{
									Group: new("foo-group"),
								},
							},
							MTLSAuths: []*kong.MTLSAuth{
								{
									ID:          new("533c259e-bf71-4d77-99d2-97944c70a6a4"),
									SubjectName: new("test@example.com"),
								},
							},
						},
					},
					Info: &Info{},
				},
				currentState: existingConsumerCredState(),
				kongVersion:  &kong130Version,
			},
			want: &utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Username: new("foo"),
					},
				},
				KeyAuths: []*kong.KeyAuth{
					{
						ID:  new("5f1ef1ea-a2a5-4a1b-adbb-b0d3434013e5"),
						Key: new("foo-apikey"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				BasicAuths: []*kong.BasicAuthOptions{
					{
						BasicAuth: kong.BasicAuth{
							ID:       new("92f4c849-960b-43af-aad3-f307051408d3"),
							Username: new("basic-username"),
							Password: new("basic-password"),
							Consumer: &kong.Consumer{
								ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
								Username: new("foo"),
							},
						},
					},
				},
				HMACAuths: []*kong.HMACAuth{
					{
						ID:       new("e5d81b73-bf9e-42b0-9d68-30a1d791b9c9"),
						Username: new("hmac-username"),
						Secret:   new("hmac-secret"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				JWTAuths: []*kong.JWTAuth{
					{
						ID:     new("917b9402-1be0-49d2-b482-ca4dccc2054e"),
						Key:    new("jwt-key"),
						Secret: new("jwt-secret"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				Oauth2Creds: []*kong.Oauth2Credential{
					{
						ID:       new("4eef5285-3d6a-4f6b-b659-8957a940e2ca"),
						ClientID: new("oauth2-clientid"),
						Name:     new("oauth2-name"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				ACLGroups: []*kong.ACLGroup{
					{
						ID:    new("b7c9352a-775a-4ba5-9869-98e926a3e6cb"),
						Group: new("foo-group"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
				MTLSAuths: []*kong.MTLSAuth{
					{
						ID:          new("533c259e-bf71-4d77-99d2-97944c70a6a4"),
						SubjectName: new("test@example.com"),
						Consumer: &kong.Consumer{
							ID:       new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: new("foo"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
				kongVersion:   utils.Kong140Version,
			}
			if tt.fields.kongVersion != nil {
				b.kongVersion = *tt.fields.kongVersion
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_certificates(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "generates ID for a non-existing certificate",
			fields: fields{
				targetContent: &Content{
					Certificates: []FCertificate{
						{
							Cert: new("foo"),
							Key:  new("bar"),
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Certificates: []*kong.Certificate{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Cert: new("foo"),
						Key:  new("bar"),
					},
				},
			},
		},
		{
			name: "matches ID of an existing certificate",
			fields: fields{
				targetContent: &Content{
					Certificates: []FCertificate{
						{
							Cert: new("foo"),
							Key:  new("bar"),
						},
					},
				},
				currentState: existingCertificateState(),
			},
			want: &utils.KongRawState{
				Certificates: []*kong.Certificate{
					{
						ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Cert: new("foo"),
						Key:  new("bar"),
					},
				},
			},
		},
		{
			name: "generates ID for SNIs",
			fields: fields{
				targetContent: &Content{
					Certificates: []FCertificate{
						{
							Cert: new("foo"),
							Key:  new("bar"),
							SNIs: []kong.SNI{
								{
									Name: new("foo.example.com"),
								},
								{
									Name: new("bar.example.com"),
								},
							},
						},
					},
				},
				currentState: existingCertificateState(),
			},
			want: &utils.KongRawState{
				Certificates: []*kong.Certificate{
					{
						ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Cert: new("foo"),
						Key:  new("bar"),
					},
				},
				SNIs: []*kong.SNI{
					{
						ID:   new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name: new("foo.example.com"),
						Certificate: &kong.Certificate{
							ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
					{
						ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: new("bar.example.com"),
						Certificate: &kong.Certificate{
							ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
				},
			},
		},
		{
			name: "matches ID for SNIs",
			fields: fields{
				targetContent: &Content{
					Certificates: []FCertificate{
						{
							Cert: new("foo"),
							Key:  new("bar"),
							SNIs: []kong.SNI{
								{
									Name: new("foo.example.com"),
								},
								{
									Name: new("bar.example.com"),
								},
							},
						},
					},
				},
				currentState: existingCertificateAndSNIState(),
			},
			want: &utils.KongRawState{
				Certificates: []*kong.Certificate{
					{
						ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Cert: new("foo"),
						Key:  new("bar"),
					},
				},
				SNIs: []*kong.SNI{
					{
						ID:   new("a53e9598-3a5e-4c12-a672-71a4cdcf7a47"),
						Name: new("foo.example.com"),
						Certificate: &kong.Certificate{
							ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
					{
						ID:   new("5f8e6848-4cb9-479a-a27e-860e1a77f875"),
						Name: new("bar.example.com"),
						Certificate: &kong.Certificate{
							ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_caCertificates(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "generates ID for a non-existing CACertificate",
			fields: fields{
				targetContent: &Content{
					CACertificates: []FCACertificate{
						{
							CACertificate: kong.CACertificate{
								Cert: new("foo"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				CACertificates: []*kong.CACertificate{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Cert: new("foo"),
					},
				},
			},
		},
		{
			name: "matches ID of an existing CACertificate",
			fields: fields{
				targetContent: &Content{
					CACertificates: []FCACertificate{
						{
							CACertificate: kong.CACertificate{
								Cert: new("foo"),
							},
						},
					},
				},
				currentState: existingCACertificateState(),
			},
			want: &utils.KongRawState{
				CACertificates: []*kong.CACertificate{
					{
						ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Cert: new("foo"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_upstream(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		targetContent *Content
		currentState  *state.KongState
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "process a non-existent upstream",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  new("foo"),
								Slots: new(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Upstreams: []*kong.Upstream{
					{
						ID:    new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:  new("foo"),
						Slots: new(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           new("none"),
						HashFallback:     new("none"),
						HashOnCookiePath: new("/"),
					},
				},
			},
		},
		{
			name: "matches ID of an existing service",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name: new("foo"),
							},
						},
					},
				},
				currentState: existingUpstreamState(),
			},
			want: &utils.KongRawState{
				Upstreams: []*kong.Upstream{
					{
						ID:    new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:  new("foo"),
						Slots: new(10000),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           new("none"),
						HashFallback:     new("none"),
						HashOnCookiePath: new("/"),
					},
				},
			},
		},
		{
			name: "multiple upstreams are handled correctly",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name: new("foo"),
							},
						},
						{
							Upstream: kong.Upstream{
								Name: new("bar"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Upstreams: []*kong.Upstream{
					{
						ID:    new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:  new("foo"),
						Slots: new(10000),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           new("none"),
						HashFallback:     new("none"),
						HashOnCookiePath: new("/"),
					},
					{
						ID:    new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:  new("bar"),
						Slots: new(10000),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           new("none"),
						HashFallback:     new("none"),
						HashOnCookiePath: new("/"),
					},
				},
			},
		},
		{
			name: "upstream with new 3.0 fields",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  new("foo"),
								Slots: new(42),
								// not actually valid configuration, but this only needs to check that these translate
								// into the raw state
								HashOnQueryArg:         new("foo"),
								HashFallbackQueryArg:   new("foo"),
								HashOnURICapture:       new("foo"),
								HashFallbackURICapture: new("foo"),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Upstreams: []*kong.Upstream{
					{
						ID:    new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:  new("foo"),
						Slots: new(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:                 new("none"),
						HashFallback:           new("none"),
						HashOnCookiePath:       new("/"),
						HashOnQueryArg:         new("foo"),
						HashFallbackQueryArg:   new("foo"),
						HashOnURICapture:       new("foo"),
						HashFallbackURICapture: new("foo"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_documents(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		targetContent *Content
		currentState  *state.KongState
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KonnectRawState
	}{
		{
			name: "matches ID of an existing document",
			fields: fields{
				targetContent: &Content{
					ServicePackages: []FServicePackage{
						{
							Name: new("foo"),
							Document: &FDocument{
								Path:      new("/foo.md"),
								Published: new(true),
								Content:   new("foo"),
							},
						},
					},
				},
				currentState: existingDocumentState(),
			},
			want: &utils.KonnectRawState{
				Documents: []*konnect.Document{
					{
						ID:        new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Path:      new("/foo.md"),
						Published: new(true),
						Content:   new("foo"),
						Parent: &konnect.ServicePackage{
							ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Name: new("foo"),
						},
					},
				},
				ServicePackages: []*konnect.ServicePackage{
					{
						ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name: new("foo"),
					},
				},
			},
		},
		{
			name: "process a non-existent document",
			fields: fields{
				targetContent: &Content{
					ServicePackages: []FServicePackage{
						{
							Name: new("bar"),
							Document: &FDocument{
								Path:      new("/bar.md"),
								Published: new(true),
								Content:   new("bar"),
							},
						},
					},
				},
				currentState: existingDocumentState(),
			},
			want: &utils.KonnectRawState{
				Documents: []*konnect.Document{
					{
						ID:        new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Path:      new("/bar.md"),
						Published: new(true),
						Content:   new("bar"),
						Parent: &konnect.ServicePackage{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("bar"),
						},
					},
				},
				ServicePackages: []*konnect.ServicePackage{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("bar"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()
			assert.Equal(t, tt.want, b.konnectRawState)
		})
	}
}

func Test_stateBuilder_kong370(t *testing.T) {
	type fields struct {
		targetContent *Content
		currentState  *state.KongState
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "end to end test with all entities with kong version v3.7.0",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("foo-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: new("foo-route1"),
									},
								},
								{
									Route: kong.Route{
										ID:   new("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
										Name: new("foo-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: new("bar-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: new("bar-route1"),
									},
								},
								{
									Route: kong.Route{
										Name: new("bar-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: new("large-payload-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name:              new("dont-buffer-these"),
										RequestBuffering:  new(false),
										ResponseBuffering: new(false),
									},
								},
								{
									Route: kong.Route{
										Name:              new("buffer-these"),
										RequestBuffering:  new(true),
										ResponseBuffering: new(true),
									},
								},
							},
						},
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  new("foo"),
								Slots: new(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           new("foo-service"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
					{
						ID:             new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           new("bar-service"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
					{
						ID:             new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Name:           new("large-payload-service"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
				},
				Routes: []*kong.Route{
					{
						ID:            new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:          new("foo-route1"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-service"),
						},
					},
					{
						ID:            new("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
						Name:          new("foo-route2"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-service"),
						},
					},
					{
						ID:            new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:          new("bar-route1"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: new("bar-service"),
						},
					},
					{
						ID:            new("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:          new("bar-route2"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: new("bar-service"),
						},
					},
					{
						ID:            new("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						Name:          new("dont-buffer-these"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: new("large-payload-service"),
						},
						RequestBuffering:  new(false),
						ResponseBuffering: new(false),
					},
					{
						ID:            new("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Name:          new("buffer-these"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: new("large-payload-service"),
						},
						RequestBuffering:  new(true),
						ResponseBuffering: new(true),
					},
				},
				Upstreams: []*kong.Upstream{
					{
						ID:    new("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						Name:  new("foo"),
						Slots: new(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           new("none"),
						HashFallback:     new("none"),
						HashOnCookiePath: new("/"),
					},
				},
			},
		},
		{
			name: "entities with configurable defaults with kong version v3.7.0",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: KongDefaults{
							Route: &kong.Route{
								PathHandling:     new("v0"),
								PreserveHost:     new(false),
								RegexPriority:    new(0),
								StripPath:        new(false),
								Protocols:        kong.StringSlice("http", "https"),
								RequestBuffering: new(false),
							},
							Service: &kong.Service{
								Protocol:       new("https"),
								ConnectTimeout: new(5000),
								WriteTimeout:   new(5000),
								ReadTimeout:    new(5000),
							},
							Upstream: &kong.Upstream{
								Slots: new(100),
								Healthchecks: &kong.Healthcheck{
									Active: &kong.ActiveHealthcheck{
										Concurrency: new(5),
										Healthy: &kong.Healthy{
											HTTPStatuses: []int{200, 302},
											Interval:     new(0),
											Successes:    new(0),
										},
										HTTPPath: new("/"),
										Type:     new("http"),
										Timeout:  new(1),
										Unhealthy: &kong.Unhealthy{
											HTTPFailures: new(0),
											TCPFailures:  new(0),
											Timeouts:     new(0),
											Interval:     new(0),
											HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
										},
									},
									Passive: &kong.PassiveHealthcheck{
										Healthy: &kong.Healthy{
											HTTPStatuses: []int{
												200, 201, 202, 203, 204, 205,
												206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
												306, 307, 308,
											},
											Successes: new(0),
										},
										Unhealthy: &kong.Unhealthy{
											HTTPFailures: new(0),
											TCPFailures:  new(0),
											Timeouts:     new(0),
											HTTPStatuses: []int{429, 500, 503},
										},
									},
								},
								HashOn:           new("none"),
								HashFallback:     new("none"),
								HashOnCookiePath: new("/"),
							},
						},
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("foo-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: new("foo-route1"),
									},
								},
								{
									Route: kong.Route{
										ID:   new("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
										Name: new("foo-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: new("bar-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: new("bar-route1"),
									},
								},
								{
									Route: kong.Route{
										Name: new("bar-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: new("large-payload-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name:              new("dont-buffer-these"),
										RequestBuffering:  new(false),
										ResponseBuffering: new(false),
									},
								},
								{
									Route: kong.Route{
										Name:              new("buffer-these"),
										RequestBuffering:  new(true),
										ResponseBuffering: new(true),
									},
								},
							},
						},
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  new("foo"),
								Slots: new(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           new("foo-service"),
						Protocol:       new("https"),
						ConnectTimeout: new(5000),
						WriteTimeout:   new(5000),
						ReadTimeout:    new(5000),
					},
					{
						ID:             new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           new("bar-service"),
						Protocol:       new("https"),
						ConnectTimeout: new(5000),
						WriteTimeout:   new(5000),
						ReadTimeout:    new(5000),
					},
					{
						ID:             new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Name:           new("large-payload-service"),
						Protocol:       new("https"),
						ConnectTimeout: new(5000),
						WriteTimeout:   new(5000),
						ReadTimeout:    new(5000),
					},
				},
				Routes: []*kong.Route{
					{
						ID:               new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:             new("foo-route1"),
						PreserveHost:     new(false),
						RegexPriority:    new(0),
						StripPath:        new(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: new(false),
						PathHandling:     new("v0"),
						Service: &kong.Service{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-service"),
						},
					},
					{
						ID:               new("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
						Name:             new("foo-route2"),
						PreserveHost:     new(false),
						RegexPriority:    new(0),
						StripPath:        new(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: new(false),
						PathHandling:     new("v0"),
						Service: &kong.Service{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-service"),
						},
					},
					{
						ID:               new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:             new("bar-route1"),
						PreserveHost:     new(false),
						RegexPriority:    new(0),
						StripPath:        new(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: new(false),
						PathHandling:     new("v0"),
						Service: &kong.Service{
							ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: new("bar-service"),
						},
					},
					{
						ID:               new("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:             new("bar-route2"),
						PreserveHost:     new(false),
						RegexPriority:    new(0),
						StripPath:        new(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: new(false),
						PathHandling:     new("v0"),
						Service: &kong.Service{
							ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: new("bar-service"),
						},
					},
					{
						ID:            new("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						Name:          new("dont-buffer-these"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						PathHandling:  new("v0"),
						Service: &kong.Service{
							ID:   new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: new("large-payload-service"),
						},
						RequestBuffering:  new(false),
						ResponseBuffering: new(false),
					},
					{
						ID:            new("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Name:          new("buffer-these"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						PathHandling:  new("v0"),
						Service: &kong.Service{
							ID:   new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: new("large-payload-service"),
						},
						RequestBuffering:  new(true),
						ResponseBuffering: new(true),
					},
				},
				Upstreams: []*kong.Upstream{
					{
						ID:    new("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						Name:  new("foo"),
						Slots: new(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(5),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           new("none"),
						HashFallback:     new("none"),
						HashOnCookiePath: new("/"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testRand = rand.New(rand.NewSource(42))
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
				kongVersion:   kong370Version,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_kong360(t *testing.T) {
	type fields struct {
		targetContent *Content
		currentState  *state.KongState
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "end to end test with all entities kong version v3.6.0",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("foo-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: new("foo-route1"),
									},
								},
								{
									Route: kong.Route{
										ID:   new("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
										Name: new("foo-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: new("bar-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: new("bar-route1"),
									},
								},
								{
									Route: kong.Route{
										Name: new("bar-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: new("large-payload-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name:              new("dont-buffer-these"),
										RequestBuffering:  new(false),
										ResponseBuffering: new(false),
									},
								},
								{
									Route: kong.Route{
										Name:              new("buffer-these"),
										RequestBuffering:  new(true),
										ResponseBuffering: new(true),
									},
								},
							},
						},
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  new("foo"),
								Slots: new(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           new("foo-service"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
					{
						ID:             new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           new("bar-service"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
					{
						ID:             new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Name:           new("large-payload-service"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
				},
				Routes: []*kong.Route{
					{
						ID:            new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:          new("foo-route1"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-service"),
						},
					},
					{
						ID:            new("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
						Name:          new("foo-route2"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-service"),
						},
					},
					{
						ID:            new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:          new("bar-route1"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: new("bar-service"),
						},
					},
					{
						ID:            new("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:          new("bar-route2"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: new("bar-service"),
						},
					},
					{
						ID:            new("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						Name:          new("dont-buffer-these"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: new("large-payload-service"),
						},
						RequestBuffering:  new(false),
						ResponseBuffering: new(false),
					},
					{
						ID:            new("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Name:          new("buffer-these"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: new("large-payload-service"),
						},
						RequestBuffering:  new(true),
						ResponseBuffering: new(true),
					},
				},
				Upstreams: []*kong.Upstream{
					{
						ID:    new("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						Name:  new("foo"),
						Slots: new(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           new("none"),
						HashFallback:     new("none"),
						HashOnCookiePath: new("/"),
					},
				},
			},
		},
		{
			name: "entities with configurable defaults kong version v3.6.0",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: KongDefaults{
							Route: &kong.Route{
								PreserveHost:     new(false),
								StripPath:        new(false),
								Protocols:        kong.StringSlice("http", "https"),
								RequestBuffering: new(false),
							},
							Service: &kong.Service{
								Protocol:       new("https"),
								ConnectTimeout: new(5000),
								WriteTimeout:   new(5000),
								ReadTimeout:    new(5000),
							},
							Upstream: &kong.Upstream{
								Slots: new(100),
								Healthchecks: &kong.Healthcheck{
									Active: &kong.ActiveHealthcheck{
										Concurrency: new(5),
										Healthy: &kong.Healthy{
											HTTPStatuses: []int{200, 302},
											Interval:     new(0),
											Successes:    new(0),
										},
										HTTPPath: new("/"),
										Type:     new("http"),
										Timeout:  new(1),
										Unhealthy: &kong.Unhealthy{
											HTTPFailures: new(0),
											TCPFailures:  new(0),
											Timeouts:     new(0),
											Interval:     new(0),
											HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
										},
									},
									Passive: &kong.PassiveHealthcheck{
										Healthy: &kong.Healthy{
											HTTPStatuses: []int{
												200, 201, 202, 203, 204, 205,
												206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
												306, 307, 308,
											},
											Successes: new(0),
										},
										Unhealthy: &kong.Unhealthy{
											HTTPFailures: new(0),
											TCPFailures:  new(0),
											Timeouts:     new(0),
											HTTPStatuses: []int{429, 500, 503},
										},
									},
								},
								HashOn:           new("none"),
								HashFallback:     new("none"),
								HashOnCookiePath: new("/"),
							},
						},
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("foo-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: new("foo-route1"),
									},
								},
								{
									Route: kong.Route{
										ID:   new("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
										Name: new("foo-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: new("bar-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: new("bar-route1"),
									},
								},
								{
									Route: kong.Route{
										Name: new("bar-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: new("large-payload-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name:              new("dont-buffer-these"),
										RequestBuffering:  new(false),
										ResponseBuffering: new(false),
									},
								},
								{
									Route: kong.Route{
										Name:              new("buffer-these"),
										RequestBuffering:  new(true),
										ResponseBuffering: new(true),
									},
								},
							},
						},
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  new("foo"),
								Slots: new(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           new("foo-service"),
						Protocol:       new("https"),
						ConnectTimeout: new(5000),
						WriteTimeout:   new(5000),
						ReadTimeout:    new(5000),
					},
					{
						ID:             new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           new("bar-service"),
						Protocol:       new("https"),
						ConnectTimeout: new(5000),
						WriteTimeout:   new(5000),
						ReadTimeout:    new(5000),
					},
					{
						ID:             new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Name:           new("large-payload-service"),
						Protocol:       new("https"),
						ConnectTimeout: new(5000),
						WriteTimeout:   new(5000),
						ReadTimeout:    new(5000),
					},
				},
				Routes: []*kong.Route{
					{
						ID:               new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:             new("foo-route1"),
						PreserveHost:     new(false),
						RegexPriority:    new(0),
						StripPath:        new(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: new(false),
						Service: &kong.Service{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-service"),
						},
					},
					{
						ID:               new("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
						Name:             new("foo-route2"),
						PreserveHost:     new(false),
						RegexPriority:    new(0),
						StripPath:        new(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: new(false),
						Service: &kong.Service{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-service"),
						},
					},
					{
						ID:               new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:             new("bar-route1"),
						PreserveHost:     new(false),
						RegexPriority:    new(0),
						StripPath:        new(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: new(false),
						Service: &kong.Service{
							ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: new("bar-service"),
						},
					},
					{
						ID:               new("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:             new("bar-route2"),
						PreserveHost:     new(false),
						RegexPriority:    new(0),
						StripPath:        new(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: new(false),
						Service: &kong.Service{
							ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: new("bar-service"),
						},
					},
					{
						ID:            new("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						Name:          new("dont-buffer-these"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: new("large-payload-service"),
						},
						RequestBuffering:  new(false),
						ResponseBuffering: new(false),
					},
					{
						ID:            new("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Name:          new("buffer-these"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: new("large-payload-service"),
						},
						RequestBuffering:  new(true),
						ResponseBuffering: new(true),
					},
				},
				Upstreams: []*kong.Upstream{
					{
						ID:    new("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						Name:  new("foo"),
						Slots: new(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: new(5),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     new(0),
									Successes:    new(0),
								},
								HTTPPath: new("/"),
								Type:     new("http"),
								Timeout:  new(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									Interval:     new(0),
									HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
								},
							},
							Passive: &kong.PassiveHealthcheck{
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{
										200, 201, 202, 203, 204, 205,
										206, 207, 208, 226, 300, 301, 302, 303, 304, 305,
										306, 307, 308,
									},
									Successes: new(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: new(0),
									TCPFailures:  new(0),
									Timeouts:     new(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           new("none"),
						HashFallback:     new("none"),
						HashOnCookiePath: new("/"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testRand = rand.New(rand.NewSource(42))
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
				kongVersion:   kong360Version,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_fillPluginConfig(t *testing.T) {
	type fields struct {
		targetContent *Content
	}
	type args struct {
		plugin *FPlugin
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		result  FPlugin
	}{
		{
			name:    "nil arg throws an error",
			wantErr: true,
		},
		{
			name: "no _plugin_config throws an error",
			fields: fields{
				targetContent: &Content{},
			},
			args: args{
				plugin: &FPlugin{
					ConfigSource: new("foo"),
				},
			},
			wantErr: true,
		},
		{
			name: "no _plugin_config throws an error",
			fields: fields{
				targetContent: &Content{
					PluginConfigs: map[string]kong.Configuration{
						"foo": {
							"k2":  "v3",
							"k3:": "v3",
						},
					},
				},
			},
			args: args{
				plugin: &FPlugin{
					ConfigSource: new("foo"),
					Plugin: kong.Plugin{
						Config: kong.Configuration{
							"k1": "v1",
							"k2": "v2",
						},
					},
				},
			},
			result: FPlugin{
				ConfigSource: new("foo"),
				Plugin: kong.Plugin{
					Config: kong.Configuration{
						"k1":  "v1",
						"k2":  "v2",
						"k3:": "v3",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "nested object in plugin_config fills missing nested keys",
			fields: fields{
				targetContent: &Content{
					PluginConfigs: map[string]kong.Configuration{
						"redis-shared": {
							"redis": map[string]any{
								"host":     "redis.example.com",
								"port":     float64(6379),
								"password": "secret",
							},
						},
					},
				},
			},
			args: args{
				plugin: &FPlugin{
					ConfigSource: new("redis-shared"),
					Plugin: kong.Plugin{
						Config: kong.Configuration{
							"redis": map[string]any{
								"host": "override.example.com",
							},
						},
					},
				},
			},
			result: FPlugin{
				ConfigSource: new("redis-shared"),
				Plugin: kong.Plugin{
					Config: kong.Configuration{
						"redis": map[string]any{
							"host":     "override.example.com",
							"port":     float64(6379),
							"password": "secret",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "deeply nested object: plugin values win at every level",
			fields: fields{
				targetContent: &Content{
					PluginConfigs: map[string]kong.Configuration{
						"shared": {
							"level1": map[string]any{
								"level2": map[string]any{
									"a": "from-config",
									"b": "from-config",
								},
								"other": "from-config",
							},
						},
					},
				},
			},
			args: args{
				plugin: &FPlugin{
					ConfigSource: new("shared"),
					Plugin: kong.Plugin{
						Config: kong.Configuration{
							"level1": map[string]any{
								"level2": map[string]any{
									"a": "from-plugin",
								},
							},
						},
					},
				},
			},
			result: FPlugin{
				ConfigSource: new("shared"),
				Plugin: kong.Plugin{
					Config: kong.Configuration{
						"level1": map[string]any{
							"level2": map[string]any{
								"a": "from-plugin",
								"b": "from-config",
							},
							"other": "from-config",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "top-level keys missing in plugin are filled from config-source alongside nested merge",
			fields: fields{
				targetContent: &Content{
					PluginConfigs: map[string]kong.Configuration{
						"shared": {
							"redis": map[string]any{
								"host": "redis.example.com",
								"port": float64(6379),
							},
							testLimit: float64(100),
						},
					},
				},
			},
			args: args{
				plugin: &FPlugin{
					ConfigSource: new("shared"),
					Plugin: kong.Plugin{
						Config: kong.Configuration{
							"redis": map[string]any{
								"host": "override.example.com",
							},
						},
					},
				},
			},
			result: FPlugin{
				ConfigSource: new("shared"),
				Plugin: kong.Plugin{
					Config: kong.Configuration{
						"redis": map[string]any{
							"host": "override.example.com",
							"port": float64(6379),
						},
						testLimit: float64(100),
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
			}
			if err := b.fillPluginConfig(tt.args.plugin); (err != nil) != tt.wantErr {
				t.Errorf("stateBuilder.fillPluginConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.result, tt.args.plugin) {
				assert.Equal(t, tt.result, *tt.args.plugin)
			}
		})
	}
}

func Test_getStripPathBasedOnProtocols(t *testing.T) {
	tests := []struct {
		name              string
		route             kong.Route
		wantErr           bool
		expectedStripPath *bool
	}{
		{
			name: "true strip_path and grpc protocols",
			route: kong.Route{
				Protocols: []*string{new("grpc")},
				StripPath: new(true),
			},
			wantErr: true,
		},
		{
			name: "true strip_path and grpcs protocol",
			route: kong.Route{
				Protocols: []*string{new("grpcs")},
				StripPath: new(true),
			},
			wantErr: true,
		},
		{
			name: "no strip_path and http protocol",
			route: kong.Route{
				Protocols: []*string{new("http")},
			},
			expectedStripPath: nil,
		},
		{
			name: "no strip_path and grpc protocol",
			route: kong.Route{
				Protocols: []*string{new("grpc")},
			},
			expectedStripPath: new(false),
		},
		{
			name: "no strip_path and grpcs protocol",
			route: kong.Route{
				Protocols: []*string{new("grpcs")},
			},
			expectedStripPath: new(false),
		},
		{
			name: "false strip_path and grpc protocol",
			route: kong.Route{
				Protocols: []*string{new("grpc")},
				StripPath: new(false),
			},
			expectedStripPath: new(false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stripPath, err := getStripPathBasedOnProtocols(tt.route)
			if (err != nil) != tt.wantErr {
				t.Errorf("getStripPathBasedOnProtocols() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.expectedStripPath != nil {
				assert.Equal(t, *tt.expectedStripPath, *stripPath)
			} else {
				assert.Equal(t, tt.expectedStripPath, stripPath)
			}
		})
	}
}

func Test_stateBuilder_ingestRouteKonnectTraditionalRoute(t *testing.T) {
	assert := assert.New(t)
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState *state.KongState
	}
	type args struct {
		route FRoute
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantState *utils.KongRawState
	}{
		{
			name: "traditional route",
			fields: fields{
				currentState: emptyState(),
			},
			args: args{
				route: FRoute{
					Route: kong.Route{
						Name: new("foo"),
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						Name:          new("foo"),
						PreserveHost:  new(false),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		for _, isKonnect := range []bool{true, false} {
			t.Run(tt.name, func(t *testing.T) {
				ctx := context.Background()
				b := &stateBuilder{
					currentState: tt.fields.currentState,
					isKonnect:    isKonnect,
				}
				b.rawState = &utils.KongRawState{}
				d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
				b.defaulter = d
				b.intermediate, _ = state.NewKongState()
				if err := b.ingestRoute(tt.args.route); (err != nil) != tt.wantErr {
					t.Errorf("stateBuilder.ingestRoute() error = %v, wantErr %v", err, tt.wantErr)
				}

				// Not checking ID equality, as it is unnecessary for testing functionality
				b.rawState.Routes[0].ID = nil

				assert.Equal(tt.wantState, b.rawState)
				assert.NotNil(b.rawState.Routes[0].RegexPriority, "RegexPriority should not be nil")
			})
		}
	}
}

func Test_stateBuilder_expressionRoutes_kong360(t *testing.T) {
	assert := assert.New(t)
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		targetContent *Content
		currentState  *state.KongState
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "expression routes with kong version 3.6.0",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Routes: []FRoute{
						{
							Route: kong.Route{
								Name:       new("foo"),
								Expression: new(`'(http.path == "/test") || (http.path ^= "/test/")'`),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						Name:         new("foo"),
						PreserveHost: new(false),
						Expression:   new(`'(http.path == "/test") || (http.path ^= "/test/")'`),
						Priority:     kong.Uint64(0),
						StripPath:    new(false),
						Protocols:    kong.StringSlice("http", "https"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		for _, isKonnect := range []bool{true, false} {
			t.Run(tt.name, func(_ *testing.T) {
				ctx := context.Background()
				b := &stateBuilder{
					targetContent: tt.fields.targetContent,
					currentState:  tt.fields.currentState,
					kongVersion:   kong360Version,
					isKonnect:     isKonnect,
				}
				d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
				b.defaulter = d
				b.build()

				// Not checking ID equality, as it is unnecessary for testing functionality
				b.rawState.Routes[0].ID = nil

				assert.Equal(tt.want, b.rawState)
				assert.Nil(b.rawState.Routes[0].RegexPriority, "RegexPriority should be nil")
				assert.Nil(b.rawState.Routes[0].PathHandling, "PathHandling should be nil")
			})
		}
	}
}

func Test_stateBuilder_expressionRoutes_kong370(t *testing.T) {
	assert := assert.New(t)
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		targetContent *Content
		currentState  *state.KongState
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "expression routes with kong version 3.7.0",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Routes: []FRoute{
						{
							Route: kong.Route{
								Name:       new("foo"),
								Expression: new(`'(http.path == "/test") || (http.path ^= "/test/")'`),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						Name:          new("foo"),
						PreserveHost:  new(false),
						Expression:    new(`'(http.path == "/test") || (http.path ^= "/test/")'`),
						Priority:      kong.Uint64(0),
						RegexPriority: new(0),
						StripPath:     new(false),
						Protocols:     kong.StringSlice("http", "https"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
				kongVersion:   kong370Version,
				isKonnect:     false,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()

			// Not checking ID equality, as it is unnecessary for testing functionality
			b.rawState.Routes[0].ID = nil

			assert.Equal(tt.want, b.rawState)
			assert.NotNil(b.rawState.Routes[0].RegexPriority, "RegexPriority should not be nil")
		})
	}
}

func Test_stateBuilder_expressionRoutes_kong370_withKonnect(t *testing.T) {
	assert := assert.New(t)
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		targetContent *Content
		currentState  *state.KongState
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "expression routes with kong version 3.7.0",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					Routes: []FRoute{
						{
							Route: kong.Route{
								Name:       new("foo"),
								Expression: new(`'(http.path == "/test") || (http.path ^= "/test/")'`),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						Name:         new("foo"),
						PreserveHost: new(false),
						Expression:   new(`'(http.path == "/test") || (http.path ^= "/test/")'`),
						Priority:     kong.Uint64(0),
						StripPath:    new(false),
						Protocols:    kong.StringSlice("http", "https"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
				kongVersion:   kong370Version,
				isKonnect:     true,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()

			// Not checking ID equality, as it is unnecessary for testing functionality
			b.rawState.Routes[0].ID = nil

			assert.Equal(tt.want, b.rawState)
			assert.Nil(b.rawState.Routes[0].RegexPriority, "RegexPriority should be nil")
			assert.Nil(b.rawState.Routes[0].PathHandling, "PathHandling should be nil")
		})
	}
}

func Test_stateBuilder_ingestCustomEntities(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name      string
		fields    fields
		want      *utils.KongRawState
		wantErr   bool
		errString string
	}{
		{
			name: "generates a new degraphql route from valid config passed",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   new("/foo"),
								"query": new("query { foo { bar }}"),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				DegraphqlRoutes: []*kong.DegraphqlRoute{
					{
						ID:    new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						URI:   new("/foo"),
						Query: new("query { foo { bar }}"),
						Service: &kong.Service{
							ID: new(testServiceID),
						},
						Methods: kong.StringSlice("GET"),
					},
				},
			},
		},
		{
			name: "matches ID for an existing degraphql route",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   new("/example"),
								"query": new("query{ example { foo } }"),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: existingDegraphqlRouteState(t),
			},
			want: &utils.KongRawState{
				DegraphqlRoutes: []*kong.DegraphqlRoute{
					{
						ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Service: &kong.Service{
							ID: new(testServiceID),
						},
						Methods: kong.StringSlice("GET"),
						URI:     new("/example"),
						Query:   new("query{ example { foo } }"),
					},
				},
			},
		},
		{
			name: "accepts multi line query input and service name",
			fields: fields{
				targetContent: &Content{
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("foo"),
							},
						},
					},
					CustomEntities: []FCustomEntity{
						{
							Type: new("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri": new("/foo"),
								"query": new(`query SearchPosts($filters: PostsFilters) {
		      								posts(filter: $filters) {
		        								id
		        								title
		        								author
		      								}
										}`),
								primaryRelationService: map[string]any{
									"name": "foo",
								},
								"methods": kong.StringSlice("GET", "POST"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				DegraphqlRoutes: []*kong.DegraphqlRoute{
					{
						ID: new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Service: &kong.Service{
							ID: new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						},
						Methods: kong.StringSlice("GET", "POST"),
						URI:     new("/foo"),
						Query: new(`query SearchPosts($filters: PostsFilters) {
		      								posts(filter: $filters) {
		        								id
		        								title
		        								author
		      								}
										}`),
					},
				},
				Services: []*kong.Service{
					{
						ID:             new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:           new("foo"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
				},
			},
		},
		{
			name: "handles empty plugin entities",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				DegraphqlRoutes: nil,
			},
		},
		{
			name: "handles multiple degraphql routes",
			fields: fields{
				targetContent: &Content{
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("service1"),
							},
						},
						{
							Service: kong.Service{
								Name: new("service2"),
							},
						},
					},
					CustomEntities: []FCustomEntity{
						{
							Type: new("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   new("/foo"),
								"query": new("query { foo }"),
								primaryRelationService: map[string]any{
									"name": "service1",
								},
							},
						},
						{
							Type: new("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   new("/bar"),
								"query": new("query { bar }"),
								primaryRelationService: map[string]any{
									"name": "service2",
								},
								"methods": kong.StringSlice("POST", "PUT"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				DegraphqlRoutes: []*kong.DegraphqlRoute{
					{
						ID:    new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						URI:   new("/foo"),
						Query: new("query { foo }"),
						Service: &kong.Service{
							ID: new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						},
						Methods: kong.StringSlice("GET"),
					},
					{
						ID:    new("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						URI:   new("/bar"),
						Query: new("query { bar }"),
						Service: &kong.Service{
							ID: new("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						},
						Methods: kong.StringSlice("POST", "PUT"),
					},
				},
				Services: []*kong.Service{
					{
						ID:             new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:           new("service1"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
					{
						ID:             new("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:           new("service2"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
				},
			},
		},
		{
			name: "handles missing required fields - service",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   new("/foo"),
								"query": new("query{ example { foo } }"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				DegraphqlRoutes: nil,
			},
			wantErr:   true,
			errString: "service is required for degraphql_routes",
		},
		{
			name: "handles missing required fields - uri",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"query": new("query{ example { foo } }"),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				DegraphqlRoutes: nil,
			},
			wantErr:   true,
			errString: "uri and query are required for degraphql_routes",
		},
		{
			name: "handles missing required fields - uri",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri": new("/foo"),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				DegraphqlRoutes: nil,
			},
			wantErr:   true,
			errString: "uri and query are required for degraphql_routes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			_, _, err := b.build()
			if tt.wantErr {
				require.Error(t, err, "build error was expected")
				require.ErrorContains(t, err, tt.errString)
				assert.Equal(t, tt.want, b.rawState)
				return
			}

			require.NoError(t, err, "build error is not nil")
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_ConsumerGroupPolicyOverrides(t *testing.T) {
	assert := assert.New(t)
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name                           string
		isConsumerGroupPolicyOverrides bool
		diagnosticPolicy               utils.DiagnosticPolicy
		fields                         fields
		want                           *utils.KongRawState
		wantErr                        bool
	}{
		{
			name:                           "consumer-group policy overrides set as true",
			isConsumerGroupPolicyOverrides: true,
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults:                     kongDefaults,
						ConsumerGroupPolicyOverrides: true,
					},
					ConsumerGroups: []FConsumerGroupObject{
						{
							ConsumerGroup: kong.ConsumerGroup{
								Name: new("foo-group"),
							},
							Consumers: nil,
							Plugins: []*kong.ConsumerGroupPlugin{
								{
									Name: new("rate-limiting-advanced"),
									Config: kong.Configuration{
										testLimit:     []any{float64(100)},
										"window_size": []any{float64(60)},
										"window_type": string("fixed"),
									},
								},
							},
						},
					},
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("rate-limiting-advanced"),
								Config: kong.Configuration{
									"consumer_groups":         []any{string("foo-group")},
									"dictionary_name":         string("kong_rate_limiting_counters"),
									"disable_penalty":         bool(false),
									"enforce_consumer_groups": bool(true),
									"error_code":              float64(429),
									"error_message":           "API rate limit exceeded",
									"header_name":             nil,
									"hide_client_headers":     false,
									"identifier":              string("consumer"),
									testLimit:                 []any{float64(10)},
									"namespace":               string("ZEz47TWgUrv01HenyQBQa8io06MWsp0L"),
									"path":                    nil,
									"redis": map[string]any{
										"cluster_addresses":        nil,
										"cluster_max_redirections": float64(5),
										"cluster_nodes":            nil,
										"connect_timeout":          float64(2000),
										"connection_is_proxied":    bool(false),
										"database":                 float64(0),
										"host":                     string("127.0.0.5"),
										"keepalive_backlog":        nil,
										"keepalive_pool_size":      float64(256),
										"password":                 nil,
										"port":                     float64(6380),
										"read_timeout":             float64(2000),
										"send_timeout":             float64(2000),
										"sentinel_addresses":       nil,
										"sentinel_master":          string("mymaster"),
										"sentinel_nodes":           nil,
										"sentinel_password":        nil,
										"sentinel_role":            string("master"),
										"sentinel_username":        nil,
										"server_name":              nil,
										"ssl":                      bool(false),
										"ssl_verify":               bool(false),
										"timeout":                  float64(2000),
										"username":                 nil,
									},
									"retry_after_jitter_max": float64(0),
									"strategy":               string("redis"),
									"sync_rate":              float64(10),
									"window_size":            []any{float64(60)},
									"window_type":            string("fixed"),
								},
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-group"),
						},
						Consumers: nil,
						Plugins: []*kong.ConsumerGroupPlugin{
							{
								ID:   new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
								Name: new("rate-limiting-advanced"),
								Config: kong.Configuration{
									testLimit:     []any{float64(100)},
									"window_size": []any{float64(60)},
									"window_type": string("fixed"),
								},
							},
						},
					},
				},
				Plugins: []*kong.Plugin{
					{
						ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: new("rate-limiting-advanced"),
						Config: kong.Configuration{
							"consumer_groups":         []any{string("foo-group")},
							"dictionary_name":         string("kong_rate_limiting_counters"),
							"disable_penalty":         bool(false),
							"enforce_consumer_groups": bool(true),
							"error_code":              float64(429),
							"error_message":           "API rate limit exceeded",
							"header_name":             nil,
							"hide_client_headers":     false,
							"identifier":              string("consumer"),
							testLimit:                 []any{float64(10)},
							"namespace":               string("ZEz47TWgUrv01HenyQBQa8io06MWsp0L"),
							"path":                    nil,
							"redis": map[string]any{
								"cluster_addresses":        nil,
								"cluster_max_redirections": float64(5),
								"cluster_nodes":            nil,
								"connect_timeout":          float64(2000),
								"connection_is_proxied":    bool(false),
								"database":                 float64(0),
								"host":                     string("127.0.0.5"),
								"keepalive_backlog":        nil,
								"keepalive_pool_size":      float64(256),
								"password":                 nil,
								"port":                     float64(6380),
								"read_timeout":             float64(2000),
								"send_timeout":             float64(2000),
								"sentinel_addresses":       nil,
								"sentinel_master":          string("mymaster"),
								"sentinel_nodes":           nil,
								"sentinel_password":        nil,
								"sentinel_role":            string("master"),
								"sentinel_username":        nil,
								"server_name":              nil,
								"ssl":                      bool(false),
								"ssl_verify":               bool(false),
								"timeout":                  float64(2000),
								"username":                 nil,
							},
							"retry_after_jitter_max": float64(0),
							"strategy":               string("redis"),
							"sync_rate":              float64(10),
							"window_size":            []any{float64(60)},
							"window_type":            string("fixed"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:                           "consumer-group policy overrides set as false",
			isConsumerGroupPolicyOverrides: false,
			diagnosticPolicy: utils.NewDiagnosticPolicy(
				[]utils.DiagnosticCode{utils.DiagnosticCodeRLAConsumerGroups},
				nil,
			),
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					ConsumerGroups: []FConsumerGroupObject{
						{
							ConsumerGroup: kong.ConsumerGroup{
								Name: new("foo-group"),
							},
							Consumers: nil,
							Plugins: []*kong.ConsumerGroupPlugin{
								{
									Name: new("rate-limiting-advanced"),
									Config: kong.Configuration{
										testLimit:     []any{float64(100)},
										"window_size": []any{float64(60)},
										"window_type": string("fixed"),
									},
								},
							},
						},
					},
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("rate-limiting-advanced"),
								Config: kong.Configuration{
									"consumer_groups":         []any{string("foo-group")},
									"dictionary_name":         string("kong_rate_limiting_counters"),
									"disable_penalty":         bool(false),
									"enforce_consumer_groups": bool(true),
									"error_code":              float64(429),
									"error_message":           "API rate limit exceeded",
									"header_name":             nil,
									"hide_client_headers":     false,
									"identifier":              string("consumer"),
									testLimit:                 []any{float64(10)},
									"namespace":               string("ZEz47TWgUrv01HenyQBQa8io06MWsp0L"),
									"path":                    nil,
									"redis": map[string]any{
										"cluster_addresses":        nil,
										"cluster_max_redirections": float64(5),
										"cluster_nodes":            nil,
										"connect_timeout":          float64(2000),
										"connection_is_proxied":    bool(false),
										"database":                 float64(0),
										"host":                     string("127.0.0.5"),
										"keepalive_backlog":        nil,
										"keepalive_pool_size":      float64(256),
										"password":                 nil,
										"port":                     float64(6380),
										"read_timeout":             float64(2000),
										"send_timeout":             float64(2000),
										"sentinel_addresses":       nil,
										"sentinel_master":          string("mymaster"),
										"sentinel_nodes":           nil,
										"sentinel_password":        nil,
										"sentinel_role":            string("master"),
										"sentinel_username":        nil,
										"server_name":              nil,
										"ssl":                      bool(false),
										"ssl_verify":               bool(false),
										"timeout":                  float64(2000),
										"username":                 nil,
									},
									"retry_after_jitter_max": float64(0),
									"strategy":               string("redis"),
									"sync_rate":              float64(10),
									"window_size":            []any{float64(60)},
									"window_type":            string("fixed"),
								},
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent:                    tt.fields.targetContent,
				currentState:                     tt.fields.currentState,
				kongVersion:                      kong340Version,
				isConsumerGroupPolicyOverrideSet: tt.isConsumerGroupPolicyOverrides,
				diagnosticPolicy:                 tt.diagnosticPolicy,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			_, _, err := b.build()

			if tt.wantErr {
				require.Error(t, err, "build error was expected")
				assert.ErrorContains(err, utils.ErrorConsumerGroupUpgrade.Error())
				return
			}

			assert.Equal(tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_validateOpenIDConnectPlugin(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	tests := []struct {
		name       string
		target     *Content
		policy     utils.DiagnosticPolicy
		wantErr    string
		wantConfig kong.Configuration
	}{
		{
			name: "accepts required fields from plugin config source",
			target: &Content{
				PluginConfigs: map[string]kong.Configuration{
					"oidc": {
						"cache_tokens_salt": "cache-salt",
					},
				},
				Plugins: []FPlugin{
					{
						ConfigSource: new("oidc"),
						Plugin: kong.Plugin{
							Name: new("openid-connect"),
						},
					},
				},
			},
			wantConfig: kong.Configuration{
				"cache_tokens_salt": "cache-salt",
			},
		},
		{
			name: "rejects missing required fields",
			target: &Content{
				Plugins: []FPlugin{
					{
						Plugin: kong.Plugin{
							Name:   new("openid-connect"),
							Config: kong.Configuration{},
						},
					},
				},
			},
			wantErr: "openid-connect plugin requires explicit non-empty config values for cache_tokens_salt",
		},
		{
			name: "rejects nil required field values",
			target: &Content{
				Plugins: []FPlugin{
					{
						Plugin: kong.Plugin{
							Name: new("openid-connect"),
							Config: kong.Configuration{
								"cache_tokens_salt": nil,
							},
						},
					},
				},
			},
			wantErr: "openid-connect plugin requires explicit non-empty config values for cache_tokens_salt",
		},
		{
			name:   "allows downgrade of required field validation to warning",
			policy: utils.NewDiagnosticPolicy(nil, []utils.DiagnosticCode{utils.DiagnosticCodeOIDCMissingConfig}),
			target: &Content{
				Plugins: []FPlugin{
					{
						Plugin: kong.Plugin{
							Name: new("openid-connect"),
							Config: kong.Configuration{
								"cache_tokens_salt": "",
							},
						},
					},
				},
			},
			wantConfig: kong.Configuration{
				"cache_tokens_salt": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent:    tt.target,
				currentState:     emptyState(),
				kongVersion:      kong340Version,
				skipDefaults:     true,
				diagnosticPolicy: tt.policy,
			}

			_, _, err := b.build()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Len(t, b.rawState.Plugins, 1)
			assert.Equal(t, tt.wantConfig, b.rawState.Plugins[0].Config)
		})
	}
}

func Test_stateBuilder_validateRateLimitingAdvancedDiagnosticSeverity(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	target := &Content{
		Plugins: []FPlugin{
			{
				Plugin: kong.Plugin{
					Name: new("rate-limiting-advanced"),
					Config: kong.Configuration{
						"consumer_groups":         []any{"foo-group"},
						"enforce_consumer_groups": true,
					},
				},
			},
		},
	}

	t.Run("defaults to error", func(t *testing.T) {
		b := &stateBuilder{
			targetContent: target,
			currentState:  emptyState(),
			kongVersion:   kong340Version,
			skipDefaults:  true,
		}

		_, _, err := b.build()
		require.Error(t, err)
		assert.ErrorContains(t, err, utils.ErrorConsumerGroupUpgrade.Error())
	})

	t.Run("allows downgrade to warning when configured", func(t *testing.T) {
		b := &stateBuilder{
			targetContent:    target,
			currentState:     emptyState(),
			kongVersion:      kong340Version,
			skipDefaults:     true,
			diagnosticPolicy: utils.NewDiagnosticPolicy(nil, []utils.DiagnosticCode{utils.DiagnosticCodeRLAConsumerGroups}),
		}

		_, _, err := b.build()
		require.NoError(t, err)
		require.Len(t, b.rawState.Plugins, 1)
	})
}

func Test_stateBuilder_partials(t *testing.T) {
	assert := assert.New(t)
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "creates new partial when not found in current state",
			fields: fields{
				targetContent: &Content{
					Partials: []FPartial{
						{
							Partial: kong.Partial{
								Name: new("my-foo-partial"),
								Type: new("foo"),
								Config: kong.Configuration{
									"key1": "value1",
									"key2": []any{"a", "b", "c"},
									"key3": map[string]any{
										"k1": "v1",
										"k2": "v2",
										"k3": []any{"a1", "b1"},
									},
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Partials: []*kong.Partial{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("my-foo-partial"),
						Type: new("foo"),
						Config: kong.Configuration{
							"key1": "value1",
							"key2": []any{"a", "b", "c"},
							"key3": map[string]any{
								"k1": "v1",
								"k2": "v2",
								"k3": []any{"a1", "b1"},
							},
						},
					},
				},
			},
		},
		{
			name: "uses existing partial ID when found in current state",
			fields: fields{
				targetContent: &Content{
					Partials: []FPartial{
						{
							Partial: kong.Partial{
								Name: new("existing-partial"),
								Type: new("foo"),
								Config: kong.Configuration{
									"key1": "value1",
									"key2": []any{"a", "b", "c"},
									"key3": map[string]any{
										"k1": "v1",
										"k2": "v2",
										"k3": []any{"a1", "b1"},
									},
								},
							},
						},
					},
				},
				currentState: existingPartialState(t),
			},
			want: &utils.KongRawState{
				Partials: []*kong.Partial{
					{
						ID:   new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name: new("existing-partial"),
						Type: new("foo"),
						Config: kong.Configuration{
							"key1": "value1",
							"key2": []any{"a", "b", "c"},
							"key3": map[string]any{
								"k1": "v1",
								"k2": "v2",
								"k3": []any{"a1", "b1"},
							},
						},
					},
				},
			},
		},
		{
			name: "maintains provided ID if already set",
			fields: fields{
				targetContent: &Content{
					Partials: []FPartial{
						{
							Partial: kong.Partial{
								ID:   new("provided-id"),
								Name: new("test-partial"),
								Type: new("foo"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Partials: []*kong.Partial{
					{
						ID:   new("provided-id"),
						Name: new("test-partial"),
						Type: new("foo"),
					},
				},
			},
		},
		{
			name: "handles multiple partials if provided",
			fields: fields{
				targetContent: &Content{
					Partials: []FPartial{
						{
							Partial: kong.Partial{
								Name: new("foo-partial"),
								Type: new("foo"),
							},
						},
						{
							Partial: kong.Partial{
								Name: new("bar-partial"),
								Type: new("bar"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Partials: []*kong.Partial{
					{
						ID:   new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name: new("foo-partial"),
						Type: new("foo"),
					},
					{
						ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: new("bar-partial"),
						Type: new("bar"),
					},
				},
			},
		},
		{
			name: "handles empty partial entities",
			fields: fields{
				targetContent: &Content{
					Partials: []FPartial{},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Partials: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			ctx := context.Background()
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
				kongVersion:   kong3100Version,
			}
			d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
			b.defaulter = d
			b.build()

			assert.Equal(tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_plugins(t *testing.T) {
	assert := assert.New(t)
	testRand = rand.New(rand.NewSource(42))

	type fields struct {
		targetContent *Content
		intermediate  *state.KongState
	}
	tests := []struct {
		name    string
		fields  fields
		want    []FPlugin
		wantErr string
	}{
		{
			name: "processes plugin with consumer reference",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("key-auth"),
								Consumer: &kong.Consumer{
									ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
								},
							},
						},
					},
				},
				intermediate: existingConsumerState(t),
			},
			want: []FPlugin{
				{
					Plugin: kong.Plugin{
						Name: new("key-auth"),
						Consumer: &kong.Consumer{
							ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
				},
			},
		},
		{
			name: "processes plugin with service reference",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("rate-limiting"),
								Service: &kong.Service{
									ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
								},
							},
						},
					},
				},
				intermediate: existingServiceState(),
			},
			want: []FPlugin{
				{
					Plugin: kong.Plugin{
						Name: new("rate-limiting"),
						Service: &kong.Service{
							ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
				},
			},
		},
		{
			name: "processes plugin with route reference",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("cors"),
								Route: &kong.Route{
									ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
								},
							},
						},
					},
				},
				intermediate: existingRouteState(),
			},
			want: []FPlugin{
				{
					Plugin: kong.Plugin{
						Name: new("cors"),
						Route: &kong.Route{
							ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
				},
			},
		},
		{
			name: "processes plugin with consumer group reference",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("rate-limiting"),
								ConsumerGroup: &kong.ConsumerGroup{
									ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
								},
							},
						},
					},
				},
				intermediate: existingConsumerGroupState(t),
			},
			want: []FPlugin{
				{
					Plugin: kong.Plugin{
						Name: new("rate-limiting"),
						ConsumerGroup: &kong.ConsumerGroup{
							ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
				},
			},
		},
		{
			name: "processes plugin with partials",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("custom-plugin"),
								Partials: []*kong.PartialLink{
									{
										Partial: &kong.Partial{
											ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
										},
										Path: new("config.custom_path"),
									},
								},
							},
						},
					},
				},
				intermediate: existingPartialState(t),
			},
			want: []FPlugin{
				{
					Plugin: kong.Plugin{
						Name: new("custom-plugin"),
						Partials: []*kong.PartialLink{
							{
								Partial: &kong.Partial{
									ID: new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
								},
								Path: new("config.custom_path"),
							},
						},
					},
				},
			},
		},
		{
			name: "error when consumer not found",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("key-auth"),
								Consumer: &kong.Consumer{
									ID: new("non-existent"),
								},
							},
						},
					},
				},
				intermediate: emptyState(),
			},
			wantErr: "consumer non-existent for plugin key-auth: entity not found",
		},
		{
			name: "error when service not found",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("key-auth"),
								Service: &kong.Service{
									ID: new("non-existent"),
								},
							},
						},
					},
				},
				intermediate: emptyState(),
			},
			wantErr: "service non-existent for plugin key-auth: entity not found",
		},
		{
			name: "error when route not found",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("key-auth"),
								Route: &kong.Route{
									ID: new("non-existent"),
								},
							},
						},
					},
				},
				intermediate: emptyState(),
			},
			wantErr: "route non-existent for plugin key-auth: entity not found",
		},
		{
			name: "error when consumer-group not found",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("key-auth"),
								ConsumerGroup: &kong.ConsumerGroup{
									ID: new("non-existent"),
								},
							},
						},
					},
				},
				intermediate: emptyState(),
			},
			wantErr: "consumer-group non-existent for plugin key-auth: entity not found",
		},
		{
			name: "error when linked partial is missing ID and name",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("custom-plugin"),
								Partials: []*kong.PartialLink{
									{
										Partial: &kong.Partial{},
									},
								},
							},
						},
					},
				},
				intermediate: emptyState(),
			},
			wantErr: "partial for plugin custom-plugin: either partial ID or name is required",
		},
		{
			name: "error when partial is not found",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: new("custom-plugin"),
								Partials: []*kong.PartialLink{
									{
										Partial: &kong.Partial{
											ID: new("non-existent"),
										},
									},
								},
							},
						},
					},
				},
				intermediate: emptyState(),
			},
			wantErr: "partial non-existent for plugin custom-plugin: entity not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  emptyState(),
				intermediate:  tt.fields.intermediate,
				rawState:      &utils.KongRawState{},
			}

			b.plugins()

			if tt.wantErr != "" {
				require.Error(t, b.err)
				assert.Contains(b.err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, b.err)
			assert.Equal(tt.want, b.targetContent.Plugins)
		})
	}
}

func Test_stateBuilder_keys(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "generates a new key from valid config passed",
			fields: fields{
				targetContent: &Content{
					Keys: []FKey{
						{
							Key: kong.Key{
								Name: new("foo"),
								KID:  new("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
								JWK:  new("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
							},
						},
						{
							Key: kong.Key{
								Name: new("my-pem-key"),
								KID:  new("my-pem-key"),
								PEM: &kong.PEM{
									PrivateKey: new("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD\n-----END PRIVATE KEY-----\n"), //nolint:lll
									PublicKey:  new("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"),          //nolint:lll
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Keys: []*kong.Key{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("foo"),
						KID:  new("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
						JWK:  new("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
					},
					{
						ID:   new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name: new("my-pem-key"),
						KID:  new("my-pem-key"),
						PEM: &kong.PEM{
							PrivateKey: new("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD\n-----END PRIVATE KEY-----\n"), //nolint:lll
							PublicKey:  new("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"),          //nolint:lll
						},
					},
				},
			},
		},
		{
			name: "matches ID for an existing key",
			fields: fields{
				targetContent: &Content{
					Keys: []FKey{
						{
							Key: kong.Key{
								Name: new("foo"),
								KID:  new("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
								JWK:  new("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
							},
						},
					},
				},
				currentState: existingKeyState(t),
			},
			want: &utils.KongRawState{
				Keys: []*kong.Key{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("foo"),
						KID:  new("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
						JWK:  new("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
					},
				},
			},
		},
		{
			name: "handles empty key entities",
			fields: fields{
				targetContent: &Content{
					Keys: []FKey{},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Keys: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			_, _, err := b.build()

			require.NoError(t, err, "build error is not nil")
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_keySets(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "generates a new key-set and associate it with keys from valid config passed",
			fields: fields{
				targetContent: &Content{
					KeySets: []FKeySet{
						{
							KeySet: kong.KeySet{
								Name: new("set-1"),
							},
						},
					},
					Keys: []FKey{
						{
							Key: kong.Key{
								Name: new("foo"),
								KID:  new("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
								JWK:  new("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
								Set: &kong.KeySet{
									Name: new("set-1"),
								},
							},
						},
						{
							Key: kong.Key{
								Name: new("my-pem-key"),
								KID:  new("my-pem-key"),
								PEM: &kong.PEM{
									PrivateKey: new("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD\n-----END PRIVATE KEY-----\n"), //nolint:lll
									PublicKey:  new("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"),          //nolint:lll
								},
								Set: &kong.KeySet{
									Name: new("set-1"),
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				KeySets: []*kong.KeySet{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("set-1"),
					},
				},
				Keys: []*kong.Key{
					{
						ID:   new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name: new("foo"),
						KID:  new("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
						JWK:  new("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
						Set: &kong.KeySet{
							ID: new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						},
					},
					{
						ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: new("my-pem-key"),
						KID:  new("my-pem-key"),
						PEM: &kong.PEM{
							PrivateKey: new("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD\n-----END PRIVATE KEY-----\n"), //nolint:lll
							PublicKey:  new("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"),          //nolint:lll
						},
						Set: &kong.KeySet{
							ID: new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						},
					},
				},
			},
		},
		{
			name: "matches ID for an existing keyset",
			fields: fields{
				targetContent: &Content{
					KeySets: []FKeySet{
						{
							KeySet: kong.KeySet{
								Name: new("existing-set"),
							},
						},
					},
				},
				currentState: existingKeySetState(t),
			},
			want: &utils.KongRawState{
				KeySets: []*kong.KeySet{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("existing-set"),
					},
				},
			},
		},
		{
			name: "handles empty key-set entities",
			fields: fields{
				targetContent: &Content{
					KeySets: []FKeySet{},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				KeySets: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			_, _, err := b.build()

			require.NoError(t, err, "build error is not nil")
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func existingClonedPluginDefinitionState(t *testing.T) *state.KongState {
	t.Helper()
	s, err := state.NewKongState()
	require.NoError(t, err, "error in getting new kongState")

	s.ClonedPluginDefinitions.Add(
		state.ClonedPluginDefinition{
			ClonedPluginDefinition: kong.ClonedPluginDefinition{
				ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
				Name: new("file-log-clone"),
				Ref:  new("file-log"),
			},
		})
	return s
}

func Test_stateBuilder_clonedPluginDefinitions(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "generates a new cloned plugin definition from valid config",
			fields: fields{
				targetContent: &Content{
					ClonedPluginDefinitions: []FClonedPluginDefinition{
						{
							ClonedPluginDefinition: kong.ClonedPluginDefinition{
								Name: new("file-log-clone"),
								Ref:  new("file-log"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				ClonedPluginDefinitions: []*kong.ClonedPluginDefinition{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("file-log-clone"),
						Ref:  new("file-log"),
					},
				},
			},
		},
		{
			name: "generates multiple cloned plugin definitions with priority and tags",
			fields: fields{
				targetContent: &Content{
					ClonedPluginDefinitions: []FClonedPluginDefinition{
						{
							ClonedPluginDefinition: kong.ClonedPluginDefinition{
								Name:     new("file-log-clone"),
								Ref:      new("file-log"),
								Priority: new(50000),
								Tags:     kong.StringSlice("t1", "t2"),
							},
						},
						{
							ClonedPluginDefinition: kong.ClonedPluginDefinition{
								Name: new("http-log-clone"),
								Ref:  new("http-log"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				ClonedPluginDefinitions: []*kong.ClonedPluginDefinition{
					{
						ID:       new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:     new("file-log-clone"),
						Ref:      new("file-log"),
						Priority: new(50000),
						Tags:     kong.StringSlice("t1", "t2"),
					},
					{
						ID:   new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: new("http-log-clone"),
						Ref:  new("http-log"),
					},
				},
			},
		},
		{
			name: "matches ID for an existing cloned plugin definition",
			fields: fields{
				targetContent: &Content{
					ClonedPluginDefinitions: []FClonedPluginDefinition{
						{
							ClonedPluginDefinition: kong.ClonedPluginDefinition{
								Name: new("file-log-clone"),
								Ref:  new("file-log"),
							},
						},
					},
				},
				currentState: existingClonedPluginDefinitionState(t),
			},
			want: &utils.KongRawState{
				ClonedPluginDefinitions: []*kong.ClonedPluginDefinition{
					{
						ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: new("file-log-clone"),
						Ref:  new("file-log"),
					},
				},
			},
		},
		{
			name: "uses provided ID when explicitly set",
			fields: fields{
				targetContent: &Content{
					ClonedPluginDefinitions: []FClonedPluginDefinition{
						{
							ClonedPluginDefinition: kong.ClonedPluginDefinition{
								ID:   new("1234abcd-0000-0000-0000-000000000000"),
								Name: new("file-log-clone"),
								Ref:  new("file-log"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				ClonedPluginDefinitions: []*kong.ClonedPluginDefinition{
					{
						ID:   new("1234abcd-0000-0000-0000-000000000000"),
						Name: new("file-log-clone"),
						Ref:  new("file-log"),
					},
				},
			},
		},
		{
			name: "handles empty cloned plugin definition entities",
			fields: fields{
				targetContent: &Content{
					ClonedPluginDefinitions: []FClonedPluginDefinition{},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				ClonedPluginDefinitions: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			_, _, err := b.build()

			require.NoError(t, err, "build error is not nil")
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_ingestConsumerGroupConsumer(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))

	type fields struct {
		targetContent            *Content
		currentState             *state.KongState
		lookupTagsConsumers      []string
		lookupTagsConsumerGroups []string
		checkIntermediateState   bool
	}
	type args struct {
		cgID *string
		c    *FConsumer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *kong.Consumer
		wantErr bool
	}{
		{
			name: "matches existing consumer by username from target content with consumer lookup tags",
			fields: fields{
				targetContent: &Content{
					Consumers: []FConsumer{
						{
							Consumer: kong.Consumer{
								ID:       new("existing-consumer-id"),
								Username: new("test-user"),
								Tags:     kong.StringSlice("lookup-tag"),
							},
						},
					},
				},
				currentState:        emptyState(),
				lookupTagsConsumers: []string{"lookup-tag"},
			},
			args: args{
				cgID: new("cg-123"),
				c: &FConsumer{
					Consumer: kong.Consumer{
						Username: new("test-user"),
					},
				},
			},
			want: &kong.Consumer{
				ID:       new("existing-consumer-id"),
				Username: new("test-user"),
				Tags:     kong.StringSlice("lookup-tag"),
			},
			wantErr: false,
		},
		{
			name: "matches existing consumer by custom ID from target content with consumer lookup tags",
			fields: fields{
				targetContent: &Content{
					Consumers: []FConsumer{
						{
							Consumer: kong.Consumer{
								ID:       new("existing-consumer-id"),
								CustomID: new("custom-123"),
								Tags:     kong.StringSlice("lookup-tag"),
							},
						},
					},
				},
				currentState:        emptyState(),
				lookupTagsConsumers: []string{"lookup-tag"},
			},
			args: args{
				cgID: new("cg-123"),
				c: &FConsumer{
					Consumer: kong.Consumer{
						CustomID: new("custom-123"),
					},
				},
			},
			want: &kong.Consumer{
				ID:       new("existing-consumer-id"),
				CustomID: new("custom-123"),
				Tags:     kong.StringSlice("lookup-tag"),
			},
			wantErr: false,
		},
		{
			name: "matches existing consumer from from target content with consumer-group lookup tags",
			fields: fields{
				targetContent: &Content{
					Consumers: []FConsumer{
						{
							Consumer: kong.Consumer{
								ID:       new("existing-consumer-id"),
								Username: new("test-user"),
							},
							Groups: []*kong.ConsumerGroup{
								{
									ID:   new("cg-123"),
									Tags: kong.StringSlice("cg-lookup-tag"),
								},
							},
						},
					},
				},
				currentState:             emptyState(),
				lookupTagsConsumerGroups: []string{"cg-lookup-tag"},
				checkIntermediateState:   true,
			},
			args: args{
				cgID: new("cg-123"),
				c: &FConsumer{
					Consumer: kong.Consumer{
						ID:       new("existing-consumer-id"),
						Username: new("test-user"),
					},
				},
			},
			want: &kong.Consumer{
				ID:       new("existing-consumer-id"),
				Username: new("test-user"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent:            tt.fields.targetContent,
				currentState:             tt.fields.currentState,
				lookupTagsConsumers:      tt.fields.lookupTagsConsumers,
				lookupTagsConsumerGroups: tt.fields.lookupTagsConsumerGroups,
				rawState:                 &utils.KongRawState{},
			}

			intermediate, err := state.NewKongState()
			require.NoError(t, err)
			b.intermediate = intermediate

			got, err := b.ingestConsumerGroupConsumer(tt.args.cgID, tt.args.c)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			if !tt.fields.checkIntermediateState {
				return
			}

			// Verify consumer group consumer relationship was added
			cgConsumers, err := b.intermediate.ConsumerGroupConsumers.GetAll()
			require.NoError(t, err)
			assert.Len(t, cgConsumers, 1)
			assert.Equal(t, tt.args.cgID, cgConsumers[0].ConsumerGroup.ID)
			assert.Equal(t, got.ID, cgConsumers[0].Consumer.ID)
		})
	}
}

func Test_InstanceName_ConsumerGroupPlugin(t *testing.T) {
	assert := assert.New(t)
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "consumergroup plugin with instance_name set",
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					ConsumerGroups: []FConsumerGroupObject{
						{
							ConsumerGroup: kong.ConsumerGroup{
								Name: new("foo-group"),
							},
							Consumers: nil,
							Plugins: []*kong.ConsumerGroupPlugin{
								{
									Name:         new("rate-limiting-advanced"),
									InstanceName: new("custom-instance-name"),
									Config: kong.Configuration{
										testLimit:     []any{float64(100)},
										"window_size": []any{float64(60)},
										"window_type": string("fixed"),
									},
								},
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							ID:   new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: new("foo-group"),
						},
						Consumers: nil,
					},
				},
				Plugins: []*kong.Plugin{
					{
						ID:           new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:         new("rate-limiting-advanced"),
						InstanceName: new("custom-instance-name"),
						Config: kong.Configuration{
							testLimit:     []any{float64(100)},
							"window_size": []any{float64(60)},
							"window_type": string("fixed"),
						},
						ConsumerGroup: &kong.ConsumerGroup{
							ID: new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						},
					},
				},
			},
		},
	}

	// Define versions to test against
	kongVersions := []semver.Version{
		kong340Version,
		kong370Version,
		kong380Version,
		kong390Version,
		kong3100Version,
		kong3110Version,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			for _, version := range kongVersions {
				t.Run(version.String(), func(t *testing.T) {
					testRand = rand.New(rand.NewSource(42))
					b := &stateBuilder{
						targetContent: tt.fields.targetContent,
						currentState:  tt.fields.currentState,
						kongVersion:   version,
					}
					d, _ := utils.GetDefaulter(ctx, defaulterTestOpts)
					b.defaulter = d
					_, _, err := b.build()
					require.NoError(t, err, "build error is not nil")
					assert.Equal(tt.want, b.rawState)
				})
			}
		})
	}
}

func Test_stateBuilder_ingestGqlRateLimitingCostDecoration(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name      string
		fields    fields
		want      *utils.KongRawState
		wantErr   bool
		errString string
	}{
		{
			name: "generates a new graphql_ratelimiting_cost_decoration from valid config passed",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":     "Query.allFields",
								"add_constant":  float64(5),
								"mul_constant":  float64(3),
								"add_arguments": []*string{new("skip"), new("take")},
								"mul_arguments": []*string{new("count"), new("size")},
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: []*kong.GraphqlRateLimitingCostDecoration{
					{
						ID:           new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						TypePath:     new("Query.allFields"),
						AddConstant:  kong.Float64(5),
						MulConstant:  kong.Float64(3),
						AddArguments: []*string{new("skip"), new("take")},
						MulArguments: []*string{new("count"), new("size")},
						Service: &kong.Service{
							ID: new(testServiceID),
						},
					},
				},
			},
		},
		{
			name: "matches ID for an existing graphql_ratelimiting_cost_decoration",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":    new("Query.users"),
								"add_constant": kong.Float64(1),
								"mul_constant": kong.Float64(1),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: existingGqlCostDecorationState(t),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: []*kong.GraphqlRateLimitingCostDecoration{
					{
						ID:          new("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						TypePath:    new("Query.users"),
						AddConstant: kong.Float64(1),
						MulConstant: kong.Float64(1),
						Service: &kong.Service{
							ID: new(testServiceID),
						},
					},
				},
			},
		},
		{
			name: "accepts service id lookup",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":    new("Mutation.createUser"),
								"add_constant": kong.Float64(10),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: []*kong.GraphqlRateLimitingCostDecoration{
					{
						ID:          new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						TypePath:    new("Mutation.createUser"),
						AddConstant: kong.Float64(10),
						Service: &kong.Service{
							ID: new(testServiceID),
						},
					},
				},
			},
		},
		{
			name: "handles multiple graphql_ratelimiting_cost_decorations",
			fields: fields{
				targetContent: &Content{
					Services: []FService{
						{
							Service: kong.Service{
								Name: new("service1"),
							},
						},
						{
							Service: kong.Service{
								Name: new("service2"),
							},
						},
					},
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":    new("Query.users"),
								"add_constant": kong.Float64(1),
								primaryRelationService: map[string]any{
									"name": "service1",
								},
							},
						},
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":    new("Query.posts"),
								"mul_constant": kong.Float64(3),
								primaryRelationService: map[string]any{
									"name": "service2",
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: []*kong.GraphqlRateLimitingCostDecoration{
					{
						ID:          new("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						TypePath:    new("Query.users"),
						AddConstant: kong.Float64(1),
						Service: &kong.Service{
							ID: new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						},
					},
					{
						ID:          new("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						TypePath:    new("Query.posts"),
						MulConstant: kong.Float64(3),
						Service: &kong.Service{
							ID: new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						},
					},
				},
				Services: []*kong.Service{
					{
						ID:             new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           new("service1"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
					{
						ID:             new("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:           new("service2"),
						Protocol:       new("http"),
						ConnectTimeout: new(60000),
						WriteTimeout:   new(60000),
						ReadTimeout:    new(60000),
					},
				},
			},
		},
		{
			name: "handles missing required fields - type_path",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"add_constant": kong.Float64(1),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: nil,
			},
			wantErr:   true,
			errString: "type_path is required for graphql_ratelimiting_cost_decorations",
		},
		{
			name: "handles missing fields",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type:   new("graphql_ratelimiting_cost_decorations"),
							Fields: nil,
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: nil,
			},
			wantErr:   true,
			errString: "fields are required for graphql_ratelimiting_cost_decorations",
		},
		{
			name: "handles add_constant only",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":    "Query.addConstantOnly",
								"add_constant": float64(10),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: []*kong.GraphqlRateLimitingCostDecoration{
					{
						ID:          new("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						TypePath:    new("Query.addConstantOnly"),
						AddConstant: kong.Float64(10),
						Service: &kong.Service{
							ID: new(testServiceID),
						},
					},
				},
			},
		},
		{
			name: "handles mul_constant only",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":    "Query.mulConstantOnly",
								"mul_constant": float64(2.5),
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: []*kong.GraphqlRateLimitingCostDecoration{
					{
						ID:          new("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						TypePath:    new("Query.mulConstantOnly"),
						MulConstant: new(2.5),
						Service: &kong.Service{
							ID: new(testServiceID),
						},
					},
				},
			},
		},
		{
			name: "handles add_arguments",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":     "Query.withAddArguments",
								"add_constant":  float64(1),
								"add_arguments": []*string{new(testLimit), new("offset")},
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: []*kong.GraphqlRateLimitingCostDecoration{
					{
						ID:           new("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						TypePath:     new("Query.withAddArguments"),
						AddConstant:  kong.Float64(1),
						AddArguments: []*string{new(testLimit), new("offset")},
						Service: &kong.Service{
							ID: new(testServiceID),
						},
					},
				},
			},
		},
		{
			name: "handles mul_arguments",
			fields: fields{
				targetContent: &Content{
					CustomEntities: []FCustomEntity{
						{
							Type: new("graphql_ratelimiting_cost_decorations"),
							Fields: CustomEntityConfiguration{
								"type_path":     "Query.withMulArguments",
								"mul_constant":  float64(2),
								"mul_arguments": []*string{new("first"), new("last")},
								primaryRelationService: map[string]any{
									"id": testServiceID,
								},
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				GraphqlRateLimitingCostDecorations: []*kong.GraphqlRateLimitingCostDecoration{
					{
						ID:           new("aa43465a-7862-4616-978a-ed0ce3c6c4f3"),
						TypePath:     new("Query.withMulArguments"),
						MulConstant:  kong.Float64(2),
						MulArguments: []*string{new("first"), new("last")},
						Service: &kong.Service{
							ID: new(testServiceID),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			_, _, err := b.build()
			if tt.wantErr {
				require.Error(t, err, "build error was expected")
				require.ErrorContains(t, err, tt.errString)
				assert.Equal(t, tt.want, b.rawState)
				return
			}

			require.NoError(t, err, "build error is not nil")
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}

func Test_stateBuilder_customPluginDefinitions(t *testing.T) {
	testRand = rand.New(rand.NewSource(42))
	type fields struct {
		currentState  *state.KongState
		targetContent *Content
	}
	tests := []struct {
		name   string
		fields fields
		want   *utils.KongRawState
	}{
		{
			name: "generates a new custom plugin definition from valid config",
			fields: fields{
				targetContent: &Content{
					CustomPluginDefinitions: []FCustomPluginDefinition{
						{
							CustomPluginDefinition: kong.CustomPluginDefinition{
								Name:    new("my-plugin"),
								Schema:  new("return {}"),
								Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				CustomPluginDefinitions: []*kong.CustomPluginDefinition{
					{
						ID:      new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:    new("my-plugin"),
						Schema:  new("return {}"),
						Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
					},
				},
			},
		},
		{
			name: "generates multiple custom plugin definitions with tags",
			fields: fields{
				targetContent: &Content{
					CustomPluginDefinitions: []FCustomPluginDefinition{
						{
							CustomPluginDefinition: kong.CustomPluginDefinition{
								Name:    new("my-plugin"),
								Schema:  new("return {}"),
								Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
								Tags:    kong.StringSlice("t1", "t2"),
							},
						},
						{
							CustomPluginDefinition: kong.CustomPluginDefinition{
								Name:    new("other-plugin"),
								Schema:  new("return {}"),
								Handler: new("return { PRIORITY = 500, VERSION = \"0.1.0\" }"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				CustomPluginDefinitions: []*kong.CustomPluginDefinition{
					{
						ID:      new("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:    new("my-plugin"),
						Schema:  new("return {}"),
						Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
						Tags:    kong.StringSlice("t1", "t2"),
					},
					{
						ID:      new("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:    new("other-plugin"),
						Schema:  new("return {}"),
						Handler: new("return { PRIORITY = 500, VERSION = \"0.1.0\" }"),
					},
				},
			},
		},
		{
			name: "matches ID for an existing custom plugin definition",
			fields: fields{
				targetContent: &Content{
					CustomPluginDefinitions: []FCustomPluginDefinition{
						{
							CustomPluginDefinition: kong.CustomPluginDefinition{
								Name:    new("my-plugin"),
								Schema:  new("return {}"),
								Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
							},
						},
					},
				},
				currentState: existingCustomPluginDefinitionState(t),
			},
			want: &utils.KongRawState{
				CustomPluginDefinitions: []*kong.CustomPluginDefinition{
					{
						ID:      new("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:    new("my-plugin"),
						Schema:  new("return {}"),
						Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
					},
				},
			},
		},
		{
			name: "uses provided ID when explicitly set",
			fields: fields{
				targetContent: &Content{
					CustomPluginDefinitions: []FCustomPluginDefinition{
						{
							CustomPluginDefinition: kong.CustomPluginDefinition{
								ID:      new("1234abcd-0000-0000-0000-000000000000"),
								Name:    new("my-plugin"),
								Schema:  new("return {}"),
								Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				CustomPluginDefinitions: []*kong.CustomPluginDefinition{
					{
						ID:      new("1234abcd-0000-0000-0000-000000000000"),
						Name:    new("my-plugin"),
						Schema:  new("return {}"),
						Handler: new("return { PRIORITY = 1000, VERSION = \"1.0.0\" }"),
					},
				},
			},
		},
		{
			name: "handles empty custom plugin definition entities",
			fields: fields{
				targetContent: &Content{
					CustomPluginDefinitions: []FCustomPluginDefinition{},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				CustomPluginDefinitions: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				targetContent: tt.fields.targetContent,
				currentState:  tt.fields.currentState,
			}
			_, _, err := b.build()

			require.NoError(t, err, "build error is not nil")
			assert.Equal(t, tt.want, b.rawState)
		})
	}
}
