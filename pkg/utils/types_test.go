package utils

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrArrayString(t *testing.T) {
	assert := assert.New(t)
	var err ErrArray
	assert.Equal("nil", err.Error())

	err.Errors = append(err.Errors, fmt.Errorf("foo failed"))

	assert.Equal("1 errors occurred:\n\tfoo failed\n", err.Error())

	err.Errors = append(err.Errors, fmt.Errorf("bar failed"))

	assert.Equal("2 errors occurred:\n\tfoo failed\n\tbar failed\n", err.Error())
}

func Test_cleanAddress(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{
				address: "foo",
			},
			want: "foo",
		},
		{
			args: args{
				address: "http://localhost:8001",
			},
			want: "http://localhost:8001",
		},
		{
			args: args{
				address: "http://localhost:8001/",
			},
			want: "http://localhost:8001",
		},
		{
			args: args{
				address: "http://localhost:8001//",
			},
			want: "http://localhost:8001",
		},
		{
			args: args{
				address: "https://subdomain.example.com///",
			},
			want: "https://subdomain.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanAddress(tt.args.address); got != tt.want {
				t.Errorf("cleanAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseHeaders(t *testing.T) {
	type args struct {
		headers []string
	}
	tests := []struct {
		name    string
		args    args
		want    http.Header
		wantErr bool
	}{
		{
			name: "nil headers returns without an error",
			args: args{
				headers: nil,
			},
			want:    http.Header{},
			wantErr: false,
		},
		{
			name: "empty headers returns without an error",
			args: args{
				headers: []string{},
			},
			want:    http.Header{},
			wantErr: false,
		},
		{
			name: "headers returns without an error",
			args: args{
				headers: []string{
					"foo:bar",
					"baz:fubar",
				},
			},
			want: http.Header{
				"Foo": []string{"bar"},
				"Baz": []string{"fubar"},
			},
			wantErr: false,
		},
		{
			name: "invalid headers value returns an error",
			args: args{
				headers: []string{
					"fubar",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHeaders(tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHeaders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPClient(t *testing.T) {
	client := HTTPClient()

	require.NotNil(t, client)
	assert.Equal(t, defaultHTTPClientTimeout, client.Timeout)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport)
	assert.NotNil(t, transport.Proxy)
}

func TestHTTPClientWithOpts(t *testing.T) {
	tests := []struct {
		name        string
		opts        HTTPClientOptions
		wantErr     bool
		wantTimeout time.Duration
		checkTLS    func(t *testing.T, tlsCfg *tls.Config)
	}{
		{
			name:        "zero timeout defaults to 30s",
			opts:        HTTPClientOptions{},
			wantTimeout: defaultHTTPClientTimeout,
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				require.Nil(t, tlsCfg)
			},
		},
		{
			name:        "custom timeout is set",
			opts:        HTTPClientOptions{Timeout: 10 * time.Second},
			wantTimeout: 10 * time.Second,
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				require.Nil(t, tlsCfg)
			},
		},
		{
			name: "TLS skip verify is set",
			opts: HTTPClientOptions{
				TLSConfig: TLSConfig{SkipVerify: true},
			},
			wantTimeout: defaultHTTPClientTimeout,
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				require.NotNil(t, tlsCfg)
				assert.True(t, tlsCfg.InsecureSkipVerify)
			},
		},
		{
			name: "TLS server name is set",
			opts: HTTPClientOptions{
				TLSConfig: TLSConfig{ServerName: "example.com"},
			},
			wantTimeout: defaultHTTPClientTimeout,
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				require.NotNil(t, tlsCfg)
				assert.Equal(t, "example.com", tlsCfg.ServerName)
			},
		},
		{
			name: "TLS skip verify and server name are set together",
			opts: HTTPClientOptions{
				TLSConfig: TLSConfig{SkipVerify: true, ServerName: "kong.example.com"},
			},
			wantTimeout: defaultHTTPClientTimeout,
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				require.NotNil(t, tlsCfg)
				assert.True(t, tlsCfg.InsecureSkipVerify)
				assert.Equal(t, "kong.example.com", tlsCfg.ServerName)
			},
		},
		{
			name: "custom timeout and TLS are set together",
			opts: HTTPClientOptions{
				Timeout:   15 * time.Second,
				TLSConfig: TLSConfig{SkipVerify: true, ServerName: "example.com"},
			},
			wantTimeout: 15 * time.Second,
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				require.NotNil(t, tlsCfg)
				assert.True(t, tlsCfg.InsecureSkipVerify)
				assert.Equal(t, "example.com", tlsCfg.ServerName)
			},
		},
		{
			name: "invalid CA cert returns error",
			opts: HTTPClientOptions{
				TLSConfig: TLSConfig{CACert: "not-a-valid-pem"},
			},
			wantErr: true,
		},
		{
			name: "invalid client cert and key returns error",
			opts: HTTPClientOptions{
				TLSConfig: TLSConfig{ClientCert: "bad-cert", ClientKey: "bad-key"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := HTTPClientWithOpts(tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, client)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, tt.wantTimeout, client.Timeout)

			transport, ok := client.Transport.(*http.Transport)
			require.True(t, ok)
			require.NotNil(t, transport)
			assert.NotNil(t, transport.Proxy)

			if tt.checkTLS != nil {
				tt.checkTLS(t, transport.TLSClientConfig)
			}
		})
	}
}

func TestHTTPClientWithTLSConfig(t *testing.T) {
	tests := []struct {
		name     string
		opt      TLSConfig
		wantErr  bool
		checkTLS func(t *testing.T, tlsCfg *tls.Config)
	}{
		{
			name: "empty TLS config produces client with nil TLS config",
			opt:  TLSConfig{},
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				assert.Nil(t, tlsCfg)
			},
		},
		{
			name: "skip verify is propagated",
			opt:  TLSConfig{SkipVerify: true},
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				assert.NotNil(t, tlsCfg)
				assert.True(t, tlsCfg.InsecureSkipVerify)
			},
		},
		{
			name: "server name is propagated",
			opt:  TLSConfig{ServerName: "example.com"},
			checkTLS: func(t *testing.T, tlsCfg *tls.Config) {
				assert.NotNil(t, tlsCfg)
				assert.Equal(t, "example.com", tlsCfg.ServerName)
			},
		},
		{
			name:    "invalid CA cert returns error",
			opt:     TLSConfig{CACert: "not-a-valid-pem"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := HTTPClientWithTLSConfig(tt.opt)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, client)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, defaultHTTPClientTimeout, client.Timeout)

			transport, ok := client.Transport.(*http.Transport)
			assert.True(t, ok)
			assert.NotNil(t, transport)

			if tt.checkTLS != nil {
				tt.checkTLS(t, transport.TLSClientConfig)
			}
		})
	}
}
