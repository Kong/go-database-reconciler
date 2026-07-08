package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	aiModelDefinitionTableName = "aiModelDefinition"
)

var aiModelDefinitionTableSchema = &memdb.TableSchema{
	Name: aiModelDefinitionTableName,
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

// AIModelsCollection stores and indexes Kong AIModels.
type AIModelsCollection collection

// Add adds an AI model definition to the collection.
// amd.ID should not be nil else an error is thrown.
func (k *AIModelsCollection) Add(amd AIModel) error {
	if utils.Empty(amd.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	var searchBy []string
	searchBy = append(searchBy, *amd.ID)
	if !utils.Empty(amd.Name) {
		searchBy = append(searchBy, *amd.Name)
	}
	_, err := getAIModel(txn, searchBy...)
	if err == nil {
		return fmt.Errorf("inserting AI model definition %v: %w", amd.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	err = txn.Insert(aiModelDefinitionTableName, &amd)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func getAIModel(txn *memdb.Txn, IDs ...string) (*AIModel, error) {
	for _, id := range IDs {
		res, err := multiIndexLookupUsingTxn(txn, aiModelDefinitionTableName,
			[]string{nameIndex, "id"}, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		amd, ok := res.(*AIModel)
		if !ok {
			panic(unexpectedType)
		}
		return &AIModel{AIModel: *amd.DeepCopy()}, nil
	}
	return nil, ErrNotFound
}

// Get gets an AI model definition by name or ID.
func (k *AIModelsCollection) Get(nameOrID string) (*AIModel, error) {
	if nameOrID == "" {
		return nil, errIDRequired
	}
	txn := k.db.Txn(false)
	defer txn.Abort()
	return getAIModel(txn, nameOrID)
}

// Update updates an existing AI model definition.
func (k *AIModelsCollection) Update(amd AIModel) error {
	if utils.Empty(amd.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	err := deleteAIModel(txn, *amd.ID)
	if err != nil {
		return err
	}
	err = txn.Insert(aiModelDefinitionTableName, &amd)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func deleteAIModel(txn *memdb.Txn, nameOrID string) error {
	amd, err := getAIModel(txn, nameOrID)
	if err != nil {
		return err
	}
	err = txn.Delete(aiModelDefinitionTableName, amd)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes an AI model definition by its name or ID.
func (k *AIModelsCollection) Delete(nameOrID string) error {
	if nameOrID == "" {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()
	err := deleteAIModel(txn, nameOrID)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// GetAll gets all AI model definitions in the state.
func (k *AIModelsCollection) GetAll() ([]*AIModel, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()
	iter, err := txn.Get(aiModelDefinitionTableName, all, true)
	if err != nil {
		return nil, err
	}
	var res []*AIModel
	for el := iter.Next(); el != nil; el = iter.Next() {
		amd, ok := el.(*AIModel)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &AIModel{AIModel: *amd.DeepCopy()})
	}
	txn.Commit()
	return res, nil
}
