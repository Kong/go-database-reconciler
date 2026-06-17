package utils

import (
	"context"
	"reflect"
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type kongDefaultForTesting struct {
	Service  *kong.Service
	Route    *kong.Route
	Upstream *kong.Upstream
	Target   *kong.Target
}

var kongDefaults = kongDefaultForTesting{
	Service:  &serviceDefaults,
	Route:    &routeDefaults,
	Upstream: &upstreamDefaults,
	Target:   &targetDefaults,
}

var defaulterTestOpts = DefaulterOpts{
	KongDefaults:           kongDefaults,
	DisableDynamicDefaults: false,
}

func TestDefaulter(t *testing.T) {
	assert := assert.New(t)

	var d Defaulter

	require.Error(t, d.Register(nil))
	require.Error(t, d.Set(nil))

	assert.Panics(func() {
		d.MustSet(d)
	})

	type Foo struct {
		A string
		B []string
	}
	defaultFoo := &Foo{
		A: "defaultA",
		B: []string{"default1"},
	}
	require.NoError(t, d.Register(defaultFoo))

	// sets a default
	var arg Foo
	require.NoError(t, d.Set(&arg))
	assert.Equal("defaultA", arg.A)
	assert.Equal([]string{"default1"}, arg.B)

	// doesn't set a default
	arg1 := Foo{
		A: "non-default-value",
	}
	require.NoError(t, d.Set(&arg1))
	assert.Equal("non-default-value", arg1.A)

	// errors on an unregistered type
	type Bar struct {
		A string
	}
	require.Error(t, d.Set(&Bar{}))
	assert.Panics(func() {
		d.MustSet(&Bar{})
	})
}

func TestServiceSetTest(t *testing.T) {
	ctx := context.Background()
	d, err := GetDefaulter(ctx, defaulterTestOpts)
	require.NotNil(t, d)
	require.NoError(t, err)

	testCases := []struct {
		desc string
		arg  *kong.Service
		want *kong.Service
	}{
		{
			desc: "empty service",
			arg:  &kong.Service{},
			want: &serviceDefaults,
		},
		{
			desc: "retries can be set to 0",
			arg: &kong.Service{
				Retries: new(0),
			},
			want: &kong.Service{
				Retries:        new(0),
				Protocol:       new("http"),
				ConnectTimeout: new(60000),
				WriteTimeout:   new(60000),
				ReadTimeout:    new(60000),
			},
		},
		{
			desc: "timeout value value is not overridden",
			arg: &kong.Service{
				WriteTimeout: new(42),
			},
			want: &kong.Service{
				Protocol:       new("http"),
				ConnectTimeout: new(60000),
				WriteTimeout:   new(42),
				ReadTimeout:    new(60000),
			},
		},
		{
			desc: "path value is not overridden",
			arg: &kong.Service{
				Path: new("/foo"),
			},
			want: &kong.Service{
				Protocol:       new("http"),
				Path:           new("/foo"),
				ConnectTimeout: new(60000),
				WriteTimeout:   new(60000),
				ReadTimeout:    new(60000),
			},
		},
		{
			desc: "Name is not reset",
			arg: &kong.Service{
				Name: new("foo"),
				Host: new("example.com"),
				Path: new("/bar"),
			},
			want: &kong.Service{
				Name:           new("foo"),
				Host:           new("example.com"),
				Protocol:       new("http"),
				Path:           new("/bar"),
				ConnectTimeout: new(60000),
				WriteTimeout:   new(60000),
				ReadTimeout:    new(60000),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			err := d.Set(tC.arg)
			require.NoError(t, err)
			assert.Equal(t, tC.want, tC.arg)
		})
	}
}

func TestRouteSetTest(t *testing.T) {
	ctx := context.Background()
	d, err := GetDefaulter(ctx, defaulterTestOpts)
	require.NotNil(t, d)
	require.NoError(t, err)

	testCases := []struct {
		desc string
		arg  *kong.Route
		want *kong.Route
	}{
		{
			desc: "empty route",
			arg:  &kong.Route{},
			want: &routeDefaults,
		},
		{
			desc: "preserve host is not overridden",
			arg: &kong.Route{
				PreserveHost: new(true),
			},
			want: &kong.Route{
				PreserveHost:  new(true),
				RegexPriority: new(0),
				StripPath:     new(true),
				Protocols:     kong.StringSlice("http", "https"),
			},
		},
		{
			desc: "Protocols is not reset",
			arg: &kong.Route{
				Protocols: kong.StringSlice("http", "tls"),
			},
			want: &kong.Route{
				PreserveHost:  new(false),
				RegexPriority: new(0),
				StripPath:     new(true),
				Protocols:     kong.StringSlice("http", "tls"),
			},
		},
		{
			desc: "non-default feilds is not reset",
			arg: &kong.Route{
				Name:      new("foo"),
				Hosts:     kong.StringSlice("1.example.com", "2.example.com"),
				Methods:   kong.StringSlice("GET", "POST"),
				StripPath: new(true),
			},
			want: &kong.Route{
				Name:          new("foo"),
				Hosts:         kong.StringSlice("1.example.com", "2.example.com"),
				Methods:       kong.StringSlice("GET", "POST"),
				PreserveHost:  new(false),
				RegexPriority: new(0),
				StripPath:     new(true),
				Protocols:     kong.StringSlice("http", "https"),
			},
		},
		{
			desc: "strip-path can be set to false",
			arg: &kong.Route{
				StripPath: new(false),
			},
			want: &kong.Route{
				PreserveHost:  new(false),
				RegexPriority: new(0),
				StripPath:     new(false),
				Protocols:     kong.StringSlice("http", "https"),
			},
		},
		{
			desc: "strip-path can be set to true",
			arg: &kong.Route{
				StripPath: new(true),
			},
			want: &kong.Route{
				PreserveHost:  new(false),
				RegexPriority: new(0),
				StripPath:     new(true),
				Protocols:     kong.StringSlice("http", "https"),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			err := d.Set(tC.arg)
			require.NoError(t, err)
			assert.Equal(t, tC.want, tC.arg)
		})
	}
}

func TestUpstreamSetTest(t *testing.T) {
	ctx := context.Background()
	d, err := GetDefaulter(ctx, defaulterTestOpts)
	require.NotNil(t, d)
	require.NoError(t, err)

	testCases := []struct {
		desc string
		arg  *kong.Upstream
		want *kong.Upstream
	}{
		{
			desc: "empty upstream",
			arg:  &kong.Upstream{},
			want: &upstreamDefaults,
		},
		{
			desc: "Healthchecks.Active.Healthy.HTTPStatuses is not overridden",
			arg: &kong.Upstream{
				Healthchecks: &kong.Healthcheck{
					Active: &kong.ActiveHealthcheck{
						Healthy: &kong.Healthy{
							HTTPStatuses: []int{200},
						},
					},
				},
			},
			want: &kong.Upstream{
				Slots: new(10000),
				Healthchecks: &kong.Healthcheck{
					Active: &kong.ActiveHealthcheck{
						Concurrency: new(10),
						Healthy: &kong.Healthy{
							HTTPStatuses: []int{200},
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
							HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
							Interval:     new(0),
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
		{
			desc: "Healthchecks.Active.Healthy.Timeout is not overridden",
			arg: &kong.Upstream{
				Name: new("foo"),
				Healthchecks: &kong.Healthcheck{
					Active: &kong.ActiveHealthcheck{
						Healthy: &kong.Healthy{
							Interval: new(1),
						},
					},
				},
			},
			want: &kong.Upstream{
				Name:  new("foo"),
				Slots: new(10000),
				Healthchecks: &kong.Healthcheck{
					Active: &kong.ActiveHealthcheck{
						Concurrency: new(10),
						Healthy: &kong.Healthy{
							HTTPStatuses: []int{200, 302},
							Interval:     new(1),
							Successes:    new(0),
						},
						HTTPPath: new("/"),
						Type:     new("http"),
						Timeout:  new(1),
						Unhealthy: &kong.Unhealthy{
							HTTPFailures: new(0),
							TCPFailures:  new(0),
							Timeouts:     new(0),
							HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
							Interval:     new(0),
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
		{
			desc: "Healthchecks.Active.HTTPSVerifyCertificate can be set to false",
			arg: &kong.Upstream{
				Name: new("foo"),
				Healthchecks: &kong.Healthcheck{
					Active: &kong.ActiveHealthcheck{
						Healthy: &kong.Healthy{
							Interval: new(1),
						},
						HTTPSVerifyCertificate: new(false),
					},
				},
			},
			want: &kong.Upstream{
				Name:  new("foo"),
				Slots: new(10000),
				Healthchecks: &kong.Healthcheck{
					Active: &kong.ActiveHealthcheck{
						Concurrency: new(10),
						Healthy: &kong.Healthy{
							HTTPStatuses: []int{200, 302},
							Interval:     new(1),
							Successes:    new(0),
						},
						HTTPPath:               new("/"),
						HTTPSVerifyCertificate: new(false),
						Type:                   new("http"),
						Timeout:                new(1),
						Unhealthy: &kong.Unhealthy{
							HTTPFailures: new(0),
							TCPFailures:  new(0),
							Timeouts:     new(0),
							HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
							Interval:     new(0),
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
		{
			desc: "Healthchecks.Active.HTTPSVerifyCertificate can be set to true",
			arg: &kong.Upstream{
				Name: new("foo"),
				Healthchecks: &kong.Healthcheck{
					Active: &kong.ActiveHealthcheck{
						Healthy: &kong.Healthy{
							Interval: new(1),
						},
						HTTPSVerifyCertificate: new(true),
					},
				},
			},
			want: &kong.Upstream{
				Name:  new("foo"),
				Slots: new(10000),
				Healthchecks: &kong.Healthcheck{
					Active: &kong.ActiveHealthcheck{
						Concurrency: new(10),
						Healthy: &kong.Healthy{
							HTTPStatuses: []int{200, 302},
							Interval:     new(1),
							Successes:    new(0),
						},
						HTTPPath:               new("/"),
						HTTPSVerifyCertificate: new(true),
						Type:                   new("http"),
						Timeout:                new(1),
						Unhealthy: &kong.Unhealthy{
							HTTPFailures: new(0),
							TCPFailures:  new(0),
							Timeouts:     new(0),
							HTTPStatuses: []int{429, 404, 500, 501, 502, 503, 504, 505},
							Interval:     new(0),
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
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			err := d.Set(tC.arg)
			require.NoError(t, err)
			assert.Equal(t, tC.want, tC.arg)
		})
	}
}

func TestGetDefaulter_Konnect(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		desc string
		opts DefaulterOpts
		want *Defaulter
	}{
		{
			desc: "empty user defaults",
			opts: DefaulterOpts{
				KongDefaults:           &kongDefaultForTesting{},
				DisableDynamicDefaults: true,
			},
			want: &Defaulter{
				service:  &serviceDefaults,
				route:    &routeDefaults,
				upstream: &upstreamDefaults,
				target:   &targetDefaults,
			},
		},
		{
			desc: "user defaults take precedence",
			opts: DefaulterOpts{
				KongDefaults: &kongDefaultForTesting{
					Service: &kong.Service{
						Port:           new(8080),
						Path:           new("/v1"),
						Protocol:       new("http"),
						ConnectTimeout: new(defaultTimeout),
						WriteTimeout:   new(defaultTimeout),
						ReadTimeout:    new(defaultTimeout),
					},
				},
				DisableDynamicDefaults: true,
			},
			want: &Defaulter{
				service: &kong.Service{
					Port:           new(8080),
					Path:           new("/v1"),
					Protocol:       new("http"),
					ConnectTimeout: new(defaultTimeout),
					WriteTimeout:   new(defaultTimeout),
					ReadTimeout:    new(defaultTimeout),
				},
				route:    &routeDefaults,
				upstream: &upstreamDefaults,
				target:   &targetDefaults,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			d, err := GetDefaulter(ctx, tc.opts)
			assert.NotNil(d)
			require.NoError(t, err)

			if !reflect.DeepEqual(d.service, tc.want.service) {
				assert.Equal(t, tc.want.service, d.service)
			}
			if !reflect.DeepEqual(d.route, tc.want.route) {
				assert.Equal(t, tc.want.route, d.route)
			}
			if !reflect.DeepEqual(d.upstream, tc.want.upstream) {
				assert.Equal(t, tc.want.upstream, d.upstream)
			}
			if !reflect.DeepEqual(d.target, tc.want.target) {
				assert.Equal(t, tc.want.target, d.target)
			}
		})
	}
}

func TestCheckRestrictedFields(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		desc             string
		entity           *kong.Service
		restrictedFields []string
		wantErr          bool
		expectedErr      string
	}{
		{
			desc: "no restricted fields",
			entity: &kong.Service{
				ID:   new("testID"),
				Name: new("testName"),
			},
			restrictedFields: []string{},
		},
		{
			desc: "one restricted fields",
			entity: &kong.Service{
				ID:   new("testID"),
				Name: new("testName"),
			},
			restrictedFields: []string{"ID"},
			wantErr:          true,
			expectedErr:      "cannot have these restricted fields set: id",
		},
		{
			desc: "multiple restricted fields",
			entity: &kong.Service{
				ID:   new("testID"),
				Name: new("testName"),
				Port: new(80),
			},
			restrictedFields: []string{"ID", "Name", "Port"},
			wantErr:          true,
			expectedErr:      "cannot have these restricted fields set: id, name, port",
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			err := checkEntityDefaults(tC.entity, tC.restrictedFields)
			if (err != nil) != tC.wantErr {
				t.Errorf("got error = %v, expected error = %v", err, tC.wantErr)
			}
			if tC.expectedErr != "" {
				assert.Equal(tC.expectedErr, err.Error())
			}
		})
	}
}

func TestKongDefaultsRestrictedFields(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	testCases := []struct {
		desc         string
		kongDefaults *kongDefaultForTesting
		wantErr      bool
		expectedErr  string
	}{
		{
			desc: "service no restricted fields",
			kongDefaults: &kongDefaultForTesting{
				Service: &kong.Service{
					Path: new("/v1"),
				},
			},
		},
		{
			desc: "route no restricted fields",
			kongDefaults: &kongDefaultForTesting{
				Route: &kong.Route{
					StripPath: new(false),
				},
			},
		},
		{
			desc: "target no restricted fields",
			kongDefaults: &kongDefaultForTesting{
				Target: &kong.Target{
					Weight: new(42),
				},
			},
		},
		{
			desc: "upstream no restricted fields",
			kongDefaults: &kongDefaultForTesting{
				Upstream: &kong.Upstream{
					HostHeader: new("testHostHeader"),
				},
			},
		},
		{
			desc: "service restricted fields",
			kongDefaults: &kongDefaultForTesting{
				Service: &kong.Service{
					ID:   new("testID"),
					Name: new("testName"),
					Path: new("/v1"),
				},
			},
			wantErr:     true,
			expectedErr: "service defaults cannot have these restricted fields set: id, name",
		},
		{
			desc: "route restricted fields",
			kongDefaults: &kongDefaultForTesting{
				Route: &kong.Route{
					ID:        new("testID"),
					Name:      new("testName"),
					StripPath: new(false),
				},
			},
			wantErr:     true,
			expectedErr: "route defaults cannot have these restricted fields set: id, name",
		},
		{
			desc: "target restricted fields",
			kongDefaults: &kongDefaultForTesting{
				Target: &kong.Target{
					ID:     new("testID"),
					Target: new("testTarget"),
				},
			},
			wantErr:     true,
			expectedErr: "target defaults cannot have these restricted fields set: id, target",
		},
		{
			desc: "upstream restricted fields",
			kongDefaults: &kongDefaultForTesting{
				Upstream: &kong.Upstream{
					ID:         new("testID"),
					Name:       new("testName"),
					HostHeader: new("testHostHeader"),
				},
			},
			wantErr:     true,
			expectedErr: "upstream defaults cannot have these restricted fields set: id, name",
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			opts := DefaulterOpts{
				KongDefaults: tC.kongDefaults,
			}
			_, err := GetDefaulter(ctx, opts)
			if (err != nil) != tC.wantErr {
				t.Errorf("got error = %v, expected error = %v", err, tC.wantErr)
			}
			if tC.expectedErr != "" {
				assert.Contains(err.Error(), tC.expectedErr)
			}
		})
	}
}
