package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func consumerGroupsCollection() *ConsumerGroupsCollection {
	return state().ConsumerGroups
}

func TestConsumerGroupInsert(t *testing.T) {
	collection := consumerGroupsCollection()

	var cg ConsumerGroup

	require.NotNil(t, collection.Add(cg))

	cg.ID = kong.String("my-id")
	cg.Name = kong.String("first")
	require.NoError(t, collection.Add(cg))

	// re-insert
	require.NotNil(t, collection.Add(cg))
}

func TestConsumerGroupInsertIgnoreDuplicate(t *testing.T) {
	assert := assert.New(t)
	collection := consumerGroupsCollection()

	var cg ConsumerGroup
	cg.ID = kong.String("my-id")
	cg.Name = kong.String("first")
	err := collection.Add(cg)
	assert.Nil(err)
	err = collection.AddIgnoringDuplicates(cg)
	assert.Nil(err)
}
