package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	licenseTableName = "license"
)

var licenseTableSchema = &memdb.TableSchema{
	Name: licenseTableName,
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: &memdb.StringFieldIndex{Field: "ID"},
		},
		all: allIndex,
	},
}

type LicensesCollection collection

func getLicense(txn *memdb.Txn, id string) (*License, error) {
	res, err := multiIndexLookupUsingTxn(txn, licenseTableName, []string{"id"}, id)
	if err != nil {
		return nil, err
	}
	l, ok := res.(*License)
	if !ok {
		panic(unexpectedType)
	}
	return &License{License: *l.DeepCopy()}, nil
}

func (k *LicensesCollection) Add(l License) error {
	if utils.Empty(l.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()

	_, err := getLicense(txn, *l.ID)
	if err == nil {
		return fmt.Errorf("inserting license %v: %w", l.Console(), ErrAlreadyExists)
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	err = txn.Insert(licenseTableName, &l)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (k *LicensesCollection) Get(id string) (*License, error) {
	if id == "" {
		return nil, errIDRequired
	}
	txn := k.db.Txn(false)
	defer txn.Abort()

	l, err := getLicense(txn, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	txn.Commit()
	return l, nil
}

func deleteLicense(txn *memdb.Txn, id string) error {
	l, err := getLicense(txn, id)
	if err != nil {
		return err
	}

	return txn.Delete(licenseTableName, l)
}

func (k *LicensesCollection) Update(l License) error {
	if utils.Empty(l.ID) {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deleteLicense(txn, *l.ID)
	if err != nil {
		return err
	}

	err = txn.Insert(licenseTableName, &l)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (k *LicensesCollection) Delete(id string) error {
	if id == "" {
		return errIDRequired
	}
	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deleteLicense(txn, id)
	if err != nil {
		return err
	}

	txn.Commit()
	return err
}

func (k *LicensesCollection) GetAll() ([]*License, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()
	iter, err := txn.Get(licenseTableName, all, true)
	if err != nil {
		return nil, err
	}

	var res []*License
	for el := iter.Next(); el != nil; el = iter.Next() {
		l, ok := el.(*License)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &License{License: *l.DeepCopy()})
	}
	txn.Commit()
	return res, nil
}
