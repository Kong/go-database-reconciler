package types

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
)

// partialCRUD implements crud.Actions interface.
type partialCRUD struct {
	client *kong.Client
}

func partialFromStruct(arg crud.Event) *state.Partial {
	partial, ok := arg.Obj.(*state.Partial)
	if !ok {
		panic("unexpected type, expected *state.Partial")
	}
	return partial
}

// Create creates a Partial in Kong.
// The arg should be of type crud.Event, containing the partial to be created,
// else the function will panic.
// It returns a the created *state.Partial.
func (s *partialCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	partial := partialFromStruct(event)
	createdPartial, err := s.client.Partials.Create(ctx, &partial.Partial)
	if err != nil {
		return nil, err
	}
	return &state.Partial{Partial: *createdPartial}, nil
}

// Delete deletes a Partial in Kong.
// The arg should be of type crud.Event, containing the partial to be deleted,
// else the function will panic.
// It returns a the deleted *state.Partial.
func (s *partialCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	partial := partialFromStruct(event)
	err := s.client.Partials.Delete(ctx, partial.ID)
	if err != nil {
		return nil, err
	}
	return partial, nil
}

// Update updates a Partial in Kong.
// The arg should be of type crud.Event, containing the partial to be updated,
// else the function will panic.
// It returns a the updated *state.Partial.
func (s *partialCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	partial := partialFromStruct(event)

	updatedPartial, err := s.client.Partials.Create(ctx, &partial.Partial)
	if err != nil {
		return nil, err
	}
	return &state.Partial{Partial: *updatedPartial}, nil
}

type partialDiffer struct {
	kind crud.Kind

	currentState, targetState *state.KongState
	client                    *kong.Client

	schemasCache map[string]map[string]interface{}
	mu           sync.Mutex
}

// Deletes generates a memdb CRUD DELETE event for Partials
// which is then consumed by the differ and used to gate Kong client calls.
func (d *partialDiffer) Deletes(handler func(crud.Event) error) error {
	currentPartials, err := d.currentState.Partials.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching partials from state: %w", err)
	}

	for _, partial := range currentPartials {
		n, err := d.deletePartial(partial)
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

func (d *partialDiffer) deletePartial(partial *state.Partial) (*crud.Event, error) {
	_, err := d.targetState.Partials.Get(*partial.ID)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  partial,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up partial %q: %w",
			*partial.Name, err)
	}
	return nil, nil
}

// CreateAndUpdates generates a memdb CRUD CREATE/UPDATE event for Partials
// which is then consumed by the differ and used to gate Kong client calls.
func (d *partialDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetPartials, err := d.targetState.Partials.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching partials from state: %w", err)
	}

	for _, partial := range targetPartials {
		n, err := d.createUpdatePartial(partial)
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

func (d *partialDiffer) createUpdatePartial(partial *state.Partial) (*crud.Event,
	error,
) {
	partialCopy := &state.Partial{Partial: *partial.DeepCopy()}

	var searchIDOrName string
	if utils.Empty(partial.Name) {
		searchIDOrName = *partial.ID
	} else {
		searchIDOrName = *partial.Name
	}

	currentPartial, err := d.currentState.Partials.Get(searchIDOrName)

	if errors.Is(err, state.ErrNotFound) {
		// partial not present, create it
		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  partialCopy,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up partial %v: %w",
			*partial.Name, err)
	}

	// found, check if update needed
	// before checking the diff, fill in the defaults
	currentPartial = &state.Partial{Partial: *currentPartial.DeepCopy()}
	schema, err := d.getPartialSchema(context.TODO(), *partial.Type)
	if err != nil {
		return nil, fmt.Errorf("failed getting schema for partial: %w", err)
	}
	partialWithDefaults := &state.Partial{Partial: *partial.DeepCopy()}
	err = kong.FillPartialDefaults(&partialWithDefaults.Partial, schema)
	if err != nil {
		return nil, fmt.Errorf("failed processing default fields for partial: %w", err)
	}

	if !currentPartial.EqualWithOpts(partialWithDefaults, false, true) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    partialCopy,
			OldObj: currentPartial,
		}, nil
	}
	return nil, nil
}

func (d *partialDiffer) getPartialSchema(ctx context.Context, partialType string) (map[string]interface{}, error) {
	var schema map[string]interface{}

	d.mu.Lock()
	defer d.mu.Unlock()
	if schema, ok := d.schemasCache[partialType]; ok {
		return schema, nil
	}

	schema, err := d.client.Partials.GetFullSchema(ctx, &partialType)
	if err != nil {
		return schema, err
	}
	d.schemasCache[partialType] = schema
	return schema, nil
}
