package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	keySetTableName = "key_set"
)

var keySetTableSchema = &memdb.TableSchema{
	Name: keySetTableName,
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

// KeySetsCollection stores and indexes Kong KeySets.
type KeySetsCollection collection

// AddIgnoringDuplicates adds a keySet to the KeySetsCollection, ignoring duplicates.
// It first checks for duplicates by keySet ID and then by keySet Name.
// If a duplicate is found, it returns nil without adding the keySet.
// If an error occurs during the duplicate check, it returns the error.
// If no duplicates are found, it adds the keySet to the collection.
func (k *KeySetsCollection) AddIgnoringDuplicates(keySet KeySet) error {
	// Detect duplicates
	if !utils.Empty(keySet.ID) {
		k, err := k.Get(*keySet.ID)
		if k != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	if !utils.Empty(keySet.Name) {
		k, err := k.Get(*keySet.Name)
		if k != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return k.Add(keySet)
}

// Add adds a key-set to the collection.
// key-set.ID should not be nil else an error is thrown.
func (k *KeySetsCollection) Add(set KeySet) error {
	if utils.Empty(set.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	var searchBy []string
	searchBy = append(searchBy, *set.ID)
	if !utils.Empty(set.Name) {
		searchBy = append(searchBy, *set.Name)
	}
	_, err := getSet(txn, searchBy...)
	if err == nil {
		return fmt.Errorf("inserting key-set %v: %w", set.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	err = txn.Insert(keySetTableName, &set)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func getSet(txn *memdb.Txn, IDs ...string) (*KeySet, error) {
	for _, id := range IDs {
		res, err := multiIndexLookupUsingTxn(txn, keySetTableName,
			[]string{"name", "id"}, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		set, ok := res.(*KeySet)
		if !ok {
			panic(unexpectedType)
		}
		return &KeySet{KeySet: *set.DeepCopy()}, nil
	}
	return nil, ErrNotFound
}

// Get gets a key-set by name or ID.
func (k *KeySetsCollection) Get(nameOrID string) (*KeySet, error) {
	if nameOrID == "" {
		return nil, errIDRequired
	}
	txn := k.db.Txn(false)
	defer txn.Abort()
	return getSet(txn, nameOrID)
}

// Update updates an existing key-set.
func (k *KeySetsCollection) Update(set KeySet) error {
	if utils.Empty(set.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	err := deleteSet(txn, *set.ID)
	if err != nil {
		return err
	}
	err = txn.Insert(keySetTableName, &set)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func deleteSet(txn *memdb.Txn, nameOrID string) error {
	set, err := getSet(txn, nameOrID)
	if err != nil {
		return err
	}
	err = txn.Delete(keySetTableName, set)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes a key-set by its name or ID.
func (k *KeySetsCollection) Delete(nameOrID string) error {
	if nameOrID == "" {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	err := deleteSet(txn, nameOrID)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// GetAll gets all key-set in the state.
func (k *KeySetsCollection) GetAll() ([]*KeySet, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()
	iter, err := txn.Get(keySetTableName, all, true)
	if err != nil {
		return nil, err
	}
	var res []*KeySet
	for el := iter.Next(); el != nil; el = iter.Next() {
		s, ok := el.(*KeySet)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &KeySet{KeySet: *s.DeepCopy()})
	}
	txn.Commit()
	return res, nil
}
