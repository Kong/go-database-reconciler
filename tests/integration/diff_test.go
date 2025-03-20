//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kong/go-database-reconciler/pkg/utils"

	deckDiff "github.com/kong/go-database-reconciler/pkg/diff"
	deckDump "github.com/kong/go-database-reconciler/pkg/dump"
)

var (
	expectedOutputMasked = `updating service svc1  {
   "connect_timeout": 60000,
   "enabled": true,
   "host": "[masked]",
   "id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
   "name": "svc1",
   "port": 80,
   "protocol": "http",
   "read_timeout": 60000,
   "retries": 5,
   "write_timeout": 60000
+  "tags": [
+    "[masked] is an external host. I like [masked]!",
+    "foo:foo",
+    "baz:[masked]",
+    "another:[masked]",
+    "bar:[masked]"
+  ]
 }

creating plugin rate-limiting (global)
Summary:
  Created: 1
  Updated: 1
  Deleted: 0
`

	expectedOutputUnMasked = `updating service svc1  {
   "connect_timeout": 60000,
   "enabled": true,
   "host": "mockbin.org",
   "id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
   "name": "svc1",
   "port": 80,
   "protocol": "http",
   "read_timeout": 60000,
   "retries": 5,
   "write_timeout": 60000
+  "tags": [
+    "test"
+  ]
 }

creating plugin rate-limiting (global)
Summary:
  Created: 1
  Updated: 1
  Deleted: 0
`

	diffEnvVars = map[string]string{
		"DECK_SVC1_HOSTNAME": "mockbin.org",
		"DECK_BARR":          "barbar",
		"DECK_BAZZ":          "bazbaz",   // used more than once
		"DECK_FUB":           "fubfub",   // unused
		"DECK_FOO":           "foo_test", // unused, partial match
	}

	expectedOutputUnMaskedJSON = `{
	"changes": {
		"creating": [
			{
				"name": "rate-limiting (global)",
				"kind": "plugin",
				"body": {
					"new": {
						"id": "a1368a28-cb5c-4eee-86d8-03a6bdf94b5e",
						"name": "rate-limiting",
						"config": {
							"day": null,
							"error_code": 429,
							"error_message": "API rate limit exceeded",
							"fault_tolerant": true,
							"header_name": null,
							"hide_client_headers": false,
							"hour": null,
							"limit_by": "consumer",
							"minute": 123,
							"month": null,
							"path": null,
							"policy": "local",
							"redis_database": 0,
							"redis_host": null,
							"redis_password": null,
							"redis_port": 6379,
							"redis_server_name": null,
							"redis_ssl": false,
							"redis_ssl_verify": false,
							"redis_timeout": 2000,
							"redis_username": null,
							"second": null,
							"year": null
						},
						"enabled": true,
						"protocols": [
							"grpc",
							"grpcs",
							"http",
							"https"
						]
					},
					"old": null
				}
			}
		],
		"updating": [
			{
				"name": "svc1",
				"kind": "service",
				"body": {
					"new": {
						"connect_timeout": 60000,
						"enabled": true,
						"host": "mockbin.org",
						"id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
						"name": "svc1",
						"port": 80,
						"protocol": "http",
						"read_timeout": 60000,
						"retries": 5,
						"write_timeout": 60000,
						"tags": [
							"test"
						]
					},
					"old": {
						"connect_timeout": 60000,
						"enabled": true,
						"host": "mockbin.org",
						"id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
						"name": "svc1",
						"port": 80,
						"protocol": "http",
						"read_timeout": 60000,
						"retries": 5,
						"write_timeout": 60000
					}
				}
			}
		],
		"deleting": []
	},
	"summary": {
		"creating": 1,
		"updating": 1,
		"deleting": 0,
		"total": 2
	},
	"warnings": [],
	"errors": []
}

`

	expectedOutputMaskedJSON = `{
	"changes": {
		"creating": [
			{
				"name": "rate-limiting (global)",
				"kind": "plugin",
				"body": {
					"new": {
						"id": "a1368a28-cb5c-4eee-86d8-03a6bdf94b5e",
						"name": "rate-limiting",
						"config": {
							"day": null,
							"error_code": 429,
							"error_message": "API rate limit exceeded",
							"fault_tolerant": true,
							"header_name": null,
							"hide_client_headers": false,
							"hour": null,
							"limit_by": "consumer",
							"minute": 123,
							"month": null,
							"path": null,
							"policy": "local",
							"redis_database": 0,
							"redis_host": null,
							"redis_password": null,
							"redis_port": 6379,
							"redis_server_name": null,
							"redis_ssl": false,
							"redis_ssl_verify": false,
							"redis_timeout": 2000,
							"redis_username": null,
							"second": null,
							"year": null
						},
						"enabled": true,
						"protocols": [
							"grpc",
							"grpcs",
							"http",
							"https"
						]
					},
					"old": null
				}
			}
		],
		"updating": [
			{
				"name": "svc1",
				"kind": "service",
				"body": {
					"new": {
						"connect_timeout": 60000,
						"enabled": true,
						"host": "[masked]",
						"id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
						"name": "svc1",
						"port": 80,
						"protocol": "http",
						"read_timeout": 60000,
						"retries": 5,
						"write_timeout": 60000,
						"tags": [
							"[masked] is an external host. I like [masked]!",
							"foo:foo",
							"baz:[masked]",
							"another:[masked]",
							"bar:[masked]"
						]
					},
					"old": {
						"connect_timeout": 60000,
						"enabled": true,
						"host": "[masked]",
						"id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
						"name": "svc1",
						"port": 80,
						"protocol": "http",
						"read_timeout": 60000,
						"retries": 5,
						"write_timeout": 60000
					}
				}
			}
		],
		"deleting": []
	},
	"summary": {
		"creating": 1,
		"updating": 1,
		"deleting": 0,
		"total": 2
	},
	"warnings": [],
	"errors": []
}

`

	expectedOutputUnMaskedJSON30x = `{
	"changes": {
		"creating": [
			{
				"name": "rate-limiting (global)",
				"kind": "plugin",
				"body": {
					"new": {
						"id": "a1368a28-cb5c-4eee-86d8-03a6bdf94b5e",
						"name": "rate-limiting",
						"config": {
							"day": null,
							"fault_tolerant": true,
							"header_name": null,
							"hide_client_headers": false,
							"hour": null,
							"limit_by": "consumer",
							"minute": 123,
							"month": null,
							"path": null,
							"policy": "local",
							"redis_database": 0,
							"redis_host": null,
							"redis_password": null,
							"redis_port": 6379,
							"redis_server_name": null,
							"redis_ssl": false,
							"redis_ssl_verify": false,
							"redis_timeout": 2000,
							"redis_username": null,
							"second": null,
							"year": null
						},
						"enabled": true,
						"protocols": [
							"grpc",
							"grpcs",
							"http",
							"https"
						]
					},
					"old": null
				}
			}
		],
		"updating": [
			{
				"name": "svc1",
				"kind": "service",
				"body": {
					"new": {
						"connect_timeout": 60000,
						"enabled": true,
						"host": "mockbin.org",
						"id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
						"name": "svc1",
						"port": 80,
						"protocol": "http",
						"read_timeout": 60000,
						"retries": 5,
						"write_timeout": 60000,
						"tags": [
							"test"
						]
					},
					"old": {
						"connect_timeout": 60000,
						"enabled": true,
						"host": "mockbin.org",
						"id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
						"name": "svc1",
						"port": 80,
						"protocol": "http",
						"read_timeout": 60000,
						"retries": 5,
						"write_timeout": 60000
					}
				}
			}
		],
		"deleting": []
	},
	"summary": {
		"creating": 1,
		"updating": 1,
		"deleting": 0,
		"total": 2
	},
	"warnings": [],
	"errors": []
}

`

	expectedOutputMaskedJSON30x = `{
	"changes": {
		"creating": [
			{
				"name": "rate-limiting (global)",
				"kind": "plugin",
				"body": {
					"new": {
						"id": "a1368a28-cb5c-4eee-86d8-03a6bdf94b5e",
						"name": "rate-limiting",
						"config": {
							"day": null,
							"fault_tolerant": true,
							"header_name": null,
							"hide_client_headers": false,
							"hour": null,
							"limit_by": "consumer",
							"minute": 123,
							"month": null,
							"path": null,
							"policy": "local",
							"redis_database": 0,
							"redis_host": null,
							"redis_password": null,
							"redis_port": 6379,
							"redis_server_name": null,
							"redis_ssl": false,
							"redis_ssl_verify": false,
							"redis_timeout": 2000,
							"redis_username": null,
							"second": null,
							"year": null
						},
						"enabled": true,
						"protocols": [
							"grpc",
							"grpcs",
							"http",
							"https"
						]
					},
					"old": null
				}
			}
		],
		"updating": [
			{
				"name": "svc1",
				"kind": "service",
				"body": {
					"new": {
						"connect_timeout": 60000,
						"enabled": true,
						"host": "[masked]",
						"id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
						"name": "svc1",
						"port": 80,
						"protocol": "http",
						"read_timeout": 60000,
						"retries": 5,
						"write_timeout": 60000,
						"tags": [
							"[masked] is an external host. I like [masked]!",
							"foo:foo",
							"baz:[masked]",
							"another:[masked]",
							"bar:[masked]"
						]
					},
					"old": {
						"connect_timeout": 60000,
						"enabled": true,
						"host": "[masked]",
						"id": "9ecf5708-f2f4-444e-a4c7-fcd3a57f9a6d",
						"name": "svc1",
						"port": 80,
						"protocol": "http",
						"read_timeout": 60000,
						"retries": 5,
						"write_timeout": 60000
					}
				}
			}
		],
		"deleting": []
	},
	"summary": {
		"creating": 1,
		"updating": 1,
		"deleting": 0,
		"total": 2
	},
	"warnings": [],
	"errors": []
}

`
	expectedOutputPluginUpdateNoChange = `Summary:
  Created: 0
  Updated: 0
  Deleted: 0
`

	expectedOutputPluginUpdateChangedNewFieldsOpenIdConnect = `updating plugin openid-connect (global)  {
   "config": {
     "anonymous": null,
     "audience": null,
     "audience_claim": [
       "aud"
     ],
     "audience_required": null,
     "auth_methods": [
       "password",
       "client_credentials",
       "authorization_code",
       "bearer",
       "introspection",
       "userinfo",
       "kong_oauth2",
       "refresh_token",
       "session"
     ],
     "authenticated_groups_claim": null,
     "authorization_cookie_domain": null,
     "authorization_cookie_http_only": true,
     "authorization_cookie_name": "authorization",
     "authorization_cookie_path": "/",
     "authorization_cookie_same_site": "Default",
     "authorization_cookie_secure": null,
     "authorization_endpoint": null,
     "authorization_query_args_client": null,
     "authorization_query_args_names": null,
     "authorization_query_args_values": null,
     "authorization_rolling_timeout": 600,
     "bearer_token_cookie_name": null,
     "bearer_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "by_username_ignore_case": false,
     "cache_introspection": true,
     "cache_token_exchange": true,
     "cache_tokens": true,
     "cache_tokens_salt": null,
     "cache_ttl": 3600,
     "cache_ttl_max": null,
     "cache_ttl_min": null,
     "cache_ttl_neg": null,
     "cache_ttl_resurrect": null,
     "cache_user_info": true,
     "claims_forbidden": null,
     "client_alg": null,
     "client_arg": "client_id",
     "client_auth": null,
     "client_credentials_param_type": [
       "header",
       "query",
       "body"
     ],
     "client_id": null,
     "client_jwk": null,
     "client_secret": null,
     "cluster_cache_redis": {
       "cluster_max_redirections": 5,
       "cluster_nodes": null,
       "connect_timeout": 2000,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.1",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6379,
       "read_timeout": 2000,
       "send_timeout": 2000,
       "sentinel_master": null,
       "sentinel_nodes": null,
       "sentinel_password": null,
       "sentinel_role": null,
       "sentinel_username": null,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "cluster_cache_strategy": "[masked]",
     "consumer_by": [
       "username",
       "custom_id"
     ],
     "consumer_claim": null,
     "consumer_optional": false,
     "credential_claim": [
       "sub"
     ],
     "disable_session": null,
     "discovery_headers_names": null,
     "discovery_headers_values": null,
     "display_errors": false,
     "domains": null,
     "downstream_access_token_header": null,
     "downstream_access_token_jwk_header": null,
     "downstream_headers_claims": null,
     "downstream_headers_names": null,
     "downstream_id_token_header": null,
     "downstream_id_token_jwk_header": null,
     "downstream_introspection_header": null,
     "downstream_introspection_jwt_header": null,
     "downstream_refresh_token_header": null,
     "downstream_session_id_header": null,
     "downstream_user_info_header": null,
     "downstream_user_info_jwt_header": null,
     "dpop_proof_lifetime": 300,
     "dpop_use_nonce": false,
     "enable_hs_signatures": false,
     "end_session_endpoint": null,
     "expose_error_code": true,
     "extra_jwks_uris": null,
     "forbidden_destroy_session": true,
     "forbidden_error_message": "Forbidden",
     "forbidden_redirect_uri": null,
     "groups_claim": [
       "groups"
     ],
     "groups_required": null,
     "hide_credentials": false,
     "http_proxy": null,
     "http_proxy_authorization": null,
     "http_version": 1.1,
     "https_proxy": null,
     "https_proxy_authorization": null,
     "id_token_param_name": null,
     "id_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "ignore_signature": [
     ],
     "introspect_jwt_tokens": false,
     "introspection_accept": "application/json",
     "introspection_check_active": true,
     "introspection_endpoint": null,
     "introspection_endpoint_auth_method": null,
     "introspection_headers_client": null,
     "introspection_headers_names": null,
     "introspection_headers_values": null,
     "introspection_hint": "access_token",
     "introspection_post_args_client": null,
     "introspection_post_args_names": null,
     "introspection_post_args_values": null,
     "introspection_token_param_name": "token",
     "issuer": "https://accounts.google.test/.well-known/openid-configuration",
     "issuers_allowed": null,
     "jwt_session_claim": "sid",
     "jwt_session_cookie": null,
     "keepalive": true,
     "leeway": 0,
     "login_action": "upstream",
     "login_methods": [
       "authorization_code"
     ],
     "login_redirect_mode": "fragment",
     "login_redirect_uri": null,
     "login_tokens": [
       "id_token"
     ],
     "logout_methods": [
       "POST",
       "DELETE"
     ],
     "logout_post_arg": null,
     "logout_query_arg": null,
     "logout_redirect_uri": null,
     "logout_revoke": false,
     "logout_revoke_access_token": true,
     "logout_revoke_refresh_token": true,
     "logout_uri_suffix": null,
     "max_age": null,
     "mtls_introspection_endpoint": null,
     "mtls_revocation_endpoint": null,
     "mtls_token_endpoint": null,
     "no_proxy": null,
     "password_param_type": [
       "header",
       "query",
       "body"
     ],
     "preserve_query_args": false,
     "proof_of_possession_auth_methods_validation": true,
     "proof_of_possession_dpop": "[masked]",
     "proof_of_possession_mtls": "[masked]",
     "pushed_authorization_request_endpoint": null,
     "pushed_authorization_request_endpoint_auth_method": null,
     "redirect_uri": null,
     "redis": {
-      "cluster_max_redirections": null,
+      "cluster_max_redirections": 11,
-      "cluster_nodes": null,
+      "cluster_nodes": [
+        {
+          "ip": "127.0.1.0",
+          "port": 7379
+        },
+        {
+          "ip": "127.0.1.0",
+          "port": 7380
+        },
+        {
+          "ip": "127.0.1.0",
+          "port": 7381
+        }
+      ],
       "connect_timeout": 2000,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.1",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6379,
       "prefix": null,
       "read_timeout": 2000,
       "send_timeout": 2000,
       "sentinel_master": null,
       "sentinel_nodes": null,
       "sentinel_password": null,
       "sentinel_role": null,
       "sentinel_username": null,
       "server_name": null,
       "socket": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "rediscovery_lifetime": 30,
     "refresh_token_param_name": null,
     "refresh_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "refresh_tokens": true,
     "require_proof_key_for_code_exchange": null,
     "require_pushed_authorization_requests": null,
     "require_signed_request_object": null,
     "resolve_distributed_claims": false,
     "response_mode": "query",
     "response_type": [
       "code"
     ],
     "reverify": false,
     "revocation_endpoint": null,
     "revocation_endpoint_auth_method": null,
     "revocation_token_param_name": "token",
     "roles_claim": [
       "roles"
     ],
     "roles_required": null,
     "run_on_preflight": true,
     "scopes": [
       "openid"
     ],
     "scopes_claim": [
       "scope"
     ],
     "scopes_required": null,
     "search_user_info": false,
     "session_absolute_timeout": 86400,
     "session_audience": "default",
     "session_cookie_domain": null,
     "session_cookie_http_only": true,
     "session_cookie_name": "session",
     "session_cookie_path": "/",
     "session_cookie_same_site": "Lax",
     "session_cookie_secure": null,
     "session_enforce_same_subject": false,
     "session_hash_storage_key": false,
     "session_hash_subject": false,
     "session_idling_timeout": 900,
     "session_memcached_host": "127.0.0.1",
     "session_memcached_port": 11211,
     "session_memcached_prefix": null,
     "session_memcached_socket": null,
     "session_remember": false,
     "session_remember_absolute_timeout": 2.592e+06,
     "session_remember_cookie_name": "remember",
     "session_remember_rolling_timeout": 604800,
     "session_request_headers": null,
     "session_response_headers": null,
     "session_rolling_timeout": 3600,
     "session_secret": null,
     "session_storage": "cookie",
     "session_store_metadata": false,
     "ssl_verify": false,
     "timeout": 10000,
     "tls_client_auth_cert_id": null,
     "tls_client_auth_ssl_verify": true,
     "token_cache_key_include_scope": false,
     "token_endpoint": null,
     "token_endpoint_auth_method": null,
     "token_exchange_endpoint": null,
     "token_headers_client": null,
     "token_headers_grants": null,
     "token_headers_names": null,
     "token_headers_prefix": null,
     "token_headers_replay": null,
     "token_headers_values": null,
     "token_post_args_client": null,
     "token_post_args_names": null,
     "token_post_args_values": null,
     "unauthorized_destroy_session": true,
     "unauthorized_error_message": "Unauthorized",
     "unauthorized_redirect_uri": null,
     "unexpected_redirect_uri": null,
     "upstream_access_token_header": "authorization:bearer",
     "upstream_access_token_jwk_header": null,
     "upstream_headers_claims": null,
     "upstream_headers_names": null,
     "upstream_id_token_header": null,
     "upstream_id_token_jwk_header": null,
     "upstream_introspection_header": null,
     "upstream_introspection_jwt_header": null,
     "upstream_refresh_token_header": null,
     "upstream_session_id_header": null,
     "upstream_user_info_header": null,
     "upstream_user_info_jwt_header": null,
     "userinfo_accept": "application/json",
     "userinfo_endpoint": null,
     "userinfo_headers_client": null,
     "userinfo_headers_names": null,
     "userinfo_headers_values": null,
     "userinfo_query_args_client": null,
     "userinfo_query_args_names": null,
     "userinfo_query_args_values": null,
     "using_pseudo_issuer": false,
     "verify_claims": true,
     "verify_nonce": true,
     "verify_parameters": false,
     "verify_signature": true
   },
   "enabled": true,
   "id": "777496e1-8b35-4512-ad30-51f9fe5d3147",
   "name": "openid-connect",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`
	expectedOutputPluginUpdateChangedNewFieldsRLA = `updating plugin rate-limiting-advanced (global)  {
   "config": {
     "consumer_groups": null,
     "dictionary_name": "kong_rate_limiting_counters",
     "disable_penalty": false,
     "enforce_consumer_groups": false,
     "error_code": 429,
     "error_message": "API rate limit exceeded",
     "header_name": null,
     "hide_client_headers": false,
     "identifier": "consumer",
     "limit": [
       10
     ],
     "namespace": "ZEz47TWgUrv01HenyQBQa8io06MWsp0L",
     "path": null,
     "redis": {
       "cluster_max_redirections": 5,
       "cluster_nodes": [
         {
           "ip": "127.0.1.0",
-          "port": 6379
+          "port": 7379
         },
         {
           "ip": "127.0.1.0",
-          "port": 6380
+          "port": 7380
         },
         {
           "ip": "127.0.1.0",
-          "port": 6381
+          "port": 7381
         }
       ],
-      "connect_timeout": 2000,
+      "connect_timeout": 2005,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.5",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6380,
-      "read_timeout": 2000,
+      "read_timeout": 2006,
-      "send_timeout": 2000,
+      "send_timeout": 2007,
       "sentinel_master": "mymaster",
       "sentinel_nodes": [
         {
           "host": "127.0.2.0",
-          "port": 6379
+          "port": 8379
         },
         {
           "host": "127.0.2.0",
-          "port": 6380
+          "port": 8380
         },
         {
           "host": "127.0.2.0",
-          "port": 6381
+          "port": 8381
         }
       ],
       "sentinel_password": null,
       "sentinel_role": "master",
       "sentinel_username": null,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "retry_after_jitter_max": 0,
     "strategy": "redis",
     "sync_rate": 10,
     "window_size": [
       60
     ],
     "window_type": "sliding"
   },
   "enabled": true,
   "id": "a1368a28-cb5c-4eee-86d8-03a6bdf94b5e",
   "name": "rate-limiting-advanced",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`
	expectedOutputPluginUpdateChangedOldFieldsOpenIdConnect = `updating plugin openid-connect (global)  {
   "config": {
     "anonymous": null,
     "audience": null,
     "audience_claim": [
       "aud"
     ],
     "audience_required": null,
     "auth_methods": [
       "password",
       "client_credentials",
       "authorization_code",
       "bearer",
       "introspection",
       "userinfo",
       "kong_oauth2",
       "refresh_token",
       "session"
     ],
     "authenticated_groups_claim": null,
     "authorization_cookie_domain": null,
     "authorization_cookie_http_only": true,
     "authorization_cookie_name": "authorization",
     "authorization_cookie_path": "/",
     "authorization_cookie_same_site": "Default",
     "authorization_cookie_secure": null,
     "authorization_endpoint": null,
     "authorization_query_args_client": null,
     "authorization_query_args_names": null,
     "authorization_query_args_values": null,
     "authorization_rolling_timeout": 600,
     "bearer_token_cookie_name": null,
     "bearer_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "by_username_ignore_case": false,
     "cache_introspection": true,
     "cache_token_exchange": true,
     "cache_tokens": true,
     "cache_tokens_salt": null,
     "cache_ttl": 3600,
     "cache_ttl_max": null,
     "cache_ttl_min": null,
     "cache_ttl_neg": null,
     "cache_ttl_resurrect": null,
     "cache_user_info": true,
     "claims_forbidden": null,
     "client_alg": null,
     "client_arg": "client_id",
     "client_auth": null,
     "client_credentials_param_type": [
       "header",
       "query",
       "body"
     ],
     "client_id": null,
     "client_jwk": null,
     "client_secret": null,
     "cluster_cache_redis": {
       "cluster_max_redirections": 5,
       "cluster_nodes": null,
       "connect_timeout": 2000,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.1",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6379,
       "read_timeout": 2000,
       "send_timeout": 2000,
       "sentinel_master": null,
       "sentinel_nodes": null,
       "sentinel_password": null,
       "sentinel_role": null,
       "sentinel_username": null,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "cluster_cache_strategy": "[masked]",
     "consumer_by": [
       "username",
       "custom_id"
     ],
     "consumer_claim": null,
     "consumer_optional": false,
     "credential_claim": [
       "sub"
     ],
     "disable_session": null,
     "discovery_headers_names": null,
     "discovery_headers_values": null,
     "display_errors": false,
     "domains": null,
     "downstream_access_token_header": null,
     "downstream_access_token_jwk_header": null,
     "downstream_headers_claims": null,
     "downstream_headers_names": null,
     "downstream_id_token_header": null,
     "downstream_id_token_jwk_header": null,
     "downstream_introspection_header": null,
     "downstream_introspection_jwt_header": null,
     "downstream_refresh_token_header": null,
     "downstream_session_id_header": null,
     "downstream_user_info_header": null,
     "downstream_user_info_jwt_header": null,
     "dpop_proof_lifetime": 300,
     "dpop_use_nonce": false,
     "enable_hs_signatures": false,
     "end_session_endpoint": null,
     "expose_error_code": true,
     "extra_jwks_uris": null,
     "forbidden_destroy_session": true,
     "forbidden_error_message": "Forbidden",
     "forbidden_redirect_uri": null,
     "groups_claim": [
       "groups"
     ],
     "groups_required": null,
     "hide_credentials": false,
     "http_proxy": null,
     "http_proxy_authorization": null,
     "http_version": 1.1,
     "https_proxy": null,
     "https_proxy_authorization": null,
     "id_token_param_name": null,
     "id_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "ignore_signature": [
     ],
     "introspect_jwt_tokens": false,
     "introspection_accept": "application/json",
     "introspection_check_active": true,
     "introspection_endpoint": null,
     "introspection_endpoint_auth_method": null,
     "introspection_headers_client": null,
     "introspection_headers_names": null,
     "introspection_headers_values": null,
     "introspection_hint": "access_token",
     "introspection_post_args_client": null,
     "introspection_post_args_names": null,
     "introspection_post_args_values": null,
     "introspection_token_param_name": "token",
     "issuer": "https://accounts.google.test/.well-known/openid-configuration",
     "issuers_allowed": null,
     "jwt_session_claim": "sid",
     "jwt_session_cookie": null,
     "keepalive": true,
     "leeway": 0,
     "login_action": "upstream",
     "login_methods": [
       "authorization_code"
     ],
     "login_redirect_mode": "fragment",
     "login_redirect_uri": null,
     "login_tokens": [
       "id_token"
     ],
     "logout_methods": [
       "POST",
       "DELETE"
     ],
     "logout_post_arg": null,
     "logout_query_arg": null,
     "logout_redirect_uri": null,
     "logout_revoke": false,
     "logout_revoke_access_token": true,
     "logout_revoke_refresh_token": true,
     "logout_uri_suffix": null,
     "max_age": null,
     "mtls_introspection_endpoint": null,
     "mtls_revocation_endpoint": null,
     "mtls_token_endpoint": null,
     "no_proxy": null,
     "password_param_type": [
       "header",
       "query",
       "body"
     ],
     "preserve_query_args": false,
     "proof_of_possession_auth_methods_validation": true,
     "proof_of_possession_dpop": "[masked]",
     "proof_of_possession_mtls": "[masked]",
     "pushed_authorization_request_endpoint": null,
     "pushed_authorization_request_endpoint_auth_method": null,
     "redirect_uri": null,
     "redis": {
       "connect_timeout": 2000,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.1",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6379,
       "prefix": null,
       "read_timeout": 2000,
       "send_timeout": 2000,
       "sentinel_master": null,
       "sentinel_nodes": null,
       "sentinel_password": null,
       "sentinel_role": null,
       "sentinel_username": null,
       "server_name": null,
       "socket": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "rediscovery_lifetime": 30,
     "refresh_token_param_name": null,
     "refresh_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "refresh_tokens": true,
     "require_proof_key_for_code_exchange": null,
     "require_pushed_authorization_requests": null,
     "require_signed_request_object": null,
     "resolve_distributed_claims": false,
     "response_mode": "query",
     "response_type": [
       "code"
     ],
     "reverify": false,
     "revocation_endpoint": null,
     "revocation_endpoint_auth_method": null,
     "revocation_token_param_name": "token",
     "roles_claim": [
       "roles"
     ],
     "roles_required": null,
     "run_on_preflight": true,
     "scopes": [
       "openid"
     ],
     "scopes_claim": [
       "scope"
     ],
     "scopes_required": null,
     "search_user_info": false,
     "session_absolute_timeout": 86400,
     "session_audience": "default",
     "session_cookie_domain": null,
     "session_cookie_http_only": true,
     "session_cookie_name": "session",
     "session_cookie_path": "/",
     "session_cookie_same_site": "Lax",
     "session_cookie_secure": null,
     "session_enforce_same_subject": false,
     "session_hash_storage_key": false,
     "session_hash_subject": false,
     "session_idling_timeout": 900,
     "session_memcached_host": "127.0.0.1",
     "session_memcached_port": 11211,
     "session_memcached_prefix": null,
     "session_memcached_socket": null,
-    "session_redis_cluster_max_redirections": null,
+    "session_redis_cluster_max_redirections": 7,
-    "session_redis_cluster_nodes": null,
+    "session_redis_cluster_nodes": [
+      {
+        "ip": "127.0.1.0",
+        "port": 6379
+      },
+      {
+        "ip": "127.0.1.0",
+        "port": 6380
+      },
+      {
+        "ip": "127.0.1.0",
+        "port": 6381
+      }
+    ],
     "session_remember": false,
     "session_remember_absolute_timeout": 2.592e+06,
     "session_remember_cookie_name": "remember",
     "session_remember_rolling_timeout": 604800,
     "session_request_headers": null,
     "session_response_headers": null,
     "session_rolling_timeout": 3600,
     "session_secret": null,
     "session_storage": "cookie",
     "session_store_metadata": false,
     "ssl_verify": false,
     "timeout": 10000,
     "tls_client_auth_cert_id": null,
     "tls_client_auth_ssl_verify": true,
     "token_cache_key_include_scope": false,
     "token_endpoint": null,
     "token_endpoint_auth_method": null,
     "token_exchange_endpoint": null,
     "token_headers_client": null,
     "token_headers_grants": null,
     "token_headers_names": null,
     "token_headers_prefix": null,
     "token_headers_replay": null,
     "token_headers_values": null,
     "token_post_args_client": null,
     "token_post_args_names": null,
     "token_post_args_values": null,
     "unauthorized_destroy_session": true,
     "unauthorized_error_message": "Unauthorized",
     "unauthorized_redirect_uri": null,
     "unexpected_redirect_uri": null,
     "upstream_access_token_header": "authorization:bearer",
     "upstream_access_token_jwk_header": null,
     "upstream_headers_claims": null,
     "upstream_headers_names": null,
     "upstream_id_token_header": null,
     "upstream_id_token_jwk_header": null,
     "upstream_introspection_header": null,
     "upstream_introspection_jwt_header": null,
     "upstream_refresh_token_header": null,
     "upstream_session_id_header": null,
     "upstream_user_info_header": null,
     "upstream_user_info_jwt_header": null,
     "userinfo_accept": "application/json",
     "userinfo_endpoint": null,
     "userinfo_headers_client": null,
     "userinfo_headers_names": null,
     "userinfo_headers_values": null,
     "userinfo_query_args_client": null,
     "userinfo_query_args_names": null,
     "userinfo_query_args_values": null,
     "using_pseudo_issuer": false,
     "verify_claims": true,
     "verify_nonce": true,
     "verify_parameters": false,
     "verify_signature": true
   },
   "enabled": true,
   "id": "777496e1-8b35-4512-ad30-51f9fe5d3147",
   "name": "openid-connect",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`
	expectedOutputPluginUpdateChangedOldFieldsRLA = `updating plugin rate-limiting-advanced (global)  {
   "config": {
     "consumer_groups": null,
     "dictionary_name": "kong_rate_limiting_counters",
     "disable_penalty": false,
     "enforce_consumer_groups": false,
     "error_code": 429,
     "error_message": "API rate limit exceeded",
     "header_name": null,
     "hide_client_headers": false,
     "identifier": "consumer",
     "limit": [
       10
     ],
     "namespace": "ZEz47TWgUrv01HenyQBQa8io06MWsp0L",
     "path": null,
     "redis": {
       "cluster_addresses": [
-        "127.0.1.0:6379",
+        "127.0.1.0:7379",
-        "127.0.1.0:6380",
+        "127.0.1.0:7380",
-        "127.0.1.0:6381"
+        "127.0.1.0:7381"
       ],
       "cluster_max_redirections": 5,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.5",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6380,
       "sentinel_addresses": [
-        "127.0.2.0:6379",
+        "127.0.2.0:8379",
-        "127.0.2.0:6380",
+        "127.0.2.0:8380",
-        "127.0.2.0:6381"
+        "127.0.2.0:8381"
       ],
       "sentinel_master": "mymaster",
       "sentinel_password": null,
       "sentinel_role": "master",
       "sentinel_username": null,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
-      "timeout": 2000,
+      "timeout": 2007,
       "username": null
     },
     "retry_after_jitter_max": 0,
     "strategy": "redis",
-    "sync_rate": 10,
+    "sync_rate": 11,
     "window_size": [
       60
     ],
     "window_type": "sliding"
   },
   "enabled": true,
   "id": "a1368a28-cb5c-4eee-86d8-03a6bdf94b5e",
   "name": "rate-limiting-advanced",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`

	expectedOutputPluginUpdateChangedNewFields36 = `updating plugin rate-limiting (global)  {
   "config": {
     "day": null,
     "error_code": 429,
     "error_message": "API rate limit exceeded",
     "fault_tolerant": true,
     "header_name": null,
     "hide_client_headers": false,
     "hour": 10000,
     "limit_by": "consumer",
     "minute": null,
     "month": null,
     "path": null,
     "policy": "redis",
     "redis": {
-      "database": 0,
+      "database": 3,
-      "host": "localhost",
+      "host": "localhost-3",
       "password": null,
       "port": 6379,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
       "timeout": 2000,
       "username": null
     },
     "second": null,
     "sync_rate": -1,
     "year": null
   },
   "enabled": true,
   "id": "2705d985-de4b-4ca8-87fd-2b361e30a3e7",
   "name": "rate-limiting",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`
	expectedOutputPluginUpdateChangedOldFields36 = `updating plugin rate-limiting (global)  {
   "config": {
     "day": null,
     "error_code": 429,
     "error_message": "API rate limit exceeded",
     "fault_tolerant": true,
     "header_name": null,
     "hide_client_headers": false,
     "hour": 10000,
     "limit_by": "consumer",
     "minute": null,
     "month": null,
     "path": null,
     "policy": "redis",
     "redis": {
       "password": null,
       "port": 6379,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
       "timeout": 2000,
       "username": null
     },
-    "redis_database": 0,
+    "redis_database": 2,
-    "redis_host": "localhost",
+    "redis_host": "localhost-2",
     "second": null,
     "sync_rate": -1,
     "year": null
   },
   "enabled": true,
   "id": "2705d985-de4b-4ca8-87fd-2b361e30a3e7",
   "name": "rate-limiting",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`

	expectedOutputPluginUpdateChangedNewFieldsOpenIdConnect39x = `updating plugin openid-connect (global)  {
   "config": {
     "anonymous": null,
     "audience": null,
     "audience_claim": [
       "aud"
     ],
     "audience_required": null,
     "auth_methods": [
       "password",
       "client_credentials",
       "authorization_code",
       "bearer",
       "introspection",
       "userinfo",
       "kong_oauth2",
       "refresh_token",
       "session"
     ],
     "authenticated_groups_claim": null,
     "authorization_cookie_domain": null,
     "authorization_cookie_http_only": true,
     "authorization_cookie_name": "authorization",
     "authorization_cookie_path": "/",
     "authorization_cookie_same_site": "Default",
     "authorization_cookie_secure": null,
     "authorization_endpoint": null,
     "authorization_query_args_client": null,
     "authorization_query_args_names": null,
     "authorization_query_args_values": null,
     "authorization_rolling_timeout": 600,
     "bearer_token_cookie_name": null,
     "bearer_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "by_username_ignore_case": false,
     "cache_introspection": true,
     "cache_token_exchange": true,
     "cache_tokens": true,
     "cache_tokens_salt": null,
     "cache_ttl": 3600,
     "cache_ttl_max": null,
     "cache_ttl_min": null,
     "cache_ttl_neg": null,
     "cache_ttl_resurrect": null,
     "cache_user_info": true,
     "claims_forbidden": null,
     "client_alg": null,
     "client_arg": "client_id",
     "client_auth": null,
     "client_credentials_param_type": [
       "header",
       "query",
       "body"
     ],
     "client_id": null,
     "client_jwk": null,
     "client_secret": null,
     "cluster_cache_redis": {
       "cluster_max_redirections": 5,
       "cluster_nodes": null,
       "connect_timeout": 2000,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.1",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6379,
       "read_timeout": 2000,
       "send_timeout": 2000,
       "sentinel_master": null,
       "sentinel_nodes": null,
       "sentinel_password": null,
       "sentinel_role": null,
       "sentinel_username": null,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "cluster_cache_strategy": "[masked]",
     "consumer_by": [
       "username",
       "custom_id"
     ],
     "consumer_claim": null,
     "consumer_optional": false,
     "credential_claim": [
       "sub"
     ],
     "disable_session": null,
     "discovery_headers_names": null,
     "discovery_headers_values": null,
     "display_errors": false,
     "domains": null,
     "downstream_access_token_header": null,
     "downstream_access_token_jwk_header": null,
     "downstream_headers_claims": null,
     "downstream_headers_names": null,
     "downstream_id_token_header": null,
     "downstream_id_token_jwk_header": null,
     "downstream_introspection_header": null,
     "downstream_introspection_jwt_header": null,
     "downstream_refresh_token_header": null,
     "downstream_session_id_header": null,
     "downstream_user_info_header": null,
     "downstream_user_info_jwt_header": null,
     "dpop_proof_lifetime": 300,
     "dpop_use_nonce": false,
     "enable_hs_signatures": false,
     "end_session_endpoint": null,
     "expose_error_code": true,
     "extra_jwks_uris": null,
     "forbidden_destroy_session": true,
     "forbidden_error_message": "Forbidden",
     "forbidden_redirect_uri": null,
     "groups_claim": [
       "groups"
     ],
     "groups_required": null,
     "hide_credentials": false,
     "http_proxy": null,
     "http_proxy_authorization": null,
     "http_version": 1.1,
     "https_proxy": null,
     "https_proxy_authorization": null,
     "id_token_param_name": null,
     "id_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "ignore_signature": [
     ],
     "introspect_jwt_tokens": false,
     "introspection_accept": "application/json",
     "introspection_check_active": true,
     "introspection_endpoint": null,
     "introspection_endpoint_auth_method": null,
     "introspection_headers_client": null,
     "introspection_headers_names": null,
     "introspection_headers_values": null,
     "introspection_hint": "access_token",
     "introspection_post_args_client": null,
     "introspection_post_args_client_headers": null,
     "introspection_post_args_names": null,
     "introspection_post_args_values": null,
     "introspection_token_param_name": "token",
     "issuer": "https://accounts.google.test/.well-known/openid-configuration",
     "issuers_allowed": null,
     "jwt_session_claim": "sid",
     "jwt_session_cookie": null,
     "keepalive": true,
     "leeway": 0,
     "login_action": "upstream",
     "login_methods": [
       "authorization_code"
     ],
     "login_redirect_mode": "fragment",
     "login_redirect_uri": null,
     "login_tokens": [
       "id_token"
     ],
     "logout_methods": [
       "POST",
       "DELETE"
     ],
     "logout_post_arg": null,
     "logout_query_arg": null,
     "logout_redirect_uri": null,
     "logout_revoke": false,
     "logout_revoke_access_token": true,
     "logout_revoke_refresh_token": true,
     "logout_uri_suffix": null,
     "max_age": null,
     "mtls_introspection_endpoint": null,
     "mtls_revocation_endpoint": null,
     "mtls_token_endpoint": null,
     "no_proxy": null,
     "password_param_type": [
       "header",
       "query",
       "body"
     ],
     "preserve_query_args": false,
     "proof_of_possession_auth_methods_validation": true,
     "proof_of_possession_dpop": "[masked]",
     "proof_of_possession_mtls": "[masked]",
     "pushed_authorization_request_endpoint": null,
     "pushed_authorization_request_endpoint_auth_method": null,
     "redirect_uri": null,
     "redis": {
-      "cluster_max_redirections": null,
+      "cluster_max_redirections": 11,
-      "cluster_nodes": null,
+      "cluster_nodes": [
+        {
+          "ip": "127.0.1.0",
+          "port": 7379
+        },
+        {
+          "ip": "127.0.1.0",
+          "port": 7380
+        },
+        {
+          "ip": "127.0.1.0",
+          "port": 7381
+        }
+      ],
       "connect_timeout": 2000,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.1",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6379,
       "prefix": null,
       "read_timeout": 2000,
       "send_timeout": 2000,
       "sentinel_master": null,
       "sentinel_nodes": null,
       "sentinel_password": null,
       "sentinel_role": null,
       "sentinel_username": null,
       "server_name": null,
       "socket": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "rediscovery_lifetime": 30,
     "refresh_token_param_name": null,
     "refresh_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "refresh_tokens": true,
     "require_proof_key_for_code_exchange": null,
     "require_pushed_authorization_requests": null,
     "require_signed_request_object": null,
     "resolve_distributed_claims": false,
     "response_mode": "query",
     "response_type": [
       "code"
     ],
     "reverify": false,
     "revocation_endpoint": null,
     "revocation_endpoint_auth_method": null,
     "revocation_token_param_name": "token",
     "roles_claim": [
       "roles"
     ],
     "roles_required": null,
     "run_on_preflight": true,
     "scopes": [
       "openid"
     ],
     "scopes_claim": [
       "scope"
     ],
     "scopes_required": null,
     "search_user_info": false,
     "session_absolute_timeout": 86400,
     "session_audience": "default",
     "session_cookie_domain": null,
     "session_cookie_http_only": true,
     "session_cookie_name": "session",
     "session_cookie_path": "/",
     "session_cookie_same_site": "Lax",
     "session_cookie_secure": null,
     "session_enforce_same_subject": false,
     "session_hash_storage_key": false,
     "session_hash_subject": false,
     "session_idling_timeout": 900,
     "session_memcached_host": "127.0.0.1",
     "session_memcached_port": 11211,
     "session_memcached_prefix": null,
     "session_memcached_socket": null,
     "session_remember": false,
     "session_remember_absolute_timeout": 2.592e+06,
     "session_remember_cookie_name": "remember",
     "session_remember_rolling_timeout": 604800,
     "session_request_headers": null,
     "session_response_headers": null,
     "session_rolling_timeout": 3600,
     "session_secret": null,
     "session_storage": "cookie",
     "session_store_metadata": false,
     "ssl_verify": false,
     "timeout": 10000,
     "tls_client_auth_cert_id": null,
     "tls_client_auth_ssl_verify": true,
     "token_cache_key_include_scope": false,
     "token_endpoint": null,
     "token_endpoint_auth_method": null,
     "token_exchange_endpoint": null,
     "token_headers_client": null,
     "token_headers_grants": null,
     "token_headers_names": null,
     "token_headers_prefix": null,
     "token_headers_replay": null,
     "token_headers_values": null,
     "token_post_args_client": null,
     "token_post_args_names": null,
     "token_post_args_values": null,
     "unauthorized_destroy_session": true,
     "unauthorized_error_message": "Unauthorized",
     "unauthorized_redirect_uri": null,
     "unexpected_redirect_uri": null,
     "upstream_access_token_header": "authorization:bearer",
     "upstream_access_token_jwk_header": null,
     "upstream_headers_claims": null,
     "upstream_headers_names": null,
     "upstream_id_token_header": null,
     "upstream_id_token_jwk_header": null,
     "upstream_introspection_header": null,
     "upstream_introspection_jwt_header": null,
     "upstream_refresh_token_header": null,
     "upstream_session_id_header": null,
     "upstream_user_info_header": null,
     "upstream_user_info_jwt_header": null,
     "userinfo_accept": "application/json",
     "userinfo_endpoint": null,
     "userinfo_headers_client": null,
     "userinfo_headers_names": null,
     "userinfo_headers_values": null,
     "userinfo_query_args_client": null,
     "userinfo_query_args_names": null,
     "userinfo_query_args_values": null,
     "using_pseudo_issuer": false,
     "verify_claims": true,
     "verify_nonce": true,
     "verify_parameters": false,
     "verify_signature": true
   },
   "enabled": true,
   "id": "777496e1-8b35-4512-ad30-51f9fe5d3147",
   "name": "openid-connect",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`
	expectedOutputPluginUpdateChangedNewFieldsRLA39x = `updating plugin rate-limiting-advanced (global)  {
   "config": {
     "compound_identifier": null,
     "consumer_groups": null,
     "dictionary_name": "kong_rate_limiting_counters",
     "disable_penalty": false,
     "enforce_consumer_groups": false,
     "error_code": 429,
     "error_message": "API rate limit exceeded",
     "header_name": null,
     "hide_client_headers": false,
     "identifier": "consumer",
     "limit": [
       10
     ],
     "lock_dictionary_name": "kong_locks",
     "namespace": "ZEz47TWgUrv01HenyQBQa8io06MWsp0L",
     "path": null,
     "redis": {
       "cluster_max_redirections": 5,
       "cluster_nodes": [
         {
           "ip": "127.0.1.0",
-          "port": 6379
+          "port": 7379
         },
         {
           "ip": "127.0.1.0",
-          "port": 6380
+          "port": 7380
         },
         {
           "ip": "127.0.1.0",
-          "port": 6381
+          "port": 7381
         }
       ],
-      "connect_timeout": 2000,
+      "connect_timeout": 2005,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.5",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6380,
-      "read_timeout": 2000,
+      "read_timeout": 2006,
       "redis_proxy_type": null,
-      "send_timeout": 2000,
+      "send_timeout": 2007,
       "sentinel_master": "mymaster",
       "sentinel_nodes": [
         {
           "host": "127.0.2.0",
-          "port": 6379
+          "port": 8379
         },
         {
           "host": "127.0.2.0",
-          "port": 6380
+          "port": 8380
         },
         {
           "host": "127.0.2.0",
-          "port": 6381
+          "port": 8381
         }
       ],
       "sentinel_password": null,
       "sentinel_role": "master",
       "sentinel_username": null,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "retry_after_jitter_max": 0,
     "strategy": "redis",
     "sync_rate": 10,
     "window_size": [
       60
     ],
     "window_type": "sliding"
   },
   "enabled": true,
   "id": "a1368a28-cb5c-4eee-86d8-03a6bdf94b5e",
   "name": "rate-limiting-advanced",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`
	expectedOutputPluginUpdateChangedOldFieldsOpenIdConnect39x = `updating plugin openid-connect (global)  {
   "config": {
     "anonymous": null,
     "audience": null,
     "audience_claim": [
       "aud"
     ],
     "audience_required": null,
     "auth_methods": [
       "password",
       "client_credentials",
       "authorization_code",
       "bearer",
       "introspection",
       "userinfo",
       "kong_oauth2",
       "refresh_token",
       "session"
     ],
     "authenticated_groups_claim": null,
     "authorization_cookie_domain": null,
     "authorization_cookie_http_only": true,
     "authorization_cookie_name": "authorization",
     "authorization_cookie_path": "/",
     "authorization_cookie_same_site": "Default",
     "authorization_cookie_secure": null,
     "authorization_endpoint": null,
     "authorization_query_args_client": null,
     "authorization_query_args_names": null,
     "authorization_query_args_values": null,
     "authorization_rolling_timeout": 600,
     "bearer_token_cookie_name": null,
     "bearer_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "by_username_ignore_case": false,
     "cache_introspection": true,
     "cache_token_exchange": true,
     "cache_tokens": true,
     "cache_tokens_salt": null,
     "cache_ttl": 3600,
     "cache_ttl_max": null,
     "cache_ttl_min": null,
     "cache_ttl_neg": null,
     "cache_ttl_resurrect": null,
     "cache_user_info": true,
     "claims_forbidden": null,
     "client_alg": null,
     "client_arg": "client_id",
     "client_auth": null,
     "client_credentials_param_type": [
       "header",
       "query",
       "body"
     ],
     "client_id": null,
     "client_jwk": null,
     "client_secret": null,
     "cluster_cache_redis": {
       "cluster_max_redirections": 5,
       "cluster_nodes": null,
       "connect_timeout": 2000,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.1",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6379,
       "read_timeout": 2000,
       "send_timeout": 2000,
       "sentinel_master": null,
       "sentinel_nodes": null,
       "sentinel_password": null,
       "sentinel_role": null,
       "sentinel_username": null,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "cluster_cache_strategy": "[masked]",
     "consumer_by": [
       "username",
       "custom_id"
     ],
     "consumer_claim": null,
     "consumer_optional": false,
     "credential_claim": [
       "sub"
     ],
     "disable_session": null,
     "discovery_headers_names": null,
     "discovery_headers_values": null,
     "display_errors": false,
     "domains": null,
     "downstream_access_token_header": null,
     "downstream_access_token_jwk_header": null,
     "downstream_headers_claims": null,
     "downstream_headers_names": null,
     "downstream_id_token_header": null,
     "downstream_id_token_jwk_header": null,
     "downstream_introspection_header": null,
     "downstream_introspection_jwt_header": null,
     "downstream_refresh_token_header": null,
     "downstream_session_id_header": null,
     "downstream_user_info_header": null,
     "downstream_user_info_jwt_header": null,
     "dpop_proof_lifetime": 300,
     "dpop_use_nonce": false,
     "enable_hs_signatures": false,
     "end_session_endpoint": null,
     "expose_error_code": true,
     "extra_jwks_uris": null,
     "forbidden_destroy_session": true,
     "forbidden_error_message": "Forbidden",
     "forbidden_redirect_uri": null,
     "groups_claim": [
       "groups"
     ],
     "groups_required": null,
     "hide_credentials": false,
     "http_proxy": null,
     "http_proxy_authorization": null,
     "http_version": 1.1,
     "https_proxy": null,
     "https_proxy_authorization": null,
     "id_token_param_name": null,
     "id_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "ignore_signature": [
     ],
     "introspect_jwt_tokens": false,
     "introspection_accept": "application/json",
     "introspection_check_active": true,
     "introspection_endpoint": null,
     "introspection_endpoint_auth_method": null,
     "introspection_headers_client": null,
     "introspection_headers_names": null,
     "introspection_headers_values": null,
     "introspection_hint": "access_token",
     "introspection_post_args_client": null,
     "introspection_post_args_client_headers": null,
     "introspection_post_args_names": null,
     "introspection_post_args_values": null,
     "introspection_token_param_name": "token",
     "issuer": "https://accounts.google.test/.well-known/openid-configuration",
     "issuers_allowed": null,
     "jwt_session_claim": "sid",
     "jwt_session_cookie": null,
     "keepalive": true,
     "leeway": 0,
     "login_action": "upstream",
     "login_methods": [
       "authorization_code"
     ],
     "login_redirect_mode": "fragment",
     "login_redirect_uri": null,
     "login_tokens": [
       "id_token"
     ],
     "logout_methods": [
       "POST",
       "DELETE"
     ],
     "logout_post_arg": null,
     "logout_query_arg": null,
     "logout_redirect_uri": null,
     "logout_revoke": false,
     "logout_revoke_access_token": true,
     "logout_revoke_refresh_token": true,
     "logout_uri_suffix": null,
     "max_age": null,
     "mtls_introspection_endpoint": null,
     "mtls_revocation_endpoint": null,
     "mtls_token_endpoint": null,
     "no_proxy": null,
     "password_param_type": [
       "header",
       "query",
       "body"
     ],
     "preserve_query_args": false,
     "proof_of_possession_auth_methods_validation": true,
     "proof_of_possession_dpop": "[masked]",
     "proof_of_possession_mtls": "[masked]",
     "pushed_authorization_request_endpoint": null,
     "pushed_authorization_request_endpoint_auth_method": null,
     "redirect_uri": null,
     "redis": {
       "connect_timeout": 2000,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.1",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6379,
       "prefix": null,
       "read_timeout": 2000,
       "send_timeout": 2000,
       "sentinel_master": null,
       "sentinel_nodes": null,
       "sentinel_password": null,
       "sentinel_role": null,
       "sentinel_username": null,
       "server_name": null,
       "socket": null,
       "ssl": false,
       "ssl_verify": false,
       "username": null
     },
     "rediscovery_lifetime": 30,
     "refresh_token_param_name": null,
     "refresh_token_param_type": [
       "header",
       "query",
       "body"
     ],
     "refresh_tokens": true,
     "require_proof_key_for_code_exchange": null,
     "require_pushed_authorization_requests": null,
     "require_signed_request_object": null,
     "resolve_distributed_claims": false,
     "response_mode": "query",
     "response_type": [
       "code"
     ],
     "reverify": false,
     "revocation_endpoint": null,
     "revocation_endpoint_auth_method": null,
     "revocation_token_param_name": "token",
     "roles_claim": [
       "roles"
     ],
     "roles_required": null,
     "run_on_preflight": true,
     "scopes": [
       "openid"
     ],
     "scopes_claim": [
       "scope"
     ],
     "scopes_required": null,
     "search_user_info": false,
     "session_absolute_timeout": 86400,
     "session_audience": "default",
     "session_cookie_domain": null,
     "session_cookie_http_only": true,
     "session_cookie_name": "session",
     "session_cookie_path": "/",
     "session_cookie_same_site": "Lax",
     "session_cookie_secure": null,
     "session_enforce_same_subject": false,
     "session_hash_storage_key": false,
     "session_hash_subject": false,
     "session_idling_timeout": 900,
     "session_memcached_host": "127.0.0.1",
     "session_memcached_port": 11211,
     "session_memcached_prefix": null,
     "session_memcached_socket": null,
-    "session_redis_cluster_max_redirections": null,
+    "session_redis_cluster_max_redirections": 7,
-    "session_redis_cluster_nodes": null,
+    "session_redis_cluster_nodes": [
+      {
+        "ip": "127.0.1.0",
+        "port": 6379
+      },
+      {
+        "ip": "127.0.1.0",
+        "port": 6380
+      },
+      {
+        "ip": "127.0.1.0",
+        "port": 6381
+      }
+    ],
     "session_remember": false,
     "session_remember_absolute_timeout": 2.592e+06,
     "session_remember_cookie_name": "remember",
     "session_remember_rolling_timeout": 604800,
     "session_request_headers": null,
     "session_response_headers": null,
     "session_rolling_timeout": 3600,
     "session_secret": null,
     "session_storage": "cookie",
     "session_store_metadata": false,
     "ssl_verify": false,
     "timeout": 10000,
     "tls_client_auth_cert_id": null,
     "tls_client_auth_ssl_verify": true,
     "token_cache_key_include_scope": false,
     "token_endpoint": null,
     "token_endpoint_auth_method": null,
     "token_exchange_endpoint": null,
     "token_headers_client": null,
     "token_headers_grants": null,
     "token_headers_names": null,
     "token_headers_prefix": null,
     "token_headers_replay": null,
     "token_headers_values": null,
     "token_post_args_client": null,
     "token_post_args_names": null,
     "token_post_args_values": null,
     "unauthorized_destroy_session": true,
     "unauthorized_error_message": "Unauthorized",
     "unauthorized_redirect_uri": null,
     "unexpected_redirect_uri": null,
     "upstream_access_token_header": "authorization:bearer",
     "upstream_access_token_jwk_header": null,
     "upstream_headers_claims": null,
     "upstream_headers_names": null,
     "upstream_id_token_header": null,
     "upstream_id_token_jwk_header": null,
     "upstream_introspection_header": null,
     "upstream_introspection_jwt_header": null,
     "upstream_refresh_token_header": null,
     "upstream_session_id_header": null,
     "upstream_user_info_header": null,
     "upstream_user_info_jwt_header": null,
     "userinfo_accept": "application/json",
     "userinfo_endpoint": null,
     "userinfo_headers_client": null,
     "userinfo_headers_names": null,
     "userinfo_headers_values": null,
     "userinfo_query_args_client": null,
     "userinfo_query_args_names": null,
     "userinfo_query_args_values": null,
     "using_pseudo_issuer": false,
     "verify_claims": true,
     "verify_nonce": true,
     "verify_parameters": false,
     "verify_signature": true
   },
   "enabled": true,
   "id": "777496e1-8b35-4512-ad30-51f9fe5d3147",
   "name": "openid-connect",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`
	expectedOutputPluginUpdateChangedOldFieldsRLA39x = `updating plugin rate-limiting-advanced (global)  {
   "config": {
     "compound_identifier": null,
     "consumer_groups": null,
     "dictionary_name": "kong_rate_limiting_counters",
     "disable_penalty": false,
     "enforce_consumer_groups": false,
     "error_code": 429,
     "error_message": "API rate limit exceeded",
     "header_name": null,
     "hide_client_headers": false,
     "identifier": "consumer",
     "limit": [
       10
     ],
     "lock_dictionary_name": "kong_locks",
     "namespace": "ZEz47TWgUrv01HenyQBQa8io06MWsp0L",
     "path": null,
     "redis": {
       "cluster_addresses": [
-        "127.0.1.0:6379",
+        "127.0.1.0:7379",
-        "127.0.1.0:6380",
+        "127.0.1.0:7380",
-        "127.0.1.0:6381"
+        "127.0.1.0:7381"
       ],
       "cluster_max_redirections": 5,
       "connection_is_proxied": false,
       "database": 0,
       "host": "127.0.0.5",
       "keepalive_backlog": null,
       "keepalive_pool_size": 256,
       "password": null,
       "port": 6380,
       "redis_proxy_type": null,
       "sentinel_addresses": [
-        "127.0.2.0:6379",
+        "127.0.2.0:8379",
-        "127.0.2.0:6380",
+        "127.0.2.0:8380",
-        "127.0.2.0:6381"
+        "127.0.2.0:8381"
       ],
       "sentinel_master": "mymaster",
       "sentinel_password": null,
       "sentinel_role": "master",
       "sentinel_username": null,
       "server_name": null,
       "ssl": false,
       "ssl_verify": false,
-      "timeout": 2000,
+      "timeout": 2007,
       "username": null
     },
     "retry_after_jitter_max": 0,
     "strategy": "redis",
-    "sync_rate": 10,
+    "sync_rate": 11,
     "window_size": [
       60
     ],
     "window_type": "sliding"
   },
   "enabled": true,
   "id": "a1368a28-cb5c-4eee-86d8-03a6bdf94b5e",
   "name": "rate-limiting-advanced",
   "protocols": [
     "grpc",
     "grpcs",
     "http",
     "https"
   ]
 }

Summary:
  Created: 0
  Updated: 1
  Deleted: 0
`
)

