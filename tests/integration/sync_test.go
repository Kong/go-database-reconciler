//go:build integration

package integration

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	deckDiff "github.com/kong/go-database-reconciler/pkg/diff"
	deckDump "github.com/kong/go-database-reconciler/pkg/dump"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// missing Enable
	svc1 = []*kong.Service{
		{
			ID:             kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			Name:           kong.String("svc1"),
			ConnectTimeout: kong.Int(60000),
			Host:           kong.String("mockbin.org"),
			Port:           kong.Int(80),
			Protocol:       kong.String("http"),
			ReadTimeout:    kong.Int(60000),
			Retries:        kong.Int(5),
			WriteTimeout:   kong.Int(60000),
			Tags:           nil,
		},
	}

	// latest
	svc1_207 = []*kong.Service{
		{
			ID:             kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			Name:           kong.String("svc1"),
			ConnectTimeout: kong.Int(60000),
			Host:           kong.String("mockbin.org"),
			Port:           kong.Int(80),
			Protocol:       kong.String("http"),
			ReadTimeout:    kong.Int(60000),
			Retries:        kong.Int(5),
			WriteTimeout:   kong.Int(60000),
			Enabled:        kong.Bool(true),
			Tags:           nil,
		},
	}

	defaultCPService = []*kong.Service{
		{
			ID:             kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			Name:           kong.String("default"),
			ConnectTimeout: kong.Int(60000),
			Host:           kong.String("mockbin-default.org"),
			Port:           kong.Int(80),
			Protocol:       kong.String("http"),
			ReadTimeout:    kong.Int(60000),
			Retries:        kong.Int(5),
			WriteTimeout:   kong.Int(60000),
			Enabled:        kong.Bool(true),
			Tags:           nil,
		},
	}

	testCPService = []*kong.Service{
		{
			ID:             kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			Name:           kong.String("test"),
			ConnectTimeout: kong.Int(60000),
			Host:           kong.String("mockbin-test.org"),
			Port:           kong.Int(80),
			Protocol:       kong.String("http"),
			ReadTimeout:    kong.Int(60000),
			Retries:        kong.Int(5),
			WriteTimeout:   kong.Int(60000),
			Enabled:        kong.Bool(true),
			Tags:           nil,
		},
	}

	// missing RequestBuffering, ResponseBuffering, Service, PathHandling
	route1_143 = []*kong.Route{
		{
			ID:                      kong.String("87b6a97e-f3f7-4c47-857a-7464cb9e202b"),
			Name:                    kong.String("r1"),
			Paths:                   []*string{kong.String("/r1")},
			PreserveHost:            kong.Bool(false),
			Protocols:               []*string{kong.String("http"), kong.String("https")},
			RegexPriority:           kong.Int(0),
			StripPath:               kong.Bool(true),
			HTTPSRedirectStatusCode: kong.Int(301),
		},
	}

	// missing RequestBuffering, ResponseBuffering
	// PathHandling set to v1
	route1_151 = []*kong.Route{
		{
			ID:                      kong.String("87b6a97e-f3f7-4c47-857a-7464cb9e202b"),
			Name:                    kong.String("r1"),
			Paths:                   []*string{kong.String("/r1")},
			PathHandling:            kong.String("v1"),
			PreserveHost:            kong.Bool(false),
			Protocols:               []*string{kong.String("http"), kong.String("https")},
			RegexPriority:           kong.Int(0),
			StripPath:               kong.Bool(true),
			HTTPSRedirectStatusCode: kong.Int(301),
			Service: &kong.Service{
				ID: kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			},
		},
	}

	// missing RequestBuffering, ResponseBuffering
	route1_205_214 = []*kong.Route{
		{
			ID:                      kong.String("87b6a97e-f3f7-4c47-857a-7464cb9e202b"),
			Name:                    kong.String("r1"),
			Paths:                   []*string{kong.String("/r1")},
			PathHandling:            kong.String("v0"),
			PreserveHost:            kong.Bool(false),
			Protocols:               []*string{kong.String("http"), kong.String("https")},
			RegexPriority:           kong.Int(0),
			StripPath:               kong.Bool(true),
			HTTPSRedirectStatusCode: kong.Int(301),
			Service: &kong.Service{
				ID: kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			},
		},
	}

	// latest
	route1_20x = []*kong.Route{
		{
			ID:                      kong.String("87b6a97e-f3f7-4c47-857a-7464cb9e202b"),
			Name:                    kong.String("r1"),
			Paths:                   []*string{kong.String("/r1")},
			PathHandling:            kong.String("v0"),
			PreserveHost:            kong.Bool(false),
			Protocols:               []*string{kong.String("http"), kong.String("https")},
			RegexPriority:           kong.Int(0),
			StripPath:               kong.Bool(true),
			HTTPSRedirectStatusCode: kong.Int(301),
			RequestBuffering:        kong.Bool(true),
			ResponseBuffering:       kong.Bool(true),
			Service: &kong.Service{
				ID: kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			},
		},
	}

	// has run-on set to 'first'
	plugin_143_151 = []*kong.Plugin{ //nolint:revive,stylecheck
		{
			Name: kong.String("basic-auth"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"anonymous":        "58076db2-28b6-423b-ba39-a797193017f7",
				"hide_credentials": false,
			},
			RunOn: kong.String("first"),
		},
	}

	// latest
	plugin = []*kong.Plugin{
		{
			Name: kong.String("basic-auth"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"anonymous":        "58076db2-28b6-423b-ba39-a797193017f7",
				"hide_credentials": false,
			},
		},
	}

	plugin36 = []*kong.Plugin{
		{
			Name: kong.String("basic-auth"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"anonymous":        "58076db2-28b6-423b-ba39-a797193017f7",
				"hide_credentials": false,
				"realm":            string("service"),
			},
		},
	}

	plugin_on_entities = []*kong.Plugin{ //nolint:revive,stylecheck
		{
			Name: kong.String("prometheus"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"per_consumer": false,
			},
			Service: &kong.Service{
				ID: kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			},
		},
		{
			Name: kong.String("prometheus"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"per_consumer": false,
			},
			Route: &kong.Route{
				ID: kong.String("87b6a97e-f3f7-4c47-857a-7464cb9e202b"),
			},
		},
		{
			Name: kong.String("prometheus"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"per_consumer": false,
			},
			Consumer: &kong.Consumer{
				ID: kong.String("d2965b9b-0608-4458-a9f8-0b93d88d03b8"),
			},
		},
	}

	plugin_on_entities3x = []*kong.Plugin{ //nolint:revive,stylecheck
		{
			Name: kong.String("prometheus"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"bandwidth_metrics":       false,
				"latency_metrics":         false,
				"per_consumer":            false,
				"status_code_metrics":     false,
				"upstream_health_metrics": false,
			},
			Service: &kong.Service{
				ID: kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
			},
		},
		{
			Name: kong.String("prometheus"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"bandwidth_metrics":       false,
				"latency_metrics":         false,
				"per_consumer":            false,
				"status_code_metrics":     false,
				"upstream_health_metrics": false,
			},
			Route: &kong.Route{
				ID: kong.String("87b6a97e-f3f7-4c47-857a-7464cb9e202b"),
			},
		},
		{
			Name: kong.String("prometheus"),
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Enabled: kong.Bool(true),
			Config: kong.Configuration{
				"bandwidth_metrics":       false,
				"latency_metrics":         false,
				"per_consumer":            false,
				"status_code_metrics":     false,
				"upstream_health_metrics": false,
			},
			Consumer: &kong.Consumer{
				ID: kong.String("d2965b9b-0608-4458-a9f8-0b93d88d03b8"),
			},
		},
	}

	upstream_pre31 = []*kong.Upstream{ //nolint:revive,stylecheck
		{
			Name:      kong.String("upstream1"),
			Algorithm: kong.String("round-robin"),
			Slots:     kong.Int(10000),
			Healthchecks: &kong.Healthcheck{
				Threshold: kong.Float64(0),
				Active: &kong.ActiveHealthcheck{
					Concurrency: kong.Int(10),
					Healthy: &kong.Healthy{
						HTTPStatuses: []int{200, 302},
						Interval:     kong.Int(0),
						Successes:    kong.Int(0),
					},
					HTTPPath:               kong.String("/"),
					Type:                   kong.String("http"),
					Timeout:                kong.Int(1),
					HTTPSVerifyCertificate: kong.Bool(true),
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
					Type: kong.String("http"),
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
	}

	// latest
	upstream = []*kong.Upstream{
		{
			Name:      kong.String("upstream1"),
			Algorithm: kong.String("round-robin"),
			Slots:     kong.Int(10000),
			Healthchecks: &kong.Healthcheck{
				Threshold: kong.Float64(0),
				Active: &kong.ActiveHealthcheck{
					Concurrency: kong.Int(10),
					Healthy: &kong.Healthy{
						HTTPStatuses: []int{200, 302},
						Interval:     kong.Int(0),
						Successes:    kong.Int(0),
					},
					HTTPPath:               kong.String("/"),
					Type:                   kong.String("http"),
					Timeout:                kong.Int(1),
					HTTPSVerifyCertificate: kong.Bool(true),
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
					Type: kong.String("http"),
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
			UseSrvName:       kong.Bool(false),
		},
	}

	target = []*kong.Target{
		{
			Target: kong.String("198.51.100.11:80"),
			Upstream: &kong.Upstream{
				ID: kong.String("a6f89ffc-1e53-4b01-9d3d-7a142bcd"),
			},
			Weight: kong.Int(100),
		},
	}

	targetZeroWeight = []*kong.Target{
		{
			Target: kong.String("198.51.100.11:80"),
			Upstream: &kong.Upstream{
				ID: kong.String("a6f89ffc-1e53-4b01-9d3d-7a142bcd"),
			},
			Weight: kong.Int(0),
		},
	}

	rateLimitingPlugin = []*kong.Plugin{
		{
			Name: kong.String("rate-limiting"),
			Config: kong.Configuration{
				"day":                 nil,
				"fault_tolerant":      true,
				"header_name":         nil,
				"hide_client_headers": false,
				"hour":                nil,
				"limit_by":            "consumer",
				"minute":              float64(123),
				"month":               nil,
				"path":                nil,
				"policy":              "cluster",
				"redis_database":      float64(0),
				"redis_host":          nil,
				"redis_password":      nil,
				"redis_port":          float64(6379),
				"redis_server_name":   nil,
				"redis_ssl":           false,
				"redis_ssl_verify":    false,
				"redis_timeout":       float64(2000),
				"second":              nil,
				"year":                nil,
			},
			Enabled: kong.Bool(true),
			RunOn:   nil,
			Protocols: []*string{
				kong.String("grpc"),
				kong.String("grpcs"),
				kong.String("http"),
				kong.String("https"),
			},
			Tags: nil,
		},
	}

	consumer = []*kong.Consumer{
		{
			Username: kong.String("yolo"),
			ID:       kong.String("d2965b9b-0608-4458-a9f8-0b93d88d03b8"),
		},
	}

	consumerGroupsConsumers = []*kong.Consumer{
		{
			Username: kong.String("foo"),
		},
		{
			Username: kong.String("bar"),
		},
		{
			Username: kong.String("baz"),
		},
	}

	consumerGroups = []*kong.ConsumerGroupObject{
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("silver"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("bar"),
				},
				{
					Username: kong.String("baz"),
				},
			},
		},
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("gold"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("foo"),
				},
			},
		},
	}

	consumerGroupsWithTags = []*kong.ConsumerGroupObject{
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("silver"),
				Tags: kong.StringSlice("tag1", "tag3"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("bar"),
				},
				{
					Username: kong.String("baz"),
				},
			},
		},
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("gold"),
				Tags: kong.StringSlice("tag1", "tag2"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("foo"),
				},
			},
		},
	}

	consumerGroupsWithRLA = []*kong.ConsumerGroupObject{
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("silver"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("bar"),
				},
			},
			Plugins: []*kong.ConsumerGroupPlugin{
				{
					Name: kong.String("rate-limiting-advanced"),
					Config: kong.Configuration{
						"limit":                  []any{float64(7)},
						"retry_after_jitter_max": float64(1),
						"window_size":            []any{float64(60)},
						"window_type":            "sliding",
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: kong.String("521a90ad-36cb-4e31-a5db-1d979aee40d1"),
					},
				},
			},
		},
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("gold"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("foo"),
				},
			},
			Plugins: []*kong.ConsumerGroupPlugin{
				{
					Name: kong.String("rate-limiting-advanced"),
					Config: kong.Configuration{
						"limit":                  []any{float64(10)},
						"retry_after_jitter_max": float64(1),
						"window_size":            []any{float64(60)},
						"window_type":            "sliding",
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: kong.String("92177268-b134-42f9-909a-36f9d2d3d5e7"),
					},
				},
			},
		},
	}

	consumerGroupsWithTagsAndRLA = []*kong.ConsumerGroupObject{
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("silver"),
				Tags: kong.StringSlice("tag1", "tag3"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("bar"),
				},
			},
			Plugins: []*kong.ConsumerGroupPlugin{
				{
					Name: kong.String("rate-limiting-advanced"),
					Config: kong.Configuration{
						"limit":                  []any{float64(7)},
						"retry_after_jitter_max": float64(1),
						"window_size":            []any{float64(60)},
						"window_type":            "sliding",
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: kong.String("521a90ad-36cb-4e31-a5db-1d979aee40d1"),
					},
				},
			},
		},
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("gold"),
				Tags: kong.StringSlice("tag1", "tag2"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("foo"),
				},
			},
			Plugins: []*kong.ConsumerGroupPlugin{
				{
					Name: kong.String("rate-limiting-advanced"),
					Config: kong.Configuration{
						"limit":                  []any{float64(10)},
						"retry_after_jitter_max": float64(1),
						"window_size":            []any{float64(60)},
						"window_type":            "sliding",
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: kong.String("92177268-b134-42f9-909a-36f9d2d3d5e7"),
					},
				},
			},
		},
	}

	consumerGroupsWithRLAApp = []*kong.ConsumerGroupObject{
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("silver"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("bar"),
				},
			},
			Plugins: []*kong.ConsumerGroupPlugin{
				{
					Name: kong.String("rate-limiting-advanced"),
					Config: kong.Configuration{
						"limit":                  []any{float64(7)},
						"retry_after_jitter_max": float64(1),
						"window_size":            []any{float64(60)},
						"window_type":            string("sliding"),
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: kong.String("f79972fe-e9a0-40b5-8dc6-f1bf3758b86b"),
					},
				},
			},
		},
		{
			ConsumerGroup: &kong.ConsumerGroup{
				Name: kong.String("gold"),
			},
			Consumers: []*kong.Consumer{
				{
					Username: kong.String("foo"),
				},
			},
			Plugins: []*kong.ConsumerGroupPlugin{
				{
					Name: kong.String("rate-limiting-advanced"),
					Config: kong.Configuration{
						"limit":                  []any{float64(10)},
						"retry_after_jitter_max": float64(1),
						"window_size":            []any{float64(60)},
						"window_type":            string("sliding"),
					},
					ConsumerGroup: &kong.ConsumerGroup{
						ID: kong.String("8eea863e-460c-4019-895a-1e80cb08699d"),
					},
				},
			},
		},
	}

	consumerGroupAppPlugins = []*kong.Plugin{
		{
			Name: kong.String("rate-limiting-advanced"),
			Config: kong.Configuration{
				"consumer_groups":         []any{string("silver"), string("gold")},
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"enforce_consumer_groups": bool(true),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(5)},
				"namespace":               string("dNRC6xKsRL8Koc1uVYA4Nki6DLW7XIdx"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(30),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(0),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("key-auth"),
			Config: kong.Configuration{
				"anonymous":        nil,
				"hide_credentials": false,
				"key_in_body":      false,
				"key_in_header":    true,
				"key_in_query":     true,
				"key_names":        []interface{}{"apikey"},
				"run_on_preflight": true,
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("http"), kong.String("https")},
		},
	}

	consumerGroupScopedPlugins = []*kong.Plugin{
		{
			Name: kong.String("rate-limiting-advanced"),
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("77e6691d-67c0-446a-9401-27be2b141aae"),
			},
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(10)},
				"namespace":               string("gold"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(30),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("rate-limiting-advanced"),
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("5bcbd3a7-030b-4310-bd1d-2721ff85d236"),
			},
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(7)},
				"namespace":               string("silver"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(30),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("rate-limiting-advanced"),
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(5)},
				"namespace":               string("silver"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(30),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("key-auth"),
			Config: kong.Configuration{
				"anonymous":        nil,
				"hide_credentials": false,
				"key_in_body":      false,
				"key_in_header":    true,
				"key_in_query":     true,
				"key_names":        []interface{}{"apikey"},
				"run_on_preflight": true,
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("http"), kong.String("https")},
		},
	}

	consumerGroupScopedPlugins35x = []*kong.Plugin{
		{
			Name: kong.String("rate-limiting-advanced"),
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("77e6691d-67c0-446a-9401-27be2b141aae"),
			},
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(10)},
				"namespace":               string("gold"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(256),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("rate-limiting-advanced"),
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("5bcbd3a7-030b-4310-bd1d-2721ff85d236"),
			},
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(7)},
				"namespace":               string("silver"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(256),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("rate-limiting-advanced"),
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(5)},
				"namespace":               string("silver"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(256),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("key-auth"),
			Config: kong.Configuration{
				"anonymous":        nil,
				"hide_credentials": false,
				"key_in_body":      false,
				"key_in_header":    true,
				"key_in_query":     true,
				"key_names":        []interface{}{"apikey"},
				"run_on_preflight": true,
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("http"), kong.String("https")},
		},
	}

	consumerGroupScopedPlugins37x = []*kong.Plugin{
		{
			Name: kong.String("rate-limiting-advanced"),
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("77e6691d-67c0-446a-9401-27be2b141aae"),
			},
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(10)},
				"namespace":               string("gold"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(256),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("rate-limiting-advanced"),
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("5bcbd3a7-030b-4310-bd1d-2721ff85d236"),
			},
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(7)},
				"namespace":               string("silver"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(256),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("rate-limiting-advanced"),
			Config: kong.Configuration{
				"consumer_groups":         nil,
				"dictionary_name":         string("kong_rate_limiting_counters"),
				"disable_penalty":         bool(false),
				"enforce_consumer_groups": bool(false),
				"error_code":              float64(429),
				"error_message":           string("API rate limit exceeded"),
				"header_name":             nil,
				"hide_client_headers":     bool(false),
				"identifier":              string("consumer"),
				"limit":                   []any{float64(5)},
				"namespace":               string("silver"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":   nil,
					"connect_timeout":     nil,
					"database":            float64(0),
					"host":                nil,
					"keepalive_backlog":   nil,
					"keepalive_pool_size": float64(256),
					"password":            nil,
					"port":                nil,
					"read_timeout":        nil,
					"send_timeout":        nil,
					"sentinel_addresses":  nil,
					"sentinel_master":     nil,
					"sentinel_password":   nil,
					"sentinel_role":       nil,
					"sentinel_username":   nil,
					"server_name":         nil,
					"ssl":                 false,
					"ssl_verify":          false,
					"timeout":             float64(2000),
					"username":            nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		},
		{
			Name: kong.String("key-auth"),
			Config: kong.Configuration{
				"anonymous":        nil,
				"hide_credentials": false,
				"key_in_body":      false,
				"key_in_header":    true,
				"key_in_query":     true,
				"key_names":        []interface{}{"apikey"},
				"realm":            nil,
				"run_on_preflight": true,
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("http"), kong.String("https")},
		},
	}

	consumerGroupScopedPluginWithTags = []*kong.Plugin{{
		Name: kong.String("request-transformer"),
		Config: kong.Configuration{
			"add":         map[string]any{"body": []any{}, "headers": []any{}, "querystring": []any{}},
			"append":      map[string]any{"body": []any{}, "headers": []any{}, "querystring": []any{}},
			"http_method": string("GET"),
			"remove":      map[string]any{"body": []any{}, "headers": []any{string("test-header")}, "querystring": []any{}},
			"rename":      map[string]any{"body": []any{}, "headers": []any{}, "querystring": []any{}},
			"replace":     map[string]any{"body": []any{}, "headers": []any{}, "querystring": []any{}, "uri": nil},
		},
		ConsumerGroup: &kong.ConsumerGroup{
			ID: kong.String("58076db2-28b6-423b-ba39-a79719301700"),
		},
		Tags:      kong.StringSlice("tag1", "tag2"),
		Enabled:   kong.Bool(true),
		Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
	}}
)

