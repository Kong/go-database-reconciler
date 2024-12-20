package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
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
			name: "returns an error if no empty degraphql route is provided",
			degraphqlRoute: DegraphqlRoute{
				DegraphqlRoute: kong.DegraphqlRoute{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := collection.Add(tt.degraphqlRoute); (err != nil) != tt.wantErr {
				t.Errorf("DegraphqlRoutesCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDegraphqlRouteGet(t *testing.T) {
	assert := assert.New(t)
	collection := degraphqlRoutesCollection()

	var degraphqlRoute DegraphqlRoute
	degraphqlRoute.ID = kong.String("example")
	degraphqlRoute.URI = kong.String("/foo")
	degraphqlRoute.Query = kong.String("query { hello }")
	degraphqlRoute.Service = &kong.Service{
		ID: kong.String("some-service"),
	}
	degraphqlRoute.Methods = kong.StringSlice("GET")

	err := collection.Add(degraphqlRoute)
	assert.Nil(err)

	// Fetch the currently added entity
	res, err := collection.Get("example")
	assert.Nil(err)
	assert.NotNil(res)
	assert.Equal("example", *res.ID)
	assert.Equal("some-service", *res.Service.ID)

	// Fetch non-existent entity
	res, err = collection.Get("does-not-exist")
	assert.NotNil(err)
	assert.Nil(res)
}

func TestDegraphqlRouteUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := degraphqlRoutesCollection()

	var degraphqlRoute DegraphqlRoute
	degraphqlRoute.ID = kong.String("example")
	degraphqlRoute.URI = kong.String("/foo")
	degraphqlRoute.Query = kong.String("query { hello }")
	degraphqlRoute.Service = &kong.Service{
		ID: kong.String("some-service"),
	}
	degraphqlRoute.Methods = kong.StringSlice("GET")

	err := collection.Add(degraphqlRoute)
	assert.Nil(err)

	// Fetch the currently added entity
	res, err := collection.Get("example")
	assert.Nil(err)
	assert.NotNil(res)
	assert.Equal("query { hello }", *res.Query)

	// Update query field
	res.Query = kong.String("query { hello world }")
	err = collection.Update(*res)
	assert.Nil(err)

	// Fetch again
	res, err = collection.Get("example")
	assert.Nil(err)
	assert.NotNil(res)
	assert.Equal("query { hello world }", *res.Query)
}

func TestDegraphqlRouteDelete(t *testing.T) {
	assert := assert.New(t)
	collection := degraphqlRoutesCollection()

	var degraphqlRoute DegraphqlRoute
	degraphqlRoute.ID = kong.String("example")
	degraphqlRoute.URI = kong.String("/foo")
	degraphqlRoute.Query = kong.String("query { hello }")
	degraphqlRoute.Service = &kong.Service{
		ID: kong.String("some-service"),
	}
	degraphqlRoute.Methods = kong.StringSlice("GET")

	err := collection.Add(degraphqlRoute)
	assert.Nil(err)

	// Fetch the currently added entity
	res, err := collection.Get("example")
	assert.Nil(err)
	assert.NotNil(res)

	// Delete entity
	err = collection.Delete(*res.ID)
	assert.Nil(err)

	// Fetch again
	res, err = collection.Get("example")
	assert.NotNil(err)
	assert.Nil(res)
}

func TestDegraphqlRouteGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := degraphqlRoutesCollection()

	populateDegraphqlRoutes(assert, collection)

	degraphqlRoutes, err := collection.GetAll()
	assert.Nil(err)
	assert.Equal(5, len(degraphqlRoutes))
	assert.IsType([]*DegraphqlRoute{}, degraphqlRoutes)
}

func populateDegraphqlRoutes(assert *assert.Assertions,
	collection *DegraphqlRoutesCollection,
) {
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
		assert.Nil(err)
	}
}
