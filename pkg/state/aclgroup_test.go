package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func aclGroupsCollection() *ACLGroupsCollection {
	return state().ACLGroups
}

func TestACLGroupInsert(t *testing.T) {
	collection := aclGroupsCollection()

	var aclGroup ACLGroup
	require.Error(t, collection.Add(aclGroup))

	aclGroup.Group = new("my-group")
	aclGroup.ID = new("first")
	err := collection.Add(aclGroup)
	require.Error(t, err)

	var aclGroup2 ACLGroup
	aclGroup2.Group = new("my-group")
	aclGroup2.ID = new("first")
	aclGroup2.Consumer = &kong.Consumer{
		ID: new("consumer-id"),
	}
	err = collection.Add(aclGroup2)
	require.NoError(t, err)

	// re-insert
	err = collection.Add(aclGroup2)
	require.Error(t, err)

	// re-insert with a different ID
	aclGroup2.ID = new("second")
	err = collection.Add(aclGroup2)
	require.Error(t, err)

	// re-insert for different consumer
	aclGroup2.Consumer = &kong.Consumer{
		ID: new("consumer2-id"),
	}
	err = collection.Add(aclGroup2)
	require.NoError(t, err)
}

func TestACLGroupGetByID(t *testing.T) {
	assert := assert.New(t)
	collection := aclGroupsCollection()

	var aclGroup ACLGroup
	aclGroup.Group = new("my-group")
	aclGroup.ID = new("first")
	aclGroup.Consumer = &kong.Consumer{
		ID: new("consumer1-id"),
	}

	err := collection.Add(aclGroup)
	require.NoError(t, err)

	res, err := collection.GetByID("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-group", *res.Group)

	res, err = collection.GetByID("my-group")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.GetByID("does-not-exist")
	require.Error(t, err)
	assert.Nil(res)
}

func TestACLGroupGet(t *testing.T) {
	assert := assert.New(t)
	collection := aclGroupsCollection()

	populateWithACLGroupFixtures(t, collection)

	res, err := collection.Get("first", "does-not-exist")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.Get("does-not-exist", "my-group12")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.Get("consumer1-id", "my-group12")
	require.NoError(t, err)
	assert.NotNil(res)
}

func TestACLGroupUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := aclGroupsCollection()

	var aclGroup ACLGroup
	aclGroup.Group = new("my-group")
	aclGroup.ID = new("first")
	aclGroup.Consumer = &kong.Consumer{
		ID: new("consumer1-id"),
	}

	err := collection.Add(aclGroup)
	require.NoError(t, err)

	res, err := collection.Get("consumer1-id", "first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-group", *res.Group)

	res.Group = new("my-group2")
	err = collection.Update(*res)
	require.NoError(t, err)

	res, err = collection.Get("consumer1-id", "my-group")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.Get("consumer1-id", "my-group2")
	require.NoError(t, err)
	assert.Equal("first", *res.ID)
}

func TestACLGroupDelete(t *testing.T) {
	assert := assert.New(t)
	collection := aclGroupsCollection()

	var aclGroup ACLGroup
	aclGroup.Group = new("my-group1")
	aclGroup.ID = new("first")
	aclGroup.Consumer = &kong.Consumer{
		ID: new("consumer1-id"),
	}
	err := collection.Add(aclGroup)
	require.NoError(t, err)

	res, err := collection.Get("consumer1-id", "my-group1")
	require.NoError(t, err)
	assert.NotNil(res)

	err = collection.Delete(*res.ID)
	require.NoError(t, err)

	res, err = collection.Get("consumer1-id", "my-group1")
	require.Error(t, err)
	assert.Nil(res)

	// delete a non-existing one
	err = collection.Delete("first")
	require.Error(t, err)

	err = collection.Delete("my-group1")
	require.Error(t, err)
}

func TestACLGroupGetAll(t *testing.T) {
	collection := aclGroupsCollection()

	populateWithACLGroupFixtures(t, collection)

	aclGroups, err := collection.GetAll()
	require.NoError(t, err)
	require.Len(t, aclGroups, 5)
}

func TestACLGroupGetByConsumer(t *testing.T) {
	collection := aclGroupsCollection()

	populateWithACLGroupFixtures(t, collection)

	aclGroups, err := collection.GetAllByConsumerID("consumer1-id")
	require.NoError(t, err)
	require.Len(t, aclGroups, 3)
}

func populateWithACLGroupFixtures(
	t *testing.T,
	collection *ACLGroupsCollection,
) {
	aclGroups := []ACLGroup{
		{
			ACLGroup: kong.ACLGroup{
				Group: new("my-group11"),
				ID:    new("first"),
				Consumer: &kong.Consumer{
					ID: new("consumer1-id"),
				},
			},
		},
		{
			ACLGroup: kong.ACLGroup{
				Group: new("my-group12"),
				ID:    new("second"),
				Consumer: &kong.Consumer{
					ID: new("consumer1-id"),
				},
			},
		},
		{
			ACLGroup: kong.ACLGroup{
				Group: new("my-group13"),
				ID:    new("third"),
				Consumer: &kong.Consumer{
					ID: new("consumer1-id"),
				},
			},
		},
		{
			ACLGroup: kong.ACLGroup{
				Group: new("my-group21"),
				ID:    new("fourth"),
				Consumer: &kong.Consumer{
					ID: new("consumer2-id"),
				},
			},
		},
		{
			ACLGroup: kong.ACLGroup{
				Group: new("my-group22"),
				ID:    new("fifth"),
				Consumer: &kong.Consumer{
					ID: new("consumer2-id"),
				},
			},
		},
	}

	for _, k := range aclGroups {
		err := collection.Add(k)
		require.NoError(t, err)
	}
}