// test scope:
//   - 1.4.3
func Test_Sync_ServicesRoutes_Till_1_4_3(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// ignore entities fields based on Kong version
	ignoreFields := []cmp.Option{
		cmpopts.IgnoreFields(kong.Route{}, "Service"),
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/001-create-a-service/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1,
			},
		},
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/002-create-services-and-routes/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1,
				Routes:   route1_143,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", "<=1.4.3")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, ignoreFields)
		})
	}
}

// test scope:
//   - 1.5.1
//   - 1.5.0.11+enterprise
func Test_Sync_ServicesRoutes_Till_1_5_1(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/001-create-a-service/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1,
			},
		},
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/002-create-services-and-routes/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1,
				Routes:   route1_151,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">1.4.3 <=1.5.1")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.0.5
//   - 2.1.4
func Test_Sync_ServicesRoutes_From_2_0_5_To_2_1_4(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/001-create-a-service/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1,
			},
		},
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/002-create-services-and-routes/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1,
				Routes:   route1_205_214,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.0.5 <=2.1.4")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.2.2
//   - 2.3.3
//   - 2.4.1
//   - 2.5.1
//   - 2.6.0
//   - 2.2.1.3+enterprise
//   - 2.3.3.4+enterprise
//   - 2.4.1.3+enterprise
//   - 2.5.1.2+enterprise
func Test_Sync_ServicesRoutes_From_2_2_1_to_2_6_0(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/001-create-a-service/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1,
			},
		},
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/002-create-services-and-routes/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1,
				Routes:   route1_20x,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">2.2.1 <=2.6.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.7.0
//   - 2.6.0.2+enterprise
//   - 2.7.0.0+enterprise
//   - 2.8.0.0+enterprise
func Test_Sync_ServicesRoutes_From_2_6_9_Till_2_8_0(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/001-create-a-service/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
			},
		},
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/002-create-services-and-routes/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
				Routes:   route1_20x,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">2.6.9 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.x
func Test_Sync_ServicesRoutes_From_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/001-create-a-service/kong3x.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
			},
		},
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/002-create-services-and-routes/kong3x.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
				Routes:   route1_20x,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - konnect
func Test_Sync_ServicesRoutes_Konnect(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/001-create-a-service/kong3x.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
			},
		},
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/002-create-services-and-routes/kong3x.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
				Routes:   route1_20x,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "konnect", "")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 1.4.3
