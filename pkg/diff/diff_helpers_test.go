package diff

import (
	"testing"

	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
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
						Path: kong.String("foo"),
						Parent: &konnect.ServiceVersion{
							ID: kong.String("abc"),
						},
						Content: kong.String(`{"foo":"foo","bar":"bar"}`),
					},
				},
				docB: &state.Document{
					Document: konnect.Document{
						Path: kong.String("foo"),
						Parent: &konnect.ServiceVersion{
							ID: kong.String("abc"),
						},
						Content: kong.String(`{"foo":"foo","bar":"bar","baz":"baz"}`),
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
						Path: kong.String("foo"),
						Parent: &konnect.ServiceVersion{
							ID: kong.String("abc"),
						},
						Content: kong.String(`foo
`),
					},
				},
				docB: &state.Document{
					Document: konnect.Document{
						Path: kong.String("foo"),
						Parent: &konnect.ServiceVersion{
							ID: kong.String("abc"),
						},
						Content: kong.String(`foo
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
