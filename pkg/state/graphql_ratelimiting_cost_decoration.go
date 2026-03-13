package state

import (
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
)

const graphqlRateLimitingCostDecorationEntityType = "graphql_ratelimiting_cost_decorations"

// GraphqlRateLimitingCostDecorationsCollection stores and indexes graphql_ratelimiting_cost_decorations.
type GraphqlRateLimitingCostDecorationsCollection struct {
	customEntitiesCollection
}

func newGraphqlRateLimitingCostDecorationsCollection(common collection) *GraphqlRateLimitingCostDecorationsCollection {
	return &GraphqlRateLimitingCostDecorationsCollection{
		customEntitiesCollection: customEntitiesCollection{
			collection:       common,
			CustomEntityType: graphqlRateLimitingCostDecorationEntityType,
			customIndexes: map[string]*memdb.IndexSchema{
				"typePath": {
					Name:    "typePath",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "TypePath"},
				},
			},
		},
	}
}

func getGraphqlRateLimitingCostDecorationByTypePath(txn *memdb.Txn,
	typePath string,
) (*GraphqlRateLimitingCostDecoration, error) {
	res, err := txn.First(graphqlRateLimitingCostDecorationEntityType, "typePath", typePath)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrNotFound
	}

	g, ok := res.(*GraphqlRateLimitingCostDecoration)
	if !ok {
		panic(unexpectedType)
	}
	return &GraphqlRateLimitingCostDecoration{GraphqlRateLimitingCostDecoration: *g.DeepCopy()}, nil
}

// GetByTypePath gets a graphql ratelimiting cost decoration with
// the same type_path from the collection.
func (k *GraphqlRateLimitingCostDecorationsCollection) GetByTypePath(
	typePath string,
) (*GraphqlRateLimitingCostDecoration, error) {
	if typePath == "" {
		return nil, fmt.Errorf("typePath cannot be empty string")
	}

	txn := k.db.Txn(false)
	defer txn.Abort()

	return getGraphqlRateLimitingCostDecorationByTypePath(txn, typePath)
}

// Add adds a graphql ratelimiting cost decoration to the collection
func (k *GraphqlRateLimitingCostDecorationsCollection) Add(decoration GraphqlRateLimitingCostDecoration) error {
	e := (customEntity)(&decoration)
	return k.customEntitiesCollection.Add(e)
}

// Get gets a graphql ratelimiting cost decoration by ID.
func (k *GraphqlRateLimitingCostDecorationsCollection) Get(id string) (*GraphqlRateLimitingCostDecoration, error) {
	e, err := k.customEntitiesCollection.Get(id)
	if err != nil {
		return nil, err
	}

	decoration, ok := e.(*GraphqlRateLimitingCostDecoration)
	if !ok {
		panic(unexpectedType)
	}
	return &GraphqlRateLimitingCostDecoration{GraphqlRateLimitingCostDecoration: *decoration.DeepCopy()}, nil
}

// Update updates an existing graphql ratelimiting cost decoration
func (k *GraphqlRateLimitingCostDecorationsCollection) Update(decoration GraphqlRateLimitingCostDecoration) error {
	e := (customEntity)(&decoration)
	return k.customEntitiesCollection.Update(e)
}

// Delete deletes a graphql ratelimiting cost decoration by ID.
func (k *GraphqlRateLimitingCostDecorationsCollection) Delete(id string) error {
	return k.customEntitiesCollection.Delete(id)
}

// GetAll gets all graphql ratelimiting cost decorations
func (k *GraphqlRateLimitingCostDecorationsCollection) GetAll() ([]*GraphqlRateLimitingCostDecoration, error) {
	customEntities, err := k.customEntitiesCollection.GetAll()
	if err != nil {
		return nil, err
	}

	var res []*GraphqlRateLimitingCostDecoration
	for _, e := range customEntities {
		r, ok := e.(*GraphqlRateLimitingCostDecoration)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &GraphqlRateLimitingCostDecoration{GraphqlRateLimitingCostDecoration: *r.DeepCopy()})
	}
	return res, nil
}