// test scope:
//   - 1.x
//   - 2.x
func Test_Diff_Workspace_OlderThan3x(t *testing.T) {
	tests := []struct {
		name          string
		stateFile     string
		expectedState utils.KongRawState
	}{
		{
			name:      "diff with not existent workspace doesn't error out",
			stateFile: "testdata/diff/001-not-existing-workspace/kong.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", "<3.0.0")
			setup(t)

			_, err := diff(tc.stateFile)
			require.NoError(t, err)
		})
	}
}

// test scope:
//   - 3.x
func Test_Diff_Workspace_NewerThan3x(t *testing.T) {
	tests := []struct {
		name          string
		stateFile     string
		expectedState utils.KongRawState
	}{
		{
			name:      "diff with not existent workspace doesn't error out",
			stateFile: "testdata/diff/001-not-existing-workspace/kong3x.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runWhen(t, "kong", ">=3.0.0")
			setup(t)

			_, err := diff(tc.stateFile)
			require.NoError(t, err)
		})
	}
}

// test scope:
//   - 2.8.0
func Test_Diff_Masked_OlderThan3x(t *testing.T) {
	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
		envVars          map[string]string
	}{
		{
			name:             "env variable are masked",
			initialStateFile: "testdata/diff/002-mask/initial.yaml",
			stateFile:        "testdata/diff/002-mask/kong.yaml",
			envVars:          diffEnvVars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", "==2.8.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile)
			require.NoError(t, err)
			assert.Equal(t, expectedOutputMasked, out)
		})
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", "==2.8.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile, "--json-output")
			require.NoError(t, err)
			assert.Equal(t, expectedOutputMaskedJSON, out)
		})
	}
}