func Test_Sync_BasicAuth_Plugin_1_4_3(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin",
			kongFile: "testdata/sync/003-create-a-plugin/kong.yaml",
			expectedState: utils.KongRawState{
				Plugins: plugin_143_151,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", "==1.4.3")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 1.5.0.11+enterprise
func Test_Sync_BasicAuth_Plugin_Earlier_Than_1_5_1(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin",
			kongFile: "testdata/sync/003-create-a-plugin/kong.yaml",
			expectedState: utils.KongRawState{
				Plugins: plugin,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", "<1.5.1 !1.4.3")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 1.5.1
func Test_Sync_BasicAuth_Plugin_1_5_1(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin",
			kongFile: "testdata/sync/003-create-a-plugin/kong.yaml",
			expectedState: utils.KongRawState{
				Plugins: plugin_143_151,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", "==1.5.1")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.0.5
//   - 2.1.4
//   - 2.2.2
//   - 2.3.3
//   - 2.4.1
//   - 2.5.1
//   - 2.6.0
//   - 2.7.0
//   - 2.1.4.6+enterprise
//   - 2.2.1.3+enterprise
//   - 2.3.3.4+enterprise
//   - 2.4.1.3+enterprise
//   - 2.5.1.2+enterprise
//   - 2.6.0.2+enterprise
//   - 2.7.0.0+enterprise
//   - 2.8.0.0+enterprise
func Test_Sync_BasicAuth_Plugin_From_2_0_5_Till_2_8_0(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin",
			kongFile: "testdata/sync/003-create-a-plugin/kong.yaml",
			expectedState: utils.KongRawState{
				Plugins: plugin,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.0.5 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - >=3.0 <3.6.0
func Test_Sync_BasicAuth_Plugin_From_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin",
			kongFile: "testdata/sync/003-create-a-plugin/kong3x.yaml",
			expectedState: utils.KongRawState{
				Plugins: plugin,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.0.0 <3.6.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.6+
func Test_Sync_BasicAuth_Plugin_From_36(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin",
			kongFile: "testdata/sync/003-create-a-plugin/kong3x.yaml",
			expectedState: utils.KongRawState{
				Plugins: plugin36,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.6.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - konnect
func Test_Sync_BasicAuth_Plugin_Konnect(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin",
			kongFile: "testdata/sync/003-create-a-plugin/kong3x.yaml",
			expectedState: utils.KongRawState{
				Plugins: plugin,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "konnect", "")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 1.4.3
//   - 1.5.1
//   - 1.5.0.11+enterprise
func Test_Sync_Upstream_Target_Till_1_5_2(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// ignore entities fields based on Kong version
	ignoreFields := []cmp.Option{
		cmpopts.IgnoreFields(kong.Healthcheck{}, "Threshold"),
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target",
			kongFile: "testdata/sync/004-create-upstream-and-target/kong.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream_pre31,
				Targets:   target,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", "<=1.5.2")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, ignoreFields)
		})
	}
}

// test scope:
//   - 2.0.5
//   - 2.1.4
//   - 2.2.2
//   - 2.3.3
//   - 2.4.1
//   - 2.5.1
//   - 2.6.0
//   - 2.7.0
//   - 2.1.4.6+enterprise
//   - 2.2.1.3+enterprise
//   - 2.3.3.4+enterprise
//   - 2.4.1.3+enterprise
//   - 2.5.1.2+enterprise
//   - 2.6.0.2+enterprise
//   - 2.7.0.0+enterprise
//   - 2.8.0.0+enterprise
func Test_Sync_Upstream_Target_From_2x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target",
			kongFile: "testdata/sync/004-create-upstream-and-target/kong.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream_pre31,
				Targets:   target,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.1.0 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.0
func Test_Sync_Upstream_Target_From_30(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target",
			kongFile: "testdata/sync/004-create-upstream-and-target/kong3x.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream_pre31,
				Targets:   target,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=3.0.0 <3.1.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.x
func Test_Sync_Upstream_Target_From_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target",
			kongFile: "testdata/sync/004-create-upstream-and-target/kong3x.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream,
				Targets:   target,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.1.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - konnect
func Test_Sync_Upstream_Target_Konnect(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target",
			kongFile: "testdata/sync/004-create-upstream-and-target/kong3x.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream,
				Targets:   target,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "konnect", "")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.4.1
//   - 2.5.1
//   - 2.6.0
//   - 2.7.0
//   - 2.4.1.3+enterprise
//   - 2.5.1.2+enterprise
//   - 2.6.0.2+enterprise
//   - 2.7.0.0+enterprise
//   - 2.8.0.0+enterprise
func Test_Sync_Upstreams_Target_ZeroWeight_2x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target with weight equals to zero",
			kongFile: "testdata/sync/005-create-upstream-and-target-weight/kong.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream_pre31,
				Targets:   targetZeroWeight,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.4.1 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.0
func Test_Sync_Upstreams_Target_ZeroWeight_30(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target with weight equals to zero",
			kongFile: "testdata/sync/005-create-upstream-and-target-weight/kong3x.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream_pre31,
				Targets:   targetZeroWeight,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=3.0.0 <3.1.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.x
func Test_Sync_Upstreams_Target_ZeroWeight_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target with weight equals to zero",
			kongFile: "testdata/sync/005-create-upstream-and-target-weight/kong3x.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream,
				Targets:   targetZeroWeight,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.1.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - konnect
func Test_Sync_Upstreams_Target_ZeroWeight_Konnect(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates an upstream and target with weight equals to zero",
			kongFile: "testdata/sync/005-create-upstream-and-target-weight/kong3x.yaml",
			expectedState: utils.KongRawState{
				Upstreams: upstream,
				Targets:   targetZeroWeight,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "konnect", "")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_RateLimitingPlugin(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "fill defaults",
			kongFile: "testdata/sync/006-fill-defaults-rate-limiting/kong.yaml",
			expectedState: utils.KongRawState{
				Plugins: rateLimitingPlugin,
			},
		},
		{
			name:     "fill defaults with dedup",
			kongFile: "testdata/sync/007-fill-defaults-rate-limiting-dedup/kong.yaml",
			expectedState: utils.KongRawState{
				Plugins: rateLimitingPlugin,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", "==2.7.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 1.5.0.11+enterprise
func Test_Sync_FillDefaults_Earlier_Than_1_5_1(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// ignore entities fields based on Kong version
	ignoreFields := []cmp.Option{
		cmpopts.IgnoreFields(kong.Route{}, "Service"),
		cmpopts.IgnoreFields(kong.Healthcheck{}, "Threshold"),
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/008-create-simple-entities/kong.yaml",
			expectedState: utils.KongRawState{
				Services:  svc1,
				Routes:    route1_151,
				Plugins:   plugin,
				Targets:   target,
				Upstreams: upstream_pre31,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", "<1.5.1 !1.4.3")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, ignoreFields)
		})
	}
}

// test scope:
//   - 2.0.5
//   - 2.1.4
func Test_Sync_FillDefaults_From_2_0_5_To_2_1_4(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/008-create-simple-entities/kong.yaml",
			expectedState: utils.KongRawState{
				Services:  svc1,
				Routes:    route1_205_214,
				Upstreams: upstream_pre31,
				Targets:   target,
				Plugins:   plugin,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.0.5 <=2.1.4")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.2.2
//   - 2.3.3
//   - 2.4.1
//   - 2.5.1
//   - 2.6.0
//   - 2.2.1.3+enterprise
//   - 2.3.3.4+enterprise
//   - 2.4.1.3+enterprise
//   - 2.5.1.2+enterprise
func Test_Sync_FillDefaults_From_2_2_1_to_2_6_0(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "create services and routes",
			kongFile: "testdata/sync/008-create-simple-entities/kong.yaml",
			expectedState: utils.KongRawState{
				Services:  svc1,
				Routes:    route1_20x,
				Upstreams: upstream_pre31,
				Targets:   target,
				Plugins:   plugin,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">2.2.1 <=2.6.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.7.0
//   - 2.6.0.2+enterprise
//   - 2.7.0.0+enterprise
//   - 2.8.0.0+enterprise
func Test_Sync_FillDefaults_From_2_6_9(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates entities with minimum configuration",
			kongFile: "testdata/sync/008-create-simple-entities/kong.yaml",
			expectedState: utils.KongRawState{
				Services:  svc1_207,
				Routes:    route1_20x,
				Plugins:   plugin,
				Targets:   target,
				Upstreams: upstream_pre31,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">2.6.9 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_SkipCACert_2x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "syncing with --skip-ca-certificates should ignore CA certs",
			kongFile: "testdata/sync/009-skip-ca-cert/kong.yaml",
			expectedState: utils.KongRawState{
				Services:       svc1_207,
				CACertificates: []*kong.CACertificate{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ca_certificates first appeared in 1.3, but we limit to 2.7+
			// here because the schema changed and the entities aren't the same
			// across all versions, even though the skip functionality works the same.
			runWhen(t, "kong", ">=2.7.0 <3.0.0")
			setup(t)

			sync(tc.kongFile, "--skip-ca-certificates")
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_SkipCACert_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "syncing with --skip-ca-certificates should ignore CA certs",
			kongFile: "testdata/sync/009-skip-ca-cert/kong3x.yaml",
			expectedState: utils.KongRawState{
				Services:       svc1_207,
				CACertificates: []*kong.CACertificate{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ca_certificates first appeared in 1.3, but we limit to 2.7+
			// here because the schema changed and the entities aren't the same
			// across all versions, even though the skip functionality works the same.
			runWhenKongOrKonnect(t, ">=3.0.0")
			setup(t)

			sync(tc.kongFile, "--skip-ca-certificates")
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_RBAC_2x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "rbac",
			kongFile: "testdata/sync/xxx-rbac-endpoint-permissions/kong.yaml",
			expectedState: utils.KongRawState{
				RBACRoles: []*kong.RBACRole{
					{
						Name:    kong.String("workspace-portal-admin"),
						Comment: kong.String("Full access to Dev Portal related endpoints in the workspace"),
					},
				},
				RBACEndpointPermissions: []*kong.RBACEndpointPermission{
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/developers"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/developers/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/files"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/files/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/kong"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*/*/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*/*/*/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*/*/*/*/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/workspaces/default"),
						Actions:   []*string{kong.String("read"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=2.7.0 <3.0.0")
			setup(t)

			sync(tc.kongFile, "--rbac-resources-only")
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_RBAC_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "rbac",
			kongFile: "testdata/sync/xxx-rbac-endpoint-permissions/kong3x.yaml",
			expectedState: utils.KongRawState{
				RBACRoles: []*kong.RBACRole{
					{
						Name:    kong.String("workspace-portal-admin"),
						Comment: kong.String("Full access to Dev Portal related endpoints in the workspace"),
					},
				},
				RBACEndpointPermissions: []*kong.RBACEndpointPermission{
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/developers"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/developers/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/files"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/files/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/kong"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*/*/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*/*/*/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/rbac/*/*/*/*/*"),
						Actions:   []*string{kong.String("read"), kong.String("delete"), kong.String("create"), kong.String("update")},
						Negative:  kong.Bool(true),
					},
					{
						Workspace: kong.String("default"),
						Endpoint:  kong.String("/workspaces/default"),
						Actions:   []*string{kong.String("read"), kong.String("update")},
						Negative:  kong.Bool(false),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=3.0.0")
			setup(t)

			sync(tc.kongFile, "--rbac-resources-only")
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_Create_Route_With_Service_Name_Reference_2x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "create a route with a service name reference",
			kongFile: "testdata/sync/010-create-route-with-service-name-reference/kong.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
				Routes:   route1_20x,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.7.0 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_Create_Route_With_Service_Name_Reference_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "create a route with a service name reference",
			kongFile: "testdata/sync/010-create-route-with-service-name-reference/kong3x.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
				Routes:   route1_20x,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.7.0 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 1.x.x
//   - 2.x.x
func Test_Sync_PluginsOnEntitiesTill_3_0_0(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "create plugins on services, routes and consumers",
			kongFile: "testdata/sync/xxx-plugins-on-entities/kong.yaml",
			expectedState: utils.KongRawState{
				Services:  svc1_207,
				Routes:    route1_20x,
				Plugins:   plugin_on_entities,
				Consumers: consumer,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.8.0 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.0.0+
func Test_Sync_PluginsOnEntitiesFrom_3_0_0(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "create plugins on services, routes and consumers",
			kongFile: "testdata/sync/xxx-plugins-on-entities/kong.yaml",
			expectedState: utils.KongRawState{
				Services:  svc1_207,
				Routes:    route1_20x,
				Plugins:   plugin_on_entities3x,
				Consumers: consumer,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.4.0+
func Test_Sync_PluginsOnConsumerGroupsWithTagsFrom_3_4_0(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "create plugins on consumer-groups",
			kongFile: "testdata/sync/xxx-plugins-on-entities/kong-cg-plugin.yaml",
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("foo"),
						},
					},
				},
				Plugins: consumerGroupScopedPluginWithTags,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenEnterpriseOrKonnect(t, ">=3.4.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.0.0+
func Test_Sync_PluginOrdering(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin with ordering",
			kongFile: "testdata/sync/011-plugin-ordering/kong.yaml",
			expectedState: utils.KongRawState{
				Plugins: []*kong.Plugin{
					{
						Name: kong.String("request-termination"),
						Protocols: []*string{
							kong.String("grpc"),
							kong.String("grpcs"),
							kong.String("http"),
							kong.String("https"),
						},
						Enabled: kong.Bool(true),
						Config: kong.Configuration{
							"status_code":  float64(200),
							"echo":         false,
							"content_type": nil,
							"body":         nil,
							"message":      nil,
							"trigger":      nil,
						},
						Ordering: &kong.PluginOrdering{
							Before: kong.PluginOrderingPhase{
								"access": []string{"basic-auth"},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.x
func Test_Sync_Unsupported_Formats(t *testing.T) {
	tests := []struct {
		name          string
		kongFile      string
		expectedError error
	}{
		{
			name:     "creates a service",
			kongFile: "testdata/sync/001-create-a-service/kong.yaml",
			expectedError: errors.New(
				"cannot apply '1.1' config format version to Kong version 3.0 or above.\n" +
					utils.UpgradeMessage),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=3.0.0")
			setup(t)

			err := sync(tc.kongFile)
			assert.Equal(t, err, tc.expectedError)
		})
	}
}

var (
	goodCACertPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIE6DCCAtACCQCjgi452nKnUDANBgkqhkiG9w0BAQsFADA2MQswCQYDVQQGEwJV
UzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEAwwJbG9jYWxob3N0MB4XDTIy
MTAwNDE4NTEyOFoXDTMyMTAwMTE4NTEyOFowNjELMAkGA1UEBhMCVVMxEzARBgNV
BAgMCkNhbGlmb3JuaWExEjAQBgNVBAMMCWxvY2FsaG9zdDCCAiIwDQYJKoZIhvcN
AQEBBQADggIPADCCAgoCggIBALUwleXMo+CxQFvgtmJbWHO4k3YBJwzWqcr2xWn+
vgeoLiKFDQC11F/nnWNKkPZyilLeJda5c9YEVaA9IW6/PZhxQ430RM53EJHoiIPB
B9j7BHGzsvWYHEkjXvGQWeD3mR4TAkoCVTfPAjBji/SL+WvLpgPW5hKRVuedD8ja
cTvkNfk6u2TwPYGgekh9+wS9zcEQs4OwsEiQxmi3Z8if1m1uD09tjqAHb0klPEzM
64tPvlzJrIcH3Z5iF+B9qr91PCQJVYOCjGWlUgPULaqIoTVtY+AnaNnNcol0LM/i
oq7uD0JbeyIFDFMDJVqZwDf/zowzLLlP8Hkok4M8JTefXvB0puQoxmGwOAhwlA0G
KF5etrmhg+dOb+f3nWdgbyjPEytyOeMOOA/4Lb8dHRlf9JnEc4DJqwRVPM9BMeUu
9ZlrSWvURRk8nUZfkjTstLqO2aeubfOvb+tDKUq5Ue2B+AFs0ETLy3bds8TU9syV
5Kl+tIwek2TXzc7afvmeCDoRunAx5nVhmW8dpGhknOmJM0GxOi5s2tiu8/3T9XdH
WcH/GMrocZrkhvzkZccSLYoo1jcDn9LwxHVr/BZ43NymjVa6T3QRTta4Kg5wWpfS
yXi4gIW7VJM12CmNfSDEXqhF03+fjFzoWH+YfBK/9GgUMNjnXWIL9PgFFOBomwEL
tv5zAgMBAAEwDQYJKoZIhvcNAQELBQADggIBAKH8eUGgH/OSS3mHB3Gqv1m2Ea04
Cs03KNEt1weelcHIBWVnPp+jGcSIIfMBnDFAwgxtBKhwptJ9ZKXIzjh7YFxbOT01
NU+KQ6tD+NFDf+SAUC4AWV9Cam63JIaCVNDoo5UjVMlssnng7NefM1q2+ucoP+gs
+bvUCTJcp3FZsq8aUI9Rka575HqRhl/8kyhcwICCgT5UHQJvCQYrInJ0Faem6dr0
tHw+PZ1bo6qB7uxBjK9kyu7dK/vEKliUGM4/MXMDKIc5qXUs47wPLbjxvKsuDglK
KftgUWNYRxx9Bf9ylbjd+ayo3+1Lb9cbvdZnh0UHN6677NvXlWNheCmeysLGQHtm
5H6iIhZ75r6QuC7m6hBSJYtLU3fsQECrmaS/+xBGoSSZjacciO7b7qjQdWOfQREn
7vc5eu0N+CJkp8t3SsyQP6v2Su3ILeTt2EWrmmE4K7SYlJe1HrUVj0AWUwzLa6+Z
+Dx16p3M0RBdFMGNNhLqvG3WRfE5c5md34Aq/C5ePjN7pQGmJhI6weowuX9wCrnh
nJJJRfqyJvqgnVBZ6IawNcOyIofITZHlYVKuaDB1odzWCDNEvFftgJvH0MnO7OY9
Pb9hILPoCy+91jQAVh6Z/ghIcZKHV+N6zV3uS3t5vCejhCNK8mUPSOwAeDf3Bq5r
wQPXd0DdsYGmXVIh
-----END CERTIFICATE-----`)

	badCACertPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIDkzCCAnugAwIBAgIUYGc07pbHSjOBPreXh7OcNT2+sD4wDQYJKoZIhvcNAQEL
BQAwWTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMRUwEwYDVQQKDAxZb2xvNDIs
IEluYy4xJjAkBgNVBAMMHVlvbG80MiBzZWxmLXNpZ25lZCB0ZXN0aW5nIENBMB4X
DTIyMDMyOTE5NDczM1oXDTMyMDMyNjE5NDczM1owWTELMAkGA1UEBhMCVVMxCzAJ
BgNVBAgMAkNBMRUwEwYDVQQKDAxZb2xvNDIsIEluYy4xJjAkBgNVBAMMHVlvbG80
MiBzZWxmLXNpZ25lZCB0ZXN0aW5nIENBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAvnhTgdJALnuLKDA0ZUZRVMqcaaC+qvfJkiEFGYwX2ZJiFtzU65F/
sB2L0ToFqY4tmMVlOmiSZFnRLDZecmQDbbNwc3wtNikmxIOzx4qR4kbRP8DDdyIf
gaNmGCuaXTM5+FYy2iNBn6CeibIjqdErQlAbFLwQs5t3mLsjii2U4cyvfRtO+0RV
HdJ6Np5LsVziN0c5gVIesIrrbxLcOjtXDzwd/w/j5NXqL/OwD5EBH2vqd3QKKX4t
s83BLl2EsbUse47VAImavrwDhmV6S/p/NuJHqjJ6dIbXLYxNS7g26ijcrXxvNhiu
YoZTykSgdI3BXMNAm1ahP/BtJPZpU7CVdQIDAQABo1MwUTAdBgNVHQ4EFgQUe1WZ
fMfZQ9QIJIttwTmcrnl40ccwHwYDVR0jBBgwFoAUe1WZfMfZQ9QIJIttwTmcrnl4
0ccwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAs4Z8VYbvEs93
haTHdbbaKk0V6xAL/Q8I8GitK9E8cgf8C5rwwn+wU/Gf39dtMUlnW8uxyzRPx53u
CAAcJAWkabT+xwrlrqjO68H3MgIAwgWA5yZC+qW7ECA8xYEK6DzEHIaOpagJdKcL
IaZr/qTJlEQClvwDs4x/BpHRB5XbmJs86GqEB7XWAm+T2L8DluHAXvek+welF4Xo
fQtLlNS/vqTDqPxkSbJhFv1L7/4gdwfAz51wH/iL7AG/ubFEtoGZPK9YCJ40yTWz
8XrUoqUC+2WIZdtmo6dFFJcLfQg4ARJZjaK6lmxJun3iRMZjKJdQKm/NEKz4y9kA
u8S6yNlu2Q==
-----END CERTIFICATE-----`)
)

// test scope:
//   - 3.0.0+
//
// This test does two things:
// 1. makes sure decK can correctly configure a Vault entity
// 2. makes sure secrets management works as expected end-to-end
//
// Specifically, for (2) we make use of:
// - a Service and a Route to verify the overall flow works end-to-end
// - a Certificate with secret references
// - an {env} Vault using 'MY_SECRET_' as env variables prefix
//
// The Kong EE instance running in the CI includes the MY_SECRET_CERT
// and MY_SECRET_KEY env variables storing cert/key signed with `caCert`.
// These variables are pulled into the {env} Vault after decK deploy
// the configuration.
//
// After the `deck sync` and the configuration verification step,
// an HTTPS client is created using the `caCert` used to sign the
// deployed certificate, and then a GET is performed to test the
// proxy functionality, which should return a 200.
func Test_Sync_Vault(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create an SSL service/route using an ENV vault",
			kongFile: "testdata/sync/012-vaults/kong3x.yaml",
			expectedState: utils.KongRawState{
				Vaults: []*kong.Vault{
					{
						Name:        kong.String("env"),
						Prefix:      kong.String("my-env-vault"),
						Description: kong.String("ENV vault for secrets"),
						Config: kong.Configuration{
							"prefix": "MY_SECRET_",
						},
					},
				},
				Services: []*kong.Service{
					{
						ID:             kong.String("58076db2-28b6-423b-ba39-a797193017f7"),
						Name:           kong.String("svc1"),
						ConnectTimeout: kong.Int(60000),
						Host:           kong.String("httpbin.org"),
						Port:           kong.Int(80),
						Path:           kong.String("/status/200"),
						Protocol:       kong.String("http"),
						ReadTimeout:    kong.Int(60000),
						Retries:        kong.Int(5),
						WriteTimeout:   kong.Int(60000),
						Tags:           nil,
						Enabled:        kong.Bool(true),
					},
				},
				Routes: route1_20x,
				Certificates: []*kong.Certificate{
					{
						ID:   kong.String("13c562a1-191c-4464-9b18-e5222b46035b"),
						Cert: kong.String("{vault://my-env-vault/cert}"),
						Key:  kong.String("{vault://my-env-vault/key}"),
					},
				},
				SNIs: []*kong.SNI{
					{
						Name: kong.String("localhost"),
						Certificate: &kong.Certificate{
							ID: kong.String("13c562a1-191c-4464-9b18-e5222b46035b"),
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)

			// Kong proxy may need a bit to be ready.
			time.Sleep(time.Second * 5)

			// build simple http client
			client := &http.Client{}

			// use simple http client with https should result
			// in a failure due missing certificate.
			_, err := client.Get("https://localhost:8443/r1")
			assert.NotNil(t, err)

			// use transport with wrong CA cert this should result
			// in a failure due to unknown authority.
			badCACertPool := x509.NewCertPool()
			badCACertPool.AppendCertsFromPEM(badCACertPEM)

			client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:    badCACertPool,
						ClientAuth: tls.RequireAndVerifyClientCert,
					},
				},
			}

			_, err = client.Get("https://localhost:8443/r1")
			assert.NotNil(t, err)

			// use transport with good CA cert should pass
			// if referenced secrets are resolved correctly
			// using the ENV vault.
			goodCACertPool := x509.NewCertPool()
			goodCACertPool.AppendCertsFromPEM(goodCACertPEM)

			client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:    goodCACertPool,
						ClientAuth: tls.RequireAndVerifyClientCert,
					},
				},
			}

			res, err := client.Get("https://localhost:8443/r1")
			assert.NoError(t, err)
			assert.Equal(t, res.StatusCode, http.StatusOK)
		})
	}
}

// test scope:
//   - 2.8.x
func Test_Sync_UpdateUsernameInConsumerWithCustomID(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		kongFileInitial string
		expectedState   utils.KongRawState
	}{
		{
			name:            "update username on a consumer with custom_id",
			kongFile:        "testdata/sync/013-update-username-consumer-with-custom-id/kong.yaml",
			kongFileInitial: "testdata/sync/013-update-username-consumer-with-custom-id/kong-initial.yaml",
			expectedState: utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("test_new"),
						CustomID: kong.String("custom_test"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.8.0 <3.0.0")
			setup(t)

			// set up initial state
			sync(tc.kongFileInitial)
			// update with desired final state
			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.8.x
func Test_Sync_UpdateConsumerWithCustomID(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		kongFileInitial string
		expectedState   utils.KongRawState
	}{
		{
			name:            "update username on a consumer with custom_id",
			kongFile:        "testdata/sync/014-update-consumer-with-custom-id/kong.yaml",
			kongFileInitial: "testdata/sync/014-update-consumer-with-custom-id/kong-initial.yaml",
			expectedState: utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("test"),
						CustomID: kong.String("new_custom_test"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=2.8.0 <3.0.0")
			setup(t)

			// set up initial state
			sync(tc.kongFileInitial)
			// update with desired final state
			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.x
func Test_Sync_UpdateUsernameInConsumerWithCustomID_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		kongFileInitial string
		expectedState   utils.KongRawState
	}{
		{
			name:            "update username on a consumer with custom_id",
			kongFile:        "testdata/sync/013-update-username-consumer-with-custom-id/kong3x.yaml",
			kongFileInitial: "testdata/sync/013-update-username-consumer-with-custom-id/kong3x-initial.yaml",
			expectedState: utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("test_new"),
						CustomID: kong.String("custom_test"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.0.0")
			setup(t)

			// set up initial state
			sync(tc.kongFileInitial)
			// update with desired final state
			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.x
func Test_Sync_UpdateConsumerWithCustomID_3x(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		kongFileInitial string
		expectedState   utils.KongRawState
	}{
		{
			name:            "update username on a consumer with custom_id",
			kongFile:        "testdata/sync/014-update-consumer-with-custom-id/kong3x.yaml",
			kongFileInitial: "testdata/sync/014-update-consumer-with-custom-id/kong3x-initial.yaml",
			expectedState: utils.KongRawState{
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("test_consumer_3x"),
						CustomID: kong.String("test_consumer_3x_custom_test"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.0.0")
			setup(t)

			// set up initial state
			sync(tc.kongFileInitial)
			// update with desired final state
			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 2.7+
func Test_Sync_ConsumerGroupsTill30(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}
	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates consumer groups",
			kongFile: "testdata/sync/015-consumer-groups/kong.yaml",
			expectedState: utils.KongRawState{
				Consumers:      consumerGroupsConsumers,
				ConsumerGroups: consumerGroups,
			},
		},
		{
			name:     "creates consumer groups and plugin",
			kongFile: "testdata/sync/016-consumer-groups-and-plugins/kong.yaml",
			expectedState: utils.KongRawState{
				Consumers:      consumerGroupsConsumers,
				ConsumerGroups: consumerGroupsWithRLA,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=2.7.0 <3.0.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.1
func Test_Sync_ConsumerGroups_31(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}
	tests := []struct {
		name            string
		kongFile        string
		kongFileInitial string
		expectedState   utils.KongRawState
	}{
		{
			name:            "creates consumer groups",
			kongFile:        "testdata/sync/015-consumer-groups/kong3x.yaml",
			kongFileInitial: "testdata/sync/015-consumer-groups/kong3x-initial.yaml",
			expectedState: utils.KongRawState{
				Consumers:      consumerGroupsConsumers,
				ConsumerGroups: consumerGroupsWithTags,
			},
		},
		{
			name:            "creates consumer groups and plugin",
			kongFile:        "testdata/sync/016-consumer-groups-and-plugins/kong3x.yaml",
			kongFileInitial: "testdata/sync/016-consumer-groups-and-plugins/kong3x-initial.yaml",
			expectedState: utils.KongRawState{
				Consumers:      consumerGroupsConsumers,
				ConsumerGroups: consumerGroupsWithTagsAndRLA,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", "==3.1.0")
			setup(t)

			// set up initial state
			sync(tc.kongFileInitial)
			// update with desired final state
			sync(tc.kongFile)

			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// This test has 2 goals:
//   - make sure consumer groups and their related properties
//     can be configured correctly in Kong
//   - the actual consumer groups functionality works once set
//
// This is achieved via configuring:
// - 3 consumers:
//   - 1 belonging to Gold Consumer Group
//   - 1 belonging to Silver Consumer Group
//   - 1 not belonging to any Consumer Group
//
// - 3 key-auths, one for each consumer
// - 1 global key-auth plugin
// - 1 global RLA plugin
// - 2 consumer group
// - 2 RLA override, 1 for each consumer group
// - 1 service pointing to mockbin.org
// - 1 route proxying the above service
//
// Once the configuration is verified to be matching in Kong,
// we then check whether the override is correctly applied: consumers
// not belonging to the consumer group should be limited to 5 requests
// every 30s, while consumers belonging to the 'gold' and 'silver' consumer groups
// should be allowed to run respectively 10 and 7 requests in the same timeframe.
// In order to make sure this is the case, we run requests in a loop
// for all consumers and then check at what point they start to receive 429.
func Test_Sync_ConsumerGroupsRLAFrom31(t *testing.T) {
	const (
		maxGoldRequestsNumber    = 10
		maxSilverRequestsNumber  = 7
		maxRegularRequestsNumber = 5
	)
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}
	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates consumer groups application",
			kongFile: "testdata/sync/017-consumer-groups-rla-application/kong3x.yaml",
			expectedState: utils.KongRawState{
				Consumers:      consumerGroupsConsumers,
				ConsumerGroups: consumerGroupsWithRLAApp,
				Plugins:        consumerGroupAppPlugins,
				Services:       svc1_207,
				Routes:         route1_20x,
				KeyAuths: []*kong.KeyAuth{
					{
						Consumer: &kong.Consumer{
							ID: kong.String("87095815-5395-454e-8c18-a11c9bc0ef04"),
						},
						Key: kong.String("i-am-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("5a5b9369-baeb-4faa-a902-c40ccdc2928e"),
						},
						Key: kong.String("i-am-not-so-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("e894ea9e-ad08-4acf-a960-5a23aa7701c7"),
						},
						Key: kong.String("i-am-just-average"),
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=3.0.0 <3.1.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)

			// Kong proxy may need a bit to be ready.
			time.Sleep(time.Second * 10)

			// build simple http client
			client := &http.Client{}

			// test 'foo' consumer (part of 'gold' group)
			req, err := http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-special")
			n := 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxGoldRequestsNumber, n)

			// test 'bar' consumer (part of 'silver' group)
			req, err = http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-not-so-special")
			n = 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxSilverRequestsNumber, n)

			// test 'baz' consumer (not part of any group)
			req, err = http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-just-average")
			n = 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxRegularRequestsNumber, n)
		})
	}
}

// test scope:
//   - konnect
func Test_Sync_ConsumerGroupsKonnect(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}
	tests := []struct {
		name            string
		kongFile        string
		kongFileInitial string
		expectedState   utils.KongRawState
	}{
		{
			name:            "creates consumer groups",
			kongFile:        "testdata/sync/015-consumer-groups/kong3x.yaml",
			kongFileInitial: "testdata/sync/015-consumer-groups/kong3x-initial.yaml",
			expectedState: utils.KongRawState{
				Consumers:      consumerGroupsConsumers,
				ConsumerGroups: consumerGroupsWithTags,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "konnect", "")
			setup(t)

			// set up initial state
			sync(tc.kongFileInitial)
			// update with desired final state
			sync(tc.kongFile)

			testKongState(t, client, true, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.2.0+
func Test_Sync_PluginInstanceName(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name            string
		kongFile        string
		initialKongFile string
		expectedState   utils.KongRawState
	}{
		{
			name:     "create a plugin with instance_name",
			kongFile: "testdata/sync/018-plugin-instance_name/kong-with-instance_name.yaml",
			expectedState: utils.KongRawState{
				Plugins: []*kong.Plugin{
					{
						Name:         kong.String("request-termination"),
						InstanceName: kong.String("my-plugin"),
						Protocols: []*string{
							kong.String("grpc"),
							kong.String("grpcs"),
							kong.String("http"),
							kong.String("https"),
						},
						Enabled: kong.Bool(true),
						Config: kong.Configuration{
							"status_code":  float64(200),
							"echo":         false,
							"content_type": nil,
							"body":         nil,
							"message":      nil,
							"trigger":      nil,
						},
					},
				},
			},
		},
		{
			name:     "create a plugin without instance_name",
			kongFile: "testdata/sync/018-plugin-instance_name/kong-without-instance_name.yaml",
			expectedState: utils.KongRawState{
				Plugins: []*kong.Plugin{
					{
						Name: kong.String("request-termination"),
						Protocols: []*string{
							kong.String("grpc"),
							kong.String("grpcs"),
							kong.String("http"),
							kong.String("https"),
						},
						Enabled: kong.Bool(true),
						Config: kong.Configuration{
							"status_code":  float64(200),
							"echo":         false,
							"content_type": nil,
							"body":         nil,
							"message":      nil,
							"trigger":      nil,
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKongOrKonnect(t, ">=3.2.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.2.x
//   - 3.3.x
func Test_Sync_SkipConsumers(t *testing.T) {
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		skipConsumers bool
		expectedState utils.KongRawState
	}{
		{
			name:     "skip-consumers successfully",
			kongFile: "testdata/sync/019-skip-consumers/kong3x.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
			},
			skipConsumers: true,
		},
		{
			name:     "do not skip consumers successfully",
			kongFile: "testdata/sync/019-skip-consumers/kong3x.yaml",
			expectedState: utils.KongRawState{
				Services:       svc1_207,
				Consumers:      consumerGroupsConsumers,
				ConsumerGroups: consumerGroupsWithTagsAndRLA,
			},
			skipConsumers: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=3.2.0 <3.4.0")
			setup(t)

			if tc.skipConsumers {
				sync(tc.kongFile, "--skip-consumers")
			} else {
				sync(tc.kongFile)
			}
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - 3.4.x
func Test_Sync_SkipConsumers_34x(t *testing.T) {
	runWhen(t, "enterprise", ">=3.4.0 <3.5.0")
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		skipConsumers bool
		expectedState utils.KongRawState
	}{
		{
			name:     "skip-consumers successfully",
			kongFile: "testdata/sync/019-skip-consumers/kong34.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
			},
			skipConsumers: true,
		},
		{
			name:     "do not skip consumers successfully",
			kongFile: "testdata/sync/019-skip-consumers/kong34.yaml",
			expectedState: utils.KongRawState{
				Services:  svc1_207,
				Consumers: consumerGroupsConsumers,
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
							Tags: kong.StringSlice("tag1", "tag3"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("bar"),
							},
						},
					},
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("gold"),
							Tags: kong.StringSlice("tag1", "tag2"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("foo"),
							},
						},
					},
				},
				Plugins: []*kong.Plugin{
					{
						Name: kong.String("rate-limiting-advanced"),
						ConsumerGroup: &kong.ConsumerGroup{
							ID: kong.String("77e6691d-67c0-446a-9401-27be2b141aae"),
						},
						Config: kong.Configuration{
							"consumer_groups":         nil,
							"dictionary_name":         string("kong_rate_limiting_counters"),
							"disable_penalty":         bool(false),
							"enforce_consumer_groups": bool(false),
							"error_code":              float64(429),
							"error_message":           string("API rate limit exceeded"),
							"header_name":             nil,
							"hide_client_headers":     bool(false),
							"identifier":              string("consumer"),
							"limit":                   []any{float64(10)},
							"namespace":               string("gold"),
							"path":                    nil,
							"redis": map[string]any{
								"cluster_addresses":   nil,
								"connect_timeout":     nil,
								"database":            float64(0),
								"host":                nil,
								"keepalive_backlog":   nil,
								"keepalive_pool_size": float64(30),
								"password":            nil,
								"port":                nil,
								"read_timeout":        nil,
								"send_timeout":        nil,
								"sentinel_addresses":  nil,
								"sentinel_master":     nil,
								"sentinel_password":   nil,
								"sentinel_role":       nil,
								"sentinel_username":   nil,
								"server_name":         nil,
								"ssl":                 false,
								"ssl_verify":          false,
								"timeout":             float64(2000),
								"username":            nil,
							},
							"retry_after_jitter_max": float64(1),
							"strategy":               string("local"),
							"sync_rate":              float64(-1),
							"window_size":            []any{float64(60)},
							"window_type":            string("sliding"),
						},
						Enabled:   kong.Bool(true),
						Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
					},
					{
						Name: kong.String("rate-limiting-advanced"),
						ConsumerGroup: &kong.ConsumerGroup{
							ID: kong.String("5bcbd3a7-030b-4310-bd1d-2721ff85d236"),
						},
						Config: kong.Configuration{
							"consumer_groups":         nil,
							"dictionary_name":         string("kong_rate_limiting_counters"),
							"disable_penalty":         bool(false),
							"enforce_consumer_groups": bool(false),
							"error_code":              float64(429),
							"error_message":           string("API rate limit exceeded"),
							"header_name":             nil,
							"hide_client_headers":     bool(false),
							"identifier":              string("consumer"),
							"limit":                   []any{float64(7)},
							"namespace":               string("silver"),
							"path":                    nil,
							"redis": map[string]any{
								"cluster_addresses":   nil,
								"connect_timeout":     nil,
								"database":            float64(0),
								"host":                nil,
								"keepalive_backlog":   nil,
								"keepalive_pool_size": float64(30),
								"password":            nil,
								"port":                nil,
								"read_timeout":        nil,
								"send_timeout":        nil,
								"sentinel_addresses":  nil,
								"sentinel_master":     nil,
								"sentinel_password":   nil,
								"sentinel_role":       nil,
								"sentinel_username":   nil,
								"server_name":         nil,
								"ssl":                 false,
								"ssl_verify":          false,
								"timeout":             float64(2000),
								"username":            nil,
							},
							"retry_after_jitter_max": float64(1),
							"strategy":               string("local"),
							"sync_rate":              float64(-1),
							"window_size":            []any{float64(60)},
							"window_type":            string("sliding"),
						},
						Enabled:   kong.Bool(true),
						Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
					},
				},
			},
			skipConsumers: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setup(t)

			if tc.skipConsumers {
				sync(tc.kongFile, "--skip-consumers")
			} else {
				sync(tc.kongFile)
			}
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - konnect
func Test_Sync_SkipConsumers_Konnect(t *testing.T) {
	runWhenKonnect(t)
	// setup stage
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	tests := []struct {
		name          string
		kongFile      string
		skipConsumers bool
		expectedState utils.KongRawState
	}{
		{
			name:     "skip-consumers successfully",
			kongFile: "testdata/sync/019-skip-consumers/kong34.yaml",
			expectedState: utils.KongRawState{
				Services: svc1_207,
			},
			skipConsumers: true,
		},
		{
			name:     "do not skip consumers successfully",
			kongFile: "testdata/sync/019-skip-consumers/kong34.yaml",
			expectedState: utils.KongRawState{
				Services:  svc1_207,
				Consumers: consumerGroupsConsumers,
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
							Tags: kong.StringSlice("tag1", "tag3"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("bar"),
							},
						},
					},
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("gold"),
							Tags: kong.StringSlice("tag1", "tag2"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("foo"),
							},
						},
					},
				},
				Plugins: []*kong.Plugin{
					{
						Name: kong.String("rate-limiting-advanced"),
						ConsumerGroup: &kong.ConsumerGroup{
							ID: kong.String("77e6691d-67c0-446a-9401-27be2b141aae"),
						},
						Config: kong.Configuration{
							"consumer_groups":         nil,
							"dictionary_name":         string("kong_rate_limiting_counters"),
							"disable_penalty":         bool(false),
							"enforce_consumer_groups": bool(false),
							"error_code":              float64(429),
							"error_message":           string("API rate limit exceeded"),
							"header_name":             nil,
							"hide_client_headers":     bool(false),
							"identifier":              string("consumer"),
							"limit":                   []any{float64(10)},
							"namespace":               string("gold"),
							"path":                    nil,
							"redis": map[string]any{
								"cluster_addresses":   nil,
								"connect_timeout":     nil,
								"database":            float64(0),
								"host":                nil,
								"keepalive_backlog":   nil,
								"keepalive_pool_size": float64(30),
								"password":            nil,
								"port":                nil,
								"read_timeout":        nil,
								"send_timeout":        nil,
								"sentinel_addresses":  nil,
								"sentinel_master":     nil,
								"sentinel_password":   nil,
								"sentinel_role":       nil,
								"sentinel_username":   nil,
								"server_name":         nil,
								"ssl":                 false,
								"ssl_verify":          false,
								"timeout":             float64(2000),
								"username":            nil,
							},
							"retry_after_jitter_max": float64(1),
							"strategy":               string("local"),
							"sync_rate":              nil,
							"window_size":            []any{float64(60)},
							"window_type":            string("sliding"),
						},
						Enabled:   kong.Bool(true),
						Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
					},
					{
						Name: kong.String("rate-limiting-advanced"),
						ConsumerGroup: &kong.ConsumerGroup{
							ID: kong.String("5bcbd3a7-030b-4310-bd1d-2721ff85d236"),
						},
						Config: kong.Configuration{
							"consumer_groups":         nil,
							"dictionary_name":         string("kong_rate_limiting_counters"),
							"disable_penalty":         bool(false),
							"enforce_consumer_groups": bool(false),
							"error_code":              float64(429),
							"error_message":           string("API rate limit exceeded"),
							"header_name":             nil,
							"hide_client_headers":     bool(false),
							"identifier":              string("consumer"),
							"limit":                   []any{float64(7)},
							"namespace":               string("silver"),
							"path":                    nil,
							"redis": map[string]any{
								"cluster_addresses":   nil,
								"connect_timeout":     nil,
								"database":            float64(0),
								"host":                nil,
								"keepalive_backlog":   nil,
								"keepalive_pool_size": float64(30),
								"password":            nil,
								"port":                nil,
								"read_timeout":        nil,
								"send_timeout":        nil,
								"sentinel_addresses":  nil,
								"sentinel_master":     nil,
								"sentinel_password":   nil,
								"sentinel_role":       nil,
								"sentinel_username":   nil,
								"server_name":         nil,
								"ssl":                 false,
								"ssl_verify":          false,
								"timeout":             float64(2000),
								"username":            nil,
							},
							"retry_after_jitter_max": float64(1),
							"strategy":               string("local"),
							"sync_rate":              nil,
							"window_size":            []any{float64(60)},
							"window_type":            string("sliding"),
						},
						Enabled:   kong.Bool(true),
						Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
					},
				},
			},
			skipConsumers: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=3.2.0")
			setup(t)

			if tc.skipConsumers {
				sync(tc.kongFile, "--skip-consumers")
			} else {
				sync(tc.kongFile)
			}
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// In the tests we're concerned only with the IDs and names of the entities
// we'll ignore other fields when comparing states.
var ignoreFieldsIrrelevantForIDsTests = []cmp.Option{
	cmpopts.IgnoreFields(
		kong.Plugin{},
		"Config",
		"Protocols",
		"Enabled",
	),
	cmpopts.IgnoreFields(
		kong.Service{},
		"ConnectTimeout",
		"Enabled",
		"Host",
		"Port",
		"Protocol",
		"ReadTimeout",
		"WriteTimeout",
		"Retries",
	),
	cmpopts.IgnoreFields(
		kong.Route{},
		"Paths",
		"PathHandling",
		"PreserveHost",
		"Protocols",
		"RegexPriority",
		"StripPath",
		"HTTPSRedirectStatusCode",
		"Sources",
		"Destinations",
		"RequestBuffering",
		"ResponseBuffering",
	),
}

// test scope:
//   - 3.0.0+
//   - konnect
func Test_Sync_ChangingIDsWhileKeepingNames(t *testing.T) {
	runWhenKongOrKonnect(t, ">=3.0.0")

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	// These are the IDs that should be present in Kong after the second sync in all cases.
	var (
		expectedServiceID  = kong.String("98076db2-28b6-423b-ba39-a797193017f7")
		expectedRouteID    = kong.String("97b6a97e-f3f7-4c47-857a-7464cb9e202b")
		expectedConsumerID = kong.String("9a1e49a8-2536-41fa-a4e9-605bf218a4fa")
	)

	// These are the entities that should be present in Kong after the second sync in all cases.
	var (
		expectedService = &kong.Service{
			Name: kong.String("s1"),
			ID:   expectedServiceID,
		}

		expectedRoute = &kong.Route{
			Name: kong.String("r1"),
			ID:   expectedRouteID,
			Service: &kong.Service{
				ID: expectedServiceID,
			},
		}

		expectedConsumer = &kong.Consumer{
			Username: kong.String("c1"),
			ID:       expectedConsumerID,
		}

		expectedPlugins = []*kong.Plugin{
			{
				Name: kong.String("rate-limiting"),
				Route: &kong.Route{
					ID: expectedRouteID,
				},
			},
			{
				Name: kong.String("rate-limiting"),
				Service: &kong.Service{
					ID: expectedServiceID,
				},
			},
			{
				Name: kong.String("rate-limiting"),
				Consumer: &kong.Consumer{
					ID: expectedConsumerID,
				},
			},
		}
	)

	testCases := []struct {
		name         string
		beforeConfig string
	}{
		{
			name:         "all entities have the same names, but different IDs",
			beforeConfig: "testdata/sync/020-same-names-altered-ids/1-before.yaml",
		},
		{
			name:         "service and consumer changed IDs, route did not",
			beforeConfig: "testdata/sync/020-same-names-altered-ids/2-before.yaml",
		},
		{
			name:         "route and consumer changed IDs, service did not",
			beforeConfig: "testdata/sync/020-same-names-altered-ids/3-before.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setup(t)

			// First, create the entities with the original IDs.
			err = sync(tc.beforeConfig)
			require.NoError(t, err)

			// Then, sync again with the same names, but different IDs.
			err = sync("testdata/sync/020-same-names-altered-ids/desired.yaml")
			require.NoError(t, err)

			// Finally, check that the all entities exist and have the expected IDs.
			testKongState(t, client, false, utils.KongRawState{
				Services:  []*kong.Service{expectedService},
				Routes:    []*kong.Route{expectedRoute},
				Consumers: []*kong.Consumer{expectedConsumer},
				Plugins:   expectedPlugins,
			}, ignoreFieldsIrrelevantForIDsTests)
		})
	}
}

// test scope:
//   - 3.0.0+
//   - konnect
func Test_Sync_UpdateWithExplicitIDs(t *testing.T) {
	runWhenKongOrKonnect(t, ">=3.0.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	const (
		beforeConfig = "testdata/sync/021-update-with-explicit-ids/before.yaml"
		afterConfig  = "testdata/sync/021-update-with-explicit-ids/after.yaml"
	)

	// First, create entities with IDs assigned explicitly.
	err = sync(beforeConfig)
	require.NoError(t, err)

	// Then, sync again, adding tags to every entity just to trigger an update.
	err = sync(afterConfig)
	require.NoError(t, err)

	// Finally, verify that the update was successful.
	testKongState(t, client, false, utils.KongRawState{
		Services: []*kong.Service{
			{
				Name: kong.String("s1"),
				ID:   kong.String("c75a775b-3a32-4b73-8e05-f68169c23941"),
				Tags: kong.StringSlice("after"),
			},
		},
		Routes: []*kong.Route{
			{
				Name: kong.String("r1"),
				ID:   kong.String("97b6a97e-f3f7-4c47-857a-7464cb9e202b"),
				Tags: kong.StringSlice("after"),
				Service: &kong.Service{
					ID: kong.String("c75a775b-3a32-4b73-8e05-f68169c23941"),
				},
			},
		},
		Consumers: []*kong.Consumer{
			{
				Username: kong.String("c1"),
				Tags:     kong.StringSlice("after"),
			},
		},
	}, ignoreFieldsIrrelevantForIDsTests)
}

// test scope:
//   - 3.0.0+
//   - konnect
func Test_Sync_UpdateWithExplicitIDsWithNoNames(t *testing.T) {
	runWhenKongOrKonnect(t, ">=3.0.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	const (
		beforeConfig = "testdata/sync/022-update-with-explicit-ids-with-no-names/before.yaml"
		afterConfig  = "testdata/sync/022-update-with-explicit-ids-with-no-names/after.yaml"
	)

	// First, create entities with IDs assigned explicitly.
	err = sync(beforeConfig)
	require.NoError(t, err)

	// Then, sync again, adding tags to every entity just to trigger an update.
	err = sync(afterConfig)
	require.NoError(t, err)

	// Finally, verify that the update was successful.
	testKongState(t, client, false, utils.KongRawState{
		Services: []*kong.Service{
			{
				ID:   kong.String("c75a775b-3a32-4b73-8e05-f68169c23941"),
				Tags: kong.StringSlice("after"),
			},
		},
		Routes: []*kong.Route{
			{
				ID:   kong.String("97b6a97e-f3f7-4c47-857a-7464cb9e202b"),
				Tags: kong.StringSlice("after"),
				Service: &kong.Service{
					ID: kong.String("c75a775b-3a32-4b73-8e05-f68169c23941"),
				},
			},
		},
	}, ignoreFieldsIrrelevantForIDsTests)
}

// test scope:
//   - 3.0.0+
//   - konnect
func Test_Sync_CreateCertificateWithSNIs(t *testing.T) {
	runWhenKongOrKonnect(t, ">=3.0.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	err = sync("testdata/sync/023-create-and-update-certificate-with-snis/initial.yaml")
	require.NoError(t, err)

	// To ignore noise, we ignore the Key and Cert fields because they are not relevant for this test.
	ignoredFields := []cmp.Option{
		cmpopts.IgnoreFields(
			kong.Certificate{},
			"Key",
			"Cert",
		),
	}

	testKongState(t, client, false, utils.KongRawState{
		Certificates: []*kong.Certificate{
			{
				ID:   kong.String("c75a775b-3a32-4b73-8e05-f68169c23941"),
				Tags: kong.StringSlice("before"),
			},
		},
		SNIs: []*kong.SNI{
			{
				Name: kong.String("example.com"),
				Certificate: &kong.Certificate{
					ID: kong.String("c75a775b-3a32-4b73-8e05-f68169c23941"),
				},
			},
		},
	}, ignoredFields)

	err = sync("testdata/sync/023-create-and-update-certificate-with-snis/update.yaml")
	require.NoError(t, err)

	testKongState(t, client, false, utils.KongRawState{
		Certificates: []*kong.Certificate{
			{
				ID:   kong.String("c75a775b-3a32-4b73-8e05-f68169c23941"),
				Tags: kong.StringSlice("after"), // Tag should be updated.
			},
		},
		SNIs: []*kong.SNI{
			{
				Name: kong.String("example.com"),
				Certificate: &kong.Certificate{
					ID: kong.String("c75a775b-3a32-4b73-8e05-f68169c23941"),
				},
			},
		},
	}, ignoredFields)
}

// test scope:
//   - 3.0.0+
//   - konnect
func Test_Sync_ConsumersWithCustomIDAndOrUsername(t *testing.T) {
	runWhenKongOrKonnect(t, ">=3.0.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	err = sync("testdata/sync/024-consumers-with-custom_id-and-username/kong3x.yaml")
	require.NoError(t, err)

	testKongState(t, client, false, utils.KongRawState{
		Consumers: []*kong.Consumer{
			{
				ID:       kong.String("ce49186d-7670-445d-a218-897631b29ada"),
				Username: kong.String("Foo"),
				CustomID: kong.String("foo"),
			},
			{
				ID:       kong.String("7820f383-7b77-4fcc-af7f-14ff3e256693"),
				Username: kong.String("foo"),
				CustomID: kong.String("bar"),
			},
			{
				ID:       kong.String("18c62c3c-12cc-429a-8e5a-57f2c3691a6b"),
				CustomID: kong.String("custom_id_only"),
			},
			{
				ID:       kong.String("8ef278c9-48c1-43e1-b665-e9bc18fab4c8"),
				Username: kong.String("username_only"),
			},
		},
	}, nil)

	err = sync("testdata/sync/024-consumers-with-custom_id-and-username/kong3x-reverse-order.yaml")
	require.NoError(t, err)

	testKongState(t, client, false, utils.KongRawState{
		Consumers: []*kong.Consumer{
			{
				Username: kong.String("TestUser"),
			},
			{
				Username: kong.String("OtherUser"),
				CustomID: kong.String("TestUser"),
			},
		},
	}, nil)
}

// This test has 2 goals:
//   - make sure consumer groups scoped plugins can be configured correctly in Kong
//   - the actual consumer groups functionality works once set
//
// This is achieved via configuring:
// - 3 consumers:
//   - 1 belonging to Gold Consumer Group
//   - 1 belonging to Silver Consumer Group
//   - 1 not belonging to any Consumer Group
//
// - 3 key-auths, one for each consumer
// - 1 global key-auth plugin
// - 2 consumer group
// - 1 global RLA plugin
// - 2 RLA plugins, scoped to the related consumer groups
// - 1 service pointing to mockbin.org
// - 1 route proxying the above service
//
// Once the configuration is verified to be matching in Kong,
// we then check whether the specific RLA configuration is correctly applied: consumers
// not belonging to the consumer group should be limited to 5 requests
// every 30s, while consumers belonging to the 'gold' and 'silver' consumer groups
// should be allowed to run respectively 10 and 7 requests in the same timeframe.
// In order to make sure this is the case, we run requests in a loop
// for all consumers and then check at what point they start to receive 429.
func Test_Sync_ConsumerGroupsScopedPlugins(t *testing.T) {
	const (
		maxGoldRequestsNumber    = 10
		maxSilverRequestsNumber  = 7
		maxRegularRequestsNumber = 5
	)
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}
	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates consumer groups scoped plugins",
			kongFile: "testdata/sync/025-consumer-groups-scoped-plugins/kong3x.yaml",
			expectedState: utils.KongRawState{
				Consumers: consumerGroupsConsumers,
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("bar"),
							},
						},
					},
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("gold"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("foo"),
							},
						},
					},
				},
				Plugins:  consumerGroupScopedPlugins,
				Services: svc1_207,
				Routes:   route1_20x,
				KeyAuths: []*kong.KeyAuth{
					{
						Consumer: &kong.Consumer{
							ID: kong.String("87095815-5395-454e-8c18-a11c9bc0ef04"),
						},
						Key: kong.String("i-am-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("5a5b9369-baeb-4faa-a902-c40ccdc2928e"),
						},
						Key: kong.String("i-am-not-so-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("e894ea9e-ad08-4acf-a960-5a23aa7701c7"),
						},
						Key: kong.String("i-am-just-average"),
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=3.4.0 <3.5.0")
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)

			// Kong proxy may need a bit to be ready.
			time.Sleep(time.Second * 10)

			// build simple http client
			client := &http.Client{}

			// test 'foo' consumer (part of 'gold' group)
			req, err := http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-special")
			n := 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxGoldRequestsNumber, n)

			// test 'bar' consumer (part of 'silver' group)
			req, err = http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-not-so-special")
			n = 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxSilverRequestsNumber, n)

			// test 'baz' consumer (not part of any group)
			req, err = http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-just-average")
			n = 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxRegularRequestsNumber, n)
		})
	}
}

func Test_Sync_ConsumerGroupsScopedPlugins_After350(t *testing.T) {
	const (
		maxGoldRequestsNumber    = 10
		maxSilverRequestsNumber  = 7
		maxRegularRequestsNumber = 5
	)
	client, err := getTestClient()
	require.NoError(t, err)

	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
		skipFuncCond  string
	}{
		{
			name:         "creates consumer groups scoped plugins",
			skipFuncCond: "==3.5.0",
			kongFile:     "testdata/sync/025-consumer-groups-scoped-plugins/kong3x.yaml",
			expectedState: utils.KongRawState{
				Consumers: consumerGroupsConsumers,
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("bar"),
							},
						},
					},
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("gold"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("foo"),
							},
						},
					},
				},
				Plugins:  consumerGroupScopedPlugins35x,
				Services: svc1_207,
				Routes:   route1_20x,
				KeyAuths: []*kong.KeyAuth{
					{
						Consumer: &kong.Consumer{
							ID: kong.String("87095815-5395-454e-8c18-a11c9bc0ef04"),
						},
						Key: kong.String("i-am-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("5a5b9369-baeb-4faa-a902-c40ccdc2928e"),
						},
						Key: kong.String("i-am-not-so-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("e894ea9e-ad08-4acf-a960-5a23aa7701c7"),
						},
						Key: kong.String("i-am-just-average"),
					},
				},
			},
		},
		{
			name:         "creates consumer groups scoped plugins",
			skipFuncCond: ">=3.7.0",
			kongFile:     "testdata/sync/025-consumer-groups-scoped-plugins/kong3x.yaml",
			expectedState: utils.KongRawState{
				Consumers: consumerGroupsConsumers,
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("bar"),
							},
						},
					},
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("gold"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("foo"),
							},
						},
					},
				},
				Plugins:  consumerGroupScopedPlugins37x,
				Services: svc1_207,
				Routes:   route1_20x,
				KeyAuths: []*kong.KeyAuth{
					{
						Consumer: &kong.Consumer{
							ID: kong.String("87095815-5395-454e-8c18-a11c9bc0ef04"),
						},
						Key: kong.String("i-am-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("5a5b9369-baeb-4faa-a902-c40ccdc2928e"),
						},
						Key: kong.String("i-am-not-so-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("e894ea9e-ad08-4acf-a960-5a23aa7701c7"),
						},
						Key: kong.String("i-am-just-average"),
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.skipFuncCond+"_"+tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", tc.skipFuncCond)
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)

			// Kong proxy may need a bit to be ready.
			time.Sleep(time.Second * 10)

			// build simple http client
			client := &http.Client{}

			// test 'foo' consumer (part of 'gold' group)
			req, err := http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-special")
			n := 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxGoldRequestsNumber, n)

			// test 'bar' consumer (part of 'silver' group)
			req, err = http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-not-so-special")
			n = 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxSilverRequestsNumber, n)

			// test 'baz' consumer (not part of any group)
			req, err = http.NewRequest("GET", "http://localhost:8000/r1", nil)
			assert.NoError(t, err)
			req.Header.Add("apikey", "i-am-just-average")
			n = 0
			for n < 11 {
				resp, err := client.Do(req)
				assert.NoError(t, err)
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusTooManyRequests {
					break
				}
				n++
			}
			assert.Equal(t, maxRegularRequestsNumber, n)
		})
	}
}

// test scope:
//   - > 3.4.0
func Test_Sync_ConsumerGroupsScopedPlugins_Post340(t *testing.T) {
	tests := []struct {
		name          string
		kongFile      string
		expectedError error
	}{
		{
			name:          "attempt to create deprecated consumer groups configuration with Kong version >= 3.4.0 fails",
			kongFile:      "testdata/sync/017-consumer-groups-rla-application/kong3x.yaml",
			expectedError: fmt.Errorf("building state: %w", utils.ErrorConsumerGroupUpgrade),
		},
		{
			name:     "empty deprecated consumer groups configuration fields do not fail with Kong version >= 3.4.0",
			kongFile: "testdata/sync/017-consumer-groups-rla-application/kong3x-empty-application.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "enterprise", ">=3.4.0")
			setup(t)

			err := sync(tc.kongFile)
			if tc.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError.Error())
			}
		})
	}
}

func Test_Sync_ConsumerGroupsScopedPluginsKonnect(t *testing.T) {
	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}
	tests := []struct {
		name          string
		kongFile      string
		expectedState utils.KongRawState
	}{
		{
			name:     "creates consumer groups scoped plugins",
			kongFile: "testdata/sync/025-consumer-groups-scoped-plugins/kong3x.yaml",
			expectedState: utils.KongRawState{
				Consumers: consumerGroupsConsumers,
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("bar"),
							},
						},
					},
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("gold"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("foo"),
							},
						},
					},
				},
				Plugins:  consumerGroupScopedPlugins37x,
				Services: svc1_207,
				Routes:   route1_20x,
				KeyAuths: []*kong.KeyAuth{
					{
						Consumer: &kong.Consumer{
							ID: kong.String("87095815-5395-454e-8c18-a11c9bc0ef04"),
						},
						Key: kong.String("i-am-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("5a5b9369-baeb-4faa-a902-c40ccdc2928e"),
						},
						Key: kong.String("i-am-not-so-special"),
					},
					{
						Consumer: &kong.Consumer{
							ID: kong.String("e894ea9e-ad08-4acf-a960-5a23aa7701c7"),
						},
						Key: kong.String("i-am-just-average"),
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKonnect(t)
			setup(t)

			sync(tc.kongFile)
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

// test scope:
//   - konnect
func Test_Sync_KonnectRename(t *testing.T) {
	// setup stage
	tests := []struct {
		name             string
		controlPlaneName string
		runtimeGroupName string
		kongFile         string
		flags            []string
		expectedState    utils.KongRawState
	}{
		{
			name:     "konnect-runtime-group-name flag - default",
			kongFile: "testdata/sync/026-konnect-rename/default.yaml",
			flags:    []string{"--konnect-runtime-group-name", "default"},
			expectedState: utils.KongRawState{
				Services: defaultCPService,
			},
		},
		{
			name:     "konnect-control-plane-name flag - default",
			kongFile: "testdata/sync/026-konnect-rename/default.yaml",
			flags:    []string{"--konnect-control-plane-name", "default"},
			expectedState: utils.KongRawState{
				Services: defaultCPService,
			},
		},
		{
			name:             "konnect-runtime-group-name flag - test",
			runtimeGroupName: "test",
			kongFile:         "testdata/sync/026-konnect-rename/test.yaml",
			flags:            []string{"--konnect-runtime-group-name", "test"},
			expectedState: utils.KongRawState{
				Services: testCPService,
			},
		},
		{
			name:             "konnect-control-plane-name flag - test",
			controlPlaneName: "test",
			kongFile:         "testdata/sync/026-konnect-rename/test.yaml",
			flags:            []string{"--konnect-control-plane-name", "test"},
			expectedState: utils.KongRawState{
				Services: testCPService,
			},
		},
		{
			name:     "konnect.runtime_group_name - default",
			kongFile: "testdata/sync/026-konnect-rename/konnect_default_rg.yaml",
			expectedState: utils.KongRawState{
				Services: defaultCPService,
			},
		},
		{
			name:     "konnect.control_plane_name - default",
			kongFile: "testdata/sync/026-konnect-rename/konnect_default_cp.yaml",
			expectedState: utils.KongRawState{
				Services: defaultCPService,
			},
		},
		{
			name:             "konnect.runtime_group_name - test",
			runtimeGroupName: "test",
			kongFile:         "testdata/sync/026-konnect-rename/konnect_test_rg.yaml",
			expectedState: utils.KongRawState{
				Services: testCPService,
			},
		},
		{
			name:             "konnect.control_plane_name - test",
			controlPlaneName: "test",
			kongFile:         "testdata/sync/026-konnect-rename/konnect_test_cp.yaml",
			expectedState: utils.KongRawState{
				Services: testCPService,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhenKonnect(t)
			setup(t)
			if tc.controlPlaneName != "" {
				t.Setenv("DECK_KONNECT_CONTROL_PLANE_NAME", tc.controlPlaneName)
				t.Cleanup(func() {
					reset(t, "--konnect-control-plane-name", tc.controlPlaneName)
				})
			} else if tc.runtimeGroupName != "" {
				t.Setenv("DECK_KONNECT_RUNTIME_GROUP_NAME", tc.runtimeGroupName)
				t.Cleanup(func() {
					reset(t, "--konnect-runtime-group-name", tc.runtimeGroupName)
				})
			}
			client, err := getTestClient()
			if err != nil {
				t.Fatalf(err.Error())
			}
			sync(tc.kongFile, tc.flags...)
			testKongState(t, client, true, tc.expectedState, nil)
		})
	}
}

func Test_Sync_KonnectRenameErrors(t *testing.T) {
	tests := []struct {
		name          string
		kongFile      string
		flags         []string
		expectedError error
	}{
		{
			name:     "different runtime group names fail",
			kongFile: "testdata/sync/026-konnect-rename/konnect_default_cp.yaml",
			flags:    []string{"--konnect-runtime-group-name", "rg1"},
			expectedError: errors.New(`warning: control plane 'rg1' specified via ` +
				`--konnect-[control-plane|runtime-group]-name flag is different from 'default' found in state file(s)`),
		},
		{
			name:     "different runtime group names fail",
			kongFile: "testdata/sync/026-konnect-rename/konnect_default_rg.yaml",
			flags:    []string{"--konnect-runtime-group-name", "rg1"},
			expectedError: errors.New(`warning: control plane 'rg1' specified via ` +
				`--konnect-[control-plane|runtime-group]-name flag is different from 'default' found in state file(s)`),
		},
		{
			name:     "different control plane names fail",
			kongFile: "testdata/sync/026-konnect-rename/konnect_default_cp.yaml",
			flags:    []string{"--konnect-control-plane-name", "cp1"},
			expectedError: errors.New(`warning: control plane 'cp1' specified via ` +
				`--konnect-[control-plane|runtime-group]-name flag is different from 'default' found in state file(s)`),
		},
		{
			name:     "different control plane names fail",
			kongFile: "testdata/sync/026-konnect-rename/konnect_default_rg.yaml",
			flags:    []string{"--konnect-control-plane-name", "cp1"},
			expectedError: errors.New(`warning: control plane 'cp1' specified via ` +
				`--konnect-[control-plane|runtime-group]-name flag is different from 'default' found in state file(s)`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := sync(tc.kongFile, tc.flags...)
			assert.Equal(t, err, tc.expectedError)
		})
	}
}

// test scope:
//   - 3.0.0+
func Test_Sync_DoNotUpdateCreatedAt(t *testing.T) {
	runWhen(t, "kong", ">=3.0.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	const (
		oldConfig = "testdata/sync/027-created-at/old.yaml"
		newConfig = "testdata/sync/027-created-at/new.yaml"
	)

	// provision entities
	require.NoError(t, sync(oldConfig))

	// get the current state
	ctx := context.Background()
	oldKongState, err := deckDump.Get(ctx, client, deckDump.Config{})
	if err != nil {
		t.Errorf(err.Error())
	}

	// update entities
	time.Sleep(time.Second)
	require.NoError(t, sync(newConfig))

	// get the new state
	newKongState, err := deckDump.Get(ctx, client, deckDump.Config{})
	if err != nil {
		t.Errorf(err.Error())
	}

	// verify that the created_at have not changed across deployments
	require.Equal(t, oldKongState.Services[0].CreatedAt, newKongState.Services[0].CreatedAt)
	require.Equal(t, oldKongState.Routes[0].CreatedAt, newKongState.Routes[0].CreatedAt)
	require.Equal(t, oldKongState.Plugins[0].CreatedAt, newKongState.Plugins[0].CreatedAt)
	require.Equal(t, oldKongState.Consumers[0].CreatedAt, newKongState.Consumers[0].CreatedAt)

	// verify that the updated_at have changed across deployments
	require.NotEqual(t, oldKongState.Services[0].UpdatedAt, newKongState.Services[0].UpdatedAt)
	require.NotEqual(t, oldKongState.Routes[0].UpdatedAt, newKongState.Routes[0].UpdatedAt)
	// plugins do not have an updated_at field
	// consumers do not have an updated_at field
}

// test scope:
//   - 3.0.0+
//   - konnect
func Test_Sync_ConsumerGroupConsumersWithCustomID(t *testing.T) {
	t.Setenv("DECK_KONNECT_CONTROL_PLANE_NAME", "default")
	runWhenEnterpriseOrKonnect(t, ">=3.0.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	expectedState := utils.KongRawState{
		ConsumerGroups: []*kong.ConsumerGroupObject{
			{
				ConsumerGroup: &kong.ConsumerGroup{
					ID:   kong.String("48df7cd3-1cd0-4e53-af73-8f57f257be18"),
					Name: kong.String("cg1"),
				},
				Consumers: []*kong.Consumer{
					{
						ID:       kong.String("bcb296c3-22bb-46f6-99c8-4828af750b77"),
						CustomID: kong.String("foo"),
					},
				},
			},
			{
				ConsumerGroup: &kong.ConsumerGroup{
					ID:   kong.String("1a81dc83-5329-4666-8ae7-8a966e62d076"),
					Name: kong.String("cg2"),
				},
				Consumers: []*kong.Consumer{
					{
						ID:       kong.String("562bf5c7-a7d9-4338-84dd-2c1064fb7f67"),
						Username: kong.String("foo"),
					},
				},
			},
			{
				ConsumerGroup: &kong.ConsumerGroup{
					ID:   kong.String("d140f9cc-227e-4872-8b0b-639f6922dfb0"),
					Name: kong.String("cg3"),
				},
				Consumers: []*kong.Consumer{
					{
						ID:       kong.String("7906968b-cd89-4a87-8dda-94678e7106b2"),
						Username: kong.String("bar"),
						CustomID: kong.String("custom_bar"),
					},
				},
			},
		},
		Consumers: []*kong.Consumer{
			{
				ID:       kong.String("bcb296c3-22bb-46f6-99c8-4828af750b77"),
				CustomID: kong.String("foo"),
			},
			{
				ID:       kong.String("562bf5c7-a7d9-4338-84dd-2c1064fb7f67"),
				Username: kong.String("foo"),
			},
			{
				ID:       kong.String("7906968b-cd89-4a87-8dda-94678e7106b2"),
				Username: kong.String("bar"),
				CustomID: kong.String("custom_bar"),
			},
		},
	}
	require.NoError(t, sync("testdata/sync/028-consumer-group-consumers-custom_id/kong.yaml"))
	testKongState(t, client, false, expectedState, nil)
}

// test scope:
//   - 3.5.0+
//   - konnect
func Test_Sync_PluginScopedToConsumerGroupAndRoute(t *testing.T) {
	t.Setenv("DECK_KONNECT_CONTROL_PLANE_NAME", "default")
	runWhenEnterpriseOrKonnect(t, ">=3.5.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	expectedState := utils.KongRawState{
		ConsumerGroups: []*kong.ConsumerGroupObject{
			{
				ConsumerGroup: &kong.ConsumerGroup{
					ID:   kong.String("48df7cd3-1cd0-4e53-af73-8f57f257be18"),
					Name: kong.String("cg1"),
				},
				Consumers: []*kong.Consumer{
					{
						ID:       kong.String("bcb296c3-22bb-46f6-99c8-4828af750b77"),
						Username: kong.String("foo"),
					},
				},
			},
		},
		Consumers: []*kong.Consumer{
			{
				ID:       kong.String("bcb296c3-22bb-46f6-99c8-4828af750b77"),
				Username: kong.String("foo"),
			},
		},
		Services: []*kong.Service{
			{
				ID:             kong.String("1b9d6d8e-9f0f-4a1a-8d5c-9d2a6b2b7f3c"),
				Host:           kong.String("example.com"),
				Name:           kong.String("s1"),
				ConnectTimeout: kong.Int(60000),
				Port:           kong.Int(80),
				Path:           nil,
				Protocol:       kong.String("http"),
				ReadTimeout:    kong.Int(60000),
				Retries:        kong.Int(5),
				WriteTimeout:   kong.Int(60000),
				Tags:           nil,
				Enabled:        kong.Bool(true),
			},
		},
		Routes: []*kong.Route{
			{
				Name:                    kong.String("r1"),
				ID:                      kong.String("a9730e9e-df7e-4042-8bc7-e8b99af70171"),
				Hosts:                   kong.StringSlice("10.*"),
				PathHandling:            kong.String("v0"),
				PreserveHost:            kong.Bool(false),
				Protocols:               []*string{kong.String("http"), kong.String("https")},
				RegexPriority:           kong.Int(0),
				StripPath:               kong.Bool(true),
				HTTPSRedirectStatusCode: kong.Int(426),
				RequestBuffering:        kong.Bool(true),
				ResponseBuffering:       kong.Bool(true),
				Service: &kong.Service{
					ID: kong.String("1b9d6d8e-9f0f-4a1a-8d5c-9d2a6b2b7f3c"),
				},
			},
		},
		Plugins: []*kong.Plugin{
			{
				ID:   kong.String("a0b4c8d9-0f1e-4e1f-9e3a-5c8e1c8b9f1a"),
				Name: kong.String("rate-limiting-advanced"),
				ConsumerGroup: &kong.ConsumerGroup{
					ID: kong.String("48df7cd3-1cd0-4e53-af73-8f57f257be18"),
				},
				Route: &kong.Route{
					ID: kong.String("a9730e9e-df7e-4042-8bc7-e8b99af70171"),
				},
				Config: kong.Configuration{
					"consumer_groups":         nil,
					"dictionary_name":         string("kong_rate_limiting_counters"),
					"disable_penalty":         bool(false),
					"enforce_consumer_groups": bool(false),
					"error_code":              float64(429),
					"error_message":           string("API rate limit exceeded"),
					"header_name":             nil,
					"hide_client_headers":     bool(false),
					"identifier":              string("consumer"),
					"limit":                   []any{float64(1)},
					"namespace":               string("dmHiQjaGTIYimSXQmRoUDA1XkJXZqxZf"),
					"path":                    nil,
					"redis": map[string]any{
						"cluster_addresses":   nil,
						"connect_timeout":     nil,
						"database":            float64(0),
						"host":                nil,
						"keepalive_backlog":   nil,
						"keepalive_pool_size": float64(256),
						"password":            nil,
						"port":                nil,
						"read_timeout":        nil,
						"send_timeout":        nil,
						"sentinel_addresses":  nil,
						"sentinel_master":     nil,
						"sentinel_password":   nil,
						"sentinel_role":       nil,
						"sentinel_username":   nil,
						"server_name":         nil,
						"ssl":                 false,
						"ssl_verify":          false,
						"timeout":             float64(2000),
						"username":            nil,
					},
					"retry_after_jitter_max": float64(0),
					"strategy":               string("local"),
					"sync_rate":              float64(-1),
					"window_size":            []any{float64(60)},
					"window_type":            string("sliding"),
				},
				Enabled:   kong.Bool(true),
				Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
			},
		},
	}
	require.NoError(t, sync("testdata/sync/029-plugin-scoped-to-cg-route/kong.yaml"))
	testKongState(t, client, false, expectedState, nil)

	// create a temporary file to dump the state.
	cwd, err := os.Getwd()
	require.NoError(t, err)
	file, err := os.CreateTemp(cwd, "dump.*.yaml")
	assert.NoError(t, err)

	// dump the state.
	_, err = dump("-o", file.Name(), "--yes")
	assert.NoError(t, err)

	// verify that the dumped state can be sync'd back and that
	// the end result is the same.
	require.NoError(t, sync(file.Name()))
	testKongState(t, client, false, expectedState, nil)
}

// test scope:
//   - 3.5.0+
//   - konnect
func Test_Sync_DeDupPluginsScopedToConsumerGroups(t *testing.T) {
	t.Setenv("DECK_KONNECT_CONTROL_PLANE_NAME", "default")
	runWhenEnterpriseOrKonnect(t, ">=3.5.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	expectedState := utils.KongRawState{
		ConsumerGroups: []*kong.ConsumerGroupObject{
			{
				ConsumerGroup: &kong.ConsumerGroup{
					ID:   kong.String("19275493-84d3-4c64-92e6-612e908a3a4f"),
					Name: kong.String("gold"),
				},
				Consumers: []*kong.Consumer{
					{
						ID:       kong.String("7b2c743c-2cec-4998-b9df-e7f8a9a20487"),
						Username: kong.String("jeff"),
					},
				},
			},
			{
				ConsumerGroup: &kong.ConsumerGroup{
					ID:   kong.String("48df7cd3-1cd0-4e53-af73-8f57f257be18"),
					Name: kong.String("silver"),
				},
			},
		},
		Consumers: []*kong.Consumer{
			{
				ID:       kong.String("7b2c743c-2cec-4998-b9df-e7f8a9a20487"),
				Username: kong.String("jeff"),
			},
		},
		Plugins: []*kong.Plugin{
			{
				ID:   kong.String("1c93dd1f-f188-473d-bec8-053bd526a693"),
				Name: kong.String("rate-limiting-advanced"),
				ConsumerGroup: &kong.ConsumerGroup{
					ID: kong.String("19275493-84d3-4c64-92e6-612e908a3a4f"),
				},
				Config: kong.Configuration{
					"consumer_groups":         nil,
					"dictionary_name":         string("kong_rate_limiting_counters"),
					"disable_penalty":         bool(false),
					"enforce_consumer_groups": bool(false),
					"error_code":              float64(429),
					"error_message":           string("API rate limit exceeded"),
					"header_name":             nil,
					"hide_client_headers":     bool(false),
					"identifier":              string("consumer"),
					"limit":                   []any{float64(1000)},
					"namespace":               string("OsFDaDQxdb1MFGHBdZENho51f3zqMLy"),
					"path":                    nil,
					"redis": map[string]any{
						"cluster_addresses":   nil,
						"connect_timeout":     nil,
						"database":            float64(0),
						"host":                nil,
						"keepalive_backlog":   nil,
						"keepalive_pool_size": float64(256),
						"password":            nil,
						"port":                nil,
						"read_timeout":        nil,
						"send_timeout":        nil,
						"sentinel_addresses":  nil,
						"sentinel_master":     nil,
						"sentinel_password":   nil,
						"sentinel_role":       nil,
						"sentinel_username":   nil,
						"server_name":         nil,
						"ssl":                 false,
						"ssl_verify":          false,
						"timeout":             float64(2000),
						"username":            nil,
					},
					"retry_after_jitter_max": float64(0),
					"strategy":               string("local"),
					"sync_rate":              float64(-1),
					"window_size":            []any{float64(60)},
					"window_type":            string("sliding"),
				},
				Enabled:   kong.Bool(true),
				Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
			},
			{
				ID:   kong.String("bcb296c3-22bb-46f6-99c8-4828af750b77"),
				Name: kong.String("rate-limiting-advanced"),
				ConsumerGroup: &kong.ConsumerGroup{
					ID: kong.String("48df7cd3-1cd0-4e53-af73-8f57f257be18"),
				},
				Config: kong.Configuration{
					"consumer_groups":         nil,
					"dictionary_name":         string("kong_rate_limiting_counters"),
					"disable_penalty":         bool(false),
					"enforce_consumer_groups": bool(false),
					"error_code":              float64(429),
					"error_message":           string("API rate limit exceeded"),
					"header_name":             nil,
					"hide_client_headers":     bool(false),
					"identifier":              string("consumer"),
					"limit":                   []any{float64(100)},
					"namespace":               string("OsFDaDQxdb1MFGHBdZENho51f3zqMLy"),
					"path":                    nil,
					"redis": map[string]any{
						"cluster_addresses":   nil,
						"connect_timeout":     nil,
						"database":            float64(0),
						"host":                nil,
						"keepalive_backlog":   nil,
						"keepalive_pool_size": float64(256),
						"password":            nil,
						"port":                nil,
						"read_timeout":        nil,
						"send_timeout":        nil,
						"sentinel_addresses":  nil,
						"sentinel_master":     nil,
						"sentinel_password":   nil,
						"sentinel_role":       nil,
						"sentinel_username":   nil,
						"server_name":         nil,
						"ssl":                 false,
						"ssl_verify":          false,
						"timeout":             float64(2000),
						"username":            nil,
					},
					"retry_after_jitter_max": float64(0),
					"strategy":               string("local"),
					"sync_rate":              float64(-1),
					"window_size":            []any{float64(60)},
					"window_type":            string("sliding"),
				},
				Enabled:   kong.Bool(true),
				Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
			},
		},
	}
	require.NoError(t, sync("testdata/sync/030-plugin-dedup-consumer-groups/kong.yaml"))
	testKongState(t, client, false, expectedState, nil)
}

// test scope:
//   - 3.5.0+
//   - konnect
func Test_Sync_ConsumerGroupConsumerFromUpstream(t *testing.T) {
	t.Setenv("DECK_KONNECT_CONTROL_PLANE_NAME", "default")
	runWhenEnterpriseOrKonnect(t, ">=3.4.0")
	setup(t)

	client, err := getTestClient()
	if err != nil {
		t.Fatalf(err.Error())
	}

	expectedState := utils.KongRawState{
		ConsumerGroups: []*kong.ConsumerGroupObject{
			{
				ConsumerGroup: &kong.ConsumerGroup{
					ID:   kong.String("c0f6c818-470c-4df7-8515-c8e904765fcc"),
					Name: kong.String("group-1"),
					Tags: kong.StringSlice("project:the-project", "managed-by:deck"),
				},
				Consumers: []*kong.Consumer{
					{
						ID:       kong.String("97cab250-1b0a-4119-aa2e-0756e8931034"),
						Username: kong.String("consumer-1"),
						Tags:     kong.StringSlice("project:the-project", "managed-by:the-background-process"),
					},
				},
			},
		},
		Consumers: []*kong.Consumer{
			{
				ID:       kong.String("97cab250-1b0a-4119-aa2e-0756e8931034"),
				Username: kong.String("consumer-1"),
				Tags:     kong.StringSlice("project:the-project", "managed-by:the-background-process"),
			},
		},
	}

	// simulate the following scenario:
	// - a consumer-group defined with a set of tags, ideally managed by decK
	// - a consumer defined with another set of tags, ideally managed by an external process
	// - the consumer -> consumer-group relationship, ideally managed by an external process
	require.NoError(t, sync("testdata/sync/031-consumer-group-consumers-from-upstream/initial.yaml"))
	testKongState(t, client, false, expectedState, nil)

	// referencing the relationship in a file without the consumer would still work
	// if default_lookup_tags are defined to pull consumers from upstream.
	require.NoError(t, sync("testdata/sync/031-consumer-group-consumers-from-upstream/consumer-groups.yaml"))
	testKongState(t, client, false, expectedState, nil)
}

// test scope:
// - Enterprise & Non-Konnect
func TestSync_License(t *testing.T) {
	t.Setenv("DECK_KONNECT_CONTROL_PLANE_NAME", "default")
	runWhen(t, "enterprise", ">=3.0.0")
	kongLicensePayload := os.Getenv("KONG_LICENSE_DATA")
	if kongLicensePayload == "" {
		t.Skip("Skipping because environment variable KONG_LICENSE_DATA not found")
	}
	setup(t)

	buf, err := os.ReadFile("testdata/sync/032-licenses/config-with-license.yaml")
	require.NoError(t, err)
	fileContent := strings.ReplaceAll(string(buf), "__KONG_LICENSE_DATA__", fmt.Sprintf("'%s'", kongLicensePayload))
	configFile, err := os.CreateTemp("/tmp", "kong-license-test")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	os.WriteFile(configFile.Name(), []byte(fileContent), os.ModeTemporary)
	client, err := getTestClient()
	ctx := context.Background()

	t.Run("create_license_and_dump_results", func(t *testing.T) {
		currentState, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: true})
		require.NoError(t, err)

		targetState := stateFromFile(ctx, t, configFile.Name(), client, deckDump.Config{
			IncludeLicenses: true,
		})
		syncer, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
			CurrentState: currentState,
			TargetState:  targetState,

			KongClient:      client,
			IncludeLicenses: true,
		})
		require.NoError(t, err)

		require.NoError(t, err, "Should get test client")
		stats, errs, changes := syncer.Solve(ctx, 1, false, true)
		require.Len(t, errs, 0, "Should have no errors in syncing")
		logEntityChanges(t, stats, changes)

		newState, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: true})
		require.NoError(t, err)

		licenses, err := newState.Licenses.GetAll()
		require.NoError(t, err)
		// Avoid dumping of `licenses` to leak sensitive content.
		require.Equal(t, 1, len(licenses))
		// Compare hashes to avoid content of licenses to be leaked.
		expectedLicenseHash := sha1.Sum([]byte(kongLicensePayload))
		actualLicenseHash := sha1.Sum([]byte(*licenses[0].Payload))
		require.Equal(t, expectedLicenseHash, actualLicenseHash, "Hash of license payload should be the same as env KONG_LICENSE_DATA")
	})

	t.Run("dump_with_includeLicense_disabled", func(t *testing.T) {
		stateWithoutLicenses, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: false})
		require.NoError(t, err)
		licenses, err := stateWithoutLicenses.Licenses.GetAll()
		require.NoError(t, err)
		require.Equal(t, 0, len(licenses))
	})

	t.Run("sync_with_includeLicenses_false", func(t *testing.T) {
		currentState, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: true})
		require.NoError(t, err)
		stateWithoutLicense := stateFromFile(ctx, t,
			"testdata/sync/032-licenses/config-without-license.yaml",
			client,
			deckDump.Config{IncludeLicenses: true},
		)
		syncer, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
			CurrentState: currentState,
			TargetState:  stateWithoutLicense,

			KongClient:      client,
			IncludeLicenses: false,
		})
		require.NoError(t, err)

		stats, errs, changes := syncer.Solve(ctx, 1, false, true)
		require.Len(t, errs, 0, "Should have no errors in syncing")
		logEntityChanges(t, stats, changes)

		newState, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: true})
		require.NoError(t, err)
		licenses, err := newState.Licenses.GetAll()
		require.NoError(t, err)
		require.Len(t, licenses, 1)
	})

	t.Run("delete_existing_license", func(t *testing.T) {
		currentState, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: true})
		require.NoError(t, err)
		stateWithoutLicense := stateFromFile(ctx, t,
			"testdata/sync/032-licenses/config-without-license.yaml",
			client,
			deckDump.Config{IncludeLicenses: true},
		)

		syncer, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
			CurrentState: currentState,
			TargetState:  stateWithoutLicense,

			KongClient:      client,
			IncludeLicenses: true,
		})
		require.NoError(t, err)
		stats, errs, changes := syncer.Solve(ctx, 1, false, true)
		require.Empty(t, errs, "Should have no errors in syncing")
		logEntityChanges(t, stats, changes)

		newState, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: true})
		require.NoError(t, err)
		licenses, err := newState.Licenses.GetAll()
		require.NoError(t, err)
		require.Empty(t, licenses)
	})
}

