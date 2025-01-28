package state

import (
	"errors"
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
	"github.com/kong/go-database-reconciler/pkg/state/indexers"
	"github.com/kong/go-database-reconciler/pkg/utils"
)

const (
	consumerGroupConsumerTableName = "consumerGroupConsumer"
	consumerByGroupID              = "consumerByGroupID"
	consumerByConsumerID           = "consumerByConsumerID"
	consumerByUsername             = "consumerByUsername"
	consumerByCustomID             = "consumerByCustomID"
)

var errInvalidConsumerGroup = fmt.Errorf("consumer_group.ID is required in consumer group consumers")

var consumerGroupConsumerTableSchema = &memdb.TableSchema{
	Name: consumerGroupConsumerTableName,
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:   "id",
			Unique: true,
			Indexer: &indexers.SubFieldIndexer{
				Fields: []indexers.Field{
					{
						Struct: "Consumer",
						Sub:    "ID",
					},
					{
						Struct: "ConsumerGroup",
						Sub:    "ID",
					},
				},
			},
		},
		"username": {
			Name:   "username",
			Unique: true,
			Indexer: &indexers.SubFieldIndexer{
				Fields: []indexers.Field{
					{
						Struct: "Consumer",
						Sub:    "Username",
					},
				},
			},
		},
		all: allIndex,
		// foreign
		consumerByGroupID: {
			Name: consumerByGroupID,
			Indexer: &indexers.SubFieldIndexer{
				Fields: []indexers.Field{
					{
						Struct: "ConsumerGroup",
						Sub:    "ID",
					},
				},
			},
		},
		consumerByConsumerID: {
			Name: consumerByConsumerID,
			Indexer: &indexers.SubFieldIndexer{
				Fields: []indexers.Field{
					{
						Struct: "Consumer",
						Sub:    "ID",
					},
				},
			},
		},
		consumerByUsername: {
			Name: consumerByUsername,
			Indexer: &indexers.SubFieldIndexer{
				Fields: []indexers.Field{
					{
						Struct: "Consumer",
						Sub:    "Username",
					},
				},
			},
		},
		consumerByCustomID: {
			Name: consumerByCustomID,
			Indexer: &indexers.SubFieldIndexer{
				Fields: []indexers.Field{
					{
						Struct: "Consumer",
						Sub:    "CustomID",
					},
				},
			},
		},
	},
}

func validateConsumerGroup(consumer *ConsumerGroupConsumer) error {
	if consumer.ConsumerGroup == nil ||
		utils.Empty(consumer.ConsumerGroup.ID) {
		return errInvalidConsumerGroup
	}
	return nil
}

// ConsumerGroupConsumersCollection stores and indexes Kong consumerGroupConsumers.
type ConsumerGroupConsumersCollection collection