// test scope:
//   - 3.x
func Test_Diff_Masked_NewerThan3x(t *testing.T) {
	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
		envVars          map[string]string
	}{
		{
			name:             "env variable are masked",
			initialStateFile: "testdata/diff/002-mask/initial3x.yaml",
			stateFile:        "testdata/diff/002-mask/kong3x.yaml",
			envVars:          diffEnvVars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.0.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile)
			require.NoError(t, err)
			assert.Equal(t, expectedOutputMasked, out)
		})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.0.0 <3.1.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile, "--json-output")
			require.NoError(t, err)
			assert.Equal(t, expectedOutputMaskedJSON30x, out)
		})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.1.0 <3.4.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile, "--json-output")
			require.NoError(t, err)
			assert.Equal(t, expectedOutputMaskedJSON, out)
		})
	}
}

// test scope:
//   - 2.8.0
func Test_Diff_Unmasked_OlderThan3x(t *testing.T) {
	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
		envVars          map[string]string
	}{
		{
			name:             "env variable are unmasked",
			initialStateFile: "testdata/diff/003-unmask/initial.yaml",
			stateFile:        "testdata/diff/003-unmask/kong.yaml",
			envVars:          diffEnvVars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", "==2.8.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile, "--no-mask-deck-env-vars-value")
			require.NoError(t, err)
			assert.Equal(t, expectedOutputUnMasked, out)
		})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", "==2.8.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile, "--no-mask-deck-env-vars-value", "--json-output")
			require.NoError(t, err)
			assert.Equal(t, expectedOutputUnMaskedJSON, out)
		})
	}
}

