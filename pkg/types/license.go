package types

import (
	"context"
	"errors"

	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-kong/kong"
)

type licenseCRUD struct {
	client *kong.Client
	// isKonnect indicates whether it is syncing with Konnect and licenses should not get changed.
	isKonnect bool
}

var _ crud.Actions = &licenseCRUD{}

func licenseFromEventStruct(arg crud.Event) *state.License {
	license, ok := arg.Obj.(*state.License)
	if !ok {
		panic("unexpected type, expected *state.License")
	}
	return license
}

// Create creates a License in Kong.
// The arg should be of type crud.Event, containing the license to be created,
// else the function will panic.
// It returns a the created *state.Licen
func (s *licenseCRUD) Create(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	if s.isKonnect {
		return nil, nil
	}

	if len(arg) == 0 {
		return nil, ErrEmptyCRUDArgs
	}
	event := crud.EventFromArg(arg[0])
	license := licenseFromEventStruct(event)
	createdLicense, err := s.client.Licenses.Create(ctx, &license.License)
	if err != nil {
		return nil, err
	}
	return &state.License{License: *createdLicense}, nil
}

// Delete deletes a License in Kong.
// The arg should be of type crud.Event, containing the license to be deleted,
// else the function will panic.
// It returns a the deleted *state.License.
func (s *licenseCRUD) Delete(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	if s.isKonnect {
		return nil, nil
	}

	if len(arg) == 0 {
		return nil, ErrEmptyCRUDArgs
	}
	event := crud.EventFromArg(arg[0])
	license := licenseFromEventStruct(event)
	err := s.client.Licenses.Delete(ctx, license.ID)
	if err != nil {
		return nil, err
	}
	return license, nil
}

// Update updates a License in Kong.
// The arg should be of type crud.Event, containing the license to be updated,
// else the function will panic.
// It returns a the updated *state.License.
func (s *licenseCRUD) Update(ctx context.Context, arg ...crud.Arg) (crud.Arg, error) {
	if s.isKonnect {
		return nil, nil
	}

	if len(arg) == 0 {
		return nil, ErrEmptyCRUDArgs
	}
	event := crud.EventFromArg(arg[0])
	license := licenseFromEventStruct(event)

	updatedLicense, err := s.client.Licenses.Create(ctx, &license.License)
	if err != nil {
		return nil, err
	}
	return &state.License{License: *updatedLicense}, nil
}

type licenseDiffer struct {
	kind crud.Kind

	currentState, targetState *state.KongState
}

var _ Differ = &licenseDiffer{}

func (d *licenseDiffer) maybeCreateOrUpdateLicense(targetLicense *state.License) (*crud.Event, error) {
	licenseCopy := &state.License{License: *targetLicense.License.DeepCopy()}
	currentLicense, err := d.currentState.Licenses.Get(*targetLicense.ID)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return &crud.Event{
				Op:   crud.Create,
				Kind: "license",
				Obj:  licenseCopy,
			}, nil
		}
		return nil, err
	}

	if !currentLicense.EqualWithOpts(licenseCopy, false, true) {
		return &crud.Event{
			Op:     crud.Update,
			Kind:   "license",
			Obj:    licenseCopy,
			OldObj: currentLicense,
		}, nil
	}

	return nil, nil
}

// CreateAndUpdates generates a memdb CRUD CREATE/UPDATE event for Licenses
// which is then consumed by the differ and used to gate Kong client calls.
func (d *licenseDiffer) CreateAndUpdates(handler func(crud.Event) error) error {
	targetLicenses, err := d.targetState.Licenses.GetAll()
	if err != nil {
		return err
	}

	for _, license := range targetLicenses {
		event, err := d.maybeCreateOrUpdateLicense(license)
		if err != nil {
			return err
		}

		if event != nil {
			err := handler(*event)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *licenseDiffer) maybeDeleteLicense(currentLicense *state.License) (*crud.Event, error) {
	_, err := d.targetState.Licenses.Get(*currentLicense.ID)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return &crud.Event{
				Op:   crud.Delete,
				Kind: "license",
				Obj:  currentLicense,
			}, nil
		}

		return nil, err
	}
	return nil, nil
}

// Deletes generates a memdb CRUD DELETE event for Licenses
// which is then consumed by the differ and used to gate Kong client calls.
func (d *licenseDiffer) Deletes(handler func(crud.Event) error) error {
	currentLicenses, err := d.currentState.Licenses.GetAll()
	if err != nil {
		return err
	}

	for _, license := range currentLicenses {
		event, err := d.maybeDeleteLicense(license)
		if err != nil {
			return err
		}
		if event != nil {
			err := handler(*event)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
