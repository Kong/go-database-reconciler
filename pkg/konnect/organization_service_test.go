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

func newOrganizationTestClient(mockServer *httptest.Server) *Client {
	return &Client{
		baseURL: "https://some-geo.api.konghq.com",
		client: &http.Client{
			Transport: &mockRoundTripper{
				mockHost: mockServer.Listener.Addr().String(),
			},
		},
	}
}

func TestOrganizationService_GetIPAllowList(t *testing.T) {
	expectedResp := IPAllowList{Enabled: true, AllowedIPs: []string{"192.168.1.1", "192.168.1.0/22"}}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Host, "global.api.konghq.com")

		if r.URL.Path == "/v3/organizations/org-1/ip-allow-list" && r.Method == http.MethodGet {
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

	client := newOrganizationTestClient(mockServer)
	client.common.client = client
	client.Organization = (*OrganizationService)(&client.common)

	result, err := client.Organization.GetIPAllowList(context.Background(), "org-1")
	require.NoError(t, err)
	assert.Equal(t, expectedResp, *result)
}

func TestOrganizationService_UpdateIPAllowList(t *testing.T) {
	requestBody := IPAllowList{Enabled: true, AllowedIPs: []string{"10.0.0.1"}}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Host, "global.api.konghq.com")

		if r.URL.Path == "/v3/organizations/org-1/ip-allow-list" && r.Method == http.MethodPut {
			var body IPAllowList
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, requestBody, body)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp, err := json.Marshal(body)
			require.NoError(t, err)
			_, err = w.Write(resp)
			require.NoError(t, err)
			return
		}

		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	client := newOrganizationTestClient(mockServer)
	client.common.client = client
	client.Organization = (*OrganizationService)(&client.common)

	result, err := client.Organization.UpdateIPAllowList(context.Background(), "org-1", requestBody)
	require.NoError(t, err)
	assert.Equal(t, requestBody, *result)
}

func TestOrganizationService_GetIPAllowList_Forbidden(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer mockServer.Close()

	client := newOrganizationTestClient(mockServer)
	client.common.client = client
	client.Organization = (*OrganizationService)(&client.common)

	result, err := client.Organization.GetIPAllowList(context.Background(), "org-1")
	require.Error(t, err)
	assert.Nil(t, result)
}