// test scope:
//   - 3.x
func Test_Diff_Unmasked_NewerThan3x(t *testing.T) {
	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedState    utils.KongRawState
		envVars          map[string]string
	}{
		{
			name:             "env variable are unmasked",
			initialStateFile: "testdata/diff/003-unmask/initial3x.yaml",
			stateFile:        "testdata/diff/003-unmask/kong3x.yaml",
			envVars:          diffEnvVars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.0.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile, "--no-mask-deck-env-vars-value")
			require.NoError(t, err)
			assert.Equal(t, expectedOutputUnMasked, out)
		})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.0.0 <3.1.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile, "--no-mask-deck-env-vars-value", "--json-output")
			require.NoError(t, err)
			assert.Equal(t, expectedOutputUnMaskedJSON30x, out)
		})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.1.0 <3.4.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile, "--no-mask-deck-env-vars-value", "--json-output")
			require.NoError(t, err)
			assert.Equal(t, expectedOutputUnMaskedJSON, out)
		})
	}
}

func Test_Diff_PluginUpdate_38x(t *testing.T) {
	runWhenEnterpriseOrKonnect(t, ">=3.8.0 <3.9.0")
	setup(t)

	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedDiff     string
	}{
		{
			name:             "initial setup sent twice - no diff expected",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee.yaml",
			stateFile:        "testdata/diff/004-plugin-update/initial-ee.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		{
			name:             "initial vs updating by sending only old fields with same values",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-no-change-old-fields.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		{
			name:             "initial vs updating by sending only new fields with same values",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-no-change-new-fields.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		// Determining order in which the plugins would be updated is not fixed.
		// Hence, the diff string checking can fail.
		// Thus, we are doing one plugin check at a time to avoid failures due to ordering.
		{
			name:             "initial vs updating by sending only old fields with new values - plugin openid-connect",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee-openid.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-changed-new-fields-openid.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedNewFieldsOpenIdConnect,
		},
		{
			name:             "initial vs updating by sending only old fields with new values - plugin rate-limiting-advanced",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee-rla.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-changed-new-fields-rla.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedNewFieldsRLA,
		},
		{
			name:             "initial vs updating by sending only new fields with new values - plugin openid-connect",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee-openid.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-changed-old-fields-openid.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedOldFieldsOpenIdConnect,
		},
		{
			name:             "initial vs updating by sending only new fields with new values - plugin rate-limiting-advanced",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee-rla.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-changed-old-fields-rla.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedOldFieldsRLA,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDiff, out)
		})
	}
}

