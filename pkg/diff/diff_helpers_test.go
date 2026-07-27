package diff

import (
	"strings"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-database-reconciler/pkg/state"
)

const (
	testCertFull = `"-----BEGIN CERTIFICATE-----\n` +
		`MIIC/zCCAeegAwIBAgIUM/0MUZ+PAmeXXrzFb1pKkfzZbEkwDQYJKoZIhvcNAQEL\n` +
		`BQAwDzENMAsGA1UEAwwEdGVzdDAeFw0yNjA3MDgwNzA1NTBaFw0yNzA3MDgwNzA1\n` +
		`NTBaMA8xDTALBgNVBAMMBHRlc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEK\n` +
		`AoIBAQCm7M8qWILmeFftsYEZbDJILZN1J7fXaA0Dd6QsgZi63/bJV6f2qE892pUY\n` +
		`TO5M/paORzziovA0T97o54fyQIl7DE+p+Vt8p/rzn1QjVzUI8jiDIItj2nZwLPa3\n` +
		`wMQ0BLDcm9YX32yfLFZF3qEw8rKRk5O8DcIdOVMlBS5ZWC99fhtHDlNfSwen+Ypu\n` +
		`4IDUF9M35iRGEqf8DycCm43awpyCx7MTZFqZANeCm5Lj3LFmGR9F9Y9hoo7C9ZiT\n` +
		`ZmtA66BUmIpIqiiAFhplljxlO0FsLIGsrctz6QWcLGUQd6uU6+LeH4IOE+YfABpm\n` +
		`49w8gcF+scxxEATUucKErjf1C8bnAgMBAAGjUzBRMB0GA1UdDgQWBBQiUBlyBAhT\n` +
		`Zg7mWhhxszE6+XvRhjAfBgNVHSMEGDAWgBQiUBlyBAhTZg7mWhhxszE6+XvRhjAP\n` +
		`BgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCNSPXHZ2xCD6oYQs85\n` +
		`fR/cdmIcMcOr0XdAuIjHxZfUAzM1M1jHffHAEfAUKZQQ9mAnP8ue8x/euEcVrhuG\n` +
		`LSQbXS/nz9JvqECnlosgOBX0IVn4IsKCc0l8V54ovysDbWBOsWjIncZg+gKWjB5M\n` +
		`VnRce2rIj+B7D8gAIRnDA4tNE6u/OZtNUxBu4Rycy0jbBleGu1OSCGghdiSmPjJE\n` +
		`mp2w4FIXIFOvH/YEX87VInipnr7y4YmyTp615lb6BVptU5vceGYS3CGzJotNZ17O\n` +
		`Ir0u1oTlaV5o+Ly3vFawCTd1/iwfuzbrVvhrtPu7/82uSJ4oaE1IeVZ1iXNG/BwS\n` +
		`OFJW\n-----END CERTIFICATE-----"`

	testCertShort = `"-----BEGIN CERTIFICATE-----\n` +
		`MIIC/zCCAeegAwIBAgIUM/0MUZ+PAmeXXrzFb1pKkfzZbEkwDQYJKoZIhvcNAQEL\n` +
		`BQAwDzENMAsGA1UEAwwEdGVzdDAeFw0yNjA3MDgwNzA1NTBaFw0yNzA3MDgwNzA1\n` +
		`NTBaMA8xDTALBgNVBAMMBHRlc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEK\n` +
		`AoIBAQCm7M8qWILmeFftsYEZbDJILZN1J7fXaA0Dd6QsgZi63/bJV6f2qE892pUY\n` +
		`-----END CERTIFICATE-----"`

	testKeyShort = `"-----BEGIN PRIVATE KEY-----\n` +
		`MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCm7M8qWILmeFft\n` +
		`sYEZbDJILZN1J7fXaA0Dd6QsgZi63/bJV6f2qE892pUYTO5M/paORzziovA0T97o\n` +
		`54fyQIl7DE+p+Vt8p/rzn1QjVzUI8jiDIItj2nZwLPa3wMQ0BLDcm9YX32yfLFZF\n` +
		`-----END PRIVATE KEY-----"`
)

func Test_PrettyPrintJSONString(t *testing.T) {
	type args struct {
		jstring string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "basic JSON string",
			args: args{
				jstring: `{"foo":"foo","bar":{"a": 1, "b": 2}}`,
			},
			want: `{
	"bar": {
		"a": 1,
		"b": 2
	},
	"foo": "foo"
}`,
			wantErr: false,
		},
		{
			name: "invalid JSON string",
			args: args{
				jstring: "a large swarm of bees",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prettyPrintJSONString(tt.args.jstring)
			if (err != nil) != tt.wantErr {
				t.Errorf("prettyPrintJSONString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("prettyPrintJSONString() = %v\nwant %v", got, tt.want)
			}
		})
	}
}

