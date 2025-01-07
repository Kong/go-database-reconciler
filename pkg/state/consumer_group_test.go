package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
)

func consumerGroupsCollection() *ConsumerGroupsCollection {
	return state().ConsumerGroups
}

func TestConsumerGroupInsert(t *testing.T) {
	assert := assert.New(t)
	collection := consumerGroupsCollection()

	var cg ConsumerGroup

	assert.NotNil(collection.Add(cg))

	cg.ID = kong.String("my-id")
	cg.Name = kong.String("first")
	assert.Nil(collection.Add(cg))

	// re-insert
	assert.NotNil(collection.Add(cg))
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
