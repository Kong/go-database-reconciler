//go:build integration

package integration

import (
	"context"
	"testing"

	deckDump "github.com/kong/go-database-reconciler/pkg/dump"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/require"
)

var (
	consumerGroupInstanceNamePlugins = []*kong.Plugin{
		{
			Name:         kong.String("rate-limiting-advanced"),
			InstanceName: kong.String("default-instance"),
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
	consumerGroupInstanceNamePlugins35x = []*kong.Plugin{
		{
			Name:         kong.String("rate-limiting-advanced"),
			InstanceName: kong.String("default-instance"),
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
	}
	consumerGroupInstanceNamePlugins38x = []*kong.Plugin{
		{
			Name:         kong.String("rate-limiting-advanced"),
			InstanceName: kong.String("default-instance"),
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
					"cluster_addresses":        nil,
					"cluster_max_redirections": float64(5),
					"cluster_nodes":            nil,
					"connect_timeout":          float64(2000),
					"connection_is_proxied":    bool(false),
					"database":                 float64(0),
					"host":                     string("127.0.0.1"),
					"keepalive_backlog":        nil,
					"keepalive_pool_size":      float64(256),
					"password":                 nil,
					"port":                     float64(6379),
					"read_timeout":             float64(2000),
					"send_timeout":             float64(2000),
					"sentinel_addresses":       nil,
					"sentinel_master":          nil,
					"sentinel_nodes":           nil,
					"sentinel_password":        nil,
					"sentinel_role":            nil,
					"sentinel_username":        nil,
					"server_name":              nil,
					"ssl":                      false,
					"ssl_verify":               false,
					"timeout":                  float64(2000),
					"username":                 nil,
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
	}
	consumerGroupInstanceNamePlugins37x = []*kong.Plugin{
		{
			Name:         kong.String("rate-limiting-advanced"),
			InstanceName: kong.String("default-instance"),
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
	}
	consumerGroupInstanceNamePlugins390x = []*kong.Plugin{
		{
			Name:         kong.String("rate-limiting-advanced"),
			InstanceName: kong.String("default-instance"),
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("77e6691d-67c0-446a-9401-27be2b141aae"),
			},
			Config: kong.Configuration{
				"compound_identifier":     nil,
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
				"lock_dictionary_name":    string("kong_locks"),
				"namespace":               string("gold"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":        nil,
					"cluster_max_redirections": float64(5),
					"cluster_nodes":            nil,
					"connect_timeout":          float64(2000),
					"connection_is_proxied":    bool(false),
					"database":                 float64(0),
					"host":                     string("127.0.0.1"),
					"keepalive_backlog":        nil,
					"keepalive_pool_size":      float64(256),
					"password":                 nil,
					"port":                     float64(6379),
					"read_timeout":             float64(2000),
					"redis_proxy_type":         nil,
					"send_timeout":             float64(2000),
					"sentinel_addresses":       nil,
					"sentinel_master":          nil,
					"sentinel_nodes":           nil,
					"sentinel_password":        nil,
					"sentinel_role":            nil,
					"sentinel_username":        nil,
					"server_name":              nil,
					"ssl":                      false,
					"ssl_verify":               false,
					"timeout":                  float64(2000),
					"username":                 nil,
				},
				"retry_after_jitter_max": float64(1),
				"strategy":               string("local"),
				"sync_rate":              float64(-1),
				"window_size":            []any{float64(60)},
				"window_type":            string("sliding"),
			},
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
		}}
	consumerGroupInstanceNamePlugins310x = []*kong.Plugin{
		{
			Name:         kong.String("rate-limiting-advanced"),
			InstanceName: kong.String("default-instance"),
			ConsumerGroup: &kong.ConsumerGroup{
				ID: kong.String("77e6691d-67c0-446a-9401-27be2b141aae"),
			},
			Config: kong.Configuration{
				"compound_identifier":     nil,
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
				"lock_dictionary_name":    string("kong_locks"),
				"namespace":               string("gold"),
				"path":                    nil,
				"redis": map[string]any{
					"cluster_addresses":        nil,
					"cluster_max_redirections": float64(5),
					"cluster_nodes":            nil,
					"connect_timeout":          float64(2000),
					"connection_is_proxied":    bool(false),
					"database":                 float64(0),
					"host":                     string("127.0.0.1"),
					"keepalive_backlog":        nil,
					"keepalive_pool_size":      float64(256),
					"password":                 nil,
					"port":                     float64(6379),
					"read_timeout":             float64(2000),
					"redis_proxy_type":         nil,
					"send_timeout":             float64(2000),
					"sentinel_addresses":       nil,
					"sentinel_master":          nil,
					"sentinel_nodes":           nil,
					"sentinel_password":        nil,
					"sentinel_role":            nil,
					"sentinel_username":        nil,
					"server_name":              nil,
					"ssl":                      false,
					"ssl_verify":               false,
					"timeout":                  float64(2000),
					"username":                 nil,
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
	}
)

func Test_Apply_Custom_Entities(t *testing.T) {
	runWhen(t, "enterprise", ">=3.0.0")
	setup(t)
	client, err := getTestClient()
	if err != nil {
		t.Fatal(err.Error())
	}
	ctx := context.Background()
	tests := []struct {
		name                   string
		initialStateFile       string
		targetPartialStateFile string
	}{
		{
			name:                   "custom entity - degraphql routes",
			initialStateFile:       "testdata/apply/001-custom-entities/initial-state.yaml",
			targetPartialStateFile: "testdata/apply/001-custom-entities/partial-update.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mustResetKongState(ctx, t, client, deckDump.Config{})
			err := sync(tc.initialStateFile)
			require.NoError(t, err)

			err = apply(tc.targetPartialStateFile)
			require.NoError(t, err)
		})
	}
}

func Test_Apply_KeysAndKeySets(t *testing.T) {
	runWhenKongOrKonnect(t, ">=3.1.0")
	setup(t)

	client, err := getTestClient()
	require.NoError(t, err)
	ctx := t.Context()

	tests := []struct {
		name             string
		initialStateFile string
		updateStateFile  string
		expectedState    utils.KongRawState
	}{
		{
			name:             "keys and key_sets",
			initialStateFile: "testdata/apply/002-keys-and-key_sets/initial.yaml",
			updateStateFile:  "testdata/apply/002-keys-and-key_sets/update.yaml",
			expectedState: utils.KongRawState{
				Keys: []*kong.Key{
					{
						ID:   kong.String("f21a7073-1183-4b1c-bd87-4d5b8b18eeb4"),
						Name: kong.String("foo"),
						KID:  kong.String("vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4"),
						Set: &kong.KeySet{
							ID: kong.String("d46b0e15-ffbc-4b15-ad92-09ef67935453"),
						},
						JWK: kong.String("{\"kty\": \"RSA\", \"kid\": \"vsR8NCNV_1_LB06LqudGa2r-T0y4Z6VQVYue9IQz6A4\", \"n\": \"v2KAzzfruqctVHaE9WSCWIg1xAhMwxTIK-i56WNqPtpWBo9AqxcVea8NyVctEjUNq_mix5CklNy3ru7ARh7rBG_LU65fzs4fY_uYalul3QZSnr61Gj-cTUB3Gy4PhA63yXCbYRR3gDy6WR_wfis1MS61j0R_AjgXuVufmmC0F7R9qSWfR8ft0CbQgemEHY3ddKeW7T7fKv1jnRwYAkl5B_xtvxRFIYT-uR9NNftixNpUIW7q8qvOH7D9icXOg4_wIVxTRe5QiRYwEFoUbV1V9bFtu5FLal0vZnLaWwg5tA6enhzBpxJNdrS0v1RcPpyeNP-9r3cUDGmeftwz9v95UQ\", \"e\": \"AQAB\", \"alg\": \"A256GCM\"}"), //nolint:lll
					},
					{
						ID:   kong.String("d7cef208-23c3-46f8-94e8-fa1eddf43f0a"),
						Name: kong.String("baz"),
						KID:  kong.String("IiI4ffge7LZXPztrZVOt26zgRt0EPsWPaxAmwhbJhDQ"),
						Set: &kong.KeySet{
							ID: kong.String("d46b0e15-ffbc-4b15-ad92-09ef67935345"),
						},
						JWK: kong.String("{\n      \"kty\": \"RSA\",\n      \"kid\": \"IiI4ffge7LZXPztrZVOt26zgRt0EPsWPaxAmwhbJhDQ\",\n      \"use\": \"sig\",\n      \"alg\": \"RS256\",\n      \"e\": \"AQAB\",\n      \"n\": \"1Sn1X_y-RUzGna0hR00Wu64ZtY5N5BVzpRIby9wQ5EZVyWL9DRhU5PXqM3Y5gzgUVEQu548qQcMKOfs46PhOQudz-HPbwKWzcJCDUeNQsxdAEhW1uJR0EEV_SGJ-jTuKGqoEQc7bNrmhyXBMIeMkTeE_-ys75iiwvNjYphiOhsokC_vRTf_7TOPTe1UQasgxEVSLlTsen0vtK_FXcpbwdxZt02IysICcX5TcWX_XBuFP4cpwI9AS3M-imc01awc1t7FE5UWp62H5Ro2S5V9YwdxSjf4lX87AxYmawaWAjyO595XLuIXA3qt8-irzbCeglR1-cTB7a4I7_AclDmYrpw\"\n  }"), //nolint:lll
					},
					{
						ID:   kong.String("03ad4618-82bb-4375-b9d1-edeefced868d"),
						Name: kong.String("my-pem-key"),
						KID:  kong.String("my-pem-key"),
						Set: &kong.KeySet{
							ID: kong.String("d46b0e15-ffbc-4b15-ad92-09ef67935345"),
						},
						PEM: &kong.PEM{
							PublicKey:  kong.String("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqvxMU4LTcHBYmCuLMhMP\nDWlZdcNRXuJkw26MRjLBxXjnPAyDolmuFFMIqPDlSaJkkzu2tn7m9p8KB90wLiMC\nIbDjseruCO+7EaIRY4d6RdpE+XowCjJu7SbC2CqWBAzKkO7WWAunO3KOsQRk1NEK\nI51CoZ26LPYQvjIGIY2/pPxq0Ydl9dyURqVfmTywni1WeScgdEZXuy9WIcobqBST\n8vV5Q5HJsZNFLR7Fy61+HHfnQiWIYyi6h8QRT+Css9y5KbH7KuN6tnb94UZaOmHl\nYeoHcP/CqviZnQOf5804qcVpPKbsGU8jupTriiJZU3a8f59eHV0ybI4ORXYgDSWd\nFQIDAQAB\n-----END PUBLIC KEY-----"),                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               //nolint:lll
							PrivateKey: kong.String("-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAqvxMU4LTcHBYmCuLMhMPDWlZdcNRXuJkw26MRjLBxXjnPAyD\nolmuFFMIqPDlSaJkkzu2tn7m9p8KB90wLiMCIbDjseruCO+7EaIRY4d6RdpE+Xow\nCjJu7SbC2CqWBAzKkO7WWAunO3KOsQRk1NEKI51CoZ26LPYQvjIGIY2/pPxq0Ydl\n9dyURqVfmTywni1WeScgdEZXuy9WIcobqBST8vV5Q5HJsZNFLR7Fy61+HHfnQiWI\nYyi6h8QRT+Css9y5KbH7KuN6tnb94UZaOmHlYeoHcP/CqviZnQOf5804qcVpPKbs\nGU8jupTriiJZU3a8f59eHV0ybI4ORXYgDSWdFQIDAQABAoIBAEOOqAGfATe9y+Nj\n4P2J9jqQU15qK65XuQRWm2npCBKj8IkTULdGw7cYD6XgeFedqCtcPpbgkRUERYxR\n4oV4I5F4OJ7FegNh5QHUjRZMIw2Sbgo8Mtr0jkt5MycBvIAhJbAaDep/wDWGz8Y1\nPDmx1lW3/umoTjURjA/5594+CWiABYzuIi4WprWe4pIKqSKOMHnCYVAD243mwJ7y\nvsatO3LRKYfLw74ifCYhWNBHaZwfw+OO2P5Ku0AGhY4StOLCHobJ8/KkkmkTlYzv\nrcF4cVdvpBfdTEQed0oD7u3xfnp3GpNU3wZFsZJRSVXouhroaMC7en4uMc+5yguW\nqrPIoEkCgYEAxm1UllY9rRfGV6884hdBFKDjE825BC1VlqcRIUEB4CpJvUF/6+A3\ngx5c4nKDJAFQMrWpr4jOcq3iLiWnJ73e80b+JpWFODdt16g2KCOINs1j8vf2U6Og\nx+Vo8vHek/Uomz1n5W0oXrJ4VedHl9NYa8r/YrVXd4k4WcaA0TXmMhMCgYEA3Jit\nzrEmrQIrLK66RgXF2RafA5c3atRHWBb5ddnGk0bV90cfsTsaDMDvpy7ZYgojBNpw\n7U6AYzqnPro6cHEginV97BFb6oetMvOWvljUob+tpnYOofgwk2hw7PeChViX7iS9\nujgTygi8ZIc2G0r7xntH+v6WHKp4yNQiCAyfGTcCgYAYKgZMDJKUOrn3wapraiON\nzI36wmnOnWq33v6SCyWcU+oI9yoJ4pNAD3mGRiW8Q8CtfDv+2W0ywAQ0VHeHunKl\nM7cNodXIY8+nnJ+Dwdf7vIV4eEPyKZIR5dkjBNtzLz7TsOWvJdzts1Q+Od0ZGy7A\naccyER1mvDo1jJvxXlv7KwKBgQDDBK9TdUVt2eb1X5sJ4HyiiN8XO44ggX55IAZ1\n64skFJGARH5+HnPPJpo3wLEpfTCsT7lZ8faKwwWr7NNRKJHOFkS2eDo8QqoZy0NP\nEBUa0evgp6oUAuheyQxcUgwver0GKbEZeg30pHh4nxh0VHv1YnOmL3/h48tYMEHN\nv+q/TQKBgQCXQmN8cY2K7UfZJ6BYEdguQZS5XISFbLNkG8wXQX9vFiF8TuSWawDN\nTrRHVDGwoMGWxjZBLCsitA6zwrMLJZs4RuetKHFou7MiDQ69YGdfNRlRvD5QCJDc\nY0ICsYjI7VM89Qj/41WQyRHYHm7E9key3avMGdbYtxdc0Ku4LnD4zg==\n-----END RSA PRIVATE KEY-----"), //nolint:lll
						},
					},
				},
				KeySets: []*kong.KeySet{
					{
						Name: kong.String("bar"),
						ID:   kong.String("d46b0e15-ffbc-4b15-ad92-09ef67935453"),
					},
					{
						Name: kong.String("bar-new"),
						ID:   kong.String("d46b0e15-ffbc-4b15-ad92-09ef67935345"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mustResetKongState(ctx, t, client, deckDump.Config{})
			err := sync(tc.initialStateFile)
			require.NoError(t, err)

			err = apply(tc.updateStateFile)
			require.NoError(t, err)

			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Apply_NestedEntity(t *testing.T) {
	setup(t)
	client, err := getTestClient()
	require.NoError(t, err)
	ctx := t.Context()

	tests := []struct {
		name             string
		initialStateFile string
		updateStateFile  string
		expectedState    utils.KongRawState
		runWhen          func(t *testing.T)
	}{
		{
			name:             "nested route in service",
			initialStateFile: "testdata/apply/003-nested-entity/route-initial.yaml",
			updateStateFile:  "testdata/apply/003-nested-entity/route-update.yaml",
			expectedState: utils.KongRawState{
				Services: []*kong.Service{
					{
						ConnectTimeout: kong.Int(60000),
						Enabled:        kong.Bool(true),
						Host:           kong.String("httpbin.konghq.com"),
						ID:             kong.String("c34277f2-b3f0-4778-aa6a-7701fc67f65b"),
						Name:           kong.String("test_svc"),
						Path:           kong.String("/anything"),
						Port:           kong.Int(80),
						Protocol:       kong.String("http"),
						ReadTimeout:    kong.Int(60000),
						Retries:        kong.Int(5),
						WriteTimeout:   kong.Int(60000),
						Tags:           nil,
					},
				},
				Routes: []*kong.Route{
					{
						ID:                      kong.String("d533e04a-9136-4439-8522-caed769aa158"),
						Name:                    kong.String("test_rt"),
						Paths:                   []*string{kong.String("/test"), kong.String("/test/abc")},
						PathHandling:            kong.String("v0"),
						PreserveHost:            kong.Bool(false),
						Protocols:               []*string{kong.String("http"), kong.String("https")},
						RegexPriority:           kong.Int(0),
						StripPath:               kong.Bool(true),
						HTTPSRedirectStatusCode: kong.Int(426),
						RequestBuffering:        kong.Bool(true),
						ResponseBuffering:       kong.Bool(true),
						Service: &kong.Service{
							ID: kong.String("c34277f2-b3f0-4778-aa6a-7701fc67f65b"),
						},
					},
				},
			},
		},
		{
			name:             "nested consumer in consumer group",
			initialStateFile: "testdata/apply/003-nested-entity/consumer-group-initial.yaml",
			updateStateFile:  "testdata/apply/003-nested-entity/consumer-group-update.yaml",
			runWhen:          func(t *testing.T) { runWhen(t, "enterprise", ">=2.7.0") },
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("gold"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("alice"),
								ID:       kong.String("3401bb32-32b2-4d50-8533-6669b27d5a42"),
								Tags:     []*string{kong.String("internal-user"), kong.String("internal-user2")},
							},
						},
					},
				},
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("alice"),
						ID:       kong.String("3401bb32-32b2-4d50-8533-6669b27d5a42"),
						Tags:     []*string{kong.String("internal-user"), kong.String("internal-user2")},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.runWhen != nil {
				tc.runWhen(t)
			}
			mustResetKongState(ctx, t, client, deckDump.Config{})
			err := sync(tc.initialStateFile)
			require.NoError(t, err)

			err = apply(tc.updateStateFile)
			require.NoError(t, err)

			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Apply_Service_Route(t *testing.T) {
	setup(t)
	client, err := getTestClient()
	require.NoError(t, err)
	ctx := t.Context()

	tests := []struct {
		name             string
		initialStateFile string
		updateStateFile  string
		expectedState    utils.KongRawState
		runWhen          func(t *testing.T)
	}{
		{
			name:             "route addition and update for a service",
			initialStateFile: "testdata/apply/005-routes/route-initial.yaml",
			updateStateFile:  "testdata/apply/005-routes/route-update.yaml",
			expectedState: utils.KongRawState{
				Services: []*kong.Service{
					{
						ConnectTimeout: kong.Int(60000),
						Enabled:        kong.Bool(true),
						Host:           kong.String("mockbin.org"),
						ID:             kong.String("c34277f2-b3f0-4778-aa6a-7701fc67f65b"),
						Name:           kong.String("svc1"),
						Port:           kong.Int(80),
						Protocol:       kong.String("http"),
						ReadTimeout:    kong.Int(60000),
						Retries:        kong.Int(5),
						WriteTimeout:   kong.Int(60000),
						Tags:           nil,
					},
				},
				Routes: []*kong.Route{
					{
						ID:                      kong.String("87b6a97e-f3f7-4c47-857a-7464cb9e202b"),
						Name:                    kong.String("r1"),
						Paths:                   []*string{kong.String("/r1")},
						PathHandling:            kong.String("v0"),
						PreserveHost:            kong.Bool(false),
						Protocols:               []*string{kong.String("http"), kong.String("https")},
						RegexPriority:           kong.Int(0),
						StripPath:               kong.Bool(true),
						HTTPSRedirectStatusCode: kong.Int(426),
						RequestBuffering:        kong.Bool(true),
						ResponseBuffering:       kong.Bool(true),
						Service: &kong.Service{
							ID: kong.String("c34277f2-b3f0-4778-aa6a-7701fc67f65b"),
						},
					},
					{
						ID:                      kong.String("87b6a97e-f3f7-4c47-857a-7464cb9e202c"),
						Name:                    kong.String("r2"),
						Paths:                   []*string{kong.String("/r2")},
						PathHandling:            kong.String("v0"),
						PreserveHost:            kong.Bool(false),
						Protocols:               []*string{kong.String("http"), kong.String("https")},
						RegexPriority:           kong.Int(0),
						StripPath:               kong.Bool(true),
						HTTPSRedirectStatusCode: kong.Int(301),
						RequestBuffering:        kong.Bool(true),
						ResponseBuffering:       kong.Bool(true),
						Service: &kong.Service{
							ID: kong.String("c34277f2-b3f0-4778-aa6a-7701fc67f65b"),
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mustResetKongState(ctx, t, client, deckDump.Config{})
			err := sync(tc.initialStateFile)
			require.NoError(t, err)

			err = apply(tc.updateStateFile)
			require.NoError(t, err)

			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Apply_Consumer_Group_Consumer(t *testing.T) {
	runWhen(t, "enterprise", ">=2.7.0")
	setup(t)
	client, err := getTestClient()
	require.NoError(t, err)
	ctx := t.Context()

	tests := []struct {
		name             string
		initialStateFile string
		updateStateFile  string
		expectedState    utils.KongRawState
		runWhen          func(t *testing.T)
	}{
		{
			name:             "consumer addition and update to consumer group",
			initialStateFile: "testdata/apply/004-consumers-and-groups/consumer-group-consumer-initial.yaml",
			updateStateFile:  "testdata/apply/004-consumers-and-groups/consumer-group-consumer-final.yaml",
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("gold"),
						},
						Consumers: []*kong.Consumer{
							{
								Username: kong.String("alice"),
								Tags:     []*string{kong.String("internal-user"), kong.String("internal-user2")},
							},
							{
								Username: kong.String("frank"),
								Tags:     []*string{kong.String("internal-user3")},
							},
						},
					},
				},
				Consumers: []*kong.Consumer{
					{
						Username: kong.String("alice"),
						Tags:     []*string{kong.String("internal-user"), kong.String("internal-user2")},
					},
					{
						Username: kong.String("frank"),
						Tags:     []*string{kong.String("internal-user3")},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mustResetKongState(ctx, t, client, deckDump.Config{})
			err := sync(tc.initialStateFile)
			require.NoError(t, err)

			err = apply(tc.updateStateFile)
			require.NoError(t, err)

			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}

func Test_Apply_Consumer_Group_Plugin(t *testing.T) {
	setup(t)
	client, err := getTestClient()
	require.NoError(t, err)
	ctx := t.Context()

	tests := []struct {
		name             string
		initialStateFile string
		updateStateFile  string
		expectedState    utils.KongRawState
		runWhen          func(t *testing.T)
	}{
		{
			name:             "plugin addition to consumer group",
			initialStateFile: "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-initial.yaml",
			updateStateFile:  "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-final.yaml",
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
					},
				},
				Plugins: consumerGroupInstanceNamePlugins,
			},
			runWhen: func(t *testing.T) { runWhen(t, "enterprise", ">=3.4.0 <3.5.0") },
		},
		{
			name:             "plugin addition to consumer group",
			initialStateFile: "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-initial.yaml",
			updateStateFile:  "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-final.yaml",
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
					},
				},
				Plugins: consumerGroupInstanceNamePlugins35x,
			},
			runWhen: func(t *testing.T) { runWhen(t, "enterprise", ">=3.5.0 <3.6.0") },
		},
		{
			name:             "plugin addition to consumer group",
			initialStateFile: "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-initial.yaml",
			updateStateFile:  "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-final.yaml",
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
					},
				},
				Plugins: consumerGroupInstanceNamePlugins37x,
			},
			runWhen: func(t *testing.T) { runWhen(t, "enterprise", ">=3.7.0 <3.8.0") },
		},
		{
			name:             "plugin addition to consumer group",
			initialStateFile: "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-initial.yaml",
			updateStateFile:  "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-final.yaml",
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
					},
				},
				Plugins: consumerGroupInstanceNamePlugins38x,
			},
			runWhen: func(t *testing.T) { runWhen(t, "enterprise", ">=3.8.0 <3.9.0") },
		},
		{
			name:             "plugin addition to consumer group",
			initialStateFile: "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-initial.yaml",
			updateStateFile:  "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-final.yaml",
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
					},
				},
				Plugins: consumerGroupInstanceNamePlugins390x,
			},
			runWhen: func(t *testing.T) { runWhen(t, "enterprise", ">=3.9.0 <3.10.0") },
		},
		{
			name:             "plugin addition to consumer group",
			initialStateFile: "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-initial.yaml",
			updateStateFile:  "testdata/apply/006-consumer-group-plugins/consumer-group-plugin-final.yaml",
			expectedState: utils.KongRawState{
				ConsumerGroups: []*kong.ConsumerGroupObject{
					{
						ConsumerGroup: &kong.ConsumerGroup{
							Name: kong.String("silver"),
						},
					},
				},
				Plugins: consumerGroupInstanceNamePlugins310x,
			},
			runWhen: func(t *testing.T) { runWhen(t, "enterprise", ">=3.10.0") },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mustResetKongState(ctx, t, client, deckDump.Config{})
			err := sync(tc.initialStateFile)
			require.NoError(t, err)

			err = apply(tc.updateStateFile)
			require.NoError(t, err)

			testKongState(t, client, false, tc.expectedState, nil)
		})
	}
}