func Test_GetDocumentDiff(t *testing.T) {
	type args struct {
		docA *state.Document
		docB *state.Document
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "JSON",
			args: args{
				docA: &state.Document{
					Document: konnect.Document{
						Path: new("foo"),
						Parent: &konnect.ServiceVersion{
							ID: new("abc"),
						},
						Content: new(`{"foo":"foo","bar":"bar"}`),
					},
				},
				docB: &state.Document{
					Document: konnect.Document{
						Path: new("foo"),
						Parent: &konnect.ServiceVersion{
							ID: new("abc"),
						},
						Content: new(`{"foo":"foo","bar":"bar","baz":"baz"}`),
					},
				},
			},
			want: ` {
   "path": "foo"
 }
--- old
+++ new
@@ -1,4 +1,5 @@
 {
 	"bar": "bar",
+	"baz": "baz",
 	"foo": "foo"
 }
\ No newline at end of file
`,
		},
		{
			name: "not JSON",
			args: args{
				docA: &state.Document{
					Document: konnect.Document{
						Path: new("foo"),
						Parent: &konnect.ServiceVersion{
							ID: new("abc"),
						},
						Content: new(`foo
`),
					},
				},
				docB: &state.Document{
					Document: konnect.Document{
						Path: new("foo"),
						Parent: &konnect.ServiceVersion{
							ID: new("abc"),
						},
						Content: new(`foo
bar
`),
					},
				},
			},
			want: ` {
   "path": "foo"
 }
--- old
+++ new
@@ -1 +1,2 @@
 foo
+bar
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := getDocumentDiff(tt.args.docA, tt.args.docB); got != tt.want {
				t.Errorf("getDocumentDiff() = %v\nwant %v", got, tt.want)
			}
		})
	}
}

func Test_MaskEnvVarsValues(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    string
		envVars map[string]string
	}{
		{
			name: "JSON string values",
			envVars: map[string]string{
				"DECK_BAR": "barbar",
				"DECK_BAZ": "bazbaz",
			},
			args: `{"foo":"foo","bar":"barbar","baz":"bazbaz"}`,
			want: `{"foo":"foo","bar":"[masked]","baz":"[masked]"}`,
		},
		{
			name: "JSON integer values produce valid JSON",
			envVars: map[string]string{
				"DECK_REDIS_DB":  "0",
				"DECK_SYNC_RATE": "1",
				"DECK_RETRIES":   "2",
				"DECK_CACHE_EXP": "5",
			},
			args: `{"id": "b35b3ec2-fa1c-4f6c-825e-c38141562c76", "retries": 2, "redis_database": 0}`,
			want: `{"id": "b35b3ec2-fa1c-4f6c-825e-c38141562c76", "retries": "[masked]", "redis_database": "[masked]"}`,
		},
		{
			name: "short values do not corrupt UUIDs or substrings",
			envVars: map[string]string{
				"DECK_REDIS_DB": "0",
			},
			args: `{"id": "b35b3ec2-fa1c-4f6c-825e-c38141562c76", "name": "my-service-01", "port": 8000}`,
			want: `{"id": "b35b3ec2-fa1c-4f6c-825e-c38141562c76", "name": "my-service-01", "port": 8000}`,
		},
		{
			name: "diff format with markers",
			envVars: map[string]string{
				"DECK_SECRET": "mysecretvalue",
			},
			args: ` {
   "name": "my-plugin",
-  "config.secret": "mysecretvalue",
+  "config.secret": "newsecretvalue"
 }`,
			want: ` {
   "name": "my-plugin",
-  "config.secret": "[masked]",
+  "config.secret": "newsecretvalue"
 }`,
		},
		{
			name: "YAML unquoted values in unified diff",
			envVars: map[string]string{
				"DECK_SECRET":  "mysecretvalue",
				"DECK_API_KEY": "sk-1234567890abcdef",
			},
			args: `--- old
+++ new
@@ -1,4 +1,4 @@
 name: my-service
-secret: mysecretvalue
+secret: newsecretvalue
 api_key: sk-1234567890abcdef
 port: 8080`,
			want: `--- old
+++ new
@@ -1,4 +1,4 @@
 name: my-service
-secret: [masked]
+secret: newsecretvalue
 api_key: [masked]
 port: 8080`,
		},
		{
			name: "YAML short numeric values do not corrupt other values",
			envVars: map[string]string{
				"DECK_REDIS_DB": "0",
				"DECK_RETRIES":  "5",
			},
			args: `--- old
+++ new
@@ -1,3 +1,3 @@
 name: my-service-500
 redis_database: 0
 retries: 5`,
			want: `--- old
+++ new
@@ -1,3 +1,3 @@
 name: my-service-500
 redis_database: [masked]
 retries: [masked]`,
		},
		{
			name: "PEM cert is masked when DECK_CLIENT_CERT env var is set (valid cert with proper base64)",
			envVars: map[string]string{
				"DECK_CLIENT_CERT": testCertFull,
			},
			args: ` {
   "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
   "cert": "-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIUM/0MUZ+PAmeXXrzFb1pKkfzZbEkwDQYJKoZIhvcNAQEL
BQAwDzENMAsGA1UEAwwEdGVzdDAeFw0yNjA3MDgwNzA1NTBaFw0yNzA3MDgwNzA1
NTBaMA8xDTALBgNVBAMMBHRlc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEK
AoIBAQCm7M8qWILmeFftsYEZbDJILZN1J7fXaA0Dd6QsgZi63/bJV6f2qE892pUY
-----END CERTIFICATE-----"
 }`,
			want: ` {
   "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
   "cert": "[masked]"
 }`,
		},
		{
			name: "rotated PEM cert - old value in minus line is masked even though it is not the current env var value",
			envVars: map[string]string{
				"DECK_CLIENT_CERT": testCertShort,
			},
			args: ` {
-  "cert": "-----BEGIN CERTIFICATE-----
OLDCERT
-----END CERTIFICATE-----",
+  "cert": "-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIUM/0MUZ+PAmeXXrzFb1pKkfzZbEkwDQYJKoZIhvcNAQEL
-----END CERTIFICATE-----"
 }`,
			want: ` {
-  "cert": "[masked]",
+  "cert": "[masked]"
 }`,
		},
		{
			name: "cert and private key both masked when both DECK_CLIENT_CERT and DECK_CLIENT_KEY are set",
			envVars: map[string]string{
				"DECK_CLIENT_CERT": testCertShort,
				"DECK_CLIENT_KEY":  testKeyShort,
			},
			args: ` {
   "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
-  "cert": "-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIUM/0MUZ+PAmeXXrzFb1pKkfzZbEkwDQYJKoZIhvcNAQEL
-----END CERTIFICATE-----",
-  "key": "-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCm7M8qWILmeFft
-----END PRIVATE KEY-----"
 }`,
			want: ` {
   "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
-  "cert": "[masked]",
-  "key": "[masked]"
 }`,
		},
		{
			name: "non-PEM secret alongside PEM cert - both masked, non-secret fields untouched",
			envVars: map[string]string{
				"DECK_CLIENT_CERT":    testCertShort,
				"DECK_REDIS_PASSWORD": "supersecretredispassword",
			},
			args: ` {
   "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
   "redis_password": "supersecretredispassword",
   "cert": "-----BEGIN CERTIFICATE-----
MIIC/zCCAeegAwIBAgIUM/0MUZ+PAmeXXrzFb1pKkfzZbEkwDQYJKoZIhvcNAQEL
-----END CERTIFICATE-----"
 }`,
			want: ` {
   "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
   "redis_password": "[masked]",
   "cert": "[masked]"
 }`,
		},
		{
			name: "JWK (JSON Web Key) on single line is masked",
			envVars: map[string]string{
				"DECK_JWK": `{"kty":"RSA","kid":"42","n":"abc123","e":"AQAB"}`,
			},
			args: ` {
   "id": "key-001",
   "jwk": {"kty":"RSA","kid":"42","n":"abc123","e":"AQAB"}
 }`,
			want: ` {
   "id": "key-001",
   "jwk": [masked]
 }`,
		},
		{
			name: "JWK with special characters in values (escaped quotes)",
			envVars: map[string]string{
				"DECK_JWK": `{"kty":"RSA","name":"key\"with\"quotes","e":"AQAB"}`,
			},
			args: `{"key":{"kty":"RSA","name":"key\"with\"quotes","e":"AQAB"}}`,
			want: `{"key":[masked]}`,
		},
		{
			name: "JWK with braces in string values",
			envVars: map[string]string{
				"DECK_JWK": `{"kty":"RSA","metadata":"value}with}braces","e":"AQAB"}`,
			},
			args: ` {
   "jwk": {"kty":"RSA","metadata":"value}with}braces","e":"AQAB"}
 }`,
			want: ` {
   "jwk": [masked]
 }`,
		},
		{
			name: "JWK with nested objects (one level)",
			envVars: map[string]string{
				"DECK_JWK": `{"kty":"RSA","headers":{"alg":"RS256"},"e":"AQAB"}`,
			},
			args: ` {
   "id": "key-002",
   "jwk": {"kty":"RSA","headers":{"alg":"RS256"},"e":"AQAB"}
 }`,
			want: ` {
   "id": "key-002",
   "jwk": [masked]
 }`,
		},
		{
			name: "Multiple JWKs in same diff - all masked when any env var is JWK (security: prevents leaking similar keys)",
			envVars: map[string]string{
				"DECK_JWK_SIGN": `{"kty":"RSA","use":"sig","e":"AQAB"}`,
			},
			args: ` {
   "signing_key": {"kty":"RSA","use":"sig","e":"AQAB"},
   "encryption_key": {"kty":"RSA","use":"enc","e":"AQAB"}
 }`,
			want: ` {
   "signing_key": [masked],
   "encryption_key": [masked]
 }`,
		},
		{
			name: "JWK array with multiple objects - all masked when any env var is JWK",
			envVars: map[string]string{
				"DECK_JWK": `{"kty":"EC","crv":"P-256","e":"AQAB"}`,
			},
			args: `"keys": [{"kty":"EC","crv":"P-256","e":"AQAB"},{"kty":"RSA","e":"AQAB"}]`,
			want: `"keys": [[masked],[masked]]`,
		},
		{
			name: "JWK in YAML-like format with trailing comma and quote",
			envVars: map[string]string{
				"DECK_JWK": `{"kty":"RSA","n":"modulus","e":"AQAB"}`,
			},
			args: ` {
   "algorithms": ["RS256"],
   "jwks": {"kty":"RSA","n":"modulus","e":"AQAB"},
 }`,
			want: ` {
   "algorithms": ["RS256"],
   "jwks": [masked],
 }`,
		},
		{
			name: "Invalid JWK (missing kty) is not masked - only PEM/real JWK",
			envVars: map[string]string{
				"DECK_OBJECT": `{"name":"test","value":"data"}`,
			},
			args: `{"key":{"name":"test","value":"data"}}`,
			want: `{"key":{"name":"test","value":"data"}}`,
		},
		{
			name: "Multi-line JWK with newlines is masked",
			envVars: map[string]string{
				"DECK_JWK_MULTILINE": `{"kty":"RSA","use":"sig","kid":"key-1",` +
					`"n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR",` +
					`"e":"AQAB"}`,
			},
			args: ` {
   "id": "key-001",
   "jwk": {"kty":"RSA","use":"sig","kid":"key-1",
   "n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR",
   "e":"AQAB"}
 }`,
			want: ` {
   "id": "key-001",
   "jwk": [masked]
 }`,
		},
		{
			name: "Multi-line JWK in diff with +/- markers is masked",
			envVars: map[string]string{
				"DECK_JWK_KEY": `{"kty":"EC","crv":"P-256","use":"sig","e":"AQAB"}`,
			},
			args: ` {
   "id": "key-001",
-  "jwk": {"kty":"EC","crv":"P-256","use":"sig",
-  "e":"AQAB"},
+  "jwk": {"kty":"EC","crv":"P-256","use":"enc",
+  "e":"AQAB"}
 }`,
			want: ` {
   "id": "key-001",
-  "jwk": [masked],
+  "jwk": [masked]
 }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			if got := MaskEnvVarValue(tt.args); got != tt.want {
				t.Errorf("maskEnvVarValue() = %v\nwant %v", got, tt.want)
			}
		})
	}
}

