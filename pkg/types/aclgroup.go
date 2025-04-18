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

// aclGroupCRUD implements crud.Actions interface.
type aclGroupCRUD struct {
	client *kong.Client
}

func aclGroupFromStruct(arg crud.Event) *state.ACLGroup {
	aclGroup, ok := arg.Obj.(*state.ACLGroup)
	if !ok {
		panic("unexpected type, expected *state.ACLGroup")
	}

	return aclGroup
}

// Create creates a Route in Kong.
// The arg should be of type crud.Event, containing the aclGroup to be created,
// else the function will panic.
// It returns a the created *state.Route.
func (s *aclGroupCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	aclGroup := aclGroupFromStruct(event)
	cid := ""
	if !utils.Empty(aclGroup.Consumer.Username) {
		cid = *aclGroup.Consumer.Username
	}
	if !utils.Empty(aclGroup.Consumer.ID) {
		cid = *aclGroup.Consumer.ID
	}
	createdACLGroup, err := s.client.ACLs.Create(ctx, &cid,
		&aclGroup.ACLGroup)
	if err != nil {
		return nil, err
	}
	return &state.ACLGroup{ACLGroup: *createdACLGroup}, nil
}

// Delete deletes a Route in Kong.
// The arg should be of type crud.Event, containing the aclGroup to be deleted,
// else the function will panic.
// It returns a the deleted *state.Route.
func (s *aclGroupCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	aclGroup := aclGroupFromStruct(event)
	cid := ""
	if !utils.Empty(aclGroup.Consumer.Username) {
		cid = *aclGroup.Consumer.Username
	}
	if !utils.Empty(aclGroup.Consumer.ID) {
		cid = *aclGroup.Consumer.ID
	}
	err := s.client.ACLs.Delete(ctx, &cid, aclGroup.ID)
	if err != nil {
		return nil, err
	}
	return aclGroup, nil
}

// Update updates a Route in Kong.
// The arg should be of type crud.Event, containing the aclGroup to be updated,
// else the function will panic.
// It returns a the updated *state.Route.
func (s *aclGroupCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	event := crud.EventFromArg(arg[0])
	aclGroup := aclGroupFromStruct(event)

	cid := ""
	if !utils.Empty(aclGroup.Consumer.Username) {
		cid = *aclGroup.Consumer.Username
	}
	if !utils.Empty(aclGroup.Consumer.ID) {
		cid = *aclGroup.Consumer.ID
	}
	updatedACLGroup, err := s.client.ACLs.Create(ctx, &cid, &aclGroup.ACLGroup)
	if err != nil {
		return nil, err
	}
	return &state.ACLGroup{ACLGroup: *updatedACLGroup}, nil
}

type aclGroupDiffer struct {
	kind crud.Kind

	currentState, targetState *state.KongState

	client *kong.Client
}

func (d *aclGroupDiffer) Deletes(handler func(crud.Event) error) error {
	currentACLGroups, err := d.currentState.ACLGroups.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching acls from state: %w", err)
	}

	for _, aclGroup := range currentACLGroups {
		n, err := d.deleteACLGroup(aclGroup)
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

func (d *aclGroupDiffer) deleteACLGroup(aclGroup *state.ACLGroup) (*crud.Event, error) {
	// lookup by consumerID and Group
	_, err := d.targetState.ACLGroups.Get(*aclGroup.Consumer.ID, *aclGroup.ID)
	if errors.Is(err, state.ErrNotFound) {
		return &crud.Event{
			Op:   crud.Delete,
			Kind: d.kind,
			Obj:  aclGroup,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up acl %q: %w", *aclGroup.Group, err)
	}
	return nil, nil
}

func (d *aclGroupDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetACLGroups, err := d.targetState.ACLGroups.GetAll()
	if err != nil {
		return fmt.Errorf("error fetching acls from state: %w", err)
	}

	for _, aclGroup := range targetACLGroups {
		n, err := d.createUpdateACLGroup(aclGroup)
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

func (d *aclGroupDiffer) createUpdateACLGroup(aclGroup *state.ACLGroup) (*crud.Event, error) {
	aclGroup = &state.ACLGroup{ACLGroup: *aclGroup.DeepCopy()}
	currentACLGroup, err := d.currentState.ACLGroups.Get(
		*aclGroup.Consumer.ID, *aclGroup.ID)
	if errors.Is(err, state.ErrNotFound) {
		if aclGroup.ID != nil {
			existingACLGroup, err := d.client.ACLs.GetByID(context.TODO(), aclGroup.ID)
			if err != nil && !kong.IsNotFoundErr(err) {
				return nil, err
			}
			if existingACLGroup != nil {
				return nil, errDuplicateEntity("ACL group", *aclGroup.ID)
			}
		}

		// aclGroup not present, create it
		return &crud.Event{
			Op:   crud.Create,
			Kind: d.kind,
			Obj:  aclGroup,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up acl %q: %w",
			*aclGroup.Group, err)
	}
	// found, check if update needed

	if !currentACLGroup.EqualWithOpts(aclGroup, false, true, false) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   d.kind,
			Obj:    aclGroup,
			OldObj: currentACLGroup,
		}, nil
	}
	return nil, nil
}
