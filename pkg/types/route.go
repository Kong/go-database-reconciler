package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

// routeCRUD implements crud.Actions interface.
type routeCRUD struct {
	client *kong.Client
}

// kong and konnect APIs only require IDs for referenced entities.
func stripRouteReferencesName(route *state.Route) {
	if route.Route.Service != nil && route.Route.Service.Name != nil {
		route.Route.Service.Name = nil
	}
}

func routeFromStruct(arg crud.Event) *state.Route {
	route, ok := arg.Obj.(*state.Route)
	if !ok {
		panic("unexpected type, expected *state.Route")
	}
	stripRouteReferencesName(route)
	return route
}

// Create creates a Route in Kong.
// The arg should be of type crud.Event, containing the route to be created,
// else the function will panic.
// It returns a the created *state.Route.
func (s *routeCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	route := routeFromStruct(event)
	createdRoute, err := s.client.Routes.Create(ctx, &route.Route)
	if err != nil {
		return nil, err
	}
	return &state.Route{Route: *createdRoute}, nil
}

// Delete deletes a Route in Kong.
// The arg should be of type crud.Event, containing the route to be deleted,
// else the function will panic.
// It returns a the deleted *state.Route.
func (s *routeCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	route := routeFromStruct(event)
	err := s.client.Routes.Delete(ctx, route.ID)
	if err != nil {
		return nil, err
	}
	return route, nil
}

// Update updates a Route in Kong.
// The arg should be of type crud.Event, containing the route to be updated,
// else the function will panic.
// It returns a the updated *state.Route.
func (s *routeCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	route := routeFromStruct(event)

	updatedRoute, err := s.client.Routes.Create(ctx, &route.Route)
	if err != nil {
		return nil, err
	}
	return &state.Route{Route: *updatedRoute}, nil
}

type routeDiffer struct {
	kind crud.Kind

	currentState, targetState *state.KongState

	client *kong.Client
}

func (d *routeDiffer) Deletes(handler func(crud.Event) error) error {
	currentRoutes, err := d.currentState.Routes.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching routes from state: %w", err)
	}

	for _, route := range currentRoutes {
		n, err := d.deleteRoute(route)
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

func (d *routeDiffer) deleteRoute(route *state.Route) (*crud.Event, error) {
	_, err := d.targetState.Routes.Get(*route.ID)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  route,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up route %q: %w",
			route.FriendlyName(), err)
	}
	return nil, nil
}

func (d *routeDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetRoutes, err := d.targetState.Routes.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching routes from state: %w", err)
	}

	for _, route := range targetRoutes {
		n, err := d.createUpdateRoute(route)
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

func (d *routeDiffer) createUpdateRoute(route *state.Route) (*crud.Event, error) {
	route = &state.Route{Route: *route.DeepCopy()}
	currentRoute, err := d.currentState.Routes.Get(*route.ID)
	if errors.Is(err, state.ErrNotFound) {
		if route.ID != nil {
			existingRoute, err := d.client.Routes.Get(context.TODO(), route.ID)
			if err != nil && !kong.IsNotFoundErr(err) {
				return nil, err
			}
			if existingRoute != nil {
				return nil, errDuplicateEntity("route", *route.ID)
			}
		}

		// route not present, create it
		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  route,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up route %q: %w",
			route.FriendlyName(), err)
	}
	// found, check if update needed

	if !currentRoute.EqualWithOpts(route, false, true, false) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    route,
			OldObj: currentRoute,
		}, nil
	}
	return nil, nil
}

func (d *routeDiffer) DuplicatesDeletes() ([]crud.Event, error) {
	targetRoutes, err := d.targetState.Routes.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error fetching routes from state: %w", err)
	}

	var events []crud.Event
	for _, route := range targetRoutes {
		event, err := d.deleteDuplicateRoute(route)
		if err != nil {
			return nil, err
		}
		if event != nil {
			events = append(events, *event)
		}
	}

	return events, nil
}

func (d *routeDiffer) deleteDuplicateRoute(targetRoute *state.Route) (*crud.Event, error) {
	if targetRoute == nil || targetRoute.Name == nil {
		// Nothing to do, cannot be a duplicate with no name.
		return nil, nil
	}

	currentRoute, err := d.currentState.Routes.Get(*targetRoute.Name)
	if errors.Is(err, state.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up route %q: %w", *targetRoute.Name, err)
	}

	if *currentRoute.ID != *targetRoute.ID {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: "route",
			Obj:  currentRoute,
		}, nil
	}

	return nil, nil
}