func Test_Diff_PluginUpdate_NewerThan39x(t *testing.T) {
	runWhenEnterpriseOrKonnect(t, ">=3.9.0")
	setup(t)

	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedDiff     string
	}{
		{
			name:             "initial setup sent twice - no diff expected",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee.yaml",
			stateFile:        "testdata/diff/004-plugin-update/initial-ee.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		{
			name:             "initial vs updating by sending only old fields with same values",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-no-change-old-fields.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		{
			name:             "initial vs updating by sending only new fields with same values",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-no-change-new-fields.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		// Determining order in which the plugins would be updated is not fixed.
		// Hence, the diff string checking can fail.
		// Thus, we are doing one plugin check at a time to avoid failures due to ordering.
		{
			name:             "initial vs updating by sending only old fields with new values - plugin openid-connect",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee-openid.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-changed-new-fields-openid.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedNewFieldsOpenIdConnect39x,
		},
		{
			name:             "initial vs updating by sending only old fields with new values - plugin rate-limiting-advanced",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee-rla.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-changed-new-fields-rla.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedNewFieldsRLA39x,
		},
		{
			name:             "initial vs updating by sending only new fields with new values - plugin openid-connect",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee-openid.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-changed-old-fields-openid.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedOldFieldsOpenIdConnect39x,
		},
		{
			name:             "initial vs updating by sending only new fields with new values - plugin rate-limiting-advanced",
			initialStateFile: "testdata/diff/004-plugin-update/initial-ee-rla.yaml",
			stateFile:        "testdata/diff/004-plugin-update/kong-ee-changed-old-fields-rla.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedOldFieldsRLA39x,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDiff, out)
		})
	}
}