func Test_Sync_PluginDoNotFillDefaults(t *testing.T) {

	client, err := getTestClient()

	require.NoError(t, err)
	ctx := context.Background()
	t.Run("empty_fields_of_plugin_config", func(t *testing.T) {
		mustResetKongState(ctx, t, client, deckDump.Config{})

		currrentState, err := fetchCurrentState(ctx, client, deckDump.Config{})
		require.NoError(t, err)
		targetState := stateFromFile(ctx, t,
			"testdata/sync/033-plugin-with-empty-fields/kong.yaml",
			client,
			deckDump.Config{},
		)

		kongURL, err := url.Parse(client.BaseRootURL())
		require.NoError(t, err)
		p := NewRecordRequestProxy(kongURL)
		s := httptest.NewServer(p)
		c, err := utils.GetKongClient(utils.KongClientConfig{
			Address: s.URL,
		})
		require.NoError(t, err)

		syncer, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
			CurrentState: currrentState,
			TargetState:  targetState,

			KongClient: c,
		})
		stats, errs, changes := syncer.Solve(ctx, 1, false, true)
		require.Empty(t, errs, "Should have no errors in syncing")
		require.NoError(t, err)

		require.Equal(t, int32(1), stats.CreateOps.Count(), "Should create 1 entity")
		require.Len(t, changes.Creating, 1, "Should have 1 creating record in changes")

		// The change records which are returned in `diff` command should fill default values.
		t.Run("should fill default values in change records", func(t *testing.T) {
			body, ok := changes.Creating[0].Body.(map[string]any)
			require.True(t, ok)
			plugin, ok := body["new"].(*state.Plugin)
			require.True(t, ok)

			path, ok := plugin.Config["path"]
			require.True(t, ok)
			require.Equal(t, "/tmp/file.log", path, "path should be same as specified in file")

			reopen, ok := plugin.Config["reopen"]
			require.True(t, ok, "'reopen' field should be filled")
			require.Equal(t, false, reopen, "should be the same as default value")

			custom_fields_by_lua, ok := plugin.Config["custom_fields_by_lua"]
			require.True(t, ok, "'custom_fields_by_lua' field should be filled")
			require.Nil(t, custom_fields_by_lua, "should be an explicit nil")
		})

		// But the default values should not be filled in request sent to Kong.
		t.Run("should not fill default values in requests sent to Kong", func(t *testing.T) {
			reqs := p.dumpRequests()
			req, found := lo.Find(reqs, func(r *http.Request) bool {
				return r.Method == "PUT" && strings.Contains(r.URL.Path, "/plugins")
			})
			require.True(t, found, "Should find request to create plugin")
			buf, err := io.ReadAll(req.Body)
			require.NoError(t, err, "Should read request body from record")
			plugin := state.Plugin{}
			err = json.Unmarshal(buf, &plugin)
			require.NoError(t, err, "Should unmarshal request body to plugin type")

			path, ok := plugin.Config["path"]
			require.True(t, ok)
			require.Equal(t, "/tmp/file.log", path, "path should be same as specified in file")

			_, ok = plugin.Config["reopen"]
			require.False(t, ok, "'reopen' field should not be filled")

			_, ok = plugin.Config["custom_fields_by_lua"]
			require.False(t, ok, "'custom_fields_by_lua' field should not be filled")
		})

		// Should update Kong state successfully.
		t.Run("Should get the plugin config from update Kong", func(t *testing.T) {
			newState, err := fetchCurrentState(ctx, client, deckDump.Config{})
			require.NoError(t, err)
			plugins, err := newState.Plugins.GetAll()
			require.NoError(t, err)
			require.Len(t, plugins, 1)
			plugin := plugins[0]
			require.Equal(t, "file-log", *plugin.Name)
			path, ok := plugin.Config["path"]
			require.True(t, ok)
			require.Equal(t, "/tmp/file.log", path, "path should be same as specified in file")
		})
	})
}

