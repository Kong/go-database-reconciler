package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func degraphqlRoutesCollection() *DegraphqlRoutesCollection {
	return state().DegraphqlRoutes
}

func TestDegraphqlRouteAdd(t *testing.T) {
	collection := degraphqlRoutesCollection()

	tests := []struct {
		name           string
		degraphqlRoute DegraphqlRoute
		wantErr        bool
	}{
		{
			name: "adds a degraphql route to the collection",
			degraphqlRoute: DegraphqlRoute{
				DegraphqlRoute: kong.DegraphqlRoute{
					ID:    kong.String("first"),
					URI:   kong.String("/foo"),
					Query: kong.String("query { hello }"),
					Service: &kong.Service{
						ID: kong.String("some-service"),
					},
					Methods: kong.StringSlice("GET"),
				},
			},
			wantErr: false,
		},
		{
			name: "adds a degraphql route with complex query to the collection",
			degraphqlRoute: DegraphqlRoute{
				DegraphqlRoute: kong.DegraphqlRoute{
					ID:  kong.String("second"),
					URI: kong.String("/bar"),
					Query: kong.String(`query SearchPosts($filters: PostsFilters) {
							posts(filter: $filters) {
								id
								title
								author
							}
							}`),
					Service: &kong.Service{
						ID: kong.String("some-service"),
					},
					Methods: kong.StringSlice("GET", "POST"),
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when the degraphql route already exists",
			degraphqlRoute: DegraphqlRoute{
				DegraphqlRoute: kong.DegraphqlRoute{
					ID:    kong.String("first"),
					URI:   kong.String("/foo"),
					Query: kong.String("query { hello }"),
					Service: &kong.Service{
						ID: kong.String("some-service"),
					},
					Methods: kong.StringSlice("GET"),
				},
			},
			wantErr: true,
		},
		{
			name: "returns an error if an id is not provided",
			degraphqlRoute: DegraphqlRoute{
				DegraphqlRoute: kong.DegraphqlRoute{
					URI:   kong.String("/foo"),
					Query: kong.String("query { hello }"),
					Service: &kong.Service{
						ID: kong.String("some-service"),
					},
					Methods: kong.StringSlice("GET"),
				},
			},
			wantErr: true,
		},
		{
			name: "returns an error if an empty degraphql route is provided",
			degraphqlRoute: DegraphqlRoute{
				DegraphqlRoute: kong.DegraphqlRoute{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collection.Add(tt.degraphqlRoute); (err != nil) != tt.wantErr {
				t.Errorf("DegraphqlRoutesCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDegraphqlRouteGet(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := degraphqlRoutesCollection()

	degraphqlRoute := DegraphqlRoute{
		DegraphqlRoute: kong.DegraphqlRoute{
			ID:    kong.String("example"),
			URI:   kong.String("/foo"),
			Query: kong.String("query { hello }"),
			Service: &kong.Service{
				ID: kong.String("some-service"),
			},
			Methods: kong.StringSlice("GET"),
		},
	}

	err := collection.Add(degraphqlRoute)
	require.NoError(err, "error adding degraphql route")

	// Fetch the currently added entity
	res, err := collection.Get("example")
	require.NoError(err, "error getting degraphql route")
	require.NotNil(res)
	assert.Equal("example", *res.ID)
	assert.Equal("some-service", *res.Service.ID)

	// Fetch non-existent entity
	res, err = collection.Get("does-not-exist")
	require.Error(err)
	require.Nil(res)
}

func TestDegraphqlRouteUpdate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := degraphqlRoutesCollection()

	degraphqlRoute := DegraphqlRoute{
		DegraphqlRoute: kong.DegraphqlRoute{
			ID:    kong.String("example"),
			URI:   kong.String("/foo"),
			Query: kong.String("query { hello }"),
			Service: &kong.Service{
				ID: kong.String("some-service"),
			},
			Methods: kong.StringSlice("GET"),
		},
	}

	err := collection.Add(degraphqlRoute)
	require.NoError(err, "error adding degraphql route")

	// Fetch the currently added entity
	res, err := collection.Get("example")
	require.NoError(err, "error getting degraphql route")
	require.NotNil(res)
	assert.Equal("query { hello }", *res.Query)

	// Update query field
	res.Query = kong.String("query { hello world }")
	err = collection.Update(*res)
	require.NoError(err, "error updating degraphql route")

	// Fetch again
	res, err = collection.Get("example")
	require.NoError(err, "error getting degraphql route")
	require.NotNil(res)
	assert.Equal("query { hello world }", *res.Query)
}

func TestDegraphqlRouteDelete(t *testing.T) {
	require := require.New(t)
	collection := degraphqlRoutesCollection()

	degraphqlRoute := DegraphqlRoute{
		DegraphqlRoute: kong.DegraphqlRoute{
			ID:    kong.String("example"),
			URI:   kong.String("/foo"),
			Query: kong.String("query { hello }"),
			Service: &kong.Service{
				ID: kong.String("some-service"),
			},
			Methods: kong.StringSlice("GET"),
		},
	}

	err := collection.Add(degraphqlRoute)
	require.NoError(err, "error adding degraphql route")

	// Fetch the currently added entity
	res, err := collection.Get("example")
	require.NoError(err, "error getting degraphql route")
	require.NotNil(res)

	// Delete entity
	err = collection.Delete(*res.ID)
	require.NoError(err, "error deleting degraphql route")

	// Fetch again
	res, err = collection.Get("example")
	require.Error(err)
	require.Nil(res)
}

func TestDegraphqlRouteGetAll(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := degraphqlRoutesCollection()

	populateDegraphqlRoutes(t, collection)

	degraphqlRoutes, err := collection.GetAll()
	require.NoError(err, "error getting all degraphql routes")
	assert.Len(degraphqlRoutes, 5)
	assert.IsType([]*DegraphqlRoute{}, degraphqlRoutes)
}

func populateDegraphqlRoutes(t *testing.T,
	collection *DegraphqlRoutesCollection,
) {
	require := require.New(t)
	degraphqlRoutes := []DegraphqlRoute{
		{
			DegraphqlRoute: kong.DegraphqlRoute{
				ID:    kong.String("first"),
				URI:   kong.String("/foo"),
				Query: kong.String("query { hello }"),
				Service: &kong.Service{
					ID: kong.String("some-service"),
				},
				Methods: kong.StringSlice("GET"),
			},
		},
		{
			DegraphqlRoute: kong.DegraphqlRoute{
				ID:    kong.String("second"),
				URI:   kong.String("/bar"),
				Query: kong.String("query { hello }"),
				Service: &kong.Service{
					ID: kong.String("some-service"),
				},
				Methods: kong.StringSlice("GET"),
			},
		},
		{
			DegraphqlRoute: kong.DegraphqlRoute{
				ID:    kong.String("third"),
				URI:   kong.String("/foo"),
				Query: kong.String("query { hello }"),
				Service: &kong.Service{
					ID: kong.String("some-service"),
				},
				Methods: kong.StringSlice("GET"),
			},
		},
		{
			DegraphqlRoute: kong.DegraphqlRoute{
				ID:    kong.String("fourth"),
				URI:   kong.String("/bar"),
				Query: kong.String("query { hello }"),
				Service: &kong.Service{
					ID: kong.String("some-service"),
				},
				Methods: kong.StringSlice("GET"),
			},
		},
		{
			DegraphqlRoute: kong.DegraphqlRoute{
				ID:    kong.String("fifth"),
				URI:   kong.String("/foo"),
				Query: kong.String("query { hello }"),
				Service: &kong.Service{
					ID: kong.String("some-service"),
				},
				Methods: kong.StringSlice("GET"),
			},
		},
	}

	for _, d := range degraphqlRoutes {
		err := collection.Add(d)
		require.NoError(err, "error adding degraphql route")
	}
}
