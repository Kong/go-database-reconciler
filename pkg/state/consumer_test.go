package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func consumersCollection() *ConsumersCollection {
	return state().Consumers
}

func TestConsumerInsert(t *testing.T) {
	collection := consumersCollection()

	var consumer Consumer

	require.Error(t, collection.Add(consumer))

	consumer.ID = new("first")
	require.NoError(t, collection.Add(consumer))

	// re-insert
	consumer.Username = new("my-name")
	require.Error(t, collection.Add(consumer))
}

func TestConsumerInsertIgnoreDuplicateUsername(t *testing.T) {
	collection := consumersCollection()

	var consumer Consumer
	consumer.ID = new("first")
	consumer.Username = new("my-name")
	err := collection.Add(consumer)
	require.NoError(t, err)
	err = collection.AddIgnoringDuplicates(consumer)
	require.NoError(t, err)
}

func TestConsumerInsertIgnoreDuplicateCustomId(t *testing.T) {
	collection := consumersCollection()

	var consumer Consumer
	consumer.ID = new("first")
	consumer.CustomID = new("my-name")
	err := collection.Add(consumer)
	require.NoError(t, err)
	err = collection.AddIgnoringDuplicates(consumer)
	require.NoError(t, err)
}

func TestConsumerGetUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := consumersCollection()

	var consumer Consumer
	consumer.ID = new("first")
	consumer.Username = new("my-name")
	err := collection.Add(consumer)
	require.NoError(t, err)

	c, err := collection.GetByIDOrUsername("")
	require.Error(t, err)
	assert.Nil(c)

	c, err = collection.GetByIDOrUsername("first")
	require.NoError(t, err)
	assert.NotNil(c)

	c.ID = nil
	c.Username = new("my-updated-name")
	err = collection.Update(*c)
	require.Error(t, err)

	c.ID = new("does-not-exist")
	require.Error(t, collection.Update(*c))

	c.ID = new("first")
	require.NoError(t, collection.Update(*c))

	c, err = collection.GetByIDOrUsername("my-name")
	require.Error(t, err)
	assert.Nil(c)

	c, err = collection.GetByIDOrUsername("my-updated-name")
	require.NoError(t, err)
	assert.NotNil(c)
}

// Test to ensure that the memory reference of the pointer returned by Get()
// is different from the one stored in MemDB.
func TestConsumerGetMemoryReference(t *testing.T) {
	assert := assert.New(t)
	collection := consumersCollection()

	var consumer Consumer
	consumer.ID = new("first")
	consumer.Username = new("my-name")
	err := collection.Add(consumer)
	require.NoError(t, err)

	c, err := collection.GetByIDOrUsername("first")
	require.NoError(t, err)
	assert.NotNil(c)
	c.Username = new("update-should-not-reflect")

	c, err = collection.GetByIDOrUsername("first")
	require.NoError(t, err)
	assert.Equal("my-name", *c.Username)
}

func TestConsumersInvalidType(t *testing.T) {
	assert := assert.New(t)
	collection := consumersCollection()

	type c2 Consumer
	var c c2
	c.Username = new("my-name")
	c.ID = new("first")
	txn := collection.db.Txn(true)
	require.NoError(t, txn.Insert(consumerTableName, &c))
	txn.Commit()

	assert.Panics(func() {
		collection.GetByIDOrUsername("my-name")
	})
	assert.Panics(func() {
		collection.GetAll()
	})
}

func TestConsumerDelete(t *testing.T) {
	assert := assert.New(t)
	collection := consumersCollection()

	var consumer Consumer
	consumer.ID = new("first")
	consumer.Username = new("my-consumer")
	err := collection.Add(consumer)
	require.NoError(t, err)

	c, err := collection.GetByIDOrUsername("my-consumer")
	require.NoError(t, err)
	assert.NotNil(c)
	assert.Equal("first", *c.ID)

	err = collection.Delete("first")
	require.NoError(t, err)

	err = collection.Delete("")
	require.Error(t, err)

	err = collection.Delete(*c.ID)
	require.Error(t, err)
}

func TestConsumerGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := consumersCollection()

	consumers := []Consumer{
		{
			Consumer: kong.Consumer{
				ID:       new("first"),
				Username: new("my-consumer1"),
			},
		},
		{
			Consumer: kong.Consumer{
				ID:       new("second"),
				Username: new("my-consumer2"),
			},
		},
	}
	for _, s := range consumers {
		require.NoError(t, collection.Add(s))
	}

	allConsumers, err := collection.GetAll()

	require.NoError(t, err)
	assert.Len(allConsumers, len(consumers))
}
