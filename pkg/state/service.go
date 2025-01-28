package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	serviceTableName = "service"
)

var serviceTableSchema = &memdb.TableSchema{
	Name: serviceTableName,
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: &memdb.StringFieldIndex{Field: "ID"},
		},
		"name": {
			Name:         "name",
			Unique:       true,
			Indexer:      &memdb.StringFieldIndex{Field: "Name"},
			AllowMissing: true,
		},
		all: allIndex,
	},
}

// ServicesCollection stores and indexes Kong Services.
type ServicesCollection collection

// AddIgnoringDuplicates adds a service to the ServicesCollection, ignoring duplicates.
// It first checks for duplicates by service ID and then by service Name.
// If a duplicate is found, it returns nil without adding the service.
// If an error occurs during the duplicate check, it returns the error.
// If no duplicates are found, it adds the service to the collection.
func (k *ServicesCollection) AddIgnoringDuplicates(service Service) error {
	// Detect duplicates
	if !utils.Empty(service.ID) {
		s, err := k.Get(*service.ID)
		if s != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	if !utils.Empty(service.Name) {
		s, err := k.Get(*service.Name)
		if s != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return k.Add(service)
}

// Add adds a service to the collection.
// service.ID should not be nil else an error is thrown.
func (k *ServicesCollection) Add(service Service) error {
	// TODO abstract this check in the go-memdb library itself
	if utils.Empty(service.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()

	var searchBy []string
	searchBy = append(searchBy, *service.ID)
	if !utils.Empty(service.Name) {
		searchBy = append(searchBy, *service.Name)
	}
	_, err := getService(txn, searchBy...)
	if err == nil {
		return fmt.Errorf("inserting service %v: %w", service.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	err = txn.Insert(serviceTableName, &service)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func getService(txn *memdb.Txn, IDs ...string) (*Service, error) {
	for _, id := range IDs {
		res, err := multiIndexLookupUsingTxn(txn, serviceTableName,
			[]string{"name", "id"}, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		service, ok := res.(*Service)
		if !ok {
			panic(unexpectedType)
		}
		return &Service{Service: *service.DeepCopy()}, nil
	}
	return nil, ErrNotFound
}

// Get gets a service by name or ID.
func (k *ServicesCollection) Get(nameOrID string) (*Service, error) {
	if nameOrID == "" {
		return nil, errIDRequired
	}

	txn := k.db.Txn(false)
	defer txn.Abort()
	return getService(txn, nameOrID)
}

// Update udpates an existing service.
// It returns an error if the service is not already present.
func (k *ServicesCollection) Update(service Service) error {
	// TODO abstract this check in the go-memdb library itself
	if utils.Empty(service.ID) {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deleteService(txn, *service.ID)
	if err != nil {
		return err
	}

	err = txn.Insert(serviceTableName, &service)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func deleteService(txn *memdb.Txn, nameOrID string) error {
	service, err := getService(txn, nameOrID)
	if err != nil {
		return err
	}

	err = txn.Delete(serviceTableName, service)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes a service by name or ID.
func (k *ServicesCollection) Delete(nameOrID string) error {
	if nameOrID == "" {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deleteService(txn, nameOrID)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

// GetAll returns all the services.
func (k *ServicesCollection) GetAll() ([]*Service, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(serviceTableName, all, true)
	if err != nil {
		return nil, err
	}

	var res []*Service
	for el := iter.Next(); el != nil; el = iter.Next() {
		s, ok := el.(*Service)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &Service{Service: *s.DeepCopy()})
	}
	txn.Commit()
	return res, nil
}
