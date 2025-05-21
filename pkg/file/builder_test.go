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
)

var (
	kong130Version  = semver.MustParse("1.3.0")
	kong340Version  = semver.MustParse("3.4.0")
	kong360Version  = semver.MustParse("3.6.0")
	kong370Version  = semver.MustParse("3.7.0")
	kong3100Version = semver.MustParse("3.10.0")
)

var kongDefaults = KongDefaults{
	Service: &kong.Service{
		Protocol:       kong.String("http"),
		ConnectTimeout: kong.Int(defaultTimeout),
		WriteTimeout:   kong.Int(defaultTimeout),
		ReadTimeout:    kong.Int(defaultTimeout),
	},
	Route: &kong.Route{
		PreserveHost:  kong.Bool(false),
		RegexPriority: kong.Int(0),
		StripPath:     kong.Bool(false),
		Protocols:     kong.StringSlice("http", "https"),
	},
	Upstream: &kong.Upstream{
		Slots: kong.Int(defaultSlots),
		Healthchecks: &kong.Healthcheck{
			Active: &kong.ActiveHealthcheck{
				Concurrency: kong.Int(defaultConcurrency),
				Healthy: &kong.Healthy{
					HTTPStatuses: []int{200, 302},
					Interval:     kong.Int(0),
					Successes:    kong.Int(0),
				},
				HTTPPath: kong.String("/"),
				Type:     kong.String("http"),
				Timeout:  kong.Int(1),
				Unhealthy: &kong.Unhealthy{
					HTTPFailures: kong.Int(0),
					TCPFailures:  kong.Int(0),
					Timeouts:     kong.Int(0),
					Interval:     kong.Int(0),
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
					Successes: kong.Int(0),
				},
				Unhealthy: &kong.Unhealthy{
					HTTPFailures: kong.Int(0),
					TCPFailures:  kong.Int(0),
					Timeouts:     kong.Int(0),
					HTTPStatuses: []int{429, 500, 503},
				},
			},
		},
		HashOn:           kong.String("none"),
		HashFallback:     kong.String("none"),
		HashOnCookiePath: kong.String("/"),
	},
	Target: &kong.Target{
		Weight: kong.Int(defaultWeight),
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
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: kong.String("foo"),
		},
	})
	return s
}

func existingServiceState() *state.KongState {
	s, _ := state.NewKongState()
	s.Services.Add(state.Service{
		Service: kong.Service{
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: kong.String("foo"),
		},
	})
	return s
}

func existingConsumerCredState() *state.KongState {
	s, _ := state.NewKongState()
	s.Consumers.Add(state.Consumer{
		Consumer: kong.Consumer{
			ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Username: kong.String("foo"),
		},
	})
	s.KeyAuths.Add(state.KeyAuth{
		KeyAuth: kong.KeyAuth{
			ID:  kong.String("5f1ef1ea-a2a5-4a1b-adbb-b0d3434013e5"),
			Key: kong.String("foo-apikey"),
			Consumer: &kong.Consumer{
				ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: kong.String("foo"),
			},
		},
	})
	s.BasicAuths.Add(state.BasicAuth{
		BasicAuth: kong.BasicAuth{
			ID:       kong.String("92f4c849-960b-43af-aad3-f307051408d3"),
			Username: kong.String("basic-username"),
			Password: kong.String("basic-password"),
			Consumer: &kong.Consumer{
				ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: kong.String("foo"),
			},
		},
	})
	s.JWTAuths.Add(state.JWTAuth{
		JWTAuth: kong.JWTAuth{
			ID:     kong.String("917b9402-1be0-49d2-b482-ca4dccc2054e"),
			Key:    kong.String("jwt-key"),
			Secret: kong.String("jwt-secret"),
			Consumer: &kong.Consumer{
				ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: kong.String("foo"),
			},
		},
	})
	s.HMACAuths.Add(state.HMACAuth{
		HMACAuth: kong.HMACAuth{
			ID:       kong.String("e5d81b73-bf9e-42b0-9d68-30a1d791b9c9"),
			Username: kong.String("hmac-username"),
			Secret:   kong.String("hmac-secret"),
			Consumer: &kong.Consumer{
				ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: kong.String("foo"),
			},
		},
	})
	s.ACLGroups.Add(state.ACLGroup{
		ACLGroup: kong.ACLGroup{
			ID:    kong.String("b7c9352a-775a-4ba5-9869-98e926a3e6cb"),
			Group: kong.String("foo-group"),
			Consumer: &kong.Consumer{
				ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: kong.String("foo"),
			},
		},
	})
	s.Oauth2Creds.Add(state.Oauth2Credential{
		Oauth2Credential: kong.Oauth2Credential{
			ID:       kong.String("4eef5285-3d6a-4f6b-b659-8957a940e2ca"),
			ClientID: kong.String("oauth2-clientid"),
			Name:     kong.String("oauth2-name"),
			Consumer: &kong.Consumer{
				ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: kong.String("foo"),
			},
		},
	})
	s.MTLSAuths.Add(state.MTLSAuth{
		MTLSAuth: kong.MTLSAuth{
			ID:          kong.String("92f4c829-968b-42af-afd3-f337051508d3"),
			SubjectName: kong.String("test@example.com"),
			Consumer: &kong.Consumer{
				ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Username: kong.String("foo"),
			},
		},
	})
	return s
}

func existingUpstreamState() *state.KongState {
	s, _ := state.NewKongState()
	s.Upstreams.Add(state.Upstream{
		Upstream: kong.Upstream{
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: kong.String("foo"),
		},
	})
	return s
}

func existingCertificateState() *state.KongState {
	s, _ := state.NewKongState()
	s.Certificates.Add(state.Certificate{
		Certificate: kong.Certificate{
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Cert: kong.String("foo"),
			Key:  kong.String("bar"),
		},
	})
	return s
}

func existingCertificateAndSNIState() *state.KongState {
	s, _ := state.NewKongState()
	s.Certificates.Add(state.Certificate{
		Certificate: kong.Certificate{
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Cert: kong.String("foo"),
			Key:  kong.String("bar"),
		},
	})
	s.SNIs.Add(state.SNI{
		SNI: kong.SNI{
			ID:   kong.String("a53e9598-3a5e-4c12-a672-71a4cdcf7a47"),
			Name: kong.String("foo.example.com"),
			Certificate: &kong.Certificate{
				ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			},
		},
	})
	s.SNIs.Add(state.SNI{
		SNI: kong.SNI{
			ID:   kong.String("5f8e6848-4cb9-479a-a27e-860e1a77f875"),
			Name: kong.String("bar.example.com"),
			Certificate: &kong.Certificate{
				ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			},
		},
	})
	return s
}

func existingCACertificateState() *state.KongState {
	s, _ := state.NewKongState()
	s.CACertificates.Add(state.CACertificate{
		CACertificate: kong.CACertificate{
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Cert: kong.String("foo"),
		},
	})
	return s
}

