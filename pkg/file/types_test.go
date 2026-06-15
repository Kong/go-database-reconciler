package file

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

const testMyID = "my-id"

var (
	jsonString = `{
  "name": "rate-limiting",
  "config": {
    "minute": 10
  },
  "service": "foo",
  "route": "bar",
  "consumer": "baz",
  "enabled": true,
  "run_on": "first",
  "protocols": [
    "http"
  ]
}`
	yamlString = `
name: rate-limiting
config:
  minute: 10
service: foo
consumer: baz
route: bar
enabled: true
run_on: first
protocols:
- http
`
)

func jsonRawMessage(s string) *json.RawMessage {
	j := json.RawMessage(s)
	return &j
}

func Test_sortKey(t *testing.T) {
	tests := []struct {
		name        string
		sortable    sortable
		expectedKey string
	}{
		{
			sortable: &FService{
				Service: kong.Service{
					Name: new("my-service"),
					ID:   new(testMyID),
				},
			},
			expectedKey: "my-service",
		},
		{
			sortable: &FService{
				Service: kong.Service{
					ID: new(testMyID),
				},
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FService{},
			expectedKey: "",
		},
		{
			sortable: &FRoute{
				Route: kong.Route{
					Name: new("my-route"),
					ID:   new(testMyID),
				},
			},
			expectedKey: "my-route",
		},
		{
			sortable: FRoute{
				Route: kong.Route{
					ID: new(testMyID),
				},
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FRoute{},
			expectedKey: "",
		},
		{
			sortable: FUpstream{
				Upstream: kong.Upstream{
					Name: new("my-upstream"),
					ID:   new(testMyID),
				},
			},
			expectedKey: "my-upstream",
		},
		{
			sortable: FUpstream{
				Upstream: kong.Upstream{
					ID: new(testMyID),
				},
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FUpstream{},
			expectedKey: "",
		},
		{
			sortable: FTarget{
				Target: kong.Target{
					Target: new("my-target"),
					ID:     new(testMyID),
				},
			},
			expectedKey: "my-target",
		},
		{
			sortable: FTarget{
				Target: kong.Target{
					ID: new(testMyID),
				},
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FTarget{},
			expectedKey: "",
		},
		{
			sortable: FCertificate{
				Cert: new("my-certificate"),
				ID:   new(testMyID),
			},
			expectedKey: "my-certificate",
		},
		{
			sortable: FCertificate{
				ID: new(testMyID),
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FCertificate{},
			expectedKey: "",
		},
		{
			sortable: FCACertificate{
				CACertificate: kong.CACertificate{
					Cert: new("my-ca-certificate"),
					ID:   new(testMyID),
				},
			},
			expectedKey: "my-ca-certificate",
		},
		{
			sortable: FCACertificate{
				CACertificate: kong.CACertificate{
					ID: new(testMyID),
				},
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FCACertificate{},
			expectedKey: "",
		},
		{
			sortable: FPlugin{
				Plugin: kong.Plugin{
					Name: new("my-plugin"),
					ID:   new(testMyID),
				},
			},
			expectedKey: "my-plugin",
		},
		{
			sortable: FPlugin{
				Plugin: kong.Plugin{
					Name: new("my-plugin"),
					ID:   new(testMyID),
					Consumer: &kong.Consumer{
						ID: new("my-consumer-id"),
					},
				},
			},
			expectedKey: "my-pluginmy-consumer-id",
		},
		{
			sortable: FPlugin{
				Plugin: kong.Plugin{
					Name: new("my-plugin"),
					ID:   new(testMyID),
					Route: &kong.Route{
						ID: new("my-route-id"),
					},
				},
			},
			expectedKey: "my-pluginmy-route-id",
		},
		{
			sortable: FPlugin{
				Plugin: kong.Plugin{
					Name: new("my-plugin"),
					ID:   new(testMyID),
					Service: &kong.Service{
						ID: new("my-service-id"),
					},
				},
			},
			expectedKey: "my-pluginmy-service-id",
		},

		{
			sortable: FPlugin{
				Plugin: kong.Plugin{
					ID: new(testMyID),
				},
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FPlugin{},
			expectedKey: "",
		},
		{
			sortable: &FConsumer{
				Consumer: kong.Consumer{
					Username: new("my-consumer"),
					ID:       new(testMyID),
				},
			},
			expectedKey: "my-consumer",
		},
		{
			sortable: &FConsumer{
				Consumer: kong.Consumer{
					ID: new(testMyID),
				},
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FConsumer{},
			expectedKey: "",
		},
		{
			sortable: &FServicePackage{
				Name: new("my-service-package"),
				ID:   new(testMyID),
			},
			expectedKey: "my-service-package",
		},
		{
			sortable: &FServicePackage{
				ID: new(testMyID),
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FServicePackage{},
			expectedKey: "",
		},
		{
			sortable: &FServiceVersion{
				Version: new("my-service-version"),
				ID:      new(testMyID),
			},
			expectedKey: "my-service-version",
		},
		{
			sortable: &FServiceVersion{
				ID: new(testMyID),
			},
			expectedKey: testMyID,
		},
		{
			sortable:    FServiceVersion{},
			expectedKey: "",
		},
		{
			sortable: &FFilterChain{
				FilterChain: kong.FilterChain{
					Name: new("my-name"),
				},
			},
			expectedKey: "my-name",
		},
		{
			sortable: &FFilterChain{
				FilterChain: kong.FilterChain{
					ID:   new(testMyID),
					Name: new("my-name"),
				},
			},
			expectedKey: "my-name",
		},
		{
			sortable: &FFilterChain{
				FilterChain: kong.FilterChain{
					ID: new(testMyID),
				},
			},
			expectedKey: testMyID,
		},
		{
			sortable: &FFilterChain{
				FilterChain: kong.FilterChain{},
			},
			expectedKey: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.sortable.sortKey()
			if key != tt.expectedKey {
				t.Errorf("Expected %v, but is %v", tt.expectedKey, key)
			}
		})
	}
}

func TestPluginUnmarshalYAML(t *testing.T) {
	var p FPlugin
	require.NoError(t, yaml.Unmarshal([]byte(yamlString), &p))
	assert.Equal(t, kong.Plugin{
		Name:      p.Name,
		Config:    p.Config,
		Enabled:   p.Enabled,
		RunOn:     p.RunOn,
		Protocols: p.Protocols,
		Service: &kong.Service{
			ID: new("foo"),
		},
		Consumer: &kong.Consumer{
			ID: new("baz"),
		},
		Route: &kong.Route{
			ID: new("bar"),
		},
	}, p.Plugin)
}

func TestPluginUnmarshalJSON(t *testing.T) {
	var p FPlugin
	require.NoError(t, json.Unmarshal([]byte(jsonString), &p))
	assert.Equal(t, kong.Plugin{
		Name:      p.Name,
		Config:    p.Config,
		Enabled:   p.Enabled,
		RunOn:     p.RunOn,
		Protocols: p.Protocols,
		Service: &kong.Service{
			ID: new("foo"),
		},
		Consumer: &kong.Consumer{
			ID: new("baz"),
		},
		Route: &kong.Route{
			ID: new("bar"),
		},
	}, p.Plugin)
}

func TestFilterChainUnmarshalJSON(t *testing.T) {
	var fc FFilterChain
	fcJSON := `{
	"name": "my-filter-chain",
	"id": "fa7bd007-e0c6-4ef2-b254-e60d3a341b0c",
	"enabled": true,
	"filters": [
		{
			"name": "my-filter",
			"config": {"a":1}
		},
		{
			"name": "my-other-filter",
			"config": "config!",
			"enabled": false
		}
	]
}`

	assert := assert.New(t)
	require.NoError(t, json.Unmarshal([]byte(fcJSON), &fc))
	assert.Equal(kong.FilterChain{
		Name:    new("my-filter-chain"),
		ID:      new("fa7bd007-e0c6-4ef2-b254-e60d3a341b0c"),
		Enabled: new(true),
		Filters: []*kong.Filter{
			{
				Name:   new("my-filter"),
				Config: jsonRawMessage(`{"a":1}`),
			},
			{
				Name:    new("my-other-filter"),
				Config:  jsonRawMessage(`"config!"`),
				Enabled: new(false),
			},
		},
	}, fc.FilterChain)
}

func TestFilterChainUnmarshalYAML(t *testing.T) {
	var fc FFilterChain
	fcYaml := `
name: my-filter-chain
id: fa7bd007-e0c6-4ef2-b254-e60d3a341b0c
enabled: true
filters:
  - name: my-filter
    config:
      a: 1
  - name: my-other-filter
    config: config!
    enabled: false
`

	assert := assert.New(t)
	require.NoError(t, yaml.Unmarshal([]byte(fcYaml), &fc))
	assert.Equal(kong.FilterChain{
		Name:    new("my-filter-chain"),
		ID:      new("fa7bd007-e0c6-4ef2-b254-e60d3a341b0c"),
		Enabled: new(true),
		Filters: []*kong.Filter{
			{
				Name:   new("my-filter"),
				Config: jsonRawMessage(`{"a":1}`),
			},
			{
				Name:    new("my-other-filter"),
				Config:  jsonRawMessage(`"config!"`),
				Enabled: new(false),
			},
		},
	}, fc.FilterChain)
}

func Test_unwrapURL(t *testing.T) {
	type args struct {
		urlString string
		fService  *FService
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				urlString: "https://foo.com:8008/bar",
				fService: &FService{
					Service: kong.Service{
						Host:     new("foo.com"),
						Port:     new(8008),
						Protocol: new("https"),
						Path:     new("/bar"),
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				urlString: "https://foo.com/bar",
				fService: &FService{
					Service: kong.Service{
						Host:     new("foo.com"),
						Protocol: new("https"),
						Path:     new("/bar"),
						Port:     new(443),
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				urlString: "https://foo.com:4224/",
				fService: &FService{
					Service: kong.Service{
						Host:     new("foo.com"),
						Protocol: new("https"),
						Path:     new("/"),
						Port:     new(4224),
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				urlString: "https://foo.com/",
				fService: &FService{
					Service: kong.Service{
						Host:     new("foo.com"),
						Protocol: new("https"),
						Path:     new("/"),
						Port:     new(443),
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				urlString: "http://foo.com:4242",
				fService: &FService{
					Service: kong.Service{
						Host:     new("foo.com"),
						Protocol: new("http"),
						Port:     new(4242),
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				urlString: "http://foo.com",
				fService: &FService{
					Service: kong.Service{
						Host:     new("foo.com"),
						Protocol: new("http"),
						Port:     new(80),
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				urlString: "grpc://foocom",
				fService: &FService{
					Service: kong.Service{
						Host:     new("foocom"),
						Protocol: new("grpc"),
						Port:     new(80),
					},
				},
			},
			wantErr: false,
		},
		{
			args: args{
				urlString: "foo.com/sdf",
				fService: &FService{
					Service: kong.Service{},
				},
			},
			wantErr: true,
		},
		{
			args: args{
				urlString: "foo.com",
				fService: &FService{
					Service: kong.Service{},
				},
			},
			wantErr: true,
		},
		{
			args: args{
				urlString: "42:",
				fService: &FService{
					Service: kong.Service{},
				},
			},
			wantErr: true,
		},
		{
			args: args{
				urlString: "http://foo.com/Spaced%20Test/bar",
				fService: &FService{
					Service: kong.Service{
						Host:     new("foo.com"),
						Protocol: new("http"),
						Port:     new(80),
						Path:     new("/Spaced%20Test/bar"),
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := FService{}
			if err := unwrapURL(tt.args.urlString, &in); (err != nil) != tt.wantErr {
				t.Errorf("unwrapURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.fService, &in) {
				t.Errorf("unwrapURL() got = %v, want = %v", &in, tt.args.fService)
			}
		})
	}
}
