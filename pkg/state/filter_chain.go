package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/state/indexers"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	filterChainTableName  = "filterChain"
	filterChainsByService = "filterChainsByService"
	filterChainsByRoute   = "filterChainsByRoute"
)

var filterChainTableSchema = &memdb.TableSchema{
	Name: filterChainTableName,
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: &memdb.StringFieldIndex{Field: "ID"},
		},
		all: allIndex,
		filterChainsByService: {
			Name:         filterChainsByService,
			Unique:       true,
			AllowMissing: true,
			Indexer: &indexers.SubFieldIndexer{
				Fields: []indexers.Field{
					{
						Struct: "Service",
						Sub:    "ID",
					},
				},
			},
		},
		filterChainsByRoute: {
			Name:         filterChainsByRoute,
			Unique:       true,
			AllowMissing: true,
			Indexer: &indexers.SubFieldIndexer{
				Fields: []indexers.Field{
					{
						Struct: "Route",
						Sub:    "ID",
					},
				},
			},
		},
	},
}

// FilterChainsCollection stores and indexes Kong Services.
type FilterChainsCollection collection

// Add adds a filter chain to FilterChainsCollection
func (k *FilterChainsCollection) Add(filterChain FilterChain) error {
	txn := k.db.Txn(true)
	defer txn.Abort()

	err := insertFilterChain(txn, filterChain)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func insertFilterChain(txn *memdb.Txn, filterChain FilterChain) error {
	if utils.Empty(filterChain.ID) {
		return errIDRequired
	}

	// err out if filter chain with same ID is present
	_, err := getFilterChainByID(txn, *filterChain.ID)
	if err == nil {
		return fmt.Errorf("inserting filter chain %v: %w", filterChain.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	if len(filterChain.Filters) == 0 {
		return fmt.Errorf("inserting filter chain %v: at least one filter is required", filterChain.Console())
	}

	sID, rID := "", ""
	if filterChain.Service != nil && !utils.Empty(filterChain.Service.ID) {
		sID = *filterChain.Service.ID
	}
	if filterChain.Route != nil && !utils.Empty(filterChain.Route.ID) {
		rID = *filterChain.Route.ID
	}

	if sID != "" && rID != "" {
		return fmt.Errorf("inserting filter chain %v: only one of service/route is allowed", filterChain.Console())
	} else if sID == "" && rID == "" {
		return fmt.Errorf("inserting filter chain: %v: at least one of service/route is required", filterChain.Console())
	}

	// err out if another filter chain with exact same combination is present
	if sID != "" {
		_, err = getFilterChainBy(txn, filterChainsByService, sID)
	} else {
		_, err = getFilterChainBy(txn, filterChainsByRoute, rID)
	}

	if err == nil {
		return fmt.Errorf("inserting filter chain %v: %w", filterChain.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	err = txn.Insert(filterChainTableName, &filterChain)
	if err != nil {
		return err
	}
	return nil
}

func getFilterChainByID(txn *memdb.Txn, id string) (*FilterChain, error) {
	return getFilterChainBy(txn, "id", id)
}

// Get gets a filter chain by id.
func (k *FilterChainsCollection) Get(id string) (*FilterChain, error) {
	if id == "" {
		return nil, errIDRequired
	}

	txn := k.db.Txn(false)
	defer txn.Abort()

	filterChain, err := getFilterChainByID(txn, id)
	if err != nil {
		return nil, err
	}
	return filterChain, nil
}

func getFilterChainBy(txn *memdb.Txn, index string, id string) (
	*FilterChain, error,
) {
	if id == "" {
		return nil, errIDRequired
	}

	res, err := txn.First(filterChainTableName, index, id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrNotFound
	}
	f, ok := res.(*FilterChain)
	if !ok {
		panic(unexpectedType)
	}
	return &FilterChain{FilterChain: *f.DeepCopy()}, nil
}

// GetByProp returns a filter chain which matches the given serviceID or
// routeID.
func (k *FilterChainsCollection) GetByProp(serviceID, routeID string) (*FilterChain, error) {
	var index string
	var id string

	if serviceID != "" {
		index = filterChainsByService
		id = serviceID
	} else if routeID != "" {
		index = filterChainsByRoute
		id = routeID
	} else {
		return nil, errIDRequired
	}

	txn := k.db.Txn(false)
	defer txn.Abort()

	return getFilterChainBy(txn, index, id)
}

func (k *FilterChainsCollection) getAllFilterChainsBy(index string, identifier ...string) (
	[]*FilterChain, error,
) {
	haveID := false
	args := make([]interface{}, len(identifier))
	for i, v := range identifier {
		haveID = haveID || v != ""
		args[i] = v
	}

	if !haveID {
		return nil, errIDRequired
	}

	txn := k.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(filterChainTableName, index, args...)
	if err != nil {
		return nil, err
	}
	var res []*FilterChain
	for el := iter.Next(); el != nil; el = iter.Next() {
		f, ok := el.(*FilterChain)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &FilterChain{FilterChain: *f.DeepCopy()})
	}
	return res, nil
}

// GetAllByServiceID returns all filter chains referencing a service
// by its id.
func (k *FilterChainsCollection) GetAllByServiceID(id string) ([]*FilterChain,
	error,
) {
	return k.getAllFilterChainsBy(filterChainsByService, id)
}

// GetAllByRouteID returns all filter chains referencing a route
// by its id.
func (k *FilterChainsCollection) GetAllByRouteID(id string) ([]*FilterChain,
	error,
) {
	return k.getAllFilterChainsBy(filterChainsByRoute, id)
}

// Update updates a filter chain
func (k *FilterChainsCollection) Update(filterChain FilterChain) error {
	if utils.Empty(filterChain.ID) {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deleteFilterChain(txn, *filterChain.ID)
	if err != nil {
		return err
	}

	err = insertFilterChain(txn, filterChain)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func deleteFilterChain(txn *memdb.Txn, id string) error {
	filterChain, err := getFilterChainByID(txn, id)
	if err != nil {
		return err
	}
	return txn.Delete(filterChainTableName, filterChain)
}

// Delete deletes a filter chain by ID.
func (k *FilterChainsCollection) Delete(id string) error {
	if id == "" {
		return errIDRequired
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deleteFilterChain(txn, id)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

// GetAll gets a filter chain by name or ID.
func (k *FilterChainsCollection) GetAll() ([]*FilterChain, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(filterChainTableName, all, true)
	if err != nil {
		return nil, err
	}

	var res []*FilterChain
	for el := iter.Next(); el != nil; el = iter.Next() {
		f, ok := el.(*FilterChain)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &FilterChain{FilterChain: *f.DeepCopy()})
	}
	return res, nil
}
