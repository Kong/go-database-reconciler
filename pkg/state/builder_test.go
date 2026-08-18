package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDegraphqlRouteFromCustomEntityServiceOutsideWorkspace(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	kongState := state()

	entity := map[string]any{
		"id": "degraphql-route-1",
		"service": map[string]any{
			"id": "service-in-other-workspace",
		},
		"uri":   "/foo",
		"query": "query { hello }",
	}

	route, err := buildDegraphqlRouteFromCustomEntity(kongState, entity)
	require.NoError(err, "should not fail when the referenced service is not in the local state")
	require.NotNil(route.Service)
	assert.Equal("service-in-other-workspace", *route.Service.ID)
}

func TestBuildDegraphqlRouteFromCustomEntityServiceInWorkspace(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	kongState := state()
	require.NoError(kongState.Services.Add(Service{
		Service: kong.Service{
			ID:   new("known-service"),
			Name: new("known-service-name"),
		},
	}))

	entity := map[string]any{
		"id": "degraphql-route-1",
		"service": map[string]any{
			"id": "known-service",
		},
		"uri":   "/foo",
		"query": "query { hello }",
	}

	route, err := buildDegraphqlRouteFromCustomEntity(kongState, entity)
	require.NoError(err)
	require.NotNil(route.Service)
	assert.Equal("known-service", *route.Service.ID)
	assert.Equal("known-service-name", *route.Service.Name)
}

func TestBuildGraphqlRateLimitingCostDecorationFromCustomEntityServiceOutsideWorkspace(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	kongState := state()

	entity := map[string]any{
		"id": "cost-decoration-1",
		"service": map[string]any{
			"id": "service-in-other-workspace",
		},
		"type_path": "Query.foo",
	}

	decoration, err := buildGraphqlRateLimitingCostDecorationFromCustomEntity(kongState, entity)
	require.NoError(err, "should not fail when the referenced service is not in the local state")
	require.NotNil(decoration.Service)
	assert.Equal("service-in-other-workspace", *decoration.Service.ID)
}
