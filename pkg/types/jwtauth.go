package types

import (
	"context"
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
)

// jwtAuthCRUD implements crud.Actions interface.
type jwtAuthCRUD struct {
	client *kong.Client
}

func jwtAuthFromStruct(arg crud.Event) *state.JWTAuth {
	jwtAuth, ok := arg.Obj.(*state.JWTAuth)
	if !ok {
		panic("unexpected type, expected *state.JWTAuth")
	}

	return jwtAuth
}

// Create creates a Route in Kong.
// The arg should be of type crud.Event, containing the jwtAuth to be created,
// else the function will panic.
// It returns a the created *state.Route.
func (s *jwtAuthCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	jwtAuth := jwtAuthFromStruct(event)
	cid := ""
	if !utils.Empty(jwtAuth.Consumer.Username) {
		cid = *jwtAuth.Consumer.Username
	}
	if !utils.Empty(jwtAuth.Consumer.ID) {
		cid = *jwtAuth.Consumer.ID
	}
	createdJWTAuth, err := s.client.JWTAuths.Create(ctx, &cid,
		&jwtAuth.JWTAuth)
	if err != nil {
		return nil, err
	}
	return &state.JWTAuth{JWTAuth: *createdJWTAuth}, nil
}

// Delete deletes a Route in Kong.
// The arg should be of type crud.Event, containing the jwtAuth to be deleted,
// else the function will panic.
// It returns a the deleted *state.Route.
func (s *jwtAuthCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	jwtAuth := jwtAuthFromStruct(event)
	cid := ""
	if !utils.Empty(jwtAuth.Consumer.Username) {
		cid = *jwtAuth.Consumer.Username
	}
	if !utils.Empty(jwtAuth.Consumer.ID) {
		cid = *jwtAuth.Consumer.ID
	}
	err := s.client.JWTAuths.Delete(ctx, &cid, jwtAuth.ID)
	if err != nil {
		return nil, err
	}
	return jwtAuth, nil
}

// Update updates a Route in Kong.
// The arg should be of type crud.Event, containing the jwtAuth to be updated,
// else the function will panic.
// It returns a the updated *state.Route.
func (s *jwtAuthCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	jwtAuth := jwtAuthFromStruct(event)

	cid := ""
	if !utils.Empty(jwtAuth.Consumer.Username) {
		cid = *jwtAuth.Consumer.Username
	}
	if !utils.Empty(jwtAuth.Consumer.ID) {
		cid = *jwtAuth.Consumer.ID
	}
	updatedJWTAuth, err := s.client.JWTAuths.Create(ctx, &cid, &jwtAuth.JWTAuth)
	if err != nil {
		return nil, err
	}
	return &state.JWTAuth{JWTAuth: *updatedJWTAuth}, nil
}

type jwtAuthDiffer struct {
	kind crud.Kind

	currentState, targetState *state.KongState

	client *kong.Client
}

func (d *jwtAuthDiffer) Deletes(handler func(crud.Event) error) error {
	currentJWTAuths, err := d.currentState.JWTAuths.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching jwt-auths from state: %w", err)
	}

	for _, jwtAuth := range currentJWTAuths {
		n, err := d.deleteJWTAuth(jwtAuth)
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

func (d *jwtAuthDiffer) deleteJWTAuth(jwtAuth *state.JWTAuth) (*crud.Event, error) {
	_, err := d.targetState.JWTAuths.Get(*jwtAuth.ID)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  jwtAuth,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up jwt-auth %q: %w", *jwtAuth.Key, err)
	}
	return nil, nil
}

func (d *jwtAuthDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetJWTAuths, err := d.targetState.JWTAuths.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching jwt-auths from state: %w", err)
	}

	for _, jwtAuth := range targetJWTAuths {
		n, err := d.createUpdateJWTAuth(jwtAuth)
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

func (d *jwtAuthDiffer) createUpdateJWTAuth(jwtAuth *state.JWTAuth) (*crud.Event, error) {
	jwtAuth = &state.JWTAuth{JWTAuth: *jwtAuth.DeepCopy()}
	currentJWTAuth, err := d.currentState.JWTAuths.Get(*jwtAuth.ID)
	if errors.Is(err, state.ErrNotFound) {
		if jwtAuth.ID != nil {
			existingJWTAuth, err := d.client.JWTAuths.GetByID(context.TODO(), jwtAuth.ID)
			if err != nil && !kong.IsNotFoundErr(err) {
				return nil, err
			}
			if existingJWTAuth != nil {
				return nil, errDuplicateEntity("jwt-auth credential", *jwtAuth.ID)
			}
		}

		// jwtAuth not present, create it

		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  jwtAuth,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up jwt-auth %q: %w",
			*jwtAuth.Key, err)
	}
	// found, check if update needed

	if !currentJWTAuth.EqualWithOpts(jwtAuth, false, true, false) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    jwtAuth,
			OldObj: currentJWTAuth,
		}, nil
	}
	return nil, nil
}
