package konnect

import (
	"errors"
)

// AuthService handles authentication with the Konnect API.
type AuthService service

// LoginV2 authenticates the client using a Personal Access Token (PAT).
// The token is stored on the client and used in subsequent request Authorization headers.
func (s *AuthService) LoginV2(token string) (AuthResponse, error) {
	if token == "" {
		return AuthResponse{}, errors.New("access token must be provided")
	}
	s.client.token = token
	return AuthResponse{}, nil
}
