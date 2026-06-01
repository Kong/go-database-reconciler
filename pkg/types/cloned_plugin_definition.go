package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// clonedPluginDefinitionCRUD implements crud.Actions interface.
type clonedPluginDefinitionCRUD struct {
	client *kong.Client
}

func clonedPluginDefinitionFromStruct(arg crud.Event) *state.ClonedPluginDefinition {
	cpd, ok := arg.Obj.(*state.ClonedPluginDefinition)
	if !ok {
		panic("unexpected type, expected *state.ClonedPluginDefinition")
	}
	return cpd
}

// Create creates a ClonedPluginDefinition in Kong.
// The arg should be of type crud.Event, containing the ClonedPluginDefinition to be created,
// else the function will panic.
// It returns the created *state.ClonedPluginDefinition.
func (s *clonedPluginDefinitionCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	cpd := clonedPluginDefinitionFromStruct(event)
	created, err := s.client.ClonedPlugins.Create(ctx, &cpd.ClonedPluginDefinition)
	if err != nil {
		return nil, err
	}
	return &state.ClonedPluginDefinition{ClonedPluginDefinition: *created}, nil
}

// Delete deletes a ClonedPluginDefinition in Kong.
// The arg should be of type crud.Event, containing the ClonedPluginDefinition to be deleted,
// else the function will panic.
// It returns the deleted *state.ClonedPluginDefinition.
func (s *clonedPluginDefinitionCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	cpd := clonedPluginDefinitionFromStruct(event)
	err := s.client.ClonedPlugins.Delete(ctx, cpd.ID)
	if err != nil {
		return nil, err
	}
	return cpd, nil
}

// Update updates a ClonedPluginDefinition in Kong.
// The arg should be of type crud.Event, containing the ClonedPluginDefinition to be updated,
// else the function will panic.
// It returns the updated *state.ClonedPluginDefinition.
func (s *clonedPluginDefinitionCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	cpd := clonedPluginDefinitionFromStruct(event)
	updated, err := s.client.ClonedPlugins.Update(ctx, &cpd.ClonedPluginDefinition)
	if err != nil {
		return nil, err
	}
	return &state.ClonedPluginDefinition{ClonedPluginDefinition: *updated}, nil
}

type clonedPluginDefinitionDiffer struct {
	kind                      crud.Kind
	currentState, targetState *state.KongState
}

// Deletes generates a memdb CRUD DELETE event for ClonedPluginDefinitions
// which is then consumed by the differ and used to gate Kong client calls.
func (d *clonedPluginDefinitionDiffer) Deletes(handler func(crud.Event) error) error {
	currentCPDs, err := d.currentState.ClonedPluginDefinitions.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching cloned plugin definitions from state: %w", err)
	}
	for _, cpd := range currentCPDs {
		n, err := d.deleteClonedPluginDefinition(cpd)
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

func (d *clonedPluginDefinitionDiffer) deleteClonedPluginDefinition(cpd *state.ClonedPluginDefinition) (*crud.Event, error) {
	_, err := d.targetState.ClonedPluginDefinitions.Get(*cpd.ID)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  cpd,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up cloned plugin definition %q: %w", *cpd.Name, err)
	}
	return nil, nil
}

// CreateAndUpdates generates a memdb CRUD CREATE/UPDATE event for ClonedPluginDefinitions
// which is then consumed by the differ and used to gate Kong client calls.
func (d *clonedPluginDefinitionDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetCPDs, err := d.targetState.ClonedPluginDefinitions.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching cloned plugin definitions from state: %w", err)
	}
	for _, cpd := range targetCPDs {
		n, err := d.createUpdateClonedPluginDefinition(cpd)
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

func (d *clonedPluginDefinitionDiffer) createUpdateClonedPluginDefinition(cpd *state.ClonedPluginDefinition) (*crud.Event, error) {
	cpdCopy := &state.ClonedPluginDefinition{ClonedPluginDefinition: *cpd.ClonedPluginDefinition.DeepCopy()}
	currentCPD, err := d.currentState.ClonedPluginDefinitions.Get(*cpd.Name)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  cpdCopy,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up cloned plugin definition %v: %w", *cpd.Name, err)
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