// AddIgnoringDuplicates adds a ConsumerGroupConsumer to the collection, ignoring duplicates.
// It first checks for duplicates by Consumer ID and then by Consumer Username.
// If a duplicate is found, it returns nil without adding the consumer.
// If an error occurs during the duplicate check, it returns the error unless the error is ErrNotFound
// as this is expected when the consumer does not exist.
// If no duplicates are found, it adds the consumer to the collection.
func (k *ConsumerGroupConsumersCollection) AddIgnoringDuplicates(consumer ConsumerGroupConsumer) error {
	// Detect duplicates
	if !utils.Empty(consumer.Consumer.ID) {
		cgc, err := k.Get(*consumer.Consumer.ID, *consumer.ConsumerGroup.ID)
		if cgc != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	if !utils.Empty(consumer.Consumer.Username) {
		cgc, err := k.Get(*consumer.Consumer.Username, *consumer.ConsumerGroup.ID)
		if cgc != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}

	return k.Add(consumer)
}

// Add adds a consumerGroupConsumer to the collection.
func (k *ConsumerGroupConsumersCollection) Add(consumer ConsumerGroupConsumer) error {
	if utils.Empty(consumer.Consumer.ID) {
		return errIDRequired
	}

	if err := validateConsumerGroup(&consumer); err != nil {
		return err
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	var searchBy []string
	searchBy = append(searchBy, *consumer.Consumer.ID, *consumer.ConsumerGroup.ID)
	if !utils.Empty(consumer.Consumer.Username) {
		searchBy = append(searchBy, *consumer.Consumer.Username)
	}
	if !utils.Empty(consumer.Consumer.CustomID) {
		searchBy = append(searchBy, *consumer.Consumer.CustomID)
	}
	_, err := getConsumerGroupConsumer(txn, *consumer.ConsumerGroup.ID, searchBy...)
	if err == nil {
		return fmt.Errorf("inserting consumerGroupConsumer %v: %w", consumer.Console(), ErrAlreadyExists)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	err = txn.Insert(consumerGroupConsumerTableName, &consumer)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func getConsumerGroupConsumer(txn *memdb.Txn, consumerGroupID string, IDs ...string) (*ConsumerGroupConsumer, error) {
	indexes := []string{consumerByConsumerID, consumerByUsername, consumerByCustomID}

	for _, id := range IDs {
		for _, index := range indexes {
			res, err := txn.Get(consumerGroupConsumerTableName, index, id)
			if err != nil {
				return nil, err
			}

			for {
				resultValue := res.Next()
				if resultValue == nil {
					break
				}
				consumer, ok := resultValue.(*ConsumerGroupConsumer)
				if !ok {
					break
				}

				if *consumer.ConsumerGroup.ID == consumerGroupID {
					return consumer, nil
				}
			}
		}
	}
	return nil, ErrNotFound
}

// Get gets a consumerGroupConsumer.
func (k *ConsumerGroupConsumersCollection) Get(
	nameOrID, consumerGroupID string,
) (*ConsumerGroupConsumer, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()

	return getConsumerGroupConsumer(txn, consumerGroupID, nameOrID)
}

// Update udpates an existing consumerGroupConsumer.
func (k *ConsumerGroupConsumersCollection) Update(consumer ConsumerGroupConsumer) error {
	if utils.Empty(consumer.Consumer.ID) {
		return errIDRequired
	}

	if err := validateConsumerGroup(&consumer); err != nil {
		return err
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	res, err := txn.First(consumerGroupConsumerTableName, "id",
		*consumer.Consumer.ID, *consumer.ConsumerGroup.ID)
	if err != nil {
		return err
	}

	t, ok := res.(*ConsumerGroupConsumer)
	if !ok {
		panic(unexpectedType)
	}
	err = txn.Delete(consumerGroupConsumerTableName, *t)
	if err != nil {
		return err
	}

	err = txn.Insert(consumerGroupConsumerTableName, &consumer)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func deleteConsumerGroupConsumer(txn *memdb.Txn, nameOrID, consumerGroupID string) error {
	consumer, err := getConsumerGroupConsumer(txn, consumerGroupID, nameOrID)
	if err != nil {
		return err
	}
	err = txn.Delete(consumerGroupConsumerTableName, consumer)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes a consumerGroupConsumer by its username or ID.
func (k *ConsumerGroupConsumersCollection) Delete(nameOrID, consumerGroupID string) error {
	if nameOrID == "" {
		return errIDRequired
	}

	if consumerGroupID == "" {
		return errInvalidConsumerGroup
	}

	txn := k.db.Txn(true)
	defer txn.Abort()

	err := deleteConsumerGroupConsumer(txn, nameOrID, consumerGroupID)
	if err != nil {
		return err
	}

	txn.Commit()
	return nil
}

// GetAll gets all consumerGroupConsumers in the state.
func (k *ConsumerGroupConsumersCollection) GetAll() ([]*ConsumerGroupConsumer, error) {
	txn := k.db.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(consumerGroupConsumerTableName, all, true)
	if err != nil {
		return nil, err
	}

	var res []*ConsumerGroupConsumer
	for el := iter.Next(); el != nil; el = iter.Next() {
		u, ok := el.(*ConsumerGroupConsumer)
		if !ok {
			panic(unexpectedType)
		}
		res = append(res, &ConsumerGroupConsumer{ConsumerGroupConsumer: *u.DeepCopy()})
	}
	txn.Commit()
	return res, nil
}