// TestSecretVsNonSecretVariableDistinction verifies that the system correctly
// distinguishes between secret and non-secret DECK variables. Non-secret variables
// (configuration items, flags, paths, URLs) should NOT appear in masked output,
// while actual secrets (tokens, passwords, API keys) SHOULD be masked.
func TestSecretVsNonSecretVariableDistinction(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		input          string
		shouldBeMasked bool
	}{
		{
			name: "Non-secret flag DECK_ANALYTICS should not be masked",
			envVars: map[string]string{
				"DECK_ANALYTICS": "true",
			},
			input:          `"analytics": "true"`,
			shouldBeMasked: false,
		},
		{
			name: "Non-secret URL DECK_KONNECT_ADDR should not be masked",
			envVars: map[string]string{
				"DECK_KONNECT_ADDR": "https://konnect.example.com",
			},
			input:          `"addr": "https://konnect.example.com"`,
			shouldBeMasked: false,
		},
		{
			name: "Secret password DECK_DATABASE_PASSWORD should be masked",
			envVars: map[string]string{
				"DECK_DATABASE_PASSWORD": "super_secret_pass",
			},
			input:          `"password": "super_secret_pass"`,
			shouldBeMasked: true,
		},
		{
			name: "Secret API key DECK_API_TOKEN should be masked",
			envVars: map[string]string{
				"DECK_API_TOKEN": "sk-1234567890abcdefghijklmnop",
			},
			input:          `"api_key": "sk-1234567890abcdefghijklmnop"`,
			shouldBeMasked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			output := MaskEnvVarValue(tt.input)
			isMasked := strings.Contains(output, "[masked]")

			if isMasked != tt.shouldBeMasked {
				if tt.shouldBeMasked {
					t.Errorf("Expected %q to be masked, but got: %s", tt.input, output)
				} else {
					t.Errorf("Expected %q to NOT be masked, but got: %s", tt.input, output)
				}
			}
		})
	}
}