func Test_Diff_PluginUpdate_OlderThan38x(t *testing.T) {
	runWhen(t, "kong", ">=3.6.0 <3.8.0")
	setup(t)

	tests := []struct {
		name             string
		initialStateFile string
		stateFile        string
		expectedDiff     string
	}{
		{
			name:             "initial setup sent twice - no diff expected",
			initialStateFile: "testdata/diff/005-deprecated-fields/kong-initial.yaml",
			stateFile:        "testdata/diff/005-deprecated-fields/kong-initial.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		{
			name:             "initial vs updating by sending only old fields with same values",
			initialStateFile: "testdata/diff/005-deprecated-fields/kong-initial.yaml",
			stateFile:        "testdata/diff/005-deprecated-fields/kong-no-change-old-fields.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		{
			name:             "initial vs updating by sending only new fields with same values",
			initialStateFile: "testdata/diff/005-deprecated-fields/kong-initial.yaml",
			stateFile:        "testdata/diff/005-deprecated-fields/kong-no-change-new-fields.yaml",
			expectedDiff:     expectedOutputPluginUpdateNoChange,
		},
		{
			name:             "initial vs updating by sending only old fields with new values",
			initialStateFile: "testdata/diff/005-deprecated-fields/kong-initial.yaml",
			stateFile:        "testdata/diff/005-deprecated-fields/kong-update-old-fields.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedOldFields36,
		},
		{
			name:             "initial vs updating by sending only new fields with new values",
			initialStateFile: "testdata/diff/005-deprecated-fields/kong-initial.yaml",
			stateFile:        "testdata/diff/005-deprecated-fields/kong-update-new-fields.yaml",
			expectedDiff:     expectedOutputPluginUpdateChangedNewFields36,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			out, err := diff(tc.stateFile)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDiff, out)
		})
	}
}

