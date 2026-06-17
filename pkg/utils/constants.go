package utils

import (
	"github.com/kong/go-kong/kong"
)

const (
	defaultTimeout     = 60000
	defaultSlots       = 10000
	defaultWeight      = 100
	defaultConcurrency = 10
)

var (
	serviceDefaults = kong.Service{
		Protocol:       new("http"),
		ConnectTimeout: new(defaultTimeout),
		WriteTimeout:   new(defaultTimeout),
		ReadTimeout:    new(defaultTimeout),
	}
	routeDefaults = kong.Route{
		PreserveHost:  new(false),
		RegexPriority: new(0),
		StripPath:     new(true),
		Protocols:     kong.StringSlice("http", "https"),
	}
	targetDefaults = kong.Target{
		Weight: new(defaultWeight),
	}
	upstreamDefaults = kong.Upstream{
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
	}
	consumerGroupPluginDefault = kong.ConsumerGroupPlugin{
		Config: kong.Configuration{
			"window_type": "sliding",
		},
	}
	defaultsRestrictedFields = map[string][]string{
		"Service":  {"ID", "Name"},
		"Route":    {"ID", "Name"},
		"Target":   {"ID", "Target"},
		"Upstream": {"ID", "Name"},
	}
)

const (
	// ImplementationTypeKongGateway indicates an implementation backed by Kong Gateway.
	ImplementationTypeKongGateway = "kong-gateway"
)
