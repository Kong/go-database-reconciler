package konnect

import (
	"context"
	"fmt"
	"net/http"
)

type OrganizationService service

// IPAllowList represents the organization-level IP allow list that
// restricts which source IPs can reach Konnect's Admin API.
type IPAllowList struct {
	Enabled    bool     `json:"enabled"`
	AllowedIPs []string `json:"allowed_ips"`
}

const ipAllowListPath = "/v3/organizations/%s/ip-allow-list"

// GetIPAllowList fetches the organization's IP allow list.
func (s *OrganizationService) GetIPAllowList(ctx context.Context, orgID string) (*IPAllowList, error) {
	// replace geo-specific endpoint with global one, same as OrgUserInfo
	client := *s.client
	client.baseURL = getGlobalEndpoint(client.baseURL)

	req, err := client.NewRequest(http.MethodGet, fmt.Sprintf(ipAllowListPath, orgID), nil, nil)
	if err != nil {
		return nil, err
	}

	var result IPAllowList
	if _, err := s.client.Do(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateIPAllowList replaces the organization's IP allow list in full.
func (s *OrganizationService) UpdateIPAllowList(
	ctx context.Context, orgID string, allowList IPAllowList,
) (*IPAllowList, error) {
	client := *s.client
	client.baseURL = getGlobalEndpoint(client.baseURL)

	req, err := client.NewRequest(http.MethodPut, fmt.Sprintf(ipAllowListPath, orgID), nil, allowList)
	if err != nil {
		return nil, err
	}

	var result IPAllowList
	if _, err := s.client.Do(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
