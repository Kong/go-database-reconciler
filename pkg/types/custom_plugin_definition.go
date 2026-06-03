package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// customPluginDefinitionCRUD implements crud.Actions interface.
type customPluginDefinitionCRUD struct {
	client *kong.Client
}

func customPluginDefinitionFromStruct(arg crud.Event) *state.CustomPluginDefinition {
	cpd, ok := arg.Obj.(*state.CustomPluginDefinition)
	if !ok {
		panic("unexpected type, expected *state.CustomPluginDefinition")
	}
	return cpd
}

// Create creates a CustomPluginDefinition in Kong.
// The arg should be of type crud.Event, containing the CustomPluginDefinition to be created,
// else the function will panic.
// It returns the created *state.CustomPluginDefinition.
func (s *customPluginDefinitionCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	cpd := customPluginDefinitionFromStruct(event)
	created, err := s.client.CustomPlugins.Create(ctx, &cpd.CustomPluginDefinition)
	if err != nil {
		return nil, err
	}
	return &state.CustomPluginDefinition{CustomPluginDefinition: *created}, nil
}

// Delete deletes a CustomPluginDefinition in Kong.
// The arg should be of type crud.Event, containing the CustomPluginDefinition to be deleted,
// else the function will panic.
// It returns the deleted *state.CustomPluginDefinition.
func (s *customPluginDefinitionCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	cpd := customPluginDefinitionFromStruct(event)
	err := s.client.CustomPlugins.Delete(ctx, cpd.ID)
	if err != nil {
		return nil, err
	}
	return cpd, nil
}

// Update updates a CustomPluginDefinition in Kong.
// The arg should be of type crud.Event, containing the CustomPluginDefinition to be updated,
// else the function will panic.
// It returns the updated *state.CustomPluginDefinition.
func (s *customPluginDefinitionCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	cpd := customPluginDefinitionFromStruct(event)
	updated, err := s.client.CustomPlugins.Update(ctx, &cpd.CustomPluginDefinition)
	if err != nil {
		return nil, err
	}
	return &state.CustomPluginDefinition{CustomPluginDefinition: *updated}, nil
}

type customPluginDefinitionDiffer struct {
	kind                      crud.Kind
	currentState, targetState *state.KongState
}

// Deletes generates a memdb CRUD DELETE event for CustomPluginDefinitions
// which is then consumed by the differ and used to gate Kong client calls.
func (d *customPluginDefinitionDiffer) Deletes(handler func(crud.Event) error) error {
	currentCPDs, err := d.currentState.CustomPluginDefinitions.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching custom plugin definitions from state: %w", err)
	}
	for _, cpd := range currentCPDs {
		n, err := d.deleteCustomPluginDefinition(cpd)
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

func (d *customPluginDefinitionDiffer) deleteCustomPluginDefinition(
	cpd *state.CustomPluginDefinition,
) (*crud.Event, error) {
	_, err := d.targetState.CustomPluginDefinitions.Get(*cpd.ID)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  cpd,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up custom plugin definition %q: %w", *cpd.Name, err)
	}
	return nil, nil
}

// CreateAndUpdates generates a memdb CRUD CREATE/UPDATE event for CustomPluginDefinitions
// which is then consumed by the differ and used to gate Kong client calls.
func (d *customPluginDefinitionDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetCPDs, err := d.targetState.CustomPluginDefinitions.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching custom plugin definitions from state: %w", err)
	}
	for _, cpd := range targetCPDs {
		n, err := d.createUpdateCustomPluginDefinition(cpd)
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

func (d *customPluginDefinitionDiffer) createUpdateCustomPluginDefinition(
	cpd *state.CustomPluginDefinition,
) (*crud.Event, error) {
	cpdCopy := &state.CustomPluginDefinition{CustomPluginDefinition: *cpd.DeepCopy()}
	currentCPD, err := d.currentState.CustomPluginDefinitions.Get(*cpd.Name)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  cpdCopy,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up custom plugin definition %v: %w", *cpd.Name, err)
	}
	// found, check if update needed
	if !currentCPD.EqualWithOpts(cpdCopy, false, true) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    cpdCopy,
			OldObj: currentCPD,
		}, nil
	}
	return nil, nil
}
