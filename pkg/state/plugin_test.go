package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBar = "bar"
	testBaz = "baz"
)

func pluginsCollection() *PluginsCollection {
	return state().Plugins
}

func TestPluginsCollection_Add(t *testing.T) {
	type args struct {
		plugin Plugin
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: errorsWhenIDIsNil,
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						Name: new("foo"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors when Name is nil",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID: new("id1"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "inserts with a name and ID",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id2"),
						Name: new("bar-name"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "errors on re-insert when same ID is present",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id3"),
						Name: new("foo-name"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on re-insert when id is present",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id3"),
						Name: new("foobar-name"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on re-insert when when same association is present",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id4-new"),
						Name: new("key-auth"),
						Route: &kong.Route{
							ID: new("route1"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on re-insert when when same (multiple) association is present",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id5-new"),
						Name: new("key-auth"),
						Route: &kong.Route{
							ID: new("route1"),
						},
						Service: &kong.Service{
							ID: new("svc1"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "inserts with a partial linked",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id"),
						Name: new("rate-limiting"),
						Partials: []*kong.PartialLink{
							{
								Partial: &kong.Partial{
									ID: new("partial-id"),
								},
								Path: new("config_path"),
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	k := pluginsCollection()
	plugin1 := Plugin{
		Plugin: kong.Plugin{
			ID:   new("id3"),
			Name: new("foo-name"),
		},
	}
	plugin2 := Plugin{
		Plugin: kong.Plugin{
			ID:   new("id4"),
			Name: new("key-auth"),
			Route: &kong.Route{
				ID: new("route1"),
			},
		},
	}
	plugin3 := Plugin{
		Plugin: kong.Plugin{
			ID:   new("id5"),
			Name: new("key-auth"),
			Route: &kong.Route{
				ID: new("route1"),
			},
			Service: &kong.Service{
				ID: new("svc1"),
			},
		},
	}
	k.Add(plugin1)
	k.Add(plugin2)
	k.Add(plugin3)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := k.Add(tt.args.plugin); (err != nil) != tt.wantErr {
				t.Errorf("PluginsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPluginsCollection_Update(t *testing.T) {
	type args struct {
		plugin Plugin
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: errorsWhenIDIsNil,
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						Name: new("foo"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors when Name is nil",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID: new("id1"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors when the plugin is not present",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("does-not-exist-yet"),
						Name: new("bar-name"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "updates when ID is present",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id3"),
						Name: new("foo-name-new"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "errors on update when when same association is present",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("new-id"),
						Name: new("key-auth"),
						Route: &kong.Route{
							ID: new("route1"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "errors on update when when same (multiple) association is present",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("new-id"),
						Name: new("key-auth"),
						Route: &kong.Route{
							ID: new("route1"),
						},
						Service: &kong.Service{
							ID: new("svc1"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "updates linked partial",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id5"),
						Name: new("rate-limiting"),
						Partials: []*kong.PartialLink{
							{
								Partial: &kong.Partial{
									ID: new("partial-id-2"),
								},
								Path: new("config_path"),
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "updates linked partial path",
			args: args{
				plugin: Plugin{
					Plugin: kong.Plugin{
						ID:   new("id5"),
						Name: new("rate-limiting"),
						Partials: []*kong.PartialLink{
							{
								Partial: &kong.Partial{
									ID: new("partial-id-2"),
								},
								Path: new("config_path_new"),
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	k := pluginsCollection()
	plugin1 := Plugin{
		Plugin: kong.Plugin{
			ID:   new("id1"),
			Name: new("foo-name"),
		},
	}
	plugin2 := Plugin{
		Plugin: kong.Plugin{
			ID:   new("id2"),
			Name: new("key-auth"),
			Route: &kong.Route{
				ID: new("route1"),
			},
		},
	}
	plugin3 := Plugin{
		Plugin: kong.Plugin{
			ID:   new("id3"),
			Name: new("key-auth"),
			Route: &kong.Route{
				ID: new("route1"),
			},
			Service: &kong.Service{
				ID: new("svc1"),
			},
		},
	}
	plugin4 := Plugin{
		Plugin: kong.Plugin{
			ID:   new("id4"),
			Name: new("key-auth"),
			Route: &kong.Route{
				ID: new("route1"),
			},
			Service: &kong.Service{
				ID: new("svc1"),
			},
			ConsumerGroup: &kong.ConsumerGroup{
				ID: new("cg1"),
			},
		},
	}
	plugin5 := Plugin{
		Plugin: kong.Plugin{
			ID:   new("id5"),
			Name: new("rate-limiting"),
			Partials: []*kong.PartialLink{
				{
					Partial: &kong.Partial{
						ID: new("partial-id-1"),
					},
					Path: new("config_path"),
				},
			},
		},
	}
	k.Add(plugin1)
	k.Add(plugin2)
	k.Add(plugin3)
	k.Add(plugin4)
	k.Add(plugin5)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := k.Update(tt.args.plugin); (err != nil) != tt.wantErr {
				t.Errorf("PluginsCollection.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPluginGet(t *testing.T) {
	assert := assert.New(t)
	collection := pluginsCollection()

	var plugin Plugin
	plugin.Name = new("my-plugin")
	plugin.ID = new("first")
	plugin.Service = &kong.Service{
		ID:   new("service1-id"),
		Name: new("service1-name"),
	}
	assert.NotNil(plugin.Service)
	err := collection.Add(plugin)
	assert.NotNil(plugin.Service)
	require.NoError(t, err)

	re, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(re)
	assert.Equal("my-plugin", *re.Name)
	re.Service = &kong.Service{
		ID:   new("service2-id"),
		Name: new("service2-name"),
	}

	re, err = collection.Get("does-not-exists")
	assert.Equal(ErrNotFound, err)
	assert.Nil(re)
}

func TestGetPluginByProp(t *testing.T) {
	plugins := []Plugin{
		{
			Plugin: kong.Plugin{
				ID:   new("1"),
				Name: new("key-auth"),
				Config: map[string]any{
					testKey1: "value1",
				},
			},
		},
		{
			Plugin: kong.Plugin{
				ID:   new("2"),
				Name: new("key-auth"),
				Service: &kong.Service{
					ID: new("svc1"),
				},
				Config: map[string]any{
					testKey2: "value2",
				},
			},
		},
		{
			Plugin: kong.Plugin{
				ID:   new("3"),
				Name: new("key-auth"),
				Route: &kong.Route{
					ID: new("route1"),
				},
				Config: map[string]any{
					"key3": "value3",
				},
			},
		},
		{
			Plugin: kong.Plugin{
				ID:   new("4"),
				Name: new("key-auth"),
				Consumer: &kong.Consumer{
					ID: new("consumer1"),
				},
				Config: map[string]any{
					"key4": "value4",
				},
			},
		},
		{
			Plugin: kong.Plugin{
				ID:   new("5"),
				Name: new("key-auth"),
				ConsumerGroup: &kong.ConsumerGroup{
					ID: new("cg1"),
				},
				Config: map[string]any{
					"key5": "value5",
				},
			},
		},
	}
	assert := assert.New(t)
	collection := pluginsCollection()

	for _, p := range plugins {
		require.NoError(t, collection.Add(p))
	}

	plugin, err := collection.GetByProp("", "", "", "", "")
	assert.Nil(plugin)
	require.Error(t, err)

	plugin, err = collection.GetByProp("foo", "", "", "", "")
	assert.Nil(plugin)
	assert.Equal(ErrNotFound, err)

	plugin, err = collection.GetByProp("key-auth", "", "", "", "")
	require.NoError(t, err)
	assert.NotNil(plugin)
	assert.Equal("value1", plugin.Config[testKey1])

	plugin, err = collection.GetByProp("key-auth", "svc1", "", "", "")
	require.NoError(t, err)
	assert.NotNil(plugin)
	assert.Equal("value2", plugin.Config[testKey2])

	plugin, err = collection.GetByProp("key-auth", "", "route1", "", "")
	require.NoError(t, err)
	assert.NotNil(plugin)
	assert.Equal("value3", plugin.Config["key3"])

	plugin, err = collection.GetByProp("key-auth", "", "", "consumer1", "")
	require.NoError(t, err)
	assert.NotNil(plugin)
	assert.Equal("value4", plugin.Config["key4"])

	plugin, err = collection.GetByProp("key-auth", "", "", "", "cg1")
	require.NoError(t, err)
	assert.NotNil(plugin)
	assert.Equal("value5", plugin.Config["key5"])
}

func TestPluginsInvalidType(t *testing.T) {
	assert := assert.New(t)

	collection := pluginsCollection()

	var service Service
	service.Name = new("my-service")
	service.ID = new("first")
	txn := collection.db.Txn(true)
	txn.Insert(pluginTableName, &service)
	txn.Commit()

	assert.Panics(func() {
		collection.Get("first")
	})
	assert.Panics(func() {
		collection.GetAll()
	})
}

func TestPluginDelete(t *testing.T) {
	assert := assert.New(t)
	collection := pluginsCollection()

	var plugin Plugin
	plugin.ID = new("first")
	plugin.Name = new("my-plugin")
	plugin.Config = map[string]any{
		"foo":   testBar,
		testBaz: testBar,
	}
	plugin.Service = &kong.Service{
		ID:   new("service1-id"),
		Name: new("service1-name"),
	}
	err := collection.Add(plugin)
	require.NoError(t, err)

	p, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(p)
	assert.Equal(testBar, p.Config["foo"])

	err = collection.Delete(*p.ID)
	require.NoError(t, err)

	err = collection.Delete(*p.ID)
	require.Error(t, err)

	require.Error(t, collection.Delete(""))
}

func TestPluginGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := pluginsCollection()

	plugins := []*Plugin{
		{
			Plugin: kong.Plugin{
				ID:   new("first-id"),
				Name: new("key-auth"),
				Service: &kong.Service{
					ID:   new("service1-id"),
					Name: new("service1-name"),
				},
				Config: map[string]any{
					"foo":   testBar,
					testBaz: testBar,
				},
			},
		},
		{
			Plugin: kong.Plugin{
				ID:   new("second-id"),
				Name: new("basic-auth"),
				Service: &kong.Service{
					ID:   new("service1-id"),
					Name: new("service1-name"),
				},
			},
		},
		{
			Plugin: kong.Plugin{
				ID:   new("third-id"),
				Name: new("rate-limiting"),
				Route: &kong.Route{
					ID:   new("route1-id"),
					Name: new("route1-name"),
				},
			},
		},
		{
			Plugin: kong.Plugin{
				ID:   new("fourth-id"),
				Name: new("key-auth"),
				Route: &kong.Route{
					ID:   new("route1-id"),
					Name: new("route1-name"),
				},
			},
		},
	}

	for _, p := range plugins {
		require.NoError(t, collection.Add(*p))
	}

	allPlugins, err := collection.GetAll()
	require.NoError(t, err)
	assert.Len(allPlugins, len(plugins))

	allPlugins, err = collection.GetAllByName("")
	require.Error(t, err)
	assert.Nil(allPlugins)
	allPlugins, err = collection.GetAllByConsumerID("")
	require.Error(t, err)
	assert.Nil(allPlugins)
	allPlugins, err = collection.GetAllByRouteID("")
	require.Error(t, err)
	assert.Nil(allPlugins)
	allPlugins, err = collection.GetAllByServiceID("")
	require.Error(t, err)
	assert.Nil(allPlugins)

	allPlugins, err = collection.GetAllByName("key-auth")
	require.NoError(t, err)
	assert.Len(allPlugins, 2)

	allPlugins, err = collection.GetAllByRouteID("route1-id")
	require.NoError(t, err)
	assert.Len(allPlugins, 2)

	allPlugins, err = collection.GetAllByServiceID("service1-id")
	require.NoError(t, err)
	assert.Len(allPlugins, 2)

	allPlugins, err = collection.GetAllByServiceID("service-nope")
	require.NoError(t, err)
	assert.Empty(allPlugins)
}
