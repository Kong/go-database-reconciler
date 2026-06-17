package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func basicAuthsCollection() *BasicAuthsCollection {
	return state().BasicAuths
}

func TestBasicAuthInsert(t *testing.T) {
	collection := basicAuthsCollection()

	var basicAuth BasicAuth
	basicAuth.ID = new("first")
	err := collection.Add(basicAuth)
	require.Error(t, err)

	basicAuth.Username = new("my-username")
	err = collection.Add(basicAuth)
	require.Error(t, err)

	var basicAuth2 BasicAuth
	basicAuth2.Username = new("my-username")
	basicAuth2.ID = new("first")
	basicAuth2.Consumer = &kong.Consumer{
		ID:       new("consumer-id"),
		Username: new("my-username"),
	}
	err = collection.Add(basicAuth2)
	require.NoError(t, err)
}

func TestBasicAuthGet(t *testing.T) {
	assert := assert.New(t)
	collection := basicAuthsCollection()

	var basicAuth BasicAuth
	basicAuth.Username = new("my-username")
	basicAuth.ID = new("first")
	basicAuth.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}

	err := collection.Add(basicAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-username", *res.Username)

	res, err = collection.Get("my-username")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("first", *res.ID)
	assert.Equal("consumer1-id", *res.Consumer.ID)

	res, err = collection.Get("does-not-exist")
	require.Error(t, err)
	assert.Nil(res)
}

func TestBasicAuthUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := basicAuthsCollection()

	var basicAuth BasicAuth
	basicAuth.Username = new("my-username")
	basicAuth.ID = new("first")
	basicAuth.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}

	err := collection.Add(basicAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-username", *res.Username)

	res.Username = new("my-username2")
	res.Password = new("password")
	err = collection.Update(*res)
	require.NoError(t, err)

	res, err = collection.Get("my-username")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.Get("my-username2")
	require.NoError(t, err)
	assert.Equal("first", *res.ID)
	assert.Equal("password", *res.Password)
}

func TestBasicAuthDelete(t *testing.T) {
	assert := assert.New(t)
	collection := basicAuthsCollection()

	var basicAuth BasicAuth
	basicAuth.Username = new("my-username1")
	basicAuth.ID = new("first")
	basicAuth.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}
	err := collection.Add(basicAuth)
	require.NoError(t, err)

	res, err := collection.Get("my-username1")
	require.NoError(t, err)
	assert.NotNil(res)

	err = collection.Delete(*res.ID)
	require.NoError(t, err)

	res, err = collection.Get("my-username1")
	require.Error(t, err)
	assert.Nil(res)

	// delete a non-existing one
	err = collection.Delete("first")
	require.Error(t, err)

	err = collection.Delete("my-username1")
	require.Error(t, err)
}

func TestBasicAuthGetAll(t *testing.T) {
	collection := basicAuthsCollection()

	populateWithBasicAuthFixtures(t, collection)

	basicAuths, err := collection.GetAll()
	require.NoError(t, err)
	require.Len(t, basicAuths, 5)
}

func TestBasicAuthGetByConsumer(t *testing.T) {
	collection := basicAuthsCollection()

	populateWithBasicAuthFixtures(t, collection)

	basicAuths, err := collection.GetAllByConsumerID("consumer1-id")
	require.NoError(t, err)
	require.Len(t, basicAuths, 3)
}

func populateWithBasicAuthFixtures(
	t *testing.T,
	collection *BasicAuthsCollection,
) {
	basicAuths := []BasicAuth{
		{
			BasicAuth: kong.BasicAuth{
				Username: new("my-username11"),
				ID:       new("first"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			BasicAuth: kong.BasicAuth{
				Username: new("my-username12"),
				ID:       new("second"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			BasicAuth: kong.BasicAuth{
				Username: new("my-username13"),
				ID:       new("third"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			BasicAuth: kong.BasicAuth{
				Username: new("my-username21"),
				ID:       new("fourth"),
				Consumer: &kong.Consumer{
					ID:       new("consumer2-id"),
					Username: new("consumer2-name"),
				},
			},
		},
		{
			BasicAuth: kong.BasicAuth{
				Username: new("my-username22"),
				ID:       new("fifth"),
				Consumer: &kong.Consumer{
					ID:       new("consumer2-id"),
					Username: new("consumer2-name"),
				},
			},
		},
	}

	for _, k := range basicAuths {
		err := collection.Add(k)
		require.NoError(t, err)
	}
}