func Test_Sync_PluginAutoFields(t *testing.T) {
	client, err := getTestClient()

	require.NoError(t, err)
	ctx := context.Background()
	t.Run("plugin_with_auto_fields", func(t *testing.T) {
		mustResetKongState(ctx, t, client, deckDump.Config{})

		currentState, err := fetchCurrentState(ctx, client, deckDump.Config{})
		require.NoError(t, err)
		targetState := stateFromFile(ctx, t,
			"testdata/sync/034-fill-auto-oauth2/kong.yaml",
			client,
			deckDump.Config{},
		)

		kongURL, err := url.Parse(client.BaseRootURL())
		require.NoError(t, err)
		p := NewRecordRequestProxy(kongURL)
		s := httptest.NewServer(p)
		c, err := utils.GetKongClient(utils.KongClientConfig{
			Address: s.URL,
		})
		require.NoError(t, err)

		syncer, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
			CurrentState: currentState,
			TargetState:  targetState,

			KongClient: c,
		})
		_, errs, _ := syncer.Solve(ctx, 1, false, true)

		require.NotNil(t, errs)
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), "provision_key: required field missing",
			"Should error out due to missing provision_key")
	})
}

// test scope:
// - enterprise
// - >=3.4.0
func Test_Sync_MoreThanOneConsumerGroupForOneConsumer(t *testing.T) {
	runWhen(t, "enterprise", ">=3.4.0")
	setup(t)

	client, err := getTestClient()
	require.NoError(t, err)

	expectedState := utils.KongRawState{
		ConsumerGroups: []*kong.ConsumerGroupObject{
			{
				ConsumerGroup: &kong.ConsumerGroup{
					Name: kong.String("group1"),
				},
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("my-test-consumer"),
					},
				},
			},
			{
				ConsumerGroup: &kong.ConsumerGroup{
					Name: kong.String("group2"),
				},
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("my-test-consumer"),
					},
				},
			},
		},
		Consumers: []*kong.Consumer{
			{
				Username: kong.String("my-test-consumer"),
			},
		},
	}
	require.NoError(t, sync("testdata/sync/xxx-more-than-one-consumer-group-with-a-consumer/kong3x.yaml"))
	testKongState(t, client, false, expectedState, nil)
}

