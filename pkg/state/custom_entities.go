package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
)

// customEntitiesCollection stores and indexes custom entities.
type customEntitiesCollection struct {
	collection
	CustomEntityType string
	customIndexes    map[string]*memdb.IndexSchema
}

func (k *customEntitiesCollection) TableName() string {
	return k.CustomEntityType
}

func (k *customEntitiesCollection) Schema() *memdb.TableSchema {
	completeIndex := map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: &memdb.StringFieldIndex{Field: "ID"},
		},
		all: allIndex,
	}

	if k.customIndexes != nil {
		for key, index := range k.customIndexes {
			completeIndex[key] = index
		}
	}

	return &memdb.TableSchema{
		Name:    k.CustomEntityType,
		Indexes: completeIndex,
	}
}

func (k *customEntitiesCollection) getByCustomEntityID(txn *memdb.Txn, id string) (customEntity, error) {
	if id == "" {
		return nil, errIDRequired
	}

	res, err := txn.First(k.CustomEntityType, "id", id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrNotFound
	}

	customEntity, ok := res.(customEntity)
	if !ok {
		panic(unexpectedType)
	}
	return customEntity, nil
}

// Add adds a customEntity to customEntitiesCollection.
func (k *customEntitiesCollection) Add(e customEntity) error {
	if e.GetCustomEntityID() == "" {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()

	_, err := k.getByCustomEntityID(txn, e.GetCustomEntityID())
	if err == nil {
		return fmt.Errorf("inserting plugin-entity %v - %v : %w", k.CustomEntityType, e.GetCustomEntityID(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	err = txn.Insert(k.CustomEntityType, e)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// Get gets a customEntity by ID
func (k *customEntitiesCollection) Get(id string) (customEntity, error) {
	if id == "" {
		return nil, errIDRequired
	}

	txn := k.db.Txn(false)
	defer txn.Abort()
	return k.getByCustomEntityID(txn, id)
}

// Update updates an existing customEntity
func (k *customEntitiesCollection) Update(e customEntity) error {
	if e.GetCustomEntityID() == "" {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := k.deleteEntity(txn, e.GetCustomEntityID())
	if err != nil {
		return err
	}
	err = txn.Insert(k.CustomEntityType, e)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (k *customEntitiesCollection) deleteEntity(txn *memdb.Txn, nameOrID string) error {
	e, err := k.getByCustomEntityID(txn, nameOrID)
	if err != nil {
		return err
	}

	err = txn.Delete(k.CustomEntityType, e)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes customEntity by ID
func (k *customEntitiesCollection) Delete(id string) error {
	if id == "" {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := k.deleteEntity(txn, id)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

// GetAll gets all customEntities
func (k *customEntitiesCollection) GetAll() ([]customEntity, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(k.CustomEntityType, all, true)
	if err != nil {
		return nil, err
	}

	var res []customEntity
	for el := iter.Next(); el != nil; el = iter.Next() {
		r, ok := el.(customEntity)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, r)
	}
	return res, nil
}