func Test_Diff_NoDeletes_OlderThan3x(t *testing.T) {
	tests := []struct {
		name                string
		initialStateFile    string
		stateFile           string
		expectedState       utils.KongRawState
		envVars             map[string]string
		noDeletes           bool
		expectedDeleteCount int32
	}{
		{
			name:                "deleted plugins show in the diff by default",
			initialStateFile:    "testdata/diff/006-no-deletes/01-plugin-removed-initial.yaml",
			stateFile:           "testdata/diff/006-no-deletes/01-plugin-removed-current.yaml",
			envVars:             diffEnvVars,
			noDeletes:           false,
			expectedDeleteCount: 1,
		},
		{
			name:                "deleted plugins do not show in the diff",
			initialStateFile:    "testdata/diff/006-no-deletes/01-plugin-removed-initial.yaml",
			stateFile:           "testdata/diff/006-no-deletes/01-plugin-removed-current.yaml",
			envVars:             diffEnvVars,
			noDeletes:           true,
			expectedDeleteCount: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", "==2.8.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			client, err := getTestClient()
			ctx := context.Background()

			currentState, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: true})
			require.NoError(t, err)

			targetState := stateFromFile(ctx, t, tc.stateFile, client, deckDump.Config{
				IncludeLicenses: true,
			})

			syncer, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
				CurrentState: currentState,
				TargetState:  targetState,

				KongClient:      client,
				IncludeLicenses: true,
				NoDeletes:       tc.noDeletes,
			})

			stats, errs, _ := syncer.Solve(ctx, 1, false, true)
			assert.Equal(t, 0, len(errs))
			assert.Equal(t, tc.expectedDeleteCount, stats.DeleteOps.Count())
		})
	}
}

