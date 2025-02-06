package types

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kong/go-database-reconciler/pkg/cprint"
	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// consumerGroupPluginCRUD implements crud.Actions interface.
type consumerGroupPluginCRUD struct {
	client    *kong.Client
	isKonnect bool
}

func consumerGroupPluginFromStruct(arg crud.Event) *state.ConsumerGroupPlugin {
	plugin, ok := arg.Obj.(*state.ConsumerGroupPlugin)
	if !ok {
		panic("unexpected type, expected *state.ConsumerGroupPlugin")
	}
	return plugin
}

// Create creates a consumerGroupPlugin in Kong.
// The arg should be of type crud.Event, containing the consumerGroupPlugin to be created,
// else the function will panic.
// It returns the created *state.consumerGroupPlugin.
func (s *consumerGroupPluginCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	plugin := consumerGroupPluginFromStruct(event)
	config := map[string]kong.Configuration{
		"config": plugin.Config,
	}
	var (
		res *kong.ConsumerGroupRLA
		err error
	)
	if s.isKonnect {
		res, err = konnect.CreateRateLimitingAdvancedPlugin(ctx, s.client, plugin.ConsumerGroup.ID, plugin.Config)
		if err != nil {
			return nil, err
		}
	} else {
		res, err = s.client.ConsumerGroups.UpdateRateLimitingAdvancedPlugin(ctx, plugin.ConsumerGroup.ID, config)
		if err != nil {
			return nil, err
		}
	}
	return &state.ConsumerGroupPlugin{
		ConsumerGroupPlugin: kong.ConsumerGroupPlugin{
			Name:   res.Plugin,
			Config: res.Config,
			ConsumerGroup: &kong.ConsumerGroup{
				ID: res.ConsumerGroup,
			},
		},
	}, nil
}

// Update updates a consumerGroupConsumer in Kong.
// The arg should be of type crud.Event, containing the consumerGroupConsumer to be updated,
// else the function will panic.
// It returns the updated *state.consumerGroupConsumer.
func (s *consumerGroupPluginCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	plugin := consumerGroupPluginFromStruct(event)
	config := map[string]kong.Configuration{
		"config": plugin.Config,
	}
	var (
		res *kong.ConsumerGroupRLA
		err error
	)
	if s.isKonnect {
		res, err = konnect.UpdateRateLimitingAdvancedPlugin(ctx, s.client, plugin.ConsumerGroup.ID, plugin.Config)
		if err != nil {
			return nil, err
		}
	} else {
		res, err = s.client.ConsumerGroups.UpdateRateLimitingAdvancedPlugin(ctx, plugin.ConsumerGroup.ID, config)
		if err != nil {
			return nil, err
		}
	}
	return &state.ConsumerGroupPlugin{
		ConsumerGroupPlugin: kong.ConsumerGroupPlugin{
			ID:     plugin.ID,
			Name:   res.Plugin,
			Config: res.Config,
			ConsumerGroup: &kong.ConsumerGroup{
				ID:   plugin.ConsumerGroup.ID,
				Name: res.ConsumerGroup,
			},
		},
	}, nil
}

// Delete is just a placeholder because Admin API doesn't support DELETEs
// for consumer groups plugins.
func (s *consumerGroupPluginCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	plugin := consumerGroupPluginFromStruct(event)
	if s.isKonnect {
		err := konnect.DeleteRateLimitingAdvancedPlugin(ctx, s.client, plugin.ConsumerGroup.ID)
		if err != nil {
			return nil, err
		}
		return plugin, nil
	}
	return nil, nil
}

type consumerGroupPluginDiffer struct {
	kind crud.Kind
	once sync.Once

	currentState, targetState *state.KongState
}

func (d *consumerGroupPluginDiffer) warnConsumerGroupPlugin() {
	const (
		consumerGroupPluginWarning = "Warning: consumer-group policy overrides " +
			"are deprecated. Please use Consumer Groups scoped\n" +
			"Plugins when running against Kong Enterprise 3.4.0 and above.\n\n" +
			"Mixing of both Consumer Groups scoped plugins and policy-overrides can be problematic.\n\n" +
			"Check https://docs.konghq.com/gateway/latest/kong-enterprise/consumer-groups/ for more information."
	)
	d.once.Do(func() {
		cprint.UpdatePrintln(consumerGroupPluginWarning)
	})
}

func (d *consumerGroupPluginDiffer) Deletes(_ func(crud.Event) error) error {
	return nil
}

func (d *consumerGroupPluginDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetPlugins, err := d.targetState.ConsumerGroupPlugins.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching consumerGroupPlugins from state: %w", err)
	}

	for _, plugin := range targetPlugins {
		n, err := d.createUpdateConsumerGroupPlugin(plugin)
		if err != nil {
			return err
		}
		if n != nil {
			err = handler(*n)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *consumerGroupPluginDiffer) createUpdateConsumerGroupPlugin(
	plugin *state.ConsumerGroupPlugin,
) (*crud.Event, error) {
	d.warnConsumerGroupPlugin()
	pluginCopy := &state.ConsumerGroupPlugin{ConsumerGroupPlugin: *plugin.DeepCopy()}
	currentPlugin, err := d.currentState.ConsumerGroupPlugins.Get(
		*plugin.Name, *plugin.ConsumerGroup.ID,
	)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Create,
			Kind: "consumer-group-plugin",
			Obj:  pluginCopy,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up consumerGroupPlugin %v: %w",
			*currentPlugin.ID, err)
	}

	// found, check if update needed
	if !currentPlugin.EqualWithOpts(pluginCopy, false, true) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   "consumer-group-plugin",
			Obj:    pluginCopy,
			OldObj: currentPlugin,
		}, nil
	}
	return nil, nil
}
