package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func partialsCollection() *PartialsCollection {
	return state().Partials
}

func TestPartialAdd(t *testing.T) {
	collection := partialsCollection()

	tests := []struct {
		name    string
		partial Partial
		wantErr bool
	}{
		{
			name: "adds a partial to the collection",
			partial: Partial{
				Partial: kong.Partial{
					ID:   kong.String("first"),
					Name: kong.String("my-foo-partial"),
					Type: kong.String("foo"),
					Config: kong.Configuration{
						"key1": "value1",
						"key2": []any{"a", "b", "c"},
						"key3": map[string]interface{}{
							"k1": "v1",
							"k2": "v2",
							"k3": []any{"a1", "b1"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when the partial already exists - by id",
			partial: Partial{
				Partial: kong.Partial{
					ID:   kong.String("first"),
					Name: kong.String("my-foo-partial"),
					Type: kong.String("foo"),
					Config: kong.Configuration{
						"key1": "value1",
						"key2": []any{"a", "b", "c"},
						"key3": map[string]interface{}{
							"k1": "v1",
							"k2": "v2",
							"k3": []any{"a1", "b1"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "returns an error when the partial already exists - by name",
			partial: Partial{
				Partial: kong.Partial{
					Name: kong.String("my-foo-partial"),
					Type: kong.String("foo"),
					Config: kong.Configuration{
						"key1": "value1",
						"key2": []any{"a", "b", "c"},
						"key3": map[string]interface{}{
							"k1": "v1",
							"k2": "v2",
							"k3": []any{"a1", "b1"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "returns an error if an id is not provided",
			partial: Partial{
				Partial: kong.Partial{
					Name: kong.String("my-foo-partial"),
					Type: kong.String("foo"),
					Config: kong.Configuration{
						"key1": "value1",
						"key2": []any{"a", "b", "c"},
						"key3": map[string]interface{}{
							"k1": "v1",
							"k2": "v2",
							"k3": []any{"a1", "b1"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "returns an error if an empty partial is provided",
			partial: Partial{
				Partial: kong.Partial{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collection.Add(tt.partial); (err != nil) != tt.wantErr {
				t.Errorf("PartialsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialGet(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := partialsCollection()

	partial := Partial{
		Partial: kong.Partial{
			ID:   kong.String("example"),
			Name: kong.String("my-foo-partial"),
			Type: kong.String("foo"),
			Config: kong.Configuration{
				"key": "value",
			},
		},
	}

	err := collection.Add(partial)
	require.NoError(err, "error adding partial")

	// Fetch the currently added entity
	res, err := collection.Get("example")
	require.NoError(err, "error getting partial")
	require.NotNil(res)
	assert.Equal("example", *res.ID)
	assert.Equal("my-foo-partial", *res.Name)
	assert.Equal("foo", *res.Type)
	assert.Equal(kong.Configuration{"key": "value"}, res.Config)

	// Fetch non-existent entity
	res, err = collection.Get("does-not-exist")
	require.Error(err)
	require.Nil(res)
}

func TestPartialUpdate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := partialsCollection()

	partial := Partial{
		Partial: kong.Partial{
			ID:   kong.String("example"),
			Name: kong.String("my-foo-partial"),
			Type: kong.String("foo"),
			Config: kong.Configuration{
				"key1": "value1",
			},
		},
	}

	err := collection.Add(partial)
	require.NoError(err, "error adding partial")

	// Fetch the currently added entity
	res, err := collection.Get("example")
	require.NoError(err, "error getting partial")
	require.NotNil(res)
	assert.Equal(kong.Configuration{
		"key1": "value1",
	}, res.Config)

	// Update Ccnfig field
	res.Config = kong.Configuration{
		"key2": "value2",
	}
	err = collection.Update(*res)
	require.NoError(err, "error updating partial")

	// Fetch again
	res, err = collection.Get("example")
	require.NoError(err, "error getting partial")
	require.NotNil(res)
	assert.Equal(kong.Configuration{
		"key2": "value2",
	}, res.Config)
}

func TestPartialDelete(t *testing.T) {
	require := require.New(t)
	collection := partialsCollection()

	partial := Partial{
		Partial: kong.Partial{
			ID:   kong.String("example"),
			Name: kong.String("my-foo-partial"),
			Type: kong.String("foo"),
			Config: kong.Configuration{
				"key": "value",
			},
		},
	}

	err := collection.Add(partial)
	require.NoError(err, "error adding partial")

	// Fetch the currently added entity
	res, err := collection.Get("example")
	require.NoError(err, "error getting partial")
	require.NotNil(res)

	// Delete entity
	err = collection.Delete(*res.ID)
	require.NoError(err, "error deleting partial")

	// Fetch again
	res, err = collection.Get("example")
	require.Error(err)
	require.Nil(res)
}

func TestPartialGetAll(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	collection := partialsCollection()

	populatePartials(t, collection)

	partials, err := collection.GetAll()
	require.NoError(err, "error getting all partials")
	assert.Equal(5, len(partials))
	assert.IsType([]*Partial{}, partials)
}

func populatePartials(t *testing.T,
	collection *PartialsCollection,
) {
	require := require.New(t)
	partials := []Partial{
		{
			Partial: kong.Partial{
				ID:   kong.String("first"),
				Name: kong.String("my-foo-partial-1"),
				Type: kong.String("foo"),
				Config: kong.Configuration{
					"key": "value",
				},
			},
		},
		{
			Partial: kong.Partial{
				ID:   kong.String("second"),
				Name: kong.String("my-foo-partial-2"),
				Type: kong.String("foo"),
				Config: kong.Configuration{
					"key": "value",
				},
			},
		},
		{
			Partial: kong.Partial{
				ID:   kong.String("third"),
				Name: kong.String("my-foo-partial-3"),
				Type: kong.String("foo"),
				Config: kong.Configuration{
					"key": "value",
				},
			},
		},
		{
			Partial: kong.Partial{
				ID:   kong.String("fourth"),
				Name: kong.String("my-foo-partial-4"),
				Type: kong.String("foo"),
				Config: kong.Configuration{
					"key": "value",
				},
			},
		},
		{
			Partial: kong.Partial{
				ID:   kong.String("fifth"),
				Name: kong.String("my-foo-partial-5"),
				Type: kong.String("foo"),
				Config: kong.Configuration{
					"key": "value",
				},
			},
		},
	}

	for _, d := range partials {
		err := collection.Add(d)
		require.NoError(err, "error adding partial")
	}
}
