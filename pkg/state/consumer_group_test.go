package state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func consumerGroupsCollection() *ConsumerGroupsCollection {
	return state().ConsumerGroups
}

func TestConsumerGroupInsert(t *testing.T) {
	collection := consumerGroupsCollection()

	var cg ConsumerGroup

	require.Error(t, collection.Add(cg))

	cg.ID = new("my-id")
	cg.Name = new("first")
	require.NoError(t, collection.Add(cg))

	// re-insert
	require.Error(t, collection.Add(cg))
}

func TestConsumerGroupInsertIgnoreDuplicate(t *testing.T) {
	collection := consumerGroupsCollection()

	var cg ConsumerGroup
	cg.ID = new("my-id")
	cg.Name = new("first")
	err := collection.Add(cg)
	require.NoError(t, err)
	err = collection.AddIgnoringDuplicates(cg)
	require.NoError(t, err)
}
