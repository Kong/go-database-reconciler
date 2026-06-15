package state

import (
	"reflect"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func servicePackagesCollection() *ServicePackagesCollection {
	return state().ServicePackages
}

func TestServicePackagesCollection_Add(t *testing.T) {
	type args struct {
		servicePackage ServicePackage
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: errorsWhenIDIsNil,
			args: args{
				servicePackage: ServicePackage{
					ServicePackage: konnect.ServicePackage{
						Name: new("foo"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors out without a name",
			args: args{
				servicePackage: ServicePackage{
					ServicePackage: konnect.ServicePackage{
						ID: new("id1"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "inserts with a name and ID",
			args: args{
				servicePackage: ServicePackage{
					ServicePackage: konnect.ServicePackage{
						ID:   new("id2"),
						Name: new("foo-name"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "errors on re-insert by ID",
			args: args{
				servicePackage: ServicePackage{
					ServicePackage: konnect.ServicePackage{
						ID:   new("id3"),
						Name: new("foo-name"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on re-insert by Name",
			args: args{
				servicePackage: ServicePackage{
					ServicePackage: konnect.ServicePackage{
						ID:   new("new-id"),
						Name: new("bar-name"),
					},
				},
			},
			wantErr: true,
		},
	}
	k := servicePackagesCollection()
	svc1 := ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   new("id3"),
			Name: new("bar-name"),
		},
	}
	k.Add(svc1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := k.Add(tt.args.servicePackage); (err != nil) != tt.wantErr {
				t.Errorf("ServicePackageCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServicePackagesCollection_Get(t *testing.T) {
	type args struct {
		nameOrID string
	}
	svc1 := ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   new("foo-id"),
			Name: new("foo-name"),
		},
	}
	svc2 := ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   new("bar-id"),
			Name: new("bar-name"),
		},
	}
	tests := []struct {
		name    string
		args    args
		want    *ServicePackage
		wantErr bool
	}{
		{
			name: "gets a servicePackage by ID",
			args: args{
				nameOrID: "foo-id",
			},
			want:    &svc1,
			wantErr: false,
		},
		{
			name: "gets a servicePackage by Name",
			args: args{
				nameOrID: "bar-name",
			},
			want:    &svc2,
			wantErr: false,
		},
		{
			name: "returns an ErrNotFound when no servicePackage found",
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
	k := servicePackagesCollection()
	k.Add(svc1)
	k.Add(svc2)
	for _, tt := range tests {
		tc := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := k.Get(tc.args.nameOrID)
			if (err != nil) != tc.wantErr {
				t.Errorf("ServicePackageCollection.Get() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ServicePackageCollection.Get() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestServicePackagesCollection_Update(t *testing.T) {
	svc1 := ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   new("foo-id"),
			Name: new("foo-name"),
		},
	}
	svc2 := ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   new("bar-id"),
			Name: new("bar-name"),
		},
	}
	svc3 := ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   new("foo-id"),
			Name: new("name"),
		},
	}
	type args struct {
		servicePackage ServicePackage
	}
	tests := []struct {
		name           string
		args           args
		wantErr        bool
		updatedService *ServicePackage
	}{
		{
			name: "update errors if servicePackage.ID is nil",
			args: args{
				servicePackage: ServicePackage{
					ServicePackage: konnect.ServicePackage{
						Name: new("name"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update errors if servicePackage does not exist",
			args: args{
				servicePackage: ServicePackage{
					ServicePackage: konnect.ServicePackage{
						ID: new("does-not-exist"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "update succeeds when ID is supplied",
			args: args{
				servicePackage: svc3,
			},
			wantErr:        false,
			updatedService: &svc3,
		},
	}
	k := servicePackagesCollection()
	k.Add(svc1)
	k.Add(svc2)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel()
			if err := k.Update(tt.args.servicePackage); (err != nil) != tt.wantErr {
				t.Errorf("ServicePackageCollection.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got, _ := k.Get(*tt.updatedService.ID)

				if !reflect.DeepEqual(got, tt.updatedService) {
					t.Errorf("update servicePackage, got = %#v, want %#v", got, tt.updatedService)
				}
			}
		})
	}
}

func TestServicePackageUpdate(t *testing.T) {
	assert := assert.New(t)
	k := servicePackagesCollection()
	svc1 := ServicePackage{
		ServicePackage: konnect.ServicePackage{
			ID:   new("foo-id"),
			Name: new("foo-name"),
		},
	}
	require.NoError(t, k.Add(svc1))

	svc1.Name = new("bar-name")
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

func TestServicePackagesInvalidType(t *testing.T) {
	assert := assert.New(t)
	collection := servicePackagesCollection()

	var route Route
	route.Name = new("my-route")
	route.ID = new("first")
	txn := collection.db.Txn(true)
	txn.Insert(servicePackageTableName, &route)
	txn.Commit()

	assert.Panics(func() {
		collection.Get("my-route")
	})
	assert.Panics(func() {
		collection.GetAll()
	})
}

func TestServicePackageDelete(t *testing.T) {
	collection := servicePackagesCollection()

	var servicePackage ServicePackage
	servicePackage.ID = new("first-id")
	servicePackage.Name = new("first-name")
	err := collection.Add(servicePackage)
	require.NoError(t, err)

	err = collection.Delete("does-not-exist")
	require.Error(t, err)
	err = collection.Delete("first-id")
	require.NoError(t, err)

	err = collection.Delete("first-name")
	require.Error(t, err)

	err = collection.Delete("")
	require.Error(t, err)
}

func TestServicePackageGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := servicePackagesCollection()

	services := []ServicePackage{
		{
			ServicePackage: konnect.ServicePackage{
				ID:   new("first"),
				Name: new("my-service1"),
			},
		},
		{
			ServicePackage: konnect.ServicePackage{
				ID:   new("second"),
				Name: new("my-service2"),
			},
		},
	}
	for _, s := range services {
		require.NoError(t, collection.Add(s))
	}

	allServices, err := collection.GetAll()

	require.NoError(t, err)
	assert.Len(allServices, len(services))
}

// Regression test
// to ensure that the memory reference of the pointer returned by Get()
// is different from the one stored in MemDB.
func TestServicePackagesGetAllMemoryReference(t *testing.T) {
	assert := assert.New(t)
	collection := servicePackagesCollection()

	services := []ServicePackage{
		{
			ServicePackage: konnect.ServicePackage{
				ID:          new("first"),
				Name:        new("my-service1"),
				Description: new("service1-desc"),
			},
		},
		{
			ServicePackage: konnect.ServicePackage{
				ID:          new("second"),
				Name:        new("my-service2"),
				Description: new("service2-desc"),
			},
		},
	}
	for _, s := range services {
		require.NoError(t, collection.Add(s))
	}

	allServices, err := collection.GetAll()
	require.NoError(t, err)
	assert.Len(allServices, len(services))

	allServices[0].Description = new("new-service1-desc")
	allServices[1].Description = new("new-service2-desc")

	servicePackage, err := collection.Get("my-service1")
	require.NoError(t, err)
	assert.NotNil(servicePackage)
	assert.Equal("service1-desc", *servicePackage.Description)
}
