package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func graphqlRateLimitingCostDecorationsCollection() *GraphqlRateLimitingCostDecorationsCollection {
	return state().GraphqlRateLimitingCostDecorations
}

func TestGraphqlRateLimitingCostDecorationAdd(t *testing.T) {
	collection := graphqlRateLimitingCostDecorationsCollection()

	tests := []struct {
		name       string
		decoration GraphqlRateLimitingCostDecoration
		wantErr    bool
	}{
		{
			name: "adds a decoration to the collection",
			decoration: GraphqlRateLimitingCostDecoration{
				GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
					ID:          kong.String("first"),
					TypePath:    kong.String("Query.users"),
					AddConstant: kong.Float64(1.0),
				},
			},
			wantErr: false,
		},
		{
			name: "adds a decoration with all fields to the collection",
			decoration: GraphqlRateLimitingCostDecoration{
				GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
					ID:           kong.String("second"),
					TypePath:     kong.String("Query.posts"),
					AddConstant:  kong.Float64(2.0),
					MulConstant:  kong.Float64(1.5),
					AddArguments: kong.StringSlice("limit"),
					MulArguments: kong.StringSlice("first", "last"),
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when the decoration already exists",
			decoration: GraphqlRateLimitingCostDecoration{
				GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
					ID:          kong.String("first"),
					TypePath:    kong.String("Query.users"),
					AddConstant: kong.Float64(1.0),
				},
			},
			wantErr: true,
		},
		{
			name: "returns an error if an id is not provided",
			decoration: GraphqlRateLimitingCostDecoration{
				GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
					TypePath:    kong.String("Query.users"),
					AddConstant: kong.Float64(1.0),
				},
			},
			wantErr: true,
		},
		{
			name: "returns an error if an empty decoration is provided",
			decoration: GraphqlRateLimitingCostDecoration{
				GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collection.Add(tt.decoration); (err != nil) != tt.wantErr {
				t.Errorf("GraphqlRateLimitingCostDecorationsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGraphqlRateLimitingCostDecorationGet(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := graphqlRateLimitingCostDecorationsCollection()

	decoration := GraphqlRateLimitingCostDecoration{
		GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
			ID:          kong.String("example"),
			TypePath:    kong.String("Query.users"),
			AddConstant: kong.Float64(1.0),
		},
	}

	err := collection.Add(decoration)
	require.NoError(err, "error adding decoration")

	// Fetch the currently added entity
	res, err := collection.Get("example")
	require.NoError(err, "error getting decoration")
	require.NotNil(res)
	assert.Equal("example", *res.ID)
	assert.Equal("Query.users", *res.TypePath)
	assert.InDelta(1.0, *res.AddConstant, 1.0)

	// Fetch non-existent entity
	res, err = collection.Get("does-not-exist")
	require.Error(err)
	require.Nil(res)
}

func TestGraphqlRateLimitingCostDecorationGetByTypePath(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := graphqlRateLimitingCostDecorationsCollection()

	decoration := GraphqlRateLimitingCostDecoration{
		GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
			ID:          kong.String("typepath-example"),
			TypePath:    kong.String("Query.comments"),
			AddConstant: kong.Float64(3.0),
		},
	}

	err := collection.Add(decoration)
	require.NoError(err, "error adding decoration")

	// Fetch by TypePath
	res, err := collection.GetByTypePath("Query.comments")
	require.NoError(err, "error getting decoration by TypePath")
	require.NotNil(res)
	assert.Equal("typepath-example", *res.ID)
	assert.Equal("Query.comments", *res.TypePath)

	// Fetch non-existent TypePath
	res, err = collection.GetByTypePath("Query.nonexistent")
	require.Error(err)
	require.Nil(res)

	// Fetch with empty TypePath
	res, err = collection.GetByTypePath("")
	require.Error(err)
	require.Nil(res)
}

func TestGraphqlRateLimitingCostDecorationUpdate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := graphqlRateLimitingCostDecorationsCollection()

	decoration := GraphqlRateLimitingCostDecoration{
		GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
			ID:          kong.String("update-example"),
			TypePath:    kong.String("Query.items"),
			AddConstant: kong.Float64(1.0),
		},
	}

	err := collection.Add(decoration)
	require.NoError(err, "error adding decoration")

	// Fetch the currently added entity
	res, err := collection.Get("update-example")
	require.NoError(err, "error getting decoration")
	require.NotNil(res)
	assert.InDelta(1.0, *res.AddConstant, 1.0)

	// Update AddConstant field
	res.AddConstant = kong.Float64(5.0)
	res.MulConstant = kong.Float64(2.0)
	err = collection.Update(*res)
	require.NoError(err, "error updating decoration")

	// Fetch again
	res, err = collection.Get("update-example")
	require.NoError(err, "error getting decoration")
	require.NotNil(res)
	assert.InDelta(5.0, *res.AddConstant, 5.0)
	assert.InDelta(2.0, *res.MulConstant, 2.0)
}

