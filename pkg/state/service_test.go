package state

import (
	"reflect"
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func servicesCollection() *ServicesCollection {
	return state().Services
}

func TestServicesCollection_Add(t *testing.T) {
	type args struct {
		service Service
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "errors when ID is nil",
			args: args{
				service: Service{
					Service: kong.Service{
						Name: kong.String("foo"),
						Host: kong.String("example.com"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "inserts without a name",
			args: args{
				service: Service{
					Service: kong.Service{
						ID:   kong.String("id1"),
						Host: kong.String("example.com"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "inserts with a name and ID",
			args: args{
				service: Service{
					Service: kong.Service{
						ID:   kong.String("id2"),
						Name: kong.String("foo-name"),
						Host: kong.String("example.com"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "errors on re-insert by ID",
			args: args{
				service: Service{
					Service: kong.Service{
						ID:   kong.String("id3"),
						Name: kong.String("foo-name"),
						Host: kong.String("example.com"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on re-insert by Name",
			args: args{
				service: Service{
					Service: kong.Service{
						ID:   kong.String("new-id"),
						Name: kong.String("bar-name"),
						Host: kong.String("example.com"),
					},
				},
			},
			wantErr: true,
		},
	}
	k := servicesCollection()
	svc1 := Service{
		Service: kong.Service{
			ID:   kong.String("id3"),
			Name: kong.String("bar-name"),
			Host: kong.String("example.com"),
		},
	}
	k.Add(svc1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := k.Add(tt.args.service); (err != nil) != tt.wantErr {
				t.Errorf("ServicesCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceInsertIgnoreDuplicate(t *testing.T) {
	collection := servicesCollection()

	var s Service
	s.ID = kong.String("my-id")
	s.Name = kong.String("first")
	err := collection.Add(s)
	require.NoError(t, err)
	err = collection.AddIgnoringDuplicates(s)
	require.NoError(t, err)
}

func TestServiceInsertIgnoreDuplicateDoesNotPanic(t *testing.T) {
	collection := servicesCollection()

	var s Service
	err := collection.AddIgnoringDuplicates(s)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ID is required")
}

func TestServicesCollection_Get(t *testing.T) {
	type args struct {
		nameOrID string
	}
	svc1 := Service{
		Service: kong.Service{
			ID:   kong.String("foo-id"),
			Host: kong.String("example.com"),
		},
	}
	svc2 := Service{
		Service: kong.Service{
			ID:   kong.String("bar-id"),
			Name: kong.String("bar-name"),
			Host: kong.String("example.com"),
		},
	}
	tests := []struct {
		name    string
		args    args
		want    *Service
		wantErr bool
	}{
		{
			name: "gets a service by ID",
			args: args{
				nameOrID: "foo-id",
			},
			want:    &svc1,
			wantErr: false,
		},
		{
			name: "gets a service by Name",
			args: args{
				nameOrID: "bar-name",
			},
			want:    &svc2,
			wantErr: false,
		},
		{
			name: "returns an ErrNotFound when no service found",
			args: args{
				nameOrID: "baz-id",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns an error when ID is empty",
			args: args{
				nameOrID: "",
			},
			want:    nil,
			wantErr: true,
		},
	}
	k := servicesCollection()
	k.Add(svc1)
	k.Add(svc2)
	for _, tt := range tests {
		tc := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := k.Get(tc.args.nameOrID)
			if (err != nil) != tc.wantErr {
				t.Errorf("ServicesCollection.Get() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ServicesCollection.Get() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestServicesCollection_Update(t *testing.T) {
	svc1 := Service{
		Service: kong.Service{
			ID:   kong.String("foo-id"),
			Host: kong.String("example.com"),
		},
	}
	svc2 := Service{
		Service: kong.Service{
			ID:   kong.String("bar-id"),
			Name: kong.String("bar-name"),
			Host: kong.String("example.com"),
		},
	}
	svc3 := Service{
		Service: kong.Service{
			ID:   kong.String("foo-id"),
			Name: kong.String("name"),
			Host: kong.String("2.example.com"),
			Port: kong.Int(42),
		},
	}
	type args struct {
		service Service
	}
	tests := []struct {
		name           string
		args           args
		wantErr        bool
		updatedService *Service
	}{
		{
			name: "update errors if service.ID is nil",
			args: args{
				service: Service{
					Service: kong.Service{
						Name: kong.String("name"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update errors if service does not exist",
			args: args{
				service: Service{
					Service: kong.Service{
						ID: kong.String("does-not-exist"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update succeeds when ID is supplied",
			args: args{
				service: svc3,
			},
			wantErr:        false,
			updatedService: &svc3,
		},
	}
	k := servicesCollection()
	k.Add(svc1)
	k.Add(svc2)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel()
			if err := k.Update(tt.args.service); (err != nil) != tt.wantErr {
				t.Errorf("ServicesCollection.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got, _ := k.Get(*tt.updatedService.ID)

				if !reflect.DeepEqual(got, tt.updatedService) {
					t.Errorf("update service, got = %#v, want %#v", got, tt.updatedService)
				}
			}
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	assert := assert.New(t)
	k := servicesCollection()
	svc1 := Service{
		Service: kong.Service{
			ID:   kong.String("foo-id"),
			Name: kong.String("foo-name"),
			Host: kong.String("example.com"),
		},
	}
	require.NoError(t, k.Add(svc1))

	svc1.Name = kong.String("bar-name")
	require.NoError(t, k.Update(svc1))

	r, err := k.Get("foo-id")
	require.NoError(t, err)
	assert.NotNil(r)

	r, err = k.Get("bar-name")
	require.NoError(t, err)
	assert.NotNil(r)

	r, err = k.Get("foo-name")
	require.Error(t, err)
	assert.Nil(r)
}

// Regression test
// to ensure that the memory reference of the pointer returned by Get()
// is different from the one stored in MemDB.
func TestServiceGetMemoryReference(t *testing.T) {
	assert := assert.New(t)
	collection := servicesCollection()

	var service Service
	service.Name = kong.String("my-service")
	service.ID = kong.String("first")
	err := collection.Add(service)
	require.NoError(t, err)

	se, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(se)
	se.Host = kong.String("example.com")

	se, err = collection.Get("my-service")
	require.NoError(t, err)
	assert.NotNil(se)
	assert.Nil(se.Host)
}

func TestServicesInvalidType(t *testing.T) {
	assert := assert.New(t)
	collection := servicesCollection()

	var route Route
	route.Name = kong.String("my-route")
	route.ID = kong.String("first")
	txn := collection.db.Txn(true)
	txn.Insert(serviceTableName, &route)
	txn.Commit()

	assert.Panics(func() {
		collection.Get("my-route")
	})
	assert.Panics(func() {
		collection.GetAll()
	})
}

func TestServiceDelete(t *testing.T) {
	collection := servicesCollection()

	var service Service
	service.ID = kong.String("first")
	service.Host = kong.String("example.com")
	err := collection.Add(service)
	require.NoError(t, err)

	err = collection.Delete("does-not-exist")
	require.Error(t, err)
	err = collection.Delete("first")
	require.NoError(t, err)

	err = collection.Delete("first")
	require.Error(t, err)

	err = collection.Delete("")
	require.Error(t, err)
}

func TestServiceGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := servicesCollection()

	services := []Service{
		{
			Service: kong.Service{
				ID:   kong.String("first"),
				Name: kong.String("my-service1"),
				Host: kong.String("example.com"),
			},
		},
		{
			Service: kong.Service{
				ID:   kong.String("second"),
				Name: kong.String("my-service2"),
				Host: kong.String("example.com"),
			},
		},
	}
	for _, s := range services {
		require.NoError(t, collection.Add(s))
	}

	allServices, err := collection.GetAll()

	require.NoError(t, err)
	assert.Equal(len(services), len(allServices))
}

// Regression test
// to ensure that the memory reference of the pointer returned by Get()
// is different from the one stored in MemDB.
func TestServiceGetAllMemoryReference(t *testing.T) {
	assert := assert.New(t)
	collection := servicesCollection()

	services := []Service{
		{
			Service: kong.Service{
				ID:   kong.String("first"),
				Name: kong.String("my-service1"),
				Host: kong.String("example.com"),
			},
		},
		{
			Service: kong.Service{
				ID:   kong.String("second"),
				Name: kong.String("my-service2"),
				Host: kong.String("example.com"),
			},
		},
	}
	for _, s := range services {
		require.NoError(t, collection.Add(s))
	}

	allServices, err := collection.GetAll()
	require.NoError(t, err)
	assert.Equal(len(services), len(allServices))

	allServices[0].Host = kong.String("new.example.com")
	allServices[1].Host = kong.String("new.example.com")

	service, err := collection.Get("my-service1")
	require.NoError(t, err)
	assert.NotNil(service)
	assert.Equal("example.com", *service.Host)
}
