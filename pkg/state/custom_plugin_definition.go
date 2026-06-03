package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	customPluginDefinitionTableName = "customPluginDefinition"
)

var customPluginDefinitionTableSchema = &memdb.TableSchema{
	Name: customPluginDefinitionTableName,
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

// CustomPluginDefinitionsCollection stores and indexes Kong CustomPluginDefinitions.
type CustomPluginDefinitionsCollection collection

// Add adds a custom plugin definition to the collection.
// cpd.ID should not be nil else an error is thrown.
func (k *CustomPluginDefinitionsCollection) Add(cpd CustomPluginDefinition) error {
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
	_, err := getCustomPluginDefinition(txn, searchBy...)
	if err == nil {
		return fmt.Errorf("inserting custom plugin definition %v: %w", cpd.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	err = txn.Insert(customPluginDefinitionTableName, &cpd)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func getCustomPluginDefinition(txn *memdb.Txn, IDs ...string) (*CustomPluginDefinition, error) {
	for _, id := range IDs {
		res, err := multiIndexLookupUsingTxn(txn, customPluginDefinitionTableName,
			[]string{nameIndex, "id"}, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		cpd, ok := res.(*CustomPluginDefinition)
		if !ok {
			panic(unexpectedType)
		}
		return &CustomPluginDefinition{CustomPluginDefinition: *cpd.DeepCopy()}, nil
	}
	return nil, ErrNotFound
}

// Get gets a custom plugin definition by name or ID.
func (k *CustomPluginDefinitionsCollection) Get(nameOrID string) (*CustomPluginDefinition, error) {
	if nameOrID == "" {
		return nil, errIDRequired
	}
	txn := k.db.Txn(false)
	defer txn.Abort()
	return getCustomPluginDefinition(txn, nameOrID)
}

// Update updates an existing custom plugin definition.
func (k *CustomPluginDefinitionsCollection) Update(cpd CustomPluginDefinition) error {
	if utils.Empty(cpd.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	err := deleteCustomPluginDefinition(txn, *cpd.ID)
	if err != nil {
		return err
	}
	err = txn.Insert(customPluginDefinitionTableName, &cpd)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func deleteCustomPluginDefinition(txn *memdb.Txn, nameOrID string) error {
	cpd, err := getCustomPluginDefinition(txn, nameOrID)
	if err != nil {
		return err
	}
	err = txn.Delete(customPluginDefinitionTableName, cpd)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes a custom plugin definition by its name or ID.
func (k *CustomPluginDefinitionsCollection) Delete(nameOrID string) error {
	if nameOrID == "" {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	err := deleteCustomPluginDefinition(txn, nameOrID)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// GetAll gets all custom plugin definitions in the state.
func (k *CustomPluginDefinitionsCollection) GetAll() ([]*CustomPluginDefinition, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()
	iter, err := txn.Get(customPluginDefinitionTableName, all, true)
	if err != nil {
		return nil, err
	}
	var res []*CustomPluginDefinition
	for el := iter.Next(); el != nil; el = iter.Next() {
		cpd, ok := el.(*CustomPluginDefinition)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &CustomPluginDefinition{CustomPluginDefinition: *cpd.DeepCopy()})
	}
	txn.Commit()
	return res, nil
}