func TestGraphqlRateLimitingCostDecorationDelete(t *testing.T) {
	require := require.New(t)
	collection := graphqlRateLimitingCostDecorationsCollection()

	decoration := GraphqlRateLimitingCostDecoration{
		GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
			ID:          kong.String("delete-example"),
			TypePath:    kong.String("Query.products"),
			AddConstant: kong.Float64(1.0),
		},
	}

	err := collection.Add(decoration)
	require.NoError(err, "error adding decoration")

	// Fetch the currently added entity
	res, err := collection.Get("delete-example")
	require.NoError(err, "error getting decoration")
	require.NotNil(res)

	// Delete entity
	err = collection.Delete(*res.ID)
	require.NoError(err, "error deleting decoration")

	// Fetch again
	res, err = collection.Get("delete-example")
	require.Error(err)
	require.Nil(res)
}

func TestGraphqlRateLimitingCostDecorationGetAll(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := graphqlRateLimitingCostDecorationsCollection()

	populateGraphqlRateLimitingCostDecorations(t, collection)

	decorations, err := collection.GetAll()
	require.NoError(err, "error getting all decorations")
	assert.Len(decorations, 5)
	assert.IsType([]*GraphqlRateLimitingCostDecoration{}, decorations)
}

func populateGraphqlRateLimitingCostDecorations(t *testing.T,
	collection *GraphqlRateLimitingCostDecorationsCollection,
) {
	require := require.New(t)
	decorations := []GraphqlRateLimitingCostDecoration{
		{
			GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
				ID:          kong.String("populate-first"),
				TypePath:    kong.String("Query.allUsers"),
				AddConstant: kong.Float64(1.0),
			},
		},
		{
			GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
				ID:          kong.String("populate-second"),
				TypePath:    kong.String("Query.allPosts"),
				AddConstant: kong.Float64(2.0),
				MulConstant: kong.Float64(1.5),
			},
		},
		{
			GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
				ID:           kong.String("populate-third"),
				TypePath:     kong.String("Query.allComments"),
				AddConstant:  kong.Float64(1.0),
				AddArguments: kong.StringSlice("limit"),
			},
		},
		{
			GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
				ID:           kong.String("populate-fourth"),
				TypePath:     kong.String("Query.allProducts"),
				MulConstant:  kong.Float64(2.0),
				MulArguments: kong.StringSlice("first", "last"),
			},
		},
		{
			GraphqlRateLimitingCostDecoration: kong.GraphqlRateLimitingCostDecoration{
				ID:           kong.String("populate-fifth"),
				TypePath:     kong.String("Query.allOrders"),
				AddConstant:  kong.Float64(3.0),
				MulConstant:  kong.Float64(2.5),
				AddArguments: kong.StringSlice("offset"),
				MulArguments: kong.StringSlice("count"),
			},
		},
	}

	for _, d := range decorations {
		err := collection.Add(d)
		require.NoError(err, "error adding decoration")
	}
}
