package konnect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginV2_WithToken(t *testing.T) {
	client := &Client{}
	authService := &AuthService{client: client}

	resp, err := authService.LoginV2("test-token")
	require.NoError(t, err)
	assert.Equal(t, AuthResponse{}, resp)
	assert.Equal(t, "test-token", client.token)
}

func TestLoginV2_EmptyToken(t *testing.T) {
	client := &Client{}
	authService := &AuthService{client: client}

	_, err := authService.LoginV2("")
	require.Error(t, err)
	assert.EqualError(t, err, "access token must be provided")
}
