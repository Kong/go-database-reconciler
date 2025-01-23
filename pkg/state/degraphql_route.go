package state

import (
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/state/indexers"
)

// DegraphqlRoutesCollection stores and indexes degraphql_routes.
type DegraphqlRoutesCollection struct {
	customEntitiesCollection
}

const customEntityType = "degraphql_routes"

func newDegraphqlRoutesCollection(common collection) *DegraphqlRoutesCollection {
	return &DegraphqlRoutesCollection{
		customEntitiesCollection: customEntitiesCollection{
			collection:       common,
			CustomEntityType: customEntityType,
			customIndexes: map[string]*memdb.IndexSchema{
				"uriQuery": {
					Name:   "uriQuery",
					Unique: true,
					Indexer: &indexers.MD5FieldsIndexer{
						Fields: []string{"URI", "Query"},
					},
				},
			},
		},
	}
}

func getDegraphqlRouteByURIQuery(txn *memdb.Txn, uri, query string) (*DegraphqlRoute, error) {
	res, err := txn.First(customEntityType, "uriQuery", uri, query)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrNotFound
	}

	d, ok := res.(*DegraphqlRoute)
	if !ok {
		panic(unexpectedType)
	}
	return &DegraphqlRoute{DegraphqlRoute: *d.DeepCopy()}, nil
}

// GetByURIQuery gets a degraphql route with
// the same uri and query from the collection.
func (k *DegraphqlRoutesCollection) GetByURIQuery(uri,
	query string,
) (*DegraphqlRoute, error) {
	if uri == "" || query == "" {
		return nil, fmt.Errorf("uri/query cannot be empty string")
	}

	txn := k.db.Txn(false)
	defer txn.Abort()

	return getDegraphqlRouteByURIQuery(txn, uri, query)
}

// Add adds a degraphql route to DegraphqlRoutesCollection
func (k *DegraphqlRoutesCollection) Add(degraphqlRoute DegraphqlRoute) error {
	e := (customEntity)(&degraphqlRoute)
	return k.customEntitiesCollection.Add(e)
}

// Get gets a degraphql route  ID.
func (k *DegraphqlRoutesCollection) Get(id string) (*DegraphqlRoute, error) {
	e, err := k.customEntitiesCollection.Get(id)
	if err != nil {
		return nil, err
	}

	degraphqlRoute, ok := e.(*DegraphqlRoute)
	if !ok {
		panic(unexpectedType)
	}
	return &DegraphqlRoute{DegraphqlRoute: *degraphqlRoute.DeepCopy()}, nil
}

// Update updates an existing degraphql route
func (k *DegraphqlRoutesCollection) Update(degraphqlRoute DegraphqlRoute) error {
	e := (customEntity)(&degraphqlRoute)
	return k.customEntitiesCollection.Update(e)
}

// Delete deletes a degraphql route by ID.
func (k *DegraphqlRoutesCollection) Delete(id string) error {
	return k.customEntitiesCollection.Delete(id)
}

// GetAll gets all degraphql routes
func (k *DegraphqlRoutesCollection) GetAll() ([]*DegraphqlRoute, error) {
	customEntities, err := k.customEntitiesCollection.GetAll()
	if err != nil {
		return nil, err
	}

	var res []*DegraphqlRoute
	for _, e := range customEntities {
		r, ok := e.(*DegraphqlRoute)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &DegraphqlRoute{DegraphqlRoute: *r.DeepCopy()})
	}
	return res, nil
}