func existingPluginState() *state.KongState {
	s, _ := state.NewKongState()
	s.Consumers.Add(state.Consumer{
		Consumer: kong.Consumer{
			ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
		},
	})
	s.Routes.Add(state.Route{
		Route: kong.Route{
			ID: kong.String("700bc504-b2b1-4abd-bd38-cec92779659e"),
		},
	})
	s.ConsumerGroups.Add(state.ConsumerGroup{
		ConsumerGroup: kong.ConsumerGroup{
			ID:   kong.String("69ed4618-a653-4b54-8bb6-dc33bd6fe048"),
			Name: kong.String("test-group"),
		},
	})
	s.Plugins.Add(state.Plugin{
		Plugin: kong.Plugin{
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: kong.String("foo"),
		},
	})
	s.Plugins.Add(state.Plugin{
		Plugin: kong.Plugin{
			ID:   kong.String("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
			Name: kong.String("bar"),
			Consumer: &kong.Consumer{
				ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
			},
		},
	})
	s.Plugins.Add(state.Plugin{
		Plugin: kong.Plugin{
			ID:   kong.String("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
			Name: kong.String("foo"),
			Consumer: &kong.Consumer{
				ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
			},
			Route: &kong.Route{
				ID: kong.String("700bc504-b2b1-4abd-bd38-cec92779659e"),
			},
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("69ed4618-a653-4b54-8bb6-dc33bd6fe048"),
			},
		},
	})
	return s
}

func existingScopedPluginState() *state.KongState {
	s, _ := state.NewKongState()

	s.Consumers.Add(state.Consumer{
		Consumer: kong.Consumer{
			ID: kong.String("cID"),
		},
	})

	s.Services.Add(state.Service{
		Service: kong.Service{
			ID: kong.String("sID"),
		},
	})

	s.Routes.Add(state.Route{
		Route: kong.Route{
			ID: kong.String("rID"),
		},
	})

	s.ConsumerGroups.Add(state.ConsumerGroup{
		ConsumerGroup: kong.ConsumerGroup{
			ID:   kong.String("cgID"),
			Name: kong.String("foo"),
		},
	})

	s.Plugins.Add(state.Plugin{
		Plugin: kong.Plugin{
			ID:   kong.String("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
			Name: kong.String("foo"),
			Consumer: &kong.Consumer{
				ID: kong.String("cID"),
			},
			Route: &kong.Route{
				ID: kong.String("rID"),
			},
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("cgID"),
			},
			Service: &kong.Service{
				ID: kong.String("sID"),
			},
		},
	})

	return s
}

func existingTargetsState() *state.KongState {
	s, _ := state.NewKongState()
	s.Targets.Add(state.Target{
		Target: kong.Target{
			ID:     kong.String("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
			Target: kong.String("bar"),
			Upstream: &kong.Upstream{
				ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
			},
		},
	})
	s.Targets.Add(state.Target{
		Target: kong.Target{
			ID:     kong.String("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
			Target: kong.String("foo"),
			Upstream: &kong.Upstream{
				ID: kong.String("700bc504-b2b1-4abd-bd38-cec92779659e"),
			},
		},
	})
	return s
}

func existingDocumentState() *state.KongState {
	s, _ := state.NewKongState()
	s.ServicePackages.Add(state.ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: kong.String("foo"),
		},
	})
	parent, _ := s.ServicePackages.Get("4bfcb11f-c962-4817-83e5-9433cf20b663")
	s.Documents.Add(state.Document{
		Document: konnect.Document{
			ID:        kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Path:      kong.String("/foo.md"),
			Published: kong.Bool(true),
			Content:   kong.String("foo"),
			Parent:    parent,
		},
	})
	return s
}

func existingFilterChainState() *state.KongState {
	s, _ := state.NewKongState()
	s.FilterChains.Add(state.FilterChain{
		FilterChain: kong.FilterChain{
			Name:    kong.String("my-service-chain"),
			ID:      kong.String("fa7bd007-e0c6-4ef2-b254-e60d3a341b0c"),
			Enabled: kong.Bool(true),
			Service: &kong.Service{
				ID: kong.String("ba54b737-38aa-49d1-87c4-64e756b0c6f9"),
			},
			Filters: []*kong.Filter{
				{
					Name:    kong.String("my-filter"),
					Config:  jsonRawMessage(`"config!"`),
					Enabled: kong.Bool(false),
				},
			},
		},
	})
	s.FilterChains.Add(state.FilterChain{
		FilterChain: kong.FilterChain{
			Name:    kong.String("my-route-chain"),
			ID:      kong.String("ac6758a5-41d4-4493-827f-de9df5b75859"),
			Enabled: kong.Bool(true),
			Route: &kong.Route{
				ID: kong.String("ec9b7c35-8e95-4a7c-b0da-4fba8986d1cd"),
			},
			Filters: []*kong.Filter{
				{
					Name:    kong.String("my-filter"),
					Config:  jsonRawMessage(`"config!"`),
					Enabled: kong.Bool(false),
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
				ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Service: &kong.Service{
					ID: kong.String("fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd"),
				},
				Methods: kong.StringSlice("GET"),
				URI:     kong.String("/example"),
				Query:   kong.String("query{ example { foo } }"),
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
				ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
				Name: kong.String("existing-partial"),
				Type: kong.String("foo"),
				Config: kong.Configuration{
					"key1": "value1",
					"key2": []any{"a", "b", "c"},
					"key3": map[string]interface{}{
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
			ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Username: kong.String("foo"),
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
			ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
			Name: kong.String("foo"),
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
				ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
				Name: kong.String("foo"),
				KID:  kong.String("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
				JWK:  kong.String("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
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
				ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
				Name: kong.String("existing-set"),
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
								Name: kong.String("foo"),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:           kong.String("foo"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
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
								Name: kong.String("foo"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           kong.String("foo"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
						Tags:           kong.StringSlice("tag1"),
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
						Name: kong.String("foo"),
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						ID:            kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:          kong.String("foo"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
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
						Name: kong.String("foo"),
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						ID:            kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:          kong.String("foo"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
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
						Name:      kong.String("foo"),
						Protocols: kong.StringSlice("grpc"),
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						ID:            kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:          kong.String("foo"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
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
						Target: kong.String("foo"),
						Upstream: &kong.Upstream{
							ID: kong.String("952ddf37-e815-40b6-b119-5379a3b1f7be"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Target: kong.String("foo"),
						Weight: kong.Int(100),
						Upstream: &kong.Upstream{
							ID: kong.String("952ddf37-e815-40b6-b119-5379a3b1f7be"),
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
						Target: kong.String("bar"),
						Upstream: &kong.Upstream{
							ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
						},
					},
					{
						Target: kong.String("foo"),
						Upstream: &kong.Upstream{
							ID: kong.String("700bc504-b2b1-4abd-bd38-cec92779659e"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     kong.String("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
						Target: kong.String("bar"),
						Weight: kong.Int(100),
						Upstream: &kong.Upstream{
							ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
						},
					},
					{
						ID:     kong.String("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
						Target: kong.String("foo"),
						Weight: kong.Int(100),
						Upstream: &kong.Upstream{
							ID: kong.String("700bc504-b2b1-4abd-bd38-cec92779659e"),
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
						ID:     kong.String("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: kong.String("[2001:db8:fd73::e]:1326"),
						Upstream: &kong.Upstream{
							ID: kong.String("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     kong.String("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: kong.String("[2001:0db8:fd73:0000:0000:0000:0000:000e]:1326"),
						Weight: kong.Int(100),
						Upstream: &kong.Upstream{
							ID: kong.String("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
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
						ID:     kong.String("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: kong.String("::1"),
						Upstream: &kong.Upstream{
							ID: kong.String("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     kong.String("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: kong.String("[0000:0000:0000:0000:0000:0000:0000:0001]:8000"),
						Weight: kong.Int(100),
						Upstream: &kong.Upstream{
							ID: kong.String("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
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
						ID:     kong.String("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: kong.String("[::1]"),
						Upstream: &kong.Upstream{
							ID: kong.String("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Targets: []*kong.Target{
					{
						ID:     kong.String("d6e7f8a9-bcde-1234-5678-9abcdef01234"),
						Target: kong.String("[0000:0000:0000:0000:0000:0000:0000:0001]:8000"),
						Weight: kong.Int(100),
						Upstream: &kong.Upstream{
							ID: kong.String("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
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
						Target: kong.String("[invalid:ipv6::address]:1326"),
						Upstream: &kong.Upstream{
							ID: kong.String("b1c2d3e4-f5a6-7890-abcd-ef1234567890"),
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
						Target: kong.String("1:2:3:4"),
						Upstream: &kong.Upstream{
							ID: kong.String("b1c2d3e4-f5a6-7890-abcd-ef1234567890"),
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
						Target: kong.String("this:is:nuts:!"),
						Upstream: &kong.Upstream{
							ID: kong.String("b1c2d3e4-f5a6-7890-abcd-ef1234567890"),
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
							Name: kong.String("foo"),
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Plugins: []*kong.Plugin{
					{
						ID:     kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:   kong.String("foo"),
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
							Name: kong.String("foo"),
						},
					},
					{
						Plugin: kong.Plugin{
							Name: kong.String("bar"),
							Consumer: &kong.Consumer{
								ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
							},
						},
					},
					{
						Plugin: kong.Plugin{
							Name: kong.String("foo"),
							Consumer: &kong.Consumer{
								ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
							},
							Route: &kong.Route{
								ID: kong.String("700bc504-b2b1-4abd-bd38-cec92779659e"),
							},
							ConsumerGroup: &kong.ConsumerGroup{
								ID: kong.String("69ed4618-a653-4b54-8bb6-dc33bd6fe048"),
							},
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Plugins: []*kong.Plugin{
					{
						ID:     kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:   kong.String("foo"),
						Config: kong.Configuration{},
					},
					{
						ID:   kong.String("f7e64af5-e438-4a9b-8ff8-ec6f5f06dccb"),
						Name: kong.String("bar"),
						Consumer: &kong.Consumer{
							ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
						},
						Config: kong.Configuration{},
					},
					{
						ID:   kong.String("53ce0a9c-d518-40ee-b8ab-1ee83a20d382"),
						Name: kong.String("foo"),
						Consumer: &kong.Consumer{
							ID: kong.String("f77ca8c7-581d-45a4-a42c-c003234228e1"),
						},
						Route: &kong.Route{
							ID: kong.String("700bc504-b2b1-4abd-bd38-cec92779659e"),
						},
						ConsumerGroup: &kong.ConsumerGroup{
							ID: kong.String("69ed4618-a653-4b54-8bb6-dc33bd6fe048"),
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
			args: args{
				plugin: &kong.Plugin{
					Name: kong.String("foo"),
				},
			},
			wantCID:      "",
			wantRID:      "",
			wantSID:      "",
			wantCGID:     "",
			currentState: emptyState(),
		},
		{
			args: args{
				plugin: &kong.Plugin{
					Name: kong.String("foo"),
					Consumer: &kong.Consumer{
						ID: kong.String("cID"),
					},
					Route: &kong.Route{
						ID: kong.String("rID"),
					},
					Service: &kong.Service{
						ID: kong.String("sID"),
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: kong.String("cgID"),
					},
				},
			},
			wantCID:      "cID",
			wantRID:      "rID",
			wantSID:      "sID",
			wantCGID:     "cgID",
			currentState: existingScopedPluginState(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &stateBuilder{
				currentState: tt.currentState,
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
							Name: kong.String("my-filter-chain"),
							Service: &kong.Service{
								ID: kong.String("fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd"),
							},
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				FilterChains: []*kong.FilterChain{
					{
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: kong.String("my-filter-chain"),
						Service: &kong.Service{
							ID: kong.String("fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd"),
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
								ID: kong.String("ba54b737-38aa-49d1-87c4-64e756b0c6f9"),
							},
						},
					},
					{
						FilterChain: kong.FilterChain{
							Route: &kong.Route{
								ID: kong.String("ec9b7c35-8e95-4a7c-b0da-4fba8986d1cd"),
							},
						},
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				FilterChains: []*kong.FilterChain{
					{
						ID: kong.String("fa7bd007-e0c6-4ef2-b254-e60d3a341b0c"),
						Service: &kong.Service{
							ID: kong.String("ba54b737-38aa-49d1-87c4-64e756b0c6f9"),
						},
					},
					{
						ID: kong.String("ac6758a5-41d4-4493-827f-de9df5b75859"),
						Route: &kong.Route{
							ID: kong.String("ec9b7c35-8e95-4a7c-b0da-4fba8986d1cd"),
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
								Username: kong.String("foo"),
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
						ID:       kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Username: kong.String("foo"),
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
								Username: kong.String("foo"),
							},
							KeyAuths: []*kong.KeyAuth{
								{
									Key: kong.String("foo-key"),
								},
							},
							BasicAuths: []*kong.BasicAuth{
								{
									Username: kong.String("basic-username"),
									Password: kong.String("basic-password"),
								},
							},
							HMACAuths: []*kong.HMACAuth{
								{
									Username: kong.String("hmac-username"),
									Secret:   kong.String("hmac-secret"),
								},
							},
							JWTAuths: []*kong.JWTAuth{
								{
									Key:    kong.String("jwt-key"),
									Secret: kong.String("jwt-secret"),
								},
							},
							Oauth2Creds: []*kong.Oauth2Credential{
								{
									ClientID: kong.String("oauth2-clientid"),
									Name:     kong.String("oauth2-name"),
								},
							},
							ACLGroups: []*kong.ACLGroup{
								{
									Group: kong.String("foo-group"),
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
						ID:       kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Username: kong.String("foo"),
					},
				},
				KeyAuths: []*kong.KeyAuth{
					{
						ID:  kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Key: kong.String("foo-key"),
						Consumer: &kong.Consumer{
							ID:       kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: kong.String("foo"),
						},
					},
				},
				BasicAuths: []*kong.BasicAuth{
					{
						ID:       kong.String("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Username: kong.String("basic-username"),
						Password: kong.String("basic-password"),
						Consumer: &kong.Consumer{
							ID:       kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: kong.String("foo"),
						},
					},
				},
				HMACAuths: []*kong.HMACAuth{
					{
						ID:       kong.String("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Username: kong.String("hmac-username"),
						Secret:   kong.String("hmac-secret"),
						Consumer: &kong.Consumer{
							ID:       kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: kong.String("foo"),
						},
					},
				},
				JWTAuths: []*kong.JWTAuth{
					{
						ID:     kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Key:    kong.String("jwt-key"),
						Secret: kong.String("jwt-secret"),
						Consumer: &kong.Consumer{
							ID:       kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: kong.String("foo"),
						},
					},
				},
				Oauth2Creds: []*kong.Oauth2Credential{
					{
						ID:       kong.String("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						ClientID: kong.String("oauth2-clientid"),
						Name:     kong.String("oauth2-name"),
						Consumer: &kong.Consumer{
							ID:       kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: kong.String("foo"),
						},
					},
				},
				ACLGroups: []*kong.ACLGroup{
					{
						ID:    kong.String("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Group: kong.String("foo-group"),
						Consumer: &kong.Consumer{
							ID:       kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
							Username: kong.String("foo"),
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
								Username: kong.String("foo"),
							},
						},
					},
				},
				currentState: existingConsumerCredState(),
			},
			want: &utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Username: kong.String("foo"),
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
								Username: kong.String("foo"),
							},
							KeyAuths: []*kong.KeyAuth{
								{
									Key: kong.String("foo-apikey"),
								},
							},
							BasicAuths: []*kong.BasicAuth{
								{
									Username: kong.String("basic-username"),
									Password: kong.String("basic-password"),
								},
							},
							HMACAuths: []*kong.HMACAuth{
								{
									Username: kong.String("hmac-username"),
									Secret:   kong.String("hmac-secret"),
								},
							},
							JWTAuths: []*kong.JWTAuth{
								{
									Key:    kong.String("jwt-key"),
									Secret: kong.String("jwt-secret"),
								},
							},
							Oauth2Creds: []*kong.Oauth2Credential{
								{
									ClientID: kong.String("oauth2-clientid"),
									Name:     kong.String("oauth2-name"),
								},
							},
							ACLGroups: []*kong.ACLGroup{
								{
									Group: kong.String("foo-group"),
								},
							},
							MTLSAuths: []*kong.MTLSAuth{
								{
									ID:          kong.String("533c259e-bf71-4d77-99d2-97944c70a6a4"),
									SubjectName: kong.String("test@example.com"),
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
						ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Username: kong.String("foo"),
					},
				},
				KeyAuths: []*kong.KeyAuth{
					{
						ID:  kong.String("5f1ef1ea-a2a5-4a1b-adbb-b0d3434013e5"),
						Key: kong.String("foo-apikey"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				BasicAuths: []*kong.BasicAuth{
					{
						ID:       kong.String("92f4c849-960b-43af-aad3-f307051408d3"),
						Username: kong.String("basic-username"),
						Password: kong.String("basic-password"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				HMACAuths: []*kong.HMACAuth{
					{
						ID:       kong.String("e5d81b73-bf9e-42b0-9d68-30a1d791b9c9"),
						Username: kong.String("hmac-username"),
						Secret:   kong.String("hmac-secret"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				JWTAuths: []*kong.JWTAuth{
					{
						ID:     kong.String("917b9402-1be0-49d2-b482-ca4dccc2054e"),
						Key:    kong.String("jwt-key"),
						Secret: kong.String("jwt-secret"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				Oauth2Creds: []*kong.Oauth2Credential{
					{
						ID:       kong.String("4eef5285-3d6a-4f6b-b659-8957a940e2ca"),
						ClientID: kong.String("oauth2-clientid"),
						Name:     kong.String("oauth2-name"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				ACLGroups: []*kong.ACLGroup{
					{
						ID:    kong.String("b7c9352a-775a-4ba5-9869-98e926a3e6cb"),
						Group: kong.String("foo-group"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				MTLSAuths: []*kong.MTLSAuth{
					{
						ID:          kong.String("533c259e-bf71-4d77-99d2-97944c70a6a4"),
						SubjectName: kong.String("test@example.com"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
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
								Username: kong.String("foo"),
							},
							KeyAuths: []*kong.KeyAuth{
								{
									Key: kong.String("foo-apikey"),
								},
							},
							BasicAuths: []*kong.BasicAuth{
								{
									Username: kong.String("basic-username"),
									Password: kong.String("basic-password"),
								},
							},
							HMACAuths: []*kong.HMACAuth{
								{
									Username: kong.String("hmac-username"),
									Secret:   kong.String("hmac-secret"),
								},
							},
							JWTAuths: []*kong.JWTAuth{
								{
									Key:    kong.String("jwt-key"),
									Secret: kong.String("jwt-secret"),
								},
							},
							Oauth2Creds: []*kong.Oauth2Credential{
								{
									ClientID: kong.String("oauth2-clientid"),
									Name:     kong.String("oauth2-name"),
								},
							},
							ACLGroups: []*kong.ACLGroup{
								{
									Group: kong.String("foo-group"),
								},
							},
							MTLSAuths: []*kong.MTLSAuth{
								{
									ID:          kong.String("533c259e-bf71-4d77-99d2-97944c70a6a4"),
									SubjectName: kong.String("test@example.com"),
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
						ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Username: kong.String("foo"),
					},
				},
				KeyAuths: []*kong.KeyAuth{
					{
						ID:  kong.String("5f1ef1ea-a2a5-4a1b-adbb-b0d3434013e5"),
						Key: kong.String("foo-apikey"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				BasicAuths: []*kong.BasicAuth{
					{
						ID:       kong.String("92f4c849-960b-43af-aad3-f307051408d3"),
						Username: kong.String("basic-username"),
						Password: kong.String("basic-password"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				HMACAuths: []*kong.HMACAuth{
					{
						ID:       kong.String("e5d81b73-bf9e-42b0-9d68-30a1d791b9c9"),
						Username: kong.String("hmac-username"),
						Secret:   kong.String("hmac-secret"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				JWTAuths: []*kong.JWTAuth{
					{
						ID:     kong.String("917b9402-1be0-49d2-b482-ca4dccc2054e"),
						Key:    kong.String("jwt-key"),
						Secret: kong.String("jwt-secret"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				Oauth2Creds: []*kong.Oauth2Credential{
					{
						ID:       kong.String("4eef5285-3d6a-4f6b-b659-8957a940e2ca"),
						ClientID: kong.String("oauth2-clientid"),
						Name:     kong.String("oauth2-name"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				ACLGroups: []*kong.ACLGroup{
					{
						ID:    kong.String("b7c9352a-775a-4ba5-9869-98e926a3e6cb"),
						Group: kong.String("foo-group"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
						},
					},
				},
				MTLSAuths: []*kong.MTLSAuth{
					{
						ID:          kong.String("533c259e-bf71-4d77-99d2-97944c70a6a4"),
						SubjectName: kong.String("test@example.com"),
						Consumer: &kong.Consumer{
							ID:       kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Username: kong.String("foo"),
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
							Cert: kong.String("foo"),
							Key:  kong.String("bar"),
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Certificates: []*kong.Certificate{
					{
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Cert: kong.String("foo"),
						Key:  kong.String("bar"),
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
							Cert: kong.String("foo"),
							Key:  kong.String("bar"),
						},
					},
				},
				currentState: existingCertificateState(),
			},
			want: &utils.KongRawState{
				Certificates: []*kong.Certificate{
					{
						ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Cert: kong.String("foo"),
						Key:  kong.String("bar"),
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
							Cert: kong.String("foo"),
							Key:  kong.String("bar"),
							SNIs: []kong.SNI{
								{
									Name: kong.String("foo.example.com"),
								},
								{
									Name: kong.String("bar.example.com"),
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
						ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Cert: kong.String("foo"),
						Key:  kong.String("bar"),
					},
				},
				SNIs: []*kong.SNI{
					{
						ID:   kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name: kong.String("foo.example.com"),
						Certificate: &kong.Certificate{
							ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
					{
						ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: kong.String("bar.example.com"),
						Certificate: &kong.Certificate{
							ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
							Cert: kong.String("foo"),
							Key:  kong.String("bar"),
							SNIs: []kong.SNI{
								{
									Name: kong.String("foo.example.com"),
								},
								{
									Name: kong.String("bar.example.com"),
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
						ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Cert: kong.String("foo"),
						Key:  kong.String("bar"),
					},
				},
				SNIs: []*kong.SNI{
					{
						ID:   kong.String("a53e9598-3a5e-4c12-a672-71a4cdcf7a47"),
						Name: kong.String("foo.example.com"),
						Certificate: &kong.Certificate{
							ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						},
					},
					{
						ID:   kong.String("5f8e6848-4cb9-479a-a27e-860e1a77f875"),
						Name: kong.String("bar.example.com"),
						Certificate: &kong.Certificate{
							ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
								Cert: kong.String("foo"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				CACertificates: []*kong.CACertificate{
					{
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Cert: kong.String("foo"),
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
								Cert: kong.String("foo"),
							},
						},
					},
				},
				currentState: existingCACertificateState(),
			},
			want: &utils.KongRawState{
				CACertificates: []*kong.CACertificate{
					{
						ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Cert: kong.String("foo"),
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
								Name:  kong.String("foo"),
								Slots: kong.Int(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Upstreams: []*kong.Upstream{
					{
						ID:    kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:  kong.String("foo"),
						Slots: kong.Int(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           kong.String("none"),
						HashFallback:     kong.String("none"),
						HashOnCookiePath: kong.String("/"),
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
								Name: kong.String("foo"),
							},
						},
					},
				},
				currentState: existingUpstreamState(),
			},
			want: &utils.KongRawState{
				Upstreams: []*kong.Upstream{
					{
						ID:    kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name:  kong.String("foo"),
						Slots: kong.Int(10000),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           kong.String("none"),
						HashFallback:     kong.String("none"),
						HashOnCookiePath: kong.String("/"),
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
								Name: kong.String("foo"),
							},
						},
						{
							Upstream: kong.Upstream{
								Name: kong.String("bar"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Upstreams: []*kong.Upstream{
					{
						ID:    kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:  kong.String("foo"),
						Slots: kong.Int(10000),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           kong.String("none"),
						HashFallback:     kong.String("none"),
						HashOnCookiePath: kong.String("/"),
					},
					{
						ID:    kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:  kong.String("bar"),
						Slots: kong.Int(10000),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           kong.String("none"),
						HashFallback:     kong.String("none"),
						HashOnCookiePath: kong.String("/"),
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
								Name:  kong.String("foo"),
								Slots: kong.Int(42),
								// not actually valid configuration, but this only needs to check that these translate
								// into the raw state
								HashOnQueryArg:         kong.String("foo"),
								HashFallbackQueryArg:   kong.String("foo"),
								HashOnURICapture:       kong.String("foo"),
								HashFallbackURICapture: kong.String("foo"),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Upstreams: []*kong.Upstream{
					{
						ID:    kong.String("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:  kong.String("foo"),
						Slots: kong.Int(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:                 kong.String("none"),
						HashFallback:           kong.String("none"),
						HashOnCookiePath:       kong.String("/"),
						HashOnQueryArg:         kong.String("foo"),
						HashFallbackQueryArg:   kong.String("foo"),
						HashOnURICapture:       kong.String("foo"),
						HashFallbackURICapture: kong.String("foo"),
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
							Name: kong.String("foo"),
							Document: &FDocument{
								Path:      kong.String("/foo.md"),
								Published: kong.Bool(true),
								Content:   kong.String("foo"),
							},
						},
					},
				},
				currentState: existingDocumentState(),
			},
			want: &utils.KonnectRawState{
				Documents: []*konnect.Document{
					{
						ID:        kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Path:      kong.String("/foo.md"),
						Published: kong.Bool(true),
						Content:   kong.String("foo"),
						Parent: &konnect.ServicePackage{
							ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
							Name: kong.String("foo"),
						},
					},
				},
				ServicePackages: []*konnect.ServicePackage{
					{
						ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name: kong.String("foo"),
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
							Name: kong.String("bar"),
							Document: &FDocument{
								Path:      kong.String("/bar.md"),
								Published: kong.Bool(true),
								Content:   kong.String("bar"),
							},
						},
					},
				},
				currentState: existingDocumentState(),
			},
			want: &utils.KonnectRawState{
				Documents: []*konnect.Document{
					{
						ID:        kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Path:      kong.String("/bar.md"),
						Published: kong.Bool(true),
						Content:   kong.String("bar"),
						Parent: &konnect.ServicePackage{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("bar"),
						},
					},
				},
				ServicePackages: []*konnect.ServicePackage{
					{
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: kong.String("bar"),
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
								Name: kong.String("foo-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: kong.String("foo-route1"),
									},
								},
								{
									Route: kong.Route{
										ID:   kong.String("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
										Name: kong.String("foo-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("bar-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: kong.String("bar-route1"),
									},
								},
								{
									Route: kong.Route{
										Name: kong.String("bar-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("large-payload-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name:              kong.String("dont-buffer-these"),
										RequestBuffering:  kong.Bool(false),
										ResponseBuffering: kong.Bool(false),
									},
								},
								{
									Route: kong.Route{
										Name:              kong.String("buffer-these"),
										RequestBuffering:  kong.Bool(true),
										ResponseBuffering: kong.Bool(true),
									},
								},
							},
						},
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  kong.String("foo"),
								Slots: kong.Int(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           kong.String("foo-service"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
					},
					{
						ID:             kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           kong.String("bar-service"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
					},
					{
						ID:             kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Name:           kong.String("large-payload-service"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
					},
				},
				Routes: []*kong.Route{
					{
						ID:            kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:          kong.String("foo-route1"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-service"),
						},
					},
					{
						ID:            kong.String("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
						Name:          kong.String("foo-route2"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-service"),
						},
					},
					{
						ID:            kong.String("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:          kong.String("bar-route1"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: kong.String("bar-service"),
						},
					},
					{
						ID:            kong.String("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:          kong.String("bar-route2"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: kong.String("bar-service"),
						},
					},
					{
						ID:            kong.String("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						Name:          kong.String("dont-buffer-these"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: kong.String("large-payload-service"),
						},
						RequestBuffering:  kong.Bool(false),
						ResponseBuffering: kong.Bool(false),
					},
					{
						ID:            kong.String("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Name:          kong.String("buffer-these"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: kong.String("large-payload-service"),
						},
						RequestBuffering:  kong.Bool(true),
						ResponseBuffering: kong.Bool(true),
					},
				},
				Upstreams: []*kong.Upstream{
					{
						ID:    kong.String("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						Name:  kong.String("foo"),
						Slots: kong.Int(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           kong.String("none"),
						HashFallback:     kong.String("none"),
						HashOnCookiePath: kong.String("/"),
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
								PathHandling:     kong.String("v0"),
								PreserveHost:     kong.Bool(false),
								RegexPriority:    kong.Int(0),
								StripPath:        kong.Bool(false),
								Protocols:        kong.StringSlice("http", "https"),
								RequestBuffering: kong.Bool(false),
							},
							Service: &kong.Service{
								Protocol:       kong.String("https"),
								ConnectTimeout: kong.Int(5000),
								WriteTimeout:   kong.Int(5000),
								ReadTimeout:    kong.Int(5000),
							},
							Upstream: &kong.Upstream{
								Slots: kong.Int(100),
								Healthchecks: &kong.Healthcheck{
									Active: &kong.ActiveHealthcheck{
										Concurrency: kong.Int(5),
										Healthy: &kong.Healthy{
											HTTPStatuses: []int{200, 302},
											Interval:     kong.Int(0),
											Successes:    kong.Int(0),
										},
										HTTPPath: kong.String("/"),
										Type:     kong.String("http"),
										Timeout:  kong.Int(1),
										Unhealthy: &kong.Unhealthy{
											HTTPFailures: kong.Int(0),
											TCPFailures:  kong.Int(0),
											Timeouts:     kong.Int(0),
											Interval:     kong.Int(0),
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
											Successes: kong.Int(0),
										},
										Unhealthy: &kong.Unhealthy{
											HTTPFailures: kong.Int(0),
											TCPFailures:  kong.Int(0),
											Timeouts:     kong.Int(0),
											HTTPStatuses: []int{429, 500, 503},
										},
									},
								},
								HashOn:           kong.String("none"),
								HashFallback:     kong.String("none"),
								HashOnCookiePath: kong.String("/"),
							},
						},
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name: kong.String("foo-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: kong.String("foo-route1"),
									},
								},
								{
									Route: kong.Route{
										ID:   kong.String("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
										Name: kong.String("foo-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("bar-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: kong.String("bar-route1"),
									},
								},
								{
									Route: kong.Route{
										Name: kong.String("bar-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("large-payload-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name:              kong.String("dont-buffer-these"),
										RequestBuffering:  kong.Bool(false),
										ResponseBuffering: kong.Bool(false),
									},
								},
								{
									Route: kong.Route{
										Name:              kong.String("buffer-these"),
										RequestBuffering:  kong.Bool(true),
										ResponseBuffering: kong.Bool(true),
									},
								},
							},
						},
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  kong.String("foo"),
								Slots: kong.Int(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           kong.String("foo-service"),
						Protocol:       kong.String("https"),
						ConnectTimeout: kong.Int(5000),
						WriteTimeout:   kong.Int(5000),
						ReadTimeout:    kong.Int(5000),
					},
					{
						ID:             kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           kong.String("bar-service"),
						Protocol:       kong.String("https"),
						ConnectTimeout: kong.Int(5000),
						WriteTimeout:   kong.Int(5000),
						ReadTimeout:    kong.Int(5000),
					},
					{
						ID:             kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Name:           kong.String("large-payload-service"),
						Protocol:       kong.String("https"),
						ConnectTimeout: kong.Int(5000),
						WriteTimeout:   kong.Int(5000),
						ReadTimeout:    kong.Int(5000),
					},
				},
				Routes: []*kong.Route{
					{
						ID:               kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:             kong.String("foo-route1"),
						PreserveHost:     kong.Bool(false),
						RegexPriority:    kong.Int(0),
						StripPath:        kong.Bool(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: kong.Bool(false),
						PathHandling:     kong.String("v0"),
						Service: &kong.Service{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-service"),
						},
					},
					{
						ID:               kong.String("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
						Name:             kong.String("foo-route2"),
						PreserveHost:     kong.Bool(false),
						RegexPriority:    kong.Int(0),
						StripPath:        kong.Bool(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: kong.Bool(false),
						PathHandling:     kong.String("v0"),
						Service: &kong.Service{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-service"),
						},
					},
					{
						ID:               kong.String("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:             kong.String("bar-route1"),
						PreserveHost:     kong.Bool(false),
						RegexPriority:    kong.Int(0),
						StripPath:        kong.Bool(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: kong.Bool(false),
						PathHandling:     kong.String("v0"),
						Service: &kong.Service{
							ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: kong.String("bar-service"),
						},
					},
					{
						ID:               kong.String("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:             kong.String("bar-route2"),
						PreserveHost:     kong.Bool(false),
						RegexPriority:    kong.Int(0),
						StripPath:        kong.Bool(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: kong.Bool(false),
						PathHandling:     kong.String("v0"),
						Service: &kong.Service{
							ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: kong.String("bar-service"),
						},
					},
					{
						ID:            kong.String("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						Name:          kong.String("dont-buffer-these"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						PathHandling:  kong.String("v0"),
						Service: &kong.Service{
							ID:   kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: kong.String("large-payload-service"),
						},
						RequestBuffering:  kong.Bool(false),
						ResponseBuffering: kong.Bool(false),
					},
					{
						ID:            kong.String("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Name:          kong.String("buffer-these"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						PathHandling:  kong.String("v0"),
						Service: &kong.Service{
							ID:   kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: kong.String("large-payload-service"),
						},
						RequestBuffering:  kong.Bool(true),
						ResponseBuffering: kong.Bool(true),
					},
				},
				Upstreams: []*kong.Upstream{
					{
						ID:    kong.String("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						Name:  kong.String("foo"),
						Slots: kong.Int(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(5),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           kong.String("none"),
						HashFallback:     kong.String("none"),
						HashOnCookiePath: kong.String("/"),
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
								Name: kong.String("foo-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: kong.String("foo-route1"),
									},
								},
								{
									Route: kong.Route{
										ID:   kong.String("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
										Name: kong.String("foo-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("bar-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: kong.String("bar-route1"),
									},
								},
								{
									Route: kong.Route{
										Name: kong.String("bar-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("large-payload-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name:              kong.String("dont-buffer-these"),
										RequestBuffering:  kong.Bool(false),
										ResponseBuffering: kong.Bool(false),
									},
								},
								{
									Route: kong.Route{
										Name:              kong.String("buffer-these"),
										RequestBuffering:  kong.Bool(true),
										ResponseBuffering: kong.Bool(true),
									},
								},
							},
						},
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  kong.String("foo"),
								Slots: kong.Int(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           kong.String("foo-service"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
					},
					{
						ID:             kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           kong.String("bar-service"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
					},
					{
						ID:             kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Name:           kong.String("large-payload-service"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
					},
				},
				Routes: []*kong.Route{
					{
						ID:            kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:          kong.String("foo-route1"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-service"),
						},
					},
					{
						ID:            kong.String("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
						Name:          kong.String("foo-route2"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-service"),
						},
					},
					{
						ID:            kong.String("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:          kong.String("bar-route1"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: kong.String("bar-service"),
						},
					},
					{
						ID:            kong.String("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:          kong.String("bar-route2"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: kong.String("bar-service"),
						},
					},
					{
						ID:            kong.String("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						Name:          kong.String("dont-buffer-these"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: kong.String("large-payload-service"),
						},
						RequestBuffering:  kong.Bool(false),
						ResponseBuffering: kong.Bool(false),
					},
					{
						ID:            kong.String("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Name:          kong.String("buffer-these"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: kong.String("large-payload-service"),
						},
						RequestBuffering:  kong.Bool(true),
						ResponseBuffering: kong.Bool(true),
					},
				},
				Upstreams: []*kong.Upstream{
					{
						ID:    kong.String("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						Name:  kong.String("foo"),
						Slots: kong.Int(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(10),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           kong.String("none"),
						HashFallback:     kong.String("none"),
						HashOnCookiePath: kong.String("/"),
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
								PreserveHost:     kong.Bool(false),
								StripPath:        kong.Bool(false),
								Protocols:        kong.StringSlice("http", "https"),
								RequestBuffering: kong.Bool(false),
							},
							Service: &kong.Service{
								Protocol:       kong.String("https"),
								ConnectTimeout: kong.Int(5000),
								WriteTimeout:   kong.Int(5000),
								ReadTimeout:    kong.Int(5000),
							},
							Upstream: &kong.Upstream{
								Slots: kong.Int(100),
								Healthchecks: &kong.Healthcheck{
									Active: &kong.ActiveHealthcheck{
										Concurrency: kong.Int(5),
										Healthy: &kong.Healthy{
											HTTPStatuses: []int{200, 302},
											Interval:     kong.Int(0),
											Successes:    kong.Int(0),
										},
										HTTPPath: kong.String("/"),
										Type:     kong.String("http"),
										Timeout:  kong.Int(1),
										Unhealthy: &kong.Unhealthy{
											HTTPFailures: kong.Int(0),
											TCPFailures:  kong.Int(0),
											Timeouts:     kong.Int(0),
											Interval:     kong.Int(0),
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
											Successes: kong.Int(0),
										},
										Unhealthy: &kong.Unhealthy{
											HTTPFailures: kong.Int(0),
											TCPFailures:  kong.Int(0),
											Timeouts:     kong.Int(0),
											HTTPStatuses: []int{429, 500, 503},
										},
									},
								},
								HashOn:           kong.String("none"),
								HashFallback:     kong.String("none"),
								HashOnCookiePath: kong.String("/"),
							},
						},
					},
					Services: []FService{
						{
							Service: kong.Service{
								Name: kong.String("foo-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: kong.String("foo-route1"),
									},
								},
								{
									Route: kong.Route{
										ID:   kong.String("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
										Name: kong.String("foo-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("bar-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name: kong.String("bar-route1"),
									},
								},
								{
									Route: kong.Route{
										Name: kong.String("bar-route2"),
									},
								},
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("large-payload-service"),
							},
							Routes: []*FRoute{
								{
									Route: kong.Route{
										Name:              kong.String("dont-buffer-these"),
										RequestBuffering:  kong.Bool(false),
										ResponseBuffering: kong.Bool(false),
									},
								},
								{
									Route: kong.Route{
										Name:              kong.String("buffer-these"),
										RequestBuffering:  kong.Bool(true),
										ResponseBuffering: kong.Bool(true),
									},
								},
							},
						},
					},
					Upstreams: []FUpstream{
						{
							Upstream: kong.Upstream{
								Name:  kong.String("foo"),
								Slots: kong.Int(42),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Services: []*kong.Service{
					{
						ID:             kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name:           kong.String("foo-service"),
						Protocol:       kong.String("https"),
						ConnectTimeout: kong.Int(5000),
						WriteTimeout:   kong.Int(5000),
						ReadTimeout:    kong.Int(5000),
					},
					{
						ID:             kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name:           kong.String("bar-service"),
						Protocol:       kong.String("https"),
						ConnectTimeout: kong.Int(5000),
						WriteTimeout:   kong.Int(5000),
						ReadTimeout:    kong.Int(5000),
					},
					{
						ID:             kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						Name:           kong.String("large-payload-service"),
						Protocol:       kong.String("https"),
						ConnectTimeout: kong.Int(5000),
						WriteTimeout:   kong.Int(5000),
						ReadTimeout:    kong.Int(5000),
					},
				},
				Routes: []*kong.Route{
					{
						ID:               kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:             kong.String("foo-route1"),
						PreserveHost:     kong.Bool(false),
						RegexPriority:    kong.Int(0),
						StripPath:        kong.Bool(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: kong.Bool(false),
						Service: &kong.Service{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-service"),
						},
					},
					{
						ID:               kong.String("d125e79a-297c-414b-bc00-ad3a87be6c2b"),
						Name:             kong.String("foo-route2"),
						PreserveHost:     kong.Bool(false),
						RegexPriority:    kong.Int(0),
						StripPath:        kong.Bool(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: kong.Bool(false),
						Service: &kong.Service{
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-service"),
						},
					},
					{
						ID:               kong.String("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:             kong.String("bar-route1"),
						PreserveHost:     kong.Bool(false),
						RegexPriority:    kong.Int(0),
						StripPath:        kong.Bool(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: kong.Bool(false),
						Service: &kong.Service{
							ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: kong.String("bar-service"),
						},
					},
					{
						ID:               kong.String("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:             kong.String("bar-route2"),
						PreserveHost:     kong.Bool(false),
						RegexPriority:    kong.Int(0),
						StripPath:        kong.Bool(false),
						Protocols:        kong.StringSlice("http", "https"),
						RequestBuffering: kong.Bool(false),
						Service: &kong.Service{
							ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
							Name: kong.String("bar-service"),
						},
					},
					{
						ID:            kong.String("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						Name:          kong.String("dont-buffer-these"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: kong.String("large-payload-service"),
						},
						RequestBuffering:  kong.Bool(false),
						ResponseBuffering: kong.Bool(false),
					},
					{
						ID:            kong.String("13dd1aac-04ce-4ea2-877c-5579cfa2c78e"),
						Name:          kong.String("buffer-these"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
						Protocols:     kong.StringSlice("http", "https"),
						Service: &kong.Service{
							ID:   kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
							Name: kong.String("large-payload-service"),
						},
						RequestBuffering:  kong.Bool(true),
						ResponseBuffering: kong.Bool(true),
					},
				},
				Upstreams: []*kong.Upstream{
					{
						ID:    kong.String("1b0bafae-881b-42a7-9110-8a42ed3c903c"),
						Name:  kong.String("foo"),
						Slots: kong.Int(42),
						Healthchecks: &kong.Healthcheck{
							Active: &kong.ActiveHealthcheck{
								Concurrency: kong.Int(5),
								Healthy: &kong.Healthy{
									HTTPStatuses: []int{200, 302},
									Interval:     kong.Int(0),
									Successes:    kong.Int(0),
								},
								HTTPPath: kong.String("/"),
								Type:     kong.String("http"),
								Timeout:  kong.Int(1),
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									Interval:     kong.Int(0),
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
									Successes: kong.Int(0),
								},
								Unhealthy: &kong.Unhealthy{
									HTTPFailures: kong.Int(0),
									TCPFailures:  kong.Int(0),
									Timeouts:     kong.Int(0),
									HTTPStatuses: []int{429, 500, 503},
								},
							},
						},
						HashOn:           kong.String("none"),
						HashFallback:     kong.String("none"),
						HashOnCookiePath: kong.String("/"),
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
					ConfigSource: kong.String("foo"),
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
					ConfigSource: kong.String("foo"),
					Plugin: kong.Plugin{
						Config: kong.Configuration{
							"k1": "v1",
							"k2": "v2",
						},
					},
				},
			},
			result: FPlugin{
				ConfigSource: kong.String("foo"),
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
				Protocols: []*string{kong.String("grpc")},
				StripPath: kong.Bool(true),
			},
			wantErr: true,
		},
		{
			name: "true strip_path and grpcs protocol",
			route: kong.Route{
				Protocols: []*string{kong.String("grpcs")},
				StripPath: kong.Bool(true),
			},
			wantErr: true,
		},
		{
			name: "no strip_path and http protocol",
			route: kong.Route{
				Protocols: []*string{kong.String("http")},
			},
			expectedStripPath: nil,
		},
		{
			name: "no strip_path and grpc protocol",
			route: kong.Route{
				Protocols: []*string{kong.String("grpc")},
			},
			expectedStripPath: kong.Bool(false),
		},
		{
			name: "no strip_path and grpcs protocol",
			route: kong.Route{
				Protocols: []*string{kong.String("grpcs")},
			},
			expectedStripPath: kong.Bool(false),
		},
		{
			name: "false strip_path and grpc protocol",
			route: kong.Route{
				Protocols: []*string{kong.String("grpc")},
				StripPath: kong.Bool(false),
			},
			expectedStripPath: kong.Bool(false),
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
						Name: kong.String("foo"),
					},
				},
			},
			wantErr: false,
			wantState: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						Name:          kong.String("foo"),
						PreserveHost:  kong.Bool(false),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
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
								Name:       kong.String("foo"),
								Expression: kong.String(`'(http.path == "/test") || (http.path ^= "/test/")'`),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						Name:         kong.String("foo"),
						PreserveHost: kong.Bool(false),
						Expression:   kong.String(`'(http.path == "/test") || (http.path ^= "/test/")'`),
						Priority:     kong.Uint64(0),
						StripPath:    kong.Bool(false),
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
								Name:       kong.String("foo"),
								Expression: kong.String(`'(http.path == "/test") || (http.path ^= "/test/")'`),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						Name:          kong.String("foo"),
						PreserveHost:  kong.Bool(false),
						Expression:    kong.String(`'(http.path == "/test") || (http.path ^= "/test/")'`),
						Priority:      kong.Uint64(0),
						RegexPriority: kong.Int(0),
						StripPath:     kong.Bool(false),
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
								Name:       kong.String("foo"),
								Expression: kong.String(`'(http.path == "/test") || (http.path ^= "/test/")'`),
							},
						},
					},
				},
				currentState: existingServiceState(),
			},
			want: &utils.KongRawState{
				Routes: []*kong.Route{
					{
						Name:         kong.String("foo"),
						PreserveHost: kong.Bool(false),
						Expression:   kong.String(`'(http.path == "/test") || (http.path ^= "/test/")'`),
						Priority:     kong.Uint64(0),
						StripPath:    kong.Bool(false),
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
							Type: kong.String("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   kong.String("/foo"),
								"query": kong.String("query { foo { bar }}"),
								"service": map[string]interface{}{
									"id": "fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd",
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
						ID:    kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						URI:   kong.String("/foo"),
						Query: kong.String("query { foo { bar }}"),
						Service: &kong.Service{
							ID: kong.String("fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd"),
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
							Type: kong.String("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   kong.String("/example"),
								"query": kong.String("query{ example { foo } }"),
								"service": map[string]interface{}{
									"id": "fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd",
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
						ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Service: &kong.Service{
							ID: kong.String("fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd"),
						},
						Methods: kong.StringSlice("GET"),
						URI:     kong.String("/example"),
						Query:   kong.String("query{ example { foo } }"),
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
								Name: kong.String("foo"),
							},
						},
					},
					CustomEntities: []FCustomEntity{
						{
							Type: kong.String("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri": kong.String("/foo"),
								"query": kong.String(`query SearchPosts($filters: PostsFilters) {
		      								posts(filter: $filters) {
		        								id
		        								title
		        								author
		      								}
										}`),
								"service": map[string]interface{}{
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
						ID: kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Service: &kong.Service{
							ID: kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						},
						Methods: kong.StringSlice("GET", "POST"),
						URI:     kong.String("/foo"),
						Query: kong.String(`query SearchPosts($filters: PostsFilters) {
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
						ID:             kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name:           kong.String("foo"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
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
								Name: kong.String("service1"),
							},
						},
						{
							Service: kong.Service{
								Name: kong.String("service2"),
							},
						},
					},
					CustomEntities: []FCustomEntity{
						{
							Type: kong.String("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   kong.String("/foo"),
								"query": kong.String("query { foo }"),
								"service": map[string]interface{}{
									"name": "service1",
								},
							},
						},
						{
							Type: kong.String("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   kong.String("/bar"),
								"query": kong.String("query { bar }"),
								"service": map[string]interface{}{
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
						ID:    kong.String("9e6f82e5-4e74-4e81-a79e-4bbd6fe34cdc"),
						URI:   kong.String("/foo"),
						Query: kong.String("query { foo }"),
						Service: &kong.Service{
							ID: kong.String("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						},
						Methods: kong.StringSlice("GET"),
					},
					{
						ID:    kong.String("ba843ee8-d63e-4c4f-be1c-ebea546d8fac"),
						URI:   kong.String("/bar"),
						Query: kong.String("query { bar }"),
						Service: &kong.Service{
							ID: kong.String("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						},
						Methods: kong.StringSlice("POST", "PUT"),
					},
				},
				Services: []*kong.Service{
					{
						ID:             kong.String("0cc0d614-4c88-4535-841a-cbe0709b0758"),
						Name:           kong.String("service1"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
					},
					{
						ID:             kong.String("083f61d3-75bc-42b4-9df4-f91929e18fda"),
						Name:           kong.String("service2"),
						Protocol:       kong.String("http"),
						ConnectTimeout: kong.Int(60000),
						WriteTimeout:   kong.Int(60000),
						ReadTimeout:    kong.Int(60000),
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
							Type: kong.String("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri":   kong.String("/foo"),
								"query": kong.String("query{ example { foo } }"),
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
							Type: kong.String("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"query": kong.String("query{ example { foo } }"),
								"service": map[string]interface{}{
									"id": "fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd",
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
							Type: kong.String("degraphql_routes"),
							Fields: CustomEntityConfiguration{
								"uri": kong.String("/foo"),
								"service": map[string]interface{}{
									"id": "fdfd14cc-cd69-49a0-9e23-cd3375b6c0cd",
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
								Name: kong.String("foo-group"),
							},
							Consumers: nil,
							Plugins: []*kong.ConsumerGroupPlugin{
								{
									Name: kong.String("rate-limiting-advanced"),
									Config: kong.Configuration{
										"limit":       []any{float64(100)},
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
								Name: kong.String("rate-limiting-advanced"),
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
									"limit":                   []any{float64(10)},
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
							ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
							Name: kong.String("foo-group"),
						},
						Consumers: nil,
						Plugins: []*kong.ConsumerGroupPlugin{
							{
								ID:   kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
								Name: kong.String("rate-limiting-advanced"),
								Config: kong.Configuration{
									"limit":       []any{float64(100)},
									"window_size": []any{float64(60)},
									"window_type": string("fixed"),
								},
							},
						},
					},
				},
				Plugins: []*kong.Plugin{
					{
						ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: kong.String("rate-limiting-advanced"),
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
							"limit":                   []any{float64(10)},
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
			fields: fields{
				targetContent: &Content{
					Info: &Info{
						Defaults: kongDefaults,
					},
					ConsumerGroups: []FConsumerGroupObject{
						{
							ConsumerGroup: kong.ConsumerGroup{
								Name: kong.String("foo-group"),
							},
							Consumers: nil,
							Plugins: []*kong.ConsumerGroupPlugin{
								{
									Name: kong.String("rate-limiting-advanced"),
									Config: kong.Configuration{
										"limit":       []any{float64(100)},
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
								Name: kong.String("rate-limiting-advanced"),
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
									"limit":                   []any{float64(10)},
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
								Name: kong.String("my-foo-partial"),
								Type: kong.String("foo"),
								Config: kong.Configuration{
									"key1": "value1",
									"key2": []any{"a", "b", "c"},
									"key3": map[string]interface{}{
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
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: kong.String("my-foo-partial"),
						Type: kong.String("foo"),
						Config: kong.Configuration{
							"key1": "value1",
							"key2": []any{"a", "b", "c"},
							"key3": map[string]interface{}{
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
								Name: kong.String("existing-partial"),
								Type: kong.String("foo"),
								Config: kong.Configuration{
									"key1": "value1",
									"key2": []any{"a", "b", "c"},
									"key3": map[string]interface{}{
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
						ID:   kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
						Name: kong.String("existing-partial"),
						Type: kong.String("foo"),
						Config: kong.Configuration{
							"key1": "value1",
							"key2": []any{"a", "b", "c"},
							"key3": map[string]interface{}{
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
								ID:   kong.String("provided-id"),
								Name: kong.String("test-partial"),
								Type: kong.String("foo"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Partials: []*kong.Partial{
					{
						ID:   kong.String("provided-id"),
						Name: kong.String("test-partial"),
						Type: kong.String("foo"),
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
								Name: kong.String("foo-partial"),
								Type: kong.String("foo"),
							},
						},
						{
							Partial: kong.Partial{
								Name: kong.String("bar-partial"),
								Type: kong.String("bar"),
							},
						},
					},
				},
				currentState: emptyState(),
			},
			want: &utils.KongRawState{
				Partials: []*kong.Partial{
					{
						ID:   kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name: kong.String("foo-partial"),
						Type: kong.String("foo"),
					},
					{
						ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: kong.String("bar-partial"),
						Type: kong.String("bar"),
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
								Name: kong.String("key-auth"),
								Consumer: &kong.Consumer{
									ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
						Name: kong.String("key-auth"),
						Consumer: &kong.Consumer{
							ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
								Name: kong.String("rate-limiting"),
								Service: &kong.Service{
									ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
						Name: kong.String("rate-limiting"),
						Service: &kong.Service{
							ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
								Name: kong.String("cors"),
								Route: &kong.Route{
									ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
						Name: kong.String("cors"),
						Route: &kong.Route{
							ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
								Name: kong.String("rate-limiting"),
								ConsumerGroup: &kong.ConsumerGroup{
									ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
						Name: kong.String("rate-limiting"),
						ConsumerGroup: &kong.ConsumerGroup{
							ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
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
								Name: kong.String("custom-plugin"),
								Partials: []*kong.PartialLink{
									{
										Partial: &kong.Partial{
											ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
										},
										Path: kong.String("config.custom_path"),
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
						Name: kong.String("custom-plugin"),
						Partials: []*kong.PartialLink{
							{
								Partial: &kong.Partial{
									ID: kong.String("4bfcb11f-c962-4817-83e5-9433cf20b663"),
								},
								Path: kong.String("config.custom_path"),
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
								Name: kong.String("key-auth"),
								Consumer: &kong.Consumer{
									ID: kong.String("non-existent"),
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
								Name: kong.String("key-auth"),
								Service: &kong.Service{
									ID: kong.String("non-existent"),
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
								Name: kong.String("key-auth"),
								Route: &kong.Route{
									ID: kong.String("non-existent"),
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
								Name: kong.String("key-auth"),
								ConsumerGroup: &kong.ConsumerGroup{
									ID: kong.String("non-existent"),
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
								Name: kong.String("custom-plugin"),
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
			wantErr: "partial for plugin custom-plugin: partial ID or name is required",
		},
		{
			name: "error when partial is not found",
			fields: fields{
				targetContent: &Content{
					Plugins: []FPlugin{
						{
							Plugin: kong.Plugin{
								Name: kong.String("custom-plugin"),
								Partials: []*kong.PartialLink{
									{
										Partial: &kong.Partial{
											ID: kong.String("non-existent"),
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
								Name: kong.String("foo"),
								KID:  kong.String("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
								JWK:  kong.String("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
							},
						},
						{
							Key: kong.Key{
								Name: kong.String("my-pem-key"),
								KID:  kong.String("my-pem-key"),
								PEM: &kong.PEM{
									PrivateKey: kong.String("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD\n-----END PRIVATE KEY-----\n"), //nolint:lll
									PublicKey:  kong.String("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"),          //nolint:lll
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
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: kong.String("foo"),
						KID:  kong.String("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
						JWK:  kong.String("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
					},
					{
						ID:   kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name: kong.String("my-pem-key"),
						KID:  kong.String("my-pem-key"),
						PEM: &kong.PEM{
							PrivateKey: kong.String("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD\n-----END PRIVATE KEY-----\n"), //nolint:lll
							PublicKey:  kong.String("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"),          //nolint:lll
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
								Name: kong.String("foo"),
								KID:  kong.String("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
								JWK:  kong.String("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
							},
						},
					},
				},
				currentState: existingKeyState(t),
			},
			want: &utils.KongRawState{
				Keys: []*kong.Key{
					{
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: kong.String("foo"),
						KID:  kong.String("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
						JWK:  kong.String("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
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
								Name: kong.String("set-1"),
							},
						},
					},
					Keys: []FKey{
						{
							Key: kong.Key{
								Name: kong.String("foo"),
								KID:  kong.String("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
								JWK:  kong.String("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
								Set: &kong.KeySet{
									Name: kong.String("set-1"),
								},
							},
						},
						{
							Key: kong.Key{
								Name: kong.String("my-pem-key"),
								KID:  kong.String("my-pem-key"),
								PEM: &kong.PEM{
									PrivateKey: kong.String("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD\n-----END PRIVATE KEY-----\n"), //nolint:lll
									PublicKey:  kong.String("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"),          //nolint:lll
								},
								Set: &kong.KeySet{
									Name: kong.String("set-1"),
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
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: kong.String("set-1"),
					},
				},
				Keys: []*kong.Key{
					{
						ID:   kong.String("5b1484f2-5209-49d9-b43e-92ba09dd9d52"),
						Name: kong.String("foo"),
						KID:  kong.String("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
						JWK:  kong.String("{\"kid\":\"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\",\"kty\":\"RSA\",\"alg\":\"A256GCM\",\"n\":\"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\",\"e\":\"AQAB\"}"), //nolint:lll
						Set: &kong.KeySet{
							Name: kong.String("set-1"),
						},
					},
					{
						ID:   kong.String("dfd79b4d-7642-4b61-ba0c-9f9f0d3ba55b"),
						Name: kong.String("my-pem-key"),
						KID:  kong.String("my-pem-key"),
						PEM: &kong.PEM{
							PrivateKey: kong.String("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQD\n-----END PRIVATE KEY-----\n"), //nolint:lll
							PublicKey:  kong.String("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA\n-----END PUBLIC KEY-----\n"),          //nolint:lll
						},
						Set: &kong.KeySet{
							Name: kong.String("set-1"),
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
								Name: kong.String("existing-set"),
							},
						},
					},
				},
				currentState: existingKeySetState(t),
			},
			want: &utils.KongRawState{
				KeySets: []*kong.KeySet{
					{
						ID:   kong.String("538c7f96-b164-4f1b-97bb-9f4bb472e89f"),
						Name: kong.String("existing-set"),
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
