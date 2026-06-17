package state

import (
	"fmt"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func filterChainsCollection() *FilterChainsCollection {
	return state().FilterChains
}

type testCase struct {
	name        string
	filterChain kong.FilterChain
	wantErr     bool
}

var commonCases = []testCase{
	{
		name: errorsWhenIDIsNil,
		filterChain: kong.FilterChain{
			ID: nil,
			Service: &kong.Service{
				ID:   new("svc1"),
				Host: new("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
		wantErr: true,
	},
	{
		name: "errors when both Service and Route are nil",
		filterChain: kong.FilterChain{
			ID:      new("fc1"),
			Service: nil,
			Route:   nil,
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
		wantErr: true,
	},
	{
		name: "errors when Service and Route are both defined",
		filterChain: kong.FilterChain{
			ID: new("fc4"),
			Service: &kong.Service{
				ID:   new("svc3"),
				Host: new("example.test"),
			},
			Route: &kong.Route{
				ID:    new("r2"),
				Hosts: kong.StringSlice("example.com"),
			},
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
		wantErr: true,
	},
	{
		name: "errors when Filters is nil",
		filterChain: kong.FilterChain{
			ID: new("fc9"),
			Service: &kong.Service{
				ID:   new("svc6"),
				Host: new("example.test"),
			},
			Route:   nil,
			Filters: nil,
		},
		wantErr: true,
	},
	{
		name: "errors when Filters is empty",
		filterChain: kong.FilterChain{
			ID: new("fc9"),
			Service: &kong.Service{
				ID:   new("svc6"),
				Host: new("example.test"),
			},
			Route:   nil,
			Filters: []*kong.Filter{},
		},
		wantErr: true,
	},
}

func TestFilterChainsCollection_Add(t *testing.T) {
	tests := []testCase{
		{
			name: "inserts when Service is defined",
			filterChain: kong.FilterChain{
				ID: new("fc2"),
				Service: &kong.Service{
					ID:   new("svc2"),
					Host: new("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "inserts when Route is defined",
			filterChain: kong.FilterChain{
				ID:      new("fc3"),
				Service: nil,
				Route: &kong.Route{
					ID:    new("r1"),
					Hosts: kong.StringSlice("example.test"),
				},
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "errors on duplicate ID",
			filterChain: kong.FilterChain{
				ID: new("dupe-filter-id"),
				Service: &kong.Service{
					ID:   new("svc4"),
					Host: new("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on duplicate service ID",
			filterChain: kong.FilterChain{
				ID: new("fc6"),
				Service: &kong.Service{
					ID:   new("dupe-service-id"),
					Host: new("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on duplicate Route ID",
			filterChain: kong.FilterChain{
				ID: new("fc7"),
				Route: &kong.Route{
					ID:    new("dupe-route-id"),
					Hosts: kong.StringSlice("example.test"),
				},
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
			wantErr: true,
		},
	}

	k := filterChainsCollection()

	dupeByID := FilterChain{
		FilterChain: kong.FilterChain{
			ID: new("dupe-filter-id"),
			Service: &kong.Service{
				ID:   new("svc5"),
				Host: new("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	if err := k.Add(dupeByID); err != nil {
		t.Error(err)
	}

	dupeByService := FilterChain{
		FilterChain: kong.FilterChain{
			ID: new("fc5"),
			Service: &kong.Service{
				ID:   new("dupe-service-id"),
				Host: new("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	if err := k.Add(dupeByService); err != nil {
		t.Error(err)
	}

	dupeByRoute := FilterChain{
		FilterChain: kong.FilterChain{
			ID: new("fc8"),
			Route: &kong.Route{
				ID:    new("dupe-route-id"),
				Hosts: kong.StringSlice("example.test"),
			},
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	if err := k.Add(dupeByRoute); err != nil {
		t.Error(err)
	}

	for _, tt := range commonCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fc := FilterChain{FilterChain: tt.filterChain}
			if err := k.Add(fc); (err != nil) != tt.wantErr {
				t.Errorf("FilterChainsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fc := FilterChain{FilterChain: tt.filterChain}
			if err := k.Add(fc); (err != nil) != tt.wantErr {
				t.Errorf("FilterChainsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFilterChainsCollection_Update(t *testing.T) {
	tests := []testCase{
		{
			name: "errors when the filter chain does not exist",
			filterChain: kong.FilterChain{
				ID: new("i-dont-exist"),
				Service: &kong.Service{
					ID:   new("svc2"),
					Host: new("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on duplicate service ID",
			filterChain: kong.FilterChain{
				ID: new("fc6"),
				Service: &kong.Service{
					ID:   new("dupe-service-id"),
					Host: new("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on duplicate Route ID",
			filterChain: kong.FilterChain{
				ID: new("fc7"),
				Route: &kong.Route{
					ID:    new("dupe-route-id"),
					Hosts: kong.StringSlice("example.test"),
				},
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
			wantErr: true,
		},
	}

	k := filterChainsCollection()

	t.Run("updates existing filter chains", func(t *testing.T) {
		pre := FilterChain{
			FilterChain: kong.FilterChain{
				ID:   new("update-id-1"),
				Name: new("old-name"),
				Service: &kong.Service{
					ID:   new("my-service"),
					Host: new("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
		}
		require.NoError(t, k.Add(pre))

		post := FilterChain{
			FilterChain: kong.FilterChain{
				ID:   new("update-id-1"),
				Name: new("new-name"),
				Service: &kong.Service{
					ID:   new("my-service"),
					Host: new("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
		}
		require.NoError(t, k.Update(post))

		updated, err := k.Get("update-id-1")
		require.NoError(t, err)

		assert.Equal(t, "new-name", *updated.Name)
	})

	for _, tt := range commonCases {
		if utils.Empty(tt.filterChain.ID) {
			continue
		}

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id := tt.filterChain.ID

			old := FilterChain{
				FilterChain: kong.FilterChain{
					ID:   id,
					Name: new(fmt.Sprintf("%s-pre-update", *id)),
					Filters: []*kong.Filter{
						{
							Name: new("my-filter"),
						},
					},
				},
			}

			updated := FilterChain{FilterChain: tt.filterChain}

			if err := k.Add(old); (err != nil) != tt.wantErr {
				t.Errorf("FilterChainsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := k.Update(updated); (err != nil) != tt.wantErr {
				t.Errorf("FilterChainsCollection.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fc := FilterChain{FilterChain: tt.filterChain}
			if err := k.Update(fc); (err != nil) != tt.wantErr {
				t.Errorf("FilterChainsCollection.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFilterChainsCollection_Get(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	filterChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID:   new("get-test"),
			Name: new("old-name"),
			Service: &kong.Service{
				ID:   new("svc9"),
				Host: new("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	require.NoError(t, collection.Add(filterChain))

	re, err := collection.Get("get-test")
	require.NoError(t, err)
	assert.NotNil(re)
	assert.NotNil(re.FilterChain)
	assert.NotNil(re.ID)
	assert.Equal("get-test", *re.ID)

	re, err = collection.Get("does-not-exists")
	assert.Equal(ErrNotFound, err)
	assert.Nil(re)

	re, err = collection.Get("")
	assert.Equal(errIDRequired, err)
	assert.Nil(re)
}

func TestFilterChainsCollection_GetByProp(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	serviceChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: new("get-prop-service"),
			Service: &kong.Service{
				ID:   new("get-prop-service-id"),
				Host: new("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	require.NoError(t, collection.Add(serviceChain))

	routeChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: new("get-prop-route"),
			Route: &kong.Route{
				ID:    new("get-prop-route-id"),
				Hosts: kong.StringSlice("example.test"),
			},
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	require.NoError(t, collection.Add(routeChain))

	re, err := collection.GetByProp("get-prop-service-id", "")
	require.NoError(t, err)
	assert.NotNil(re)
	assert.NotNil(re.FilterChain)
	assert.NotNil(re.ID)
	assert.Equal("get-prop-service", *re.ID)

	re, err = collection.GetByProp("", "get-prop-route-id")
	require.NoError(t, err)
	assert.NotNil(re)
	assert.NotNil(re.FilterChain)
	assert.NotNil(re.ID)
	assert.Equal("get-prop-route", *re.ID)

	re, err = collection.GetByProp("", "")
	assert.Nil(re)
	require.Error(t, err)
	assert.Equal(errIDRequired, err)
}

func TestFilterChainsCollection_GetAllByServiceID(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	serviceChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: new("get-all-service"),
			Service: &kong.Service{
				ID:   new("get-all-service-id"),
				Host: new("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	require.NoError(t, collection.Add(serviceChain))

	re, err := collection.GetAllByServiceID("get-all-service-id")
	require.NoError(t, err)
	assert.Equal([]*FilterChain{&serviceChain}, re)

	re, err = collection.GetAllByServiceID("no-chain-with-this-service-id")
	require.NoError(t, err)
	assert.Empty(re)

	re, err = collection.GetAllByServiceID("")
	assert.Nil(re)
	require.Error(t, err)
	assert.Equal(errIDRequired, err)
}

func TestFilterChainsCollection_GetAllByRouteID(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	routeChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: new("get-all-route"),
			Route: &kong.Route{
				ID: new("get-all-route-id"),
			},
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	require.NoError(t, collection.Add(routeChain))

	re, err := collection.GetAllByRouteID("get-all-route-id")
	require.NoError(t, err)
	assert.Equal([]*FilterChain{&routeChain}, re)

	re, err = collection.GetAllByRouteID("no-chain-with-this-route-id")
	require.NoError(t, err)
	assert.Empty(re)

	re, err = collection.GetAllByRouteID("")
	assert.Nil(re)
	require.Error(t, err)
	assert.Equal(errIDRequired, err)
}

func TestFilterChainsCollection_Delete(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	filterChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: new("delete-test"),
			Service: &kong.Service{
				ID:   new("my-service"),
				Host: new("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: new("my-filter"),
				},
			},
		},
	}
	require.NoError(t, collection.Add(filterChain))

	res, err := collection.Get("delete-test")
	assert.NotNil(res)
	require.NoError(t, err)

	err = collection.Delete("delete-test")
	require.NoError(t, err)

	res, err = collection.Get("delete-test")
	assert.Equal(ErrNotFound, err)
	assert.Nil(res)
}

func TestFilterChainsCollection_GetAll(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	for i := range 3 {
		id := fmt.Sprintf("get-all-%d", i)
		serviceID := fmt.Sprintf("get-all-service-id-%d", i)

		filterChain := FilterChain{
			FilterChain: kong.FilterChain{
				ID: new(id),
				Service: &kong.Service{
					ID:   new(serviceID),
					Host: new("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: new("my-filter"),
					},
				},
			},
		}

		require.NoError(t, collection.Add(filterChain))
	}

	allFilterChains, err := collection.GetAll()
	require.NoError(t, err)
	assert.Len(allFilterChains, 3)
}
