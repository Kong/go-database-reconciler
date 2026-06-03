package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	clonedPluginDefinitionTableName = "clonedPluginDefinition"
)

var clonedPluginDefinitionTableSchema = &memdb.TableSchema{
	Name: clonedPluginDefinitionTableName,
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: &memdb.StringFieldIndex{Field: "ID"},
		},
		nameIndex: {
			Name:         nameIndex,
			Unique:       true,
			Indexer:      &memdb.StringFieldIndex{Field: nameFieldIndex},
			AllowMissing: true,
		},
		all: allIndex,
	},
}

// ClonedPluginDefinitionsCollection stores and indexes Kong ClonedPluginDefinitions.
type ClonedPluginDefinitionsCollection collection

// Add adds a cloned plugin definition to the collection.
// cpd.ID should not be nil else an error is thrown.
func (k *ClonedPluginDefinitionsCollection) Add(cpd ClonedPluginDefinition) error {
	if utils.Empty(cpd.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	var searchBy []string
	searchBy = append(searchBy, *cpd.ID)
	if !utils.Empty(cpd.Name) {
		searchBy = append(searchBy, *cpd.Name)
	}
	_, err := getClonedPluginDefinition(txn, searchBy...)
	if err == nil {
		return fmt.Errorf("inserting cloned plugin definition %v: %w", cpd.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	err = txn.Insert(clonedPluginDefinitionTableName, &cpd)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func getClonedPluginDefinition(txn *memdb.Txn, IDs ...string) (*ClonedPluginDefinition, error) {
	for _, id := range IDs {
		res, err := multiIndexLookupUsingTxn(txn, clonedPluginDefinitionTableName,
			[]string{nameIndex, "id"}, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		cpd, ok := res.(*ClonedPluginDefinition)
		if !ok {
			panic(unexpectedType)
		}
		return &ClonedPluginDefinition{ClonedPluginDefinition: *cpd.DeepCopy()}, nil
	}
	return nil, ErrNotFound
}

// Get gets a cloned plugin definition by name or ID.
func (k *ClonedPluginDefinitionsCollection) Get(nameOrID string) (*ClonedPluginDefinition, error) {
	if nameOrID == "" {
		return nil, errIDRequired
	}
	txn := k.db.Txn(false)
	defer txn.Abort()
	return getClonedPluginDefinition(txn, nameOrID)
}

// Update updates an existing cloned plugin definition.
func (k *ClonedPluginDefinitionsCollection) Update(cpd ClonedPluginDefinition) error {
	if utils.Empty(cpd.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	err := deleteClonedPluginDefinition(txn, *cpd.ID)
	if err != nil {
		return err
	}
	err = txn.Insert(clonedPluginDefinitionTableName, &cpd)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func deleteClonedPluginDefinition(txn *memdb.Txn, nameOrID string) error {
	cpd, err := getClonedPluginDefinition(txn, nameOrID)
	if err != nil {
		return err
	}
	err = txn.Delete(clonedPluginDefinitionTableName, cpd)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes a cloned plugin definition by its name or ID.
func (k *ClonedPluginDefinitionsCollection) Delete(nameOrID string) error {
	if nameOrID == "" {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	err := deleteClonedPluginDefinition(txn, nameOrID)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// GetAll gets all cloned plugin definitions in the state.
func (k *ClonedPluginDefinitionsCollection) GetAll() ([]*ClonedPluginDefinition, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()
	iter, err := txn.Get(clonedPluginDefinitionTableName, all, true)
	if err != nil {
		return nil, err
	}
	var res []*ClonedPluginDefinition
	for el := iter.Next(); el != nil; el = iter.Next() {
		cpd, ok := el.(*ClonedPluginDefinition)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &ClonedPluginDefinition{ClonedPluginDefinition: *cpd.DeepCopy()})
	}
	txn.Commit()
	return res, nil
}
