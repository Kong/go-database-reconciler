package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// graphqlRateLimitingCostDecorationCRUD implements crud.Actions interface.
type graphqlRateLimitingCostDecorationCRUD struct {
	client *kong.Client
}

func graphqlRateLimitingCostDecorationFromStruct(arg crud.Event) *state.GraphqlRateLimitingCostDecoration {
	decoration, ok := arg.Obj.(*state.GraphqlRateLimitingCostDecoration)
	if !ok {
		panic("unexpected type, expected *state.GraphqlRateLimitingCostDecoration")
	}
	return decoration
}

// Create creates a GraphqlRateLimitingCostDecoration in Kong.
// The arg should be of type crud.Event, containing the decoration to be created,
// else the function will panic.
// It returns the created *state.GraphqlRateLimitingCostDecoration.
func (s *graphqlRateLimitingCostDecorationCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	decoration := graphqlRateLimitingCostDecorationFromStruct(event)

	createdDecoration, err := s.client.GraphqlRateLimitingCostDecorations.CreateWithID(ctx, &decoration.GraphqlRateLimitingCostDecoration)
	if err != nil {
		return nil, err
	}
	return &state.GraphqlRateLimitingCostDecoration{GraphqlRateLimitingCostDecoration: *createdDecoration}, nil
}

// Delete deletes a GraphqlRateLimitingCostDecoration in Kong.
// The arg should be of type crud.Event, containing the decoration to be deleted,
// else the function will panic.
// It returns the deleted *state.GraphqlRateLimitingCostDecoration.
func (s *graphqlRateLimitingCostDecorationCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	decoration := graphqlRateLimitingCostDecorationFromStruct(event)
	err := s.client.GraphqlRateLimitingCostDecorations.Delete(ctx, decoration.ID)
	if err != nil {
		return nil, err
	}
	return decoration, nil
}

// Update updates a GraphqlRateLimitingCostDecoration in Kong.
// The arg should be of type crud.Event, containing the decoration to be updated,
// else the function will panic.
// It returns the updated *state.GraphqlRateLimitingCostDecoration.
func (s *graphqlRateLimitingCostDecorationCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	decoration := graphqlRateLimitingCostDecorationFromStruct(event)

	updatedDecoration, err := s.client.GraphqlRateLimitingCostDecorations.Update(ctx, &decoration.GraphqlRateLimitingCostDecoration)
	if err != nil {
		return nil, err
	}
	return &state.GraphqlRateLimitingCostDecoration{GraphqlRateLimitingCostDecoration: *updatedDecoration}, nil
}

type graphqlRateLimitingCostDecorationDiffer struct {
	kind crud.Kind

	currentState, targetState *state.KongState
}

func (d *graphqlRateLimitingCostDecorationDiffer) Deletes(handler func(crud.Event) error) error {
	currentDecorations, err := d.currentState.GraphqlRateLimitingCostDecorations.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching graphql ratelimiting cost decorations from state: %w", err)
	}

	for _, decoration := range currentDecorations {
		n, err := d.deleteDecoration(decoration)
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

func (d *graphqlRateLimitingCostDecorationDiffer) deleteDecoration(
	decoration *state.GraphqlRateLimitingCostDecoration,
) (*crud.Event, error) {
	// First try to find by ID
	_, err := d.targetState.GraphqlRateLimitingCostDecorations.Get(*decoration.ID)
	if err == nil {
		// Found by ID, no delete needed
		return nil, nil
	}
	if !errors.Is(err, state.ErrNotFound) {
		return nil, fmt.Errorf("looking up graphql ratelimiting cost decoration %q: %w", *decoration.ID, err)
	}

	// Not found by ID, try to find by TypePath
	if decoration.TypePath != nil {
		_, err = d.targetState.GraphqlRateLimitingCostDecorations.GetByTypePath(*decoration.TypePath)
		if err == nil {
			// Found by TypePath, no delete needed
			return nil, nil
		}
		if !errors.Is(err, state.ErrNotFound) {
			return nil, fmt.Errorf("looking up graphql ratelimiting cost decoration by type_path %q: %w", *decoration.TypePath, err)
		}
	}

	// Not found by ID or TypePath, delete it
	return &crud.Event{
		Op:   crud.Delete,
		Kind: d.kind,
		Obj:  decoration,
	}, nil
}

func (d *graphqlRateLimitingCostDecorationDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetDecorations, err := d.targetState.GraphqlRateLimitingCostDecorations.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching graphql ratelimiting cost decorations from state: %w", err)
	}

	for _, decoration := range targetDecorations {
		n, err := d.createUpdateDecoration(decoration)
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

func (d *graphqlRateLimitingCostDecorationDiffer) createUpdateDecoration(
	decoration *state.GraphqlRateLimitingCostDecoration,
) (*crud.Event, error) {
	decoration = &state.GraphqlRateLimitingCostDecoration{GraphqlRateLimitingCostDecoration: *decoration.DeepCopy()}

	// First try to find by ID
	currentDecoration, err := d.currentState.GraphqlRateLimitingCostDecorations.Get(*decoration.ID)
	if err != nil && !errors.Is(err, state.ErrNotFound) {
		return nil, fmt.Errorf("error looking up graphql ratelimiting cost decoration %q: %w",
			*decoration.ID, err)
	}

	// If not found by ID, try to find by TypePath
	if errors.Is(err, state.ErrNotFound) && decoration.TypePath != nil {
		currentDecoration, err = d.currentState.GraphqlRateLimitingCostDecorations.GetByTypePath(*decoration.TypePath)
		if err != nil && !errors.Is(err, state.ErrNotFound) {
			return nil, fmt.Errorf("error looking up graphql ratelimiting cost decoration by type_path %q: %w",
				*decoration.TypePath, err)
		}
		// If found by TypePath, use the existing ID
		if err == nil && currentDecoration != nil {
			decoration.ID = currentDecoration.ID
		}
	}

	if errors.Is(err, state.ErrNotFound) {
		// decoration not present, create it
		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  decoration,
		}, nil
	}

	// found, check if update needed
	if !currentDecoration.EqualWithOpts(decoration, false) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    decoration,
			OldObj: currentDecoration,
		}, nil
	}
	return nil, nil
}
