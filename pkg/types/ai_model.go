package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// aiModelDefinitionCRUD implements crud.Actions interface.
type aiModelDefinitionCRUD struct {
	client *kong.Client
}

func aiModelDefinitionFromStruct(arg crud.Event) *state.AIModel {
	amd, ok := arg.Obj.(*state.AIModel)
	if !ok {
		panic("unexpected type, expected *state.AIModel")
	}
	return amd
}

// Create creates an AIModel in Kong.
// The arg should be of type crud.Event, containing the AIModel to be created,
// else the function will panic.
// It returns the created *state.AIModel.
func (s *aiModelDefinitionCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	amd := aiModelDefinitionFromStruct(event)
	created, err := s.client.AIModels.Create(ctx, &amd.AIModel)
	if err != nil {
		return nil, err
	}
	return &state.AIModel{AIModel: *created}, nil
}

// Delete deletes an AIModel in Kong.
// The arg should be of type crud.Event, containing the AIModel to be deleted,
// else the function will panic.
// It returns the deleted *state.AIModel.
func (s *aiModelDefinitionCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	amd := aiModelDefinitionFromStruct(event)
	err := s.client.AIModels.Delete(ctx, amd.ID)
	if err != nil {
		return nil, err
	}
	return amd, nil
}

// Update updates an AIModel in Kong.
// The arg should be of type crud.Event, containing the AIModel to be updated,
// else the function will panic.
// It returns the updated *state.AIModel.
func (s *aiModelDefinitionCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	amd := aiModelDefinitionFromStruct(event)
	updated, err := s.client.AIModels.Update(ctx, &amd.AIModel)
	if err != nil {
		return nil, err
	}
	return &state.AIModel{AIModel: *updated}, nil
}

type aiModelDefinitionDiffer struct {
	kind                      crud.Kind
	currentState, targetState *state.KongState
}

// Deletes generates a memdb CRUD DELETE event for AIModels
// which is then consumed by the differ and used to gate Kong client calls.
func (d *aiModelDefinitionDiffer) Deletes(handler func(crud.Event) error) error {
	currentAMDs, err := d.currentState.AIModels.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching AI model definitions from state: %w", err)
	}
	for _, amd := range currentAMDs {
		n, err := d.deleteAIModel(amd)
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

func (d *aiModelDefinitionDiffer) deleteAIModel(
	amd *state.AIModel,
) (*crud.Event, error) {
	_, err := d.targetState.AIModels.Get(*amd.ID)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  amd,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up AI model definition %q: %w", *amd.Name, err)
	}
	return nil, nil
}

// CreateAndUpdates generates a memdb CRUD CREATE/UPDATE event for AIModels
// which is then consumed by the differ and used to gate Kong client calls.
func (d *aiModelDefinitionDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetAMDs, err := d.targetState.AIModels.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching AI model definitions from state: %w", err)
	}
	for _, amd := range targetAMDs {
		n, err := d.createUpdateAIModel(amd)
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

func (d *aiModelDefinitionDiffer) createUpdateAIModel(
	amd *state.AIModel,
) (*crud.Event, error) {
	amdCopy := &state.AIModel{AIModel: *amd.DeepCopy()}
	currentAMD, err := d.currentState.AIModels.Get(*amd.Name)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  amdCopy,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up AI model definition %v: %w", *amd.Name, err)
	}
	// found, check if update needed
	if !currentAMD.EqualWithOpts(amdCopy, false, true) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    amdCopy,
			OldObj: currentAMD,
		}, nil
	}
	return nil, nil
}