func Test_Diff_NoDeletes_3x(t *testing.T) {
	tests := []struct {
		name                string
		initialStateFile    string
		stateFile           string
		expectedState       utils.KongRawState
		envVars             map[string]string
		noDeletes           bool
		expectedDeleteCount int32
	}{
		{
			name:                "deleted plugins show in the diff by default",
			initialStateFile:    "testdata/diff/006-no-deletes/01-plugin-removed-initial.yaml",
			stateFile:           "testdata/diff/006-no-deletes/01-plugin-removed-current.yaml",
			envVars:             diffEnvVars,
			noDeletes:           false,
			expectedDeleteCount: 1,
		},
		{
			name:                "deleted plugins do not show in the diff",
			initialStateFile:    "testdata/diff/006-no-deletes/01-plugin-removed-initial.yaml",
			stateFile:           "testdata/diff/006-no-deletes/01-plugin-removed-current.yaml",
			envVars:             diffEnvVars,
			noDeletes:           true,
			expectedDeleteCount: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			runWhen(t, "kong", ">=3.0.0")
			setup(t)

			// initialize state
			require.NoError(t, sync(tc.initialStateFile))

			client, err := getTestClient()
			ctx := context.Background()

			currentState, err := fetchCurrentState(ctx, client, deckDump.Config{IncludeLicenses: true})
			require.NoError(t, err)

			targetState := stateFromFile(ctx, t, tc.stateFile, client, deckDump.Config{
				IncludeLicenses: true,
			})

			syncer, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
				CurrentState: currentState,
				TargetState:  targetState,

				KongClient:      client,
				IncludeLicenses: true,
				NoDeletes:       tc.noDeletes,
			})

			stats, errs, _ := syncer.Solve(ctx, 1, false, true)
			assert.Equal(t, 0, len(errs))
			assert.Equal(t, tc.expectedDeleteCount, stats.DeleteOps.Count())
		})
	}
}

func Test_Diff_Partials(t *testing.T) {
	runWhen(t, "enterprise", ">=3.10.0")
	client, err := getTestClient()
	require.NoError(t, err)

	ctx := context.Background()
	dumpConfig := deckDump.Config{}

	mustResetKongState(ctx, t, client, dumpConfig)
	currentState, err := fetchCurrentState(ctx, client, dumpConfig)
	require.NoError(t, err)

	targetState := stateFromFile(ctx, t, "testdata/sync/038-partials/kong.yaml", client, dumpConfig)
	syncer, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
		CurrentState: currentState,
		TargetState:  targetState,

		KongClient: client,
	})
	require.NoError(t, err)

	stats, errs, changes := syncer.Solve(ctx, 1, true, true)
	require.Empty(t, errs, "Should have no errors in diffing")
	logEntityChanges(t, stats, changes)

	assert.Equal(t, int32(2), stats.CreateOps.Count())
	assert.Equal(t, int32(0), stats.DeleteOps.Count())
	assert.Equal(t, int32(0), stats.UpdateOps.Count())
}