// test scope:
// - enterprise
// - 2.8.0
func Test_Sync_MoreThanOneConsumerGroupForOneConsumer_2_8(t *testing.T) {
	runWhen(t, "enterprise", ">=2.8.0 <3.0.0")
	setup(t)

	client, err := getTestClient()
	require.NoError(t, err)

	expectedState := utils.KongRawState{
		ConsumerGroups: []*kong.ConsumerGroupObject{
			{
				ConsumerGroup: &kong.ConsumerGroup{
					Name: kong.String("group1"),
				},
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("my-test-consumer"),
					},
				},
			},
			{
				ConsumerGroup: &kong.ConsumerGroup{
					Name: kong.String("group2"),
				},
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("my-test-consumer"),
					},
				},
			},
		},
		Consumers: []*kong.Consumer{
			{
				Username: kong.String("my-test-consumer"),
			},
		},
	}
	require.NoError(t, sync("testdata/sync/xxx-more-than-one-consumer-group-with-a-consumer/kong.yaml"))
	testKongState(t, client, false, expectedState, nil)
}

func Test_Sync_PluginDeprecatedFields36x(t *testing.T) {
	runWhen(t, "kong", ">=3.6.0 <3.8.0")

	client, err := getTestClient()
	require.NoError(t, err)

	ctx := context.Background()
	mustResetKongState(ctx, t, client, deckDump.Config{})

	rateLimitingConfigurationInitial := DefaultConfigFactory.RateLimitingConfiguration()
	expectedInitialState := utils.KongRawState{
		Plugins: []*kong.Plugin{
			DefaultConfigFactory.Plugin(
				"2705d985-de4b-4ca8-87fd-2b361e30a3e7", "rate-limiting", rateLimitingConfigurationInitial,
			),
		},
	}

	rateLimitingConfigurationUpdatedOldFields := rateLimitingConfigurationInitial.DeepCopy()
	rateLimitingConfigurationUpdatedOldFields["redis_host"] = string("localhost-2")
	rateLimitingConfigurationUpdatedOldFields["redis_database"] = float64(2)
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["host"] = string("localhost-2")
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["database"] = float64(2)

	expectedStateAfterChangeUsingOldFields := utils.KongRawState{
		Plugins: []*kong.Plugin{
			{
				ID:        kong.String("2705d985-de4b-4ca8-87fd-2b361e30a3e7"),
				Name:      kong.String("rate-limiting"),
				Enabled:   kong.Bool(true),
				Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
				Config:    rateLimitingConfigurationUpdatedOldFields,
			},
		},
	}

	rateLimitingConfigurationUpdatedNewFields := rateLimitingConfigurationInitial.DeepCopy()
	rateLimitingConfigurationUpdatedNewFields["redis_host"] = string("localhost-3")
	rateLimitingConfigurationUpdatedNewFields["redis_database"] = float64(3)
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["host"] = string("localhost-3")
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["database"] = float64(3)

	expectedStateAfterChangeUsingNewFields := utils.KongRawState{
		Plugins: []*kong.Plugin{
			{
				ID:        kong.String("2705d985-de4b-4ca8-87fd-2b361e30a3e7"),
				Name:      kong.String("rate-limiting"),
				Enabled:   kong.Bool(true),
				Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
				Config:    rateLimitingConfigurationUpdatedNewFields,
			},
		},
	}

	tests := []struct {
		name          string
		stateFile     string
		expectedState utils.KongRawState
	}{
		{
			name:          "initial sync",
			stateFile:     "testdata/sync/035-deprecated-fields/kong-ce/kong-initial.yaml",
			expectedState: expectedInitialState,
		},
		{
			name:          "syncing but not update - using only old (deprecated) fields",
			stateFile:     "testdata/sync/035-deprecated-fields/kong-ce/kong-no-change-old-fields.yaml",
			expectedState: expectedInitialState,
		},
		{
			name:          "syncing but not update - using only new (not deprecated) fields",
			stateFile:     "testdata/sync/035-deprecated-fields/kong-ce/kong-no-change-new-fields.yaml",
			expectedState: expectedInitialState,
		},
		{
			name:          "syncing but with update - using only old (deprecated) fields",
			stateFile:     "testdata/sync/035-deprecated-fields/kong-ce/kong-update-old-fields.yaml",
			expectedState: expectedStateAfterChangeUsingOldFields,
		},
		{
			name:          "syncing but with update - using only new (not deprecated) fields",
			stateFile:     "testdata/sync/035-deprecated-fields/kong-ce/kong-update-new-fields.yaml",
			expectedState: expectedStateAfterChangeUsingNewFields,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// initialize state
			require.NoError(t, sync(tc.stateFile))

			// test
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_PluginDeprecatedFields38x(t *testing.T) {
	runWhen(t, "kong", ">=3.8.0")

	// Setup RateLimitingAdvanced ==============================
	rateLimitingAdvancedConfigurationInitial := DefaultConfigFactory.RateLimitingAdvancedConfiguration()
	rateLimitingAdvancedConfigurationInitial["sync_rate"] = float64(10)
	rateLimitingAdvancedConfigurationInitial["redis"].(map[string]interface{})["cluster_addresses"] =
		[]any{string("127.0.1.0:6379"), string("127.0.1.0:6380"), string("127.0.1.0:6381")}
	rateLimitingAdvancedConfigurationInitial["redis"].(map[string]interface{})["cluster_nodes"] = []any{
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6379)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6380)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6381)},
	}
	rateLimitingAdvancedConfigurationInitial["redis"].(map[string]interface{})["timeout"] = float64(2000)
	rateLimitingAdvancedConfigurationInitial["redis"].(map[string]interface{})["connect_timeout"] = float64(2000)
	rateLimitingAdvancedConfigurationInitial["redis"].(map[string]interface{})["read_timeout"] = float64(2000)
	rateLimitingAdvancedConfigurationInitial["redis"].(map[string]interface{})["send_timeout"] = float64(2000)
	rateLimitingAdvancedConfigurationInitial["redis"].(map[string]interface{})["sentinel_addresses"] =
		[]any{string("127.0.2.0:6379"), string("127.0.2.0:6380"), string("127.0.2.0:6381")}
	rateLimitingAdvancedConfigurationInitial["redis"].(map[string]interface{})["sentinel_nodes"] = []any{
		map[string]any{"host": string("127.0.2.0"), "port": float64(6379)},
		map[string]any{"host": string("127.0.2.0"), "port": float64(6380)},
		map[string]any{"host": string("127.0.2.0"), "port": float64(6381)},
	}

	// Setup OpenIdConnect ==============================
	openidConnectConfigurationInitial := DefaultConfigFactory.OpenIDConnectConfiguration()
	openidConnectConfigurationInitial["redis"].(map[string]interface{})["cluster_max_redirections"] = nil
	openidConnectConfigurationInitial["session_redis_cluster_max_redirections"] = nil
	openidConnectConfigurationInitial["redis"].(map[string]interface{})["cluster_addresses"] = nil
	openidConnectConfigurationInitial["redis"].(map[string]interface{})["cluster_nodes"] = nil
	openidConnectConfigurationInitial["session_redis_cluster_nodes"] = nil

	// Initial State
	expectedInitialState := utils.KongRawState{
		Services: []*kong.Service{
			DefaultConfigFactory.Service("9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d", "mockbin.org", "svc1"),
		},
		Plugins: []*kong.Plugin{
			DefaultConfigFactory.Plugin(
				"a1368a28-cb5c-4eee-86d8-03a6bdf94b5e", "rate-limiting-advanced", rateLimitingAdvancedConfigurationInitial,
			),
			DefaultConfigFactory.Plugin(
				"777496e1-8b35-4512-ad30-51f9fe5d3147", "openid-connect", openidConnectConfigurationInitial,
			),
		},
	}

	rateLimitingConfigurationUpdatedOldFields := rateLimitingAdvancedConfigurationInitial.DeepCopy()
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["cluster_addresses"] =
		[]any{string("127.0.1.0:7379"), string("127.0.1.0:7380"), string("127.0.1.0:7381")}
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["cluster_nodes"] = []any{
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7379)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7380)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7381)},
	}
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["timeout"] = float64(2007)
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["connect_timeout"] = float64(2007)
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["read_timeout"] = float64(2007)
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["send_timeout"] = float64(2007)
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["sentinel_addresses"] =
		[]any{string("127.0.2.0:8379"), string("127.0.2.0:8380"), string("127.0.2.0:8381")}
	rateLimitingConfigurationUpdatedOldFields["redis"].(map[string]interface{})["sentinel_nodes"] = []any{
		map[string]any{"host": string("127.0.2.0"), "port": float64(8379)},
		map[string]any{"host": string("127.0.2.0"), "port": float64(8380)},
		map[string]any{"host": string("127.0.2.0"), "port": float64(8381)},
	}
	rateLimitingConfigurationUpdatedOldFields["sync_rate"] = float64(11)

	openidConnectConfigurationUpdatedOldFields := openidConnectConfigurationInitial.DeepCopy()
	openidConnectConfigurationUpdatedOldFields["redis"].(map[string]interface{})["cluster_max_redirections"] = float64(7)
	openidConnectConfigurationUpdatedOldFields["redis"].(map[string]interface{})["cluster_addresses"] =
		[]any{string("127.0.1.0:6379"), string("127.0.1.0:6380"), string("127.0.1.0:6381")}
	openidConnectConfigurationUpdatedOldFields["redis"].(map[string]interface{})["cluster_nodes"] = []any{
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6379)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6380)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6381)},
	}
	openidConnectConfigurationUpdatedOldFields["session_redis_cluster_max_redirections"] = float64(7)
	openidConnectConfigurationUpdatedOldFields["session_redis_cluster_nodes"] = []any{
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6379)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6380)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(6381)},
	}

	expectedStateAfterChangeUsingOldFields := utils.KongRawState{
		Services: []*kong.Service{
			DefaultConfigFactory.Service("9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d", "mockbin.org", "svc1"),
		},
		Plugins: []*kong.Plugin{
			DefaultConfigFactory.Plugin(
				"a1368a28-cb5c-4eee-86d8-03a6bdf94b5e", "rate-limiting-advanced", rateLimitingConfigurationUpdatedOldFields,
			),
			DefaultConfigFactory.Plugin(
				"777496e1-8b35-4512-ad30-51f9fe5d3147", "openid-connect", openidConnectConfigurationUpdatedOldFields,
			),
		},
	}

	rateLimitingConfigurationUpdatedNewFields := rateLimitingAdvancedConfigurationInitial.DeepCopy()
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["cluster_addresses"] =
		[]any{string("127.0.1.0:7379"), string("127.0.1.0:7380"), string("127.0.1.0:7381")}
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["cluster_nodes"] = []any{
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7379)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7380)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7381)},
	}
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["timeout"] = float64(2005)
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["connect_timeout"] = float64(2005)
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["read_timeout"] = float64(2006)
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["send_timeout"] = float64(2007)
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["sentinel_addresses"] =
		[]any{string("127.0.2.0:8379"), string("127.0.2.0:8380"), string("127.0.2.0:8381")}
	rateLimitingConfigurationUpdatedNewFields["redis"].(map[string]interface{})["sentinel_nodes"] = []any{
		map[string]any{"host": string("127.0.2.0"), "port": float64(8379)},
		map[string]any{"host": string("127.0.2.0"), "port": float64(8380)},
		map[string]any{"host": string("127.0.2.0"), "port": float64(8381)},
	}
	rateLimitingConfigurationUpdatedNewFields["sync_rate"] = float64(11)

	openidConnectConfigurationUpdatedNewFields := openidConnectConfigurationInitial.DeepCopy()
	openidConnectConfigurationUpdatedNewFields["redis"].(map[string]interface{})["cluster_max_redirections"] = float64(11)
	openidConnectConfigurationUpdatedNewFields["session_redis_cluster_max_redirections"] = float64(11)
	openidConnectConfigurationUpdatedNewFields["redis"].(map[string]interface{})["cluster_addresses"] =
		[]any{string("127.0.1.0:7379"), string("127.0.1.0:7380"), string("127.0.1.0:7381")}
	openidConnectConfigurationUpdatedNewFields["redis"].(map[string]interface{})["cluster_nodes"] = []any{
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7379)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7380)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7381)},
	}
	openidConnectConfigurationUpdatedNewFields["session_redis_cluster_nodes"] = []any{
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7379)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7380)},
		map[string]any{"ip": string("127.0.1.0"), "port": float64(7381)},
	}

	expectedStateAfterChangeUsingNewFields := utils.KongRawState{
		Services: []*kong.Service{
			DefaultConfigFactory.Service("9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d", "mockbin.org", "svc1"),
		},
		Plugins: []*kong.Plugin{
			DefaultConfigFactory.Plugin(
				"a1368a28-cb5c-4eee-86d8-03a6bdf94b5e", "rate-limiting-advanced", rateLimitingConfigurationUpdatedNewFields,
			),
			DefaultConfigFactory.Plugin(
				"777496e1-8b35-4512-ad30-51f9fe5d3147", "openid-connect", openidConnectConfigurationUpdatedNewFields,
			),
		},
	}

	client, err := getTestClient()
	require.NoError(t, err)
	ctx := context.Background()

	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
	}{
		{
			name:             "initial sync",
			initialStateFile: "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-initial.yaml",
			stateFile:        "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-initial.yaml",
			expectedState:    expectedInitialState,
		},
		{
			name:             "syncing but not update - using only old (deprecated) fields",
			initialStateFile: "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-initial.yaml",
			stateFile:        "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-no-change-old-fields.yaml",
			expectedState:    expectedInitialState,
		},
		{
			name:             "syncing but not update - using only new (not deprecated) fields",
			initialStateFile: "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-initial.yaml",
			stateFile:        "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-no-change-new-fields.yaml",
			expectedState:    expectedInitialState,
		},
		{
			name:             "syncing but with update - using only old (deprecated) fields",
			initialStateFile: "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-initial.yaml",
			stateFile:        "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-update-old-fields.yaml",
			expectedState:    expectedStateAfterChangeUsingOldFields,
		},
		{
			name:             "syncing but with update - using only new (not deprecated) fields",
			initialStateFile: "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-initial.yaml",
			stateFile:        "testdata/sync/035-deprecated-fields/kong-ee/kong-ee-update-new-fields.yaml",
			expectedState:    expectedStateAfterChangeUsingNewFields,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// initialize state
			mustResetKongState(ctx, t, client, deckDump.Config{})
			require.NoError(t, sync(tc.initialStateFile))

			// make tested changes
			require.NoError(t, sync(tc.stateFile))

			// test
			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Sync_Scoped_Plugins_3x(t *testing.T) {
	runWhen(t, "enterprise", ">=3.0.0")
	setup(t)

	//client, err := getTestClient()
	// require.NoError(t, err)

	tests := []struct {
		name          string
		file          string
		errorExpected string
	}{
		{
			name:          "syncing route-scoped plugin with service field set",
			file:          "testdata/sync/036-scoped-plugins-validation/route-plugins.yaml",
			errorExpected: "building state: nesting service (example-service) under route-scoped plugin (request-transformer) is not allowed",
		},
		{
			name:          "syncing service-scoped plugin with route and consumer field set",
			file:          "testdata/sync/036-scoped-plugins-validation/service-plugins.yaml",
			errorExpected: "building state: nesting consumer (foo) under service-scoped plugin plugin (request-transformer) is not allowed\nnesting route (example-route) under service-scoped plugin (request-transformer) is not allowed",
		},
		{
			name:          "syncing consumer-scoped plugin with service and route field set",
			file:          "testdata/sync/036-scoped-plugins-validation/consumer-plugins.yaml",
			errorExpected: "building state: nesting route (example-route) under consumer-scoped plugin (request-transformer) is not allowed\nnesting service (example-service) under consumer-scoped plugin (request-transformer) is not allowed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := sync(tc.file)
			require.Equal(t, tc.errorExpected, err.Error())
		})
	}
}
