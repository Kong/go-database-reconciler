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
	basicAuth.ID = kong.String("first")
	err := collection.Add(basicAuth)
	require.Error(t, err)

	basicAuth.Username = kong.String("my-username")
	err = collection.Add(basicAuth)
	require.Error(t, err)

	var basicAuth2 BasicAuth
	basicAuth2.Username = kong.String("my-username")
	basicAuth2.ID = kong.String("first")
	basicAuth2.Consumer = &kong.Consumer{
		ID:       kong.String("consumer-id"),
		Username: kong.String("my-username"),
	}
	err = collection.Add(basicAuth2)
	require.NoError(t, err)
}

func TestBasicAuthGet(t *testing.T) {
	assert := assert.New(t)
	collection := basicAuthsCollection()

	var basicAuth BasicAuth
	basicAuth.Username = kong.String("my-username")
	basicAuth.ID = kong.String("first")
	basicAuth.Consumer = &kong.Consumer{
		ID:       kong.String("consumer1-id"),
		Username: kong.String("consumer1-name"),
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
	basicAuth.Username = kong.String("my-username")
	basicAuth.ID = kong.String("first")
	basicAuth.Consumer = &kong.Consumer{
		ID:       kong.String("consumer1-id"),
		Username: kong.String("consumer1-name"),
	}

	err := collection.Add(basicAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-username", *res.Username)

	res.Username = kong.String("my-username2")
	res.Password = kong.String("password")
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
	basicAuth.Username = kong.String("my-username1")
	basicAuth.ID = kong.String("first")
	basicAuth.Consumer = &kong.Consumer{
		ID:       kong.String("consumer1-id"),
		Username: kong.String("consumer1-name"),
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
				Username: kong.String("my-username11"),
				ID:       kong.String("first"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			BasicAuth: kong.BasicAuth{
				Username: kong.String("my-username12"),
				ID:       kong.String("second"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			BasicAuth: kong.BasicAuth{
				Username: kong.String("my-username13"),
				ID:       kong.String("third"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			BasicAuth: kong.BasicAuth{
				Username: kong.String("my-username21"),
				ID:       kong.String("fourth"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer2-id"),
					Username: kong.String("consumer2-name"),
				},
			},
		},
		{
			BasicAuth: kong.BasicAuth{
				Username: kong.String("my-username22"),
				ID:       kong.String("fifth"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer2-id"),
					Username: kong.String("consumer2-name"),
				},
			},
		},
	}

	for _, k := range basicAuths {
		err := collection.Add(k)
		require.NoError(t, err)
	}
}
