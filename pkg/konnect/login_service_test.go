package konnect

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRoundTripper struct{ mockHost string }

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Host = req.URL.Host
	req.URL.Scheme = "http"
	req.URL.Host = m.mockHost

	return (&http.Client{}).Do(req)
}

func TestGetGlobalAuthEndpoint(t *testing.T) {
	tests := []struct {
		baseURL  string
		expected string
	}{
		{
			baseURL:  "https://us.api.konghq.com",
			expected: "https://global.api.konghq.com/kauth/api/v1/authenticate",
		},
		{
			baseURL:  "https://global.api.konghq.com",
			expected: "https://global.api.konghq.com/kauth/api/v1/authenticate",
		},
		{
			baseURL:  "https://eu.api.konghq.com",
			expected: "https://global.api.konghq.com/kauth/api/v1/authenticate",
		},
		{
			baseURL:  "https://au.api.konghq.com",
			expected: "https://global.api.konghq.com/kauth/api/v1/authenticate",
		},
		{
			baseURL:  "https://api.konghq.com",
			expected: "https://global.api.konghq.com/kauth/api/v1/authenticate",
		},
		{
			baseURL:  "https://eu.api.konghq.test",
			expected: "https://global.api.konghq.test/kauth/api/v1/authenticate",
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, getGlobalAuthEndpoint(tt.baseURL))
	}
}

func TestAuthService_OrgUserInfo(t *testing.T) {
	expectedResp := OrgUserInfo{Name: "test-org", OrgID: "1234"}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Host, "global.api.konghq.com")

		if r.URL.Path == "/v2/organizations/me" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			resp, err := json.Marshal(expectedResp)
			require.NoError(t, err)

			_, err = w.Write(resp)
			require.NoError(t, err)

			return
		}

		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	authService := &AuthService{
		client: &Client{
			baseURL: "https://some-geo.api.konghq.com",
			client: &http.Client{
				Transport: &mockRoundTripper{
					mockHost: mockServer.Listener.Addr().String(),
				},
			},
		},
	}

	info, err := authService.OrgUserInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedResp.Name, info.Name)
	assert.Equal(t, expectedResp.OrgID, info.OrgID)
}
