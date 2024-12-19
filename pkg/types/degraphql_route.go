package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// degraphqlRouteCRUD implements crud.Actions interface.
type degraphqlRouteCRUD struct {
	client *kong.Client
}

func degraphqlRouteFromStruct(arg crud.Event) *state.DegraphqlRoute {
	degraphqlRoute, ok := arg.Obj.(*state.DegraphqlRoute)
	if !ok {
		panic("unexpected type, expected *state.DegraphqlRoute")
	}

	return degraphqlRoute
}

// Create creates a DegraphqlRoute in Kong.
// The arg should be of type crud.Event, containing the degraphql route to be created,
// else the function will panic.
// It returns a the created *state.DegraphqlRoute.
func (s *degraphqlRouteCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	degraphqlRoute := degraphqlRouteFromStruct(event)

	createdDegraphqlRoute, err := s.client.DegraphqlRoutes.Create(ctx, &degraphqlRoute.DegraphqlRoute)
	if err != nil {
		return nil, err
	}
	return &state.DegraphqlRoute{DegraphqlRoute: *createdDegraphqlRoute}, nil
}

// Delete deletes a DegraphqlRoute in Kong.
// The arg should be of type crud.Event, containing the degraphql route to be deleted,
// else the function will panic.
// It returns a the deleted *state.DegraphqlRoute.
func (s *degraphqlRouteCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	degraphqlRoute := degraphqlRouteFromStruct(event)
	err := s.client.DegraphqlRoutes.Delete(ctx, degraphqlRoute.Service.ID, degraphqlRoute.ID)
	if err != nil {
		return nil, err
	}
	return degraphqlRoute, nil
}

// Update updates a DegraphqlRoute in Kong.
// The arg should be of type crud.Event, containing the degraphql route to be updated,
// else the function will panic.
// It returns a the updated *state.DegraphqlRoute.
func (s *degraphqlRouteCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	degraphqlRoute := degraphqlRouteFromStruct(event)

	updatedDegraphqlRoute, err := s.client.DegraphqlRoutes.Update(ctx, &degraphqlRoute.DegraphqlRoute)
	if err != nil {
		return nil, err
	}
	return &state.DegraphqlRoute{DegraphqlRoute: *updatedDegraphqlRoute}, nil
}

type degraphqlRouteDiffer struct {
	kind crud.Kind

	currentState, targetState *state.KongState
}

func (d *degraphqlRouteDiffer) Deletes(handler func(crud.Event) error) error {
	currentDegraphqlRoutes, err := d.currentState.DegraphqlRoutes.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching degraphql routes from state: %w", err)
	}

	for _, degraphqlRoute := range currentDegraphqlRoutes {
		n, err := d.deleteDegraphqlRoute(degraphqlRoute)
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

func (d *degraphqlRouteDiffer) deleteDegraphqlRoute(degraphqlRoute *state.DegraphqlRoute) (*crud.Event, error) {
	_, err := d.targetState.DegraphqlRoutes.Get(*degraphqlRoute.ID)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  degraphqlRoute,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up degraphql route %q: %w", *degraphqlRoute.ID, err)
	}
	return nil, nil
}

func (d *degraphqlRouteDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetDegraphqlRoutes, err := d.targetState.DegraphqlRoutes.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching degraphql routes from state: %w", err)
	}

	for _, degraphqlRoute := range targetDegraphqlRoutes {
		n, err := d.createUpdateDegraphqlRoute(degraphqlRoute)
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

func (d *degraphqlRouteDiffer) createUpdateDegraphqlRoute(degraphqlRoute *state.DegraphqlRoute) (*crud.Event, error) {
	degraphqlRoute = &state.DegraphqlRoute{DegraphqlRoute: *degraphqlRoute.DeepCopy()}

	currentDegraphqlRoute, err := d.currentState.DegraphqlRoutes.Get(*degraphqlRoute.ID)

	if errors.Is(err, state.ErrNotFound) {
		// degraphql route not present, create it
		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  degraphqlRoute,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up degraphql route %q: %w",
			*degraphqlRoute.ID, err)
	}

	// found, check if update needed
	if !currentDegraphqlRoute.EqualWithOpts(degraphqlRoute, false) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    degraphqlRoute,
			OldObj: currentDegraphqlRoute,
		}, nil
	}
	return nil, nil
}
