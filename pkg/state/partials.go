package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	partialTableName = "partial"
)

var partialTableSchema = &memdb.TableSchema{
	Name: partialTableName,
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

// PartialsCollection stores and indexes Kong Partials.
type PartialsCollection collection

func getPartial(txn *memdb.Txn, IDs ...string) (*Partial, error) {
	for _, id := range IDs {
		res, err := multiIndexLookupUsingTxn(txn, partialTableName,
			[]string{"name", "id"}, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		partial, ok := res.(*Partial)
		if !ok {
			return nil, fmt.Errorf("expected *Partial, got %T", res)
		}
		return &Partial{Partial: *partial.DeepCopy()}, nil
	}
	return nil, ErrNotFound
}

// Add adds a partial to the collection.
// partial.ID should not be nil or an error is thrown.
func (k *PartialsCollection) Add(partial Partial) error {
	if utils.Empty(partial.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()

	var searchBy []string
	searchBy = append(searchBy, *partial.ID)
	if !utils.Empty(partial.Name) {
		searchBy = append(searchBy, *partial.Name)
	}
	_, err := getPartial(txn, searchBy...)
	if err == nil {
		return fmt.Errorf("inserting partial %v: %w", partial.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	err = txn.Insert(partialTableName, &partial)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// AddIgnoringDuplicates adds a Partial to the collection, ignoring duplicates.
// If a Partial with the same ID or Name already exists in the collection,
// the method returns nil without adding the new Partial.
// If an error occurs during the duplicate check, it is returned unless the error is ErrNotFound
// as this is expected when the Partial does not exist.
func (k *PartialsCollection) AddIgnoringDuplicates(partial Partial) error {
	// Detect duplicates
	if !utils.Empty(partial.ID) {
		cg, err := k.Get(*partial.ID)
		if cg != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	if !utils.Empty(partial.Name) {
		cg, err := k.Get(*partial.Name)
		if cg != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	return k.Add(partial)
}

// Get gets a partial by name or ID.
func (k *PartialsCollection) Get(nameOrID string) (*Partial, error) {
	if nameOrID == "" {
		return nil, errIDRequired
	}

	txn := k.db.Txn(false)
	defer txn.Abort()
	partial, err := getPartial(txn, nameOrID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return partial, nil
}

// Update udpates an existing partial.
func (k *PartialsCollection) Update(partial Partial) error {
	if utils.Empty(partial.ID) {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deletePartial(txn, *partial.ID)
	if err != nil {
		return err
	}

	err = txn.Insert(partialTableName, &partial)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func deletePartial(txn *memdb.Txn, nameOrID string) error {
	partial, err := getPartial(txn, nameOrID)
	if err != nil {
		return err
	}

	err = txn.Delete(partialTableName, partial)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes a partial by its name or ID.
func (k *PartialsCollection) Delete(nameOrID string) error {
	if nameOrID == "" {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deletePartial(txn, nameOrID)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

// GetAll gets all partials in the state.
func (k *PartialsCollection) GetAll() ([]*Partial, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(partialTableName, all, true)
	if err != nil {
		return nil, err
	}

	var res []*Partial
	for el := iter.Next(); el != nil; el = iter.Next() {
		p, ok := el.(*Partial)
		if !ok {
			return nil, fmt.Errorf("expected *Partial, got %T", el)
		}
		res = append(res, &Partial{Partial: *p.DeepCopy()})
	}
	txn.Commit()
	return res, nil
}
