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
		name: "errors when ID is nil",
		filterChain: kong.FilterChain{
			ID: nil,
			Service: &kong.Service{
				ID:   kong.String("svc1"),
				Host: kong.String("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
		wantErr: true,
	},
	{
		name: "errors when both Service and Route are nil",
		filterChain: kong.FilterChain{
			ID:      kong.String("fc1"),
			Service: nil,
			Route:   nil,
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
		wantErr: true,
	},
	{
		name: "errors when Service and Route are both defined",
		filterChain: kong.FilterChain{
			ID: kong.String("fc4"),
			Service: &kong.Service{
				ID:   kong.String("svc3"),
				Host: kong.String("example.test"),
			},
			Route: &kong.Route{
				ID:    kong.String("r2"),
				Hosts: kong.StringSlice("example.com"),
			},
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
		wantErr: true,
	},
	{
		name: "errors when Filters is nil",
		filterChain: kong.FilterChain{
			ID: kong.String("fc9"),
			Service: &kong.Service{
				ID:   kong.String("svc6"),
				Host: kong.String("example.test"),
			},
			Route:   nil,
			Filters: nil,
		},
		wantErr: true,
	},
	{
		name: "errors when Filters is empty",
		filterChain: kong.FilterChain{
			ID: kong.String("fc9"),
			Service: &kong.Service{
				ID:   kong.String("svc6"),
				Host: kong.String("example.test"),
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
				ID: kong.String("fc2"),
				Service: &kong.Service{
					ID:   kong.String("svc2"),
					Host: kong.String("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "inserts when Route is defined",
			filterChain: kong.FilterChain{
				ID:      kong.String("fc3"),
				Service: nil,
				Route: &kong.Route{
					ID:    kong.String("r1"),
					Hosts: kong.StringSlice("example.test"),
				},
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "errors on duplicate ID",
			filterChain: kong.FilterChain{
				ID: kong.String("dupe-filter-id"),
				Service: &kong.Service{
					ID:   kong.String("svc4"),
					Host: kong.String("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on duplicate service ID",
			filterChain: kong.FilterChain{
				ID: kong.String("fc6"),
				Service: &kong.Service{
					ID:   kong.String("dupe-service-id"),
					Host: kong.String("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on duplicate Route ID",
			filterChain: kong.FilterChain{
				ID: kong.String("fc7"),
				Route: &kong.Route{
					ID:    kong.String("dupe-route-id"),
					Hosts: kong.StringSlice("example.test"),
				},
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
			wantErr: true,
		},
	}

	k := filterChainsCollection()

	dupeByID := FilterChain{
		FilterChain: kong.FilterChain{
			ID: kong.String("dupe-filter-id"),
			Service: &kong.Service{
				ID:   kong.String("svc5"),
				Host: kong.String("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	if err := k.Add(dupeByID); err != nil {
		t.Error(err)
	}

	dupeByService := FilterChain{
		FilterChain: kong.FilterChain{
			ID: kong.String("fc5"),
			Service: &kong.Service{
				ID:   kong.String("dupe-service-id"),
				Host: kong.String("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	if err := k.Add(dupeByService); err != nil {
		t.Error(err)
	}

	dupeByRoute := FilterChain{
		FilterChain: kong.FilterChain{
			ID: kong.String("fc8"),
			Route: &kong.Route{
				ID:    kong.String("dupe-route-id"),
				Hosts: kong.StringSlice("example.test"),
			},
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	if err := k.Add(dupeByRoute); err != nil {
		t.Error(err)
	}

	for _, tt := range commonCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fc := FilterChain{FilterChain: tt.filterChain}
			if err := k.Add(fc); (err != nil) != tt.wantErr {
				t.Errorf("FilterChainsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	for _, tt := range tests {
		tt := tt
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
				ID: kong.String("i-dont-exist"),
				Service: &kong.Service{
					ID:   kong.String("svc2"),
					Host: kong.String("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on duplicate service ID",
			filterChain: kong.FilterChain{
				ID: kong.String("fc6"),
				Service: &kong.Service{
					ID:   kong.String("dupe-service-id"),
					Host: kong.String("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on duplicate Route ID",
			filterChain: kong.FilterChain{
				ID: kong.String("fc7"),
				Route: &kong.Route{
					ID:    kong.String("dupe-route-id"),
					Hosts: kong.StringSlice("example.test"),
				},
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
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
				ID:   kong.String("update-id-1"),
				Name: kong.String("old-name"),
				Service: &kong.Service{
					ID:   kong.String("my-service"),
					Host: kong.String("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
		}
		require.NoError(t, k.Add(pre))

		post := FilterChain{
			FilterChain: kong.FilterChain{
				ID:   kong.String("update-id-1"),
				Name: kong.String("new-name"),
				Service: &kong.Service{
					ID:   kong.String("my-service"),
					Host: kong.String("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
		}
		require.NoError(t, k.Update(post))

		updated, err := k.Get("update-id-1")
		assert.NoError(t, err)

		assert.Equal(t, "new-name", *updated.Name)
	})

	for _, tt := range commonCases {
		tt := tt
		if utils.Empty(tt.filterChain.ID) {
			continue
		}

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id := tt.filterChain.ID

			old := FilterChain{
				FilterChain: kong.FilterChain{
					ID:   id,
					Name: kong.String(fmt.Sprintf("%s-pre-update", *id)),
					Filters: []*kong.Filter{
						{
							Name: kong.String("my-filter"),
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
		tt := tt
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
			ID:   kong.String("get-test"),
			Name: kong.String("old-name"),
			Service: &kong.Service{
				ID:   kong.String("svc9"),
				Host: kong.String("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	require.NoError(t, collection.Add(filterChain))

	re, err := collection.Get("get-test")
	assert.Nil(err)
	assert.NotNil(re)
	assert.NotNil(re.FilterChain)
	assert.NotNil(re.FilterChain.ID)
	assert.Equal("get-test", *re.FilterChain.ID)

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
			ID: kong.String("get-prop-service"),
			Service: &kong.Service{
				ID:   kong.String("get-prop-service-id"),
				Host: kong.String("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	assert.NoError(collection.Add(serviceChain))

	routeChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: kong.String("get-prop-route"),
			Route: &kong.Route{
				ID:    kong.String("get-prop-route-id"),
				Hosts: kong.StringSlice("example.test"),
			},
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	assert.NoError(collection.Add(routeChain))

	re, err := collection.GetByProp("get-prop-service-id", "")
	assert.Nil(err)
	assert.NotNil(re)
	assert.NotNil(re.FilterChain)
	assert.NotNil(re.FilterChain.ID)
	assert.Equal("get-prop-service", *re.FilterChain.ID)

	re, err = collection.GetByProp("", "get-prop-route-id")
	assert.Nil(err)
	assert.NotNil(re)
	assert.NotNil(re.FilterChain)
	assert.NotNil(re.FilterChain.ID)
	assert.Equal("get-prop-route", *re.FilterChain.ID)

	re, err = collection.GetByProp("", "")
	assert.Nil(re)
	assert.NotNil(err)
	assert.Equal(errIDRequired, err)
}

func TestFilterChainsCollection_GetAllByServiceID(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	serviceChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: kong.String("get-all-service"),
			Service: &kong.Service{
				ID:   kong.String("get-all-service-id"),
				Host: kong.String("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	assert.NoError(collection.Add(serviceChain))

	re, err := collection.GetAllByServiceID("get-all-service-id")
	assert.Nil(err)
	assert.Equal([]*FilterChain{&serviceChain}, re)

	re, err = collection.GetAllByServiceID("no-chain-with-this-service-id")
	assert.Nil(err)
	assert.Equal(0, len(re))

	re, err = collection.GetAllByServiceID("")
	assert.Nil(re)
	assert.NotNil(err)
	assert.Equal(errIDRequired, err)
}

func TestFilterChainsCollection_GetAllByRouteID(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	routeChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: kong.String("get-all-route"),
			Route: &kong.Route{
				ID: kong.String("get-all-route-id"),
			},
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	assert.NoError(collection.Add(routeChain))

	re, err := collection.GetAllByRouteID("get-all-route-id")
	assert.Nil(err)
	assert.Equal([]*FilterChain{&routeChain}, re)

	re, err = collection.GetAllByRouteID("no-chain-with-this-route-id")
	assert.Nil(err)
	assert.Equal(0, len(re))

	re, err = collection.GetAllByRouteID("")
	assert.Nil(re)
	assert.NotNil(err)
	assert.Equal(errIDRequired, err)
}

func TestFilterChainsCollection_Delete(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	filterChain := FilterChain{
		FilterChain: kong.FilterChain{
			ID: kong.String("delete-test"),
			Service: &kong.Service{
				ID:   kong.String("my-service"),
				Host: kong.String("example.test"),
			},
			Route: nil,
			Filters: []*kong.Filter{
				{
					Name: kong.String("my-filter"),
				},
			},
		},
	}
	assert.NoError(collection.Add(filterChain))

	res, err := collection.Get("delete-test")
	assert.NotNil(res)
	assert.NoError(err)

	err = collection.Delete("delete-test")
	assert.NoError(err)

	res, err = collection.Get("delete-test")
	assert.Equal(ErrNotFound, err)
	assert.Nil(res)
}

func TestFilterChainsCollection_GetAll(t *testing.T) {
	assert := assert.New(t)
	collection := filterChainsCollection()

	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("get-all-%d", i)
		serviceID := fmt.Sprintf("get-all-service-id-%d", i)

		filterChain := FilterChain{
			FilterChain: kong.FilterChain{
				ID: kong.String(id),
				Service: &kong.Service{
					ID:   kong.String(serviceID),
					Host: kong.String("example.test"),
				},
				Route: nil,
				Filters: []*kong.Filter{
					{
						Name: kong.String("my-filter"),
					},
				},
			},
		}

		assert.NoError(collection.Add(filterChain))
	}

	allFilterChains, err := collection.GetAll()
	assert.Nil(err)
	assert.Equal(3, len(allFilterChains))
}
