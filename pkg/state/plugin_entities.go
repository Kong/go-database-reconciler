package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
)

// pluginEntitiesCollection stores and indexes plugin entities.
type pluginEntitiesCollection struct {
	collection
	PluginEntityType string
	customIndexes    map[string]*memdb.IndexSchema
}

func (k *pluginEntitiesCollection) TableName() string {
	return k.PluginEntityType
}

func (k *pluginEntitiesCollection) Schema() *memdb.TableSchema {
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
		Name:    k.PluginEntityType,
		Indexes: completeIndex,
	}
}

func (k *pluginEntitiesCollection) getByPluginEntityID(txn *memdb.Txn, id string) (pluginEntity, error) {
	if id == "" {
		return nil, errIDRequired
	}

	res, err := txn.First(k.PluginEntityType, "id", id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrNotFound
	}

	pluginEntity, ok := res.(pluginEntity)
	if !ok {
		panic(unexpectedType)
	}
	return pluginEntity, nil
}

// Add adds a pluginEntity to pluginEntitiesCollection.
func (k *pluginEntitiesCollection) Add(e pluginEntity) error {
	if e.GetPluginEntityID() == "" {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()

	_, err := k.getByPluginEntityID(txn, e.GetPluginEntityID())
	if err == nil {
		return fmt.Errorf("inserting plugin-entity %v - %v : %w", k.PluginEntityType, e.GetPluginEntityID(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	err = txn.Insert(k.PluginEntityType, e)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// Get gets a pluginEntity by ID
func (k *pluginEntitiesCollection) Get(id string) (pluginEntity, error) {
	if id == "" {
		return nil, errIDRequired
	}

	txn := k.db.Txn(false)
	defer txn.Abort()
	return k.getByPluginEntityID(txn, id)
}

// Update updates an existing pluginEntity
func (k *pluginEntitiesCollection) Update(e pluginEntity) error {
	if e.GetPluginEntityID() == "" {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := k.deleteEntity(txn, e.GetPluginEntityID())
	if err != nil {
		return err
	}
	err = txn.Insert(k.PluginEntityType, e)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (k *pluginEntitiesCollection) deleteEntity(txn *memdb.Txn, nameOrID string) error {
	e, err := k.getByPluginEntityID(txn, nameOrID)
	if err != nil {
		return err
	}

	err = txn.Delete(k.PluginEntityType, e)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes pluginEntity by ID
func (k *pluginEntitiesCollection) Delete(id string) error {
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

// GetAll gets all pluginEntities
func (k *pluginEntitiesCollection) GetAll() ([]pluginEntity, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(k.PluginEntityType, all, true)
	if err != nil {
		return nil, err
	}

	var res []pluginEntity
	for el := iter.Next(); el != nil; el = iter.Next() {
		r, ok := el.(pluginEntity)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, r)
	}
	return res, nil
}
