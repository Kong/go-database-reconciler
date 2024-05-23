package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// filterChainCRUD implements crud.Actions interface.
type filterChainCRUD struct {
	client *kong.Client
}

// kong and konnect APIs only require IDs for referenced entities.
func stripFilterChainReferencesName(filterChain *state.FilterChain) {
	if filterChain.FilterChain.Service != nil && filterChain.FilterChain.Service.Name != nil {
		filterChain.FilterChain.Service.Name = nil
	}
	if filterChain.FilterChain.Route != nil && filterChain.FilterChain.Route.Name != nil {
		filterChain.FilterChain.Route.Name = nil
	}
}

func filterChainFromStruct(arg crud.Event) *state.FilterChain {
	filterChain, ok := arg.Obj.(*state.FilterChain)
	if !ok {
		panic("unexpected type, expected *state.FilterChain")
	}
	stripFilterChainReferencesName(filterChain)
	return filterChain
}

// Create creates a FilterChain in Kong.
// The arg should be of type crud.Event, containing the filter chain to be created,
// else the function will panic.
// It returns a the created *state.FilterChain.
func (s *filterChainCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	filterChain := filterChainFromStruct(event)

	createdFilterChain, err := s.client.FilterChains.Create(ctx, &filterChain.FilterChain)
	if err != nil {
		return nil, err
	}
	return &state.FilterChain{FilterChain: *createdFilterChain}, nil
}

// Delete deletes a FilterChain in Kong.
// The arg should be of type crud.Event, containing the filter chain to be deleted,
// else the function will panic.
// It returns a the deleted *state.FilterChain.
func (s *filterChainCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	filterChain := filterChainFromStruct(event)
	err := s.client.FilterChains.Delete(ctx, filterChain.ID)
	if err != nil {
		return nil, err
	}
	return filterChain, nil
}

// Update updates a FilterChain in Kong.
// The arg should be of type crud.Event, containing the filter chain to be updated,
// else the function will panic.
// It returns a the updated *state.FilterChain.
func (s *filterChainCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	filterChain := filterChainFromStruct(event)

	updatedFilterChain, err := s.client.FilterChains.Create(ctx, &filterChain.FilterChain)
	if err != nil {
		return nil, err
	}
	return &state.FilterChain{FilterChain: *updatedFilterChain}, nil
}

type filterChainDiffer struct {
	kind crud.Kind

	currentState, targetState *state.KongState
}

func (d *filterChainDiffer) Deletes(handler func(crud.Event) error) error {
	currentFilterChains, err := d.currentState.FilterChains.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching filter chains from state: %w", err)
	}

	for _, filterChain := range currentFilterChains {
		n, err := d.deleteFilterChain(filterChain)
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

func (d *filterChainDiffer) deleteFilterChain(filterChain *state.FilterChain) (*crud.Event, error) {
	filterChain = &state.FilterChain{FilterChain: *filterChain.DeepCopy()}

	serviceID, routeID := filterChainForeignNames(filterChain)
	_, err := d.targetState.FilterChains.GetByProp(
		serviceID, routeID,
	)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  filterChain,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up filter chain %q: %w", *filterChain.ID, err)
	}
	return nil, nil
}

func (d *filterChainDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetFilterChains, err := d.targetState.FilterChains.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching filter chains from state: %w", err)
	}

	for _, filterChain := range targetFilterChains {
		n, err := d.createUpdateFilterChain(filterChain)
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

func (d *filterChainDiffer) createUpdateFilterChain(filterChain *state.FilterChain) (*crud.Event, error) {
	filterChain = &state.FilterChain{FilterChain: *filterChain.DeepCopy()}

	name := ""
	if filterChain.Name != nil {
		name = *filterChain.Name
	}

	serviceID, routeID := filterChainForeignNames(filterChain)
	currentFilterChain, err := d.currentState.FilterChains.GetByProp(
		serviceID, routeID,
	)
	if errors.Is(err, state.ErrNotFound) {
		// filter chain not present, create it

		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  filterChain,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up filter chain %q: %w",
			name, err)
	}
	currentFilterChain = &state.FilterChain{FilterChain: *currentFilterChain.DeepCopy()}
	// found, check if update needed

	if !currentFilterChain.EqualWithOpts(filterChain, false, true, false) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    filterChain,
			OldObj: currentFilterChain,
		}, nil
	}
	return nil, nil
}

func filterChainForeignNames(p *state.FilterChain) (serviceID, routeID string) {
	if p == nil {
		return
	}
	if p.Service != nil && p.Service.ID != nil {
		serviceID = *p.Service.ID
	}
	if p.Route != nil && p.Route.ID != nil {
		routeID = *p.Route.ID
	}
	return
}
