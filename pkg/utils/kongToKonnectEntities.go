package utils

// Some entities in Konnect have different names compared to Kong Gateway
var KongToKonnectEntitiesMap = map[string]string{
	"services":              "service",
	"routes":                "route",
	"upstreams":             "upstream",
	"targets":               "target",
	"jwt_secrets":           "jwt",
	"consumers":             "consumer",
	"consumer_groups":       "consumer_group",
	"certificates":          "certificate",
	"ca_certificates":       "ca_certificate",
	"keys":                  "key",
	"key_sets":              "key-set",
	"hmacauth_credentials":  "hmac-auth",
	"basicauth_credentials": "basic-auth",
	"mtls_auth_credentials": "mtls-auth",
	"snis":                  "sni",
}
