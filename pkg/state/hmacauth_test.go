package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hmacAuthsCollection() *HMACAuthsCollection {
	return state().HMACAuths
}

func TestHMACAuthInsert(t *testing.T) {
	collection := hmacAuthsCollection()

	var hmacAuth HMACAuth
	hmacAuth.ID = kong.String("first")
	err := collection.Add(hmacAuth)
	require.Error(t, err)

	hmacAuth.Username = kong.String("my-username")
	err = collection.Add(hmacAuth)
	require.Error(t, err)

	var hmacAuth2 HMACAuth
	hmacAuth2.Username = kong.String("my-username")
	hmacAuth2.ID = kong.String("first")
	hmacAuth2.Consumer = &kong.Consumer{
		ID:       kong.String("consumer-id"),
		Username: kong.String("my-username"),
	}
	err = collection.Add(hmacAuth2)
	require.NoError(t, err)
}

func TestHMACAuthGet(t *testing.T) {
	assert := assert.New(t)
	collection := hmacAuthsCollection()

	var hmacAuth HMACAuth
	hmacAuth.Username = kong.String("my-username")
	hmacAuth.ID = kong.String("first")
	hmacAuth.Consumer = &kong.Consumer{
		ID:       kong.String("consumer1-id"),
		Username: kong.String("consumer1-name"),
	}

	err := collection.Add(hmacAuth)
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

func TestHMACAuthUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := hmacAuthsCollection()

	var hmacAuth HMACAuth
	hmacAuth.Username = kong.String("my-username")
	hmacAuth.ID = kong.String("first")
	hmacAuth.Consumer = &kong.Consumer{
		ID: kong.String("consumer1-id"),
	}

	err := collection.Add(hmacAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-username", *res.Username)

	res.Username = kong.String("my-username2")
	res.Secret = kong.String("secret")
	err = collection.Update(*res)
	require.NoError(t, err)

	res, err = collection.Get("my-username")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.Get("my-username2")
	require.NoError(t, err)
	assert.Equal("first", *res.ID)
	assert.Equal("secret", *res.Secret)
}

func TestHMACAuthDelete(t *testing.T) {
	assert := assert.New(t)
	collection := hmacAuthsCollection()

	var hmacAuth HMACAuth
	hmacAuth.Username = kong.String("my-username1")
	hmacAuth.ID = kong.String("first")
	hmacAuth.Consumer = &kong.Consumer{
		ID:       kong.String("consumer1-id"),
		Username: kong.String("consumer1-name"),
	}
	err := collection.Add(hmacAuth)
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

func TestHMACAuthGetAll(t *testing.T) {
	collection := hmacAuthsCollection()

	populateWithHMACAuthFixtures(t, collection)

	hmacAuths, err := collection.GetAll()
	require.NoError(t, err)
	require.Len(t, hmacAuths, 5)
}

func TestHMACAuthGetByConsumer(t *testing.T) {
	collection := hmacAuthsCollection()

	populateWithHMACAuthFixtures(t, collection)

	hmacAuths, err := collection.GetAllByConsumerID("consumer1-id")
	require.NoError(t, err)
	require.Len(t, hmacAuths, 3)
}

func populateWithHMACAuthFixtures(
	t *testing.T,
	collection *HMACAuthsCollection,
) {
	hmacAuths := []HMACAuth{
		{
			HMACAuth: kong.HMACAuth{
				Username: kong.String("my-username11"),
				ID:       kong.String("first"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			HMACAuth: kong.HMACAuth{
				Username: kong.String("my-username12"),
				ID:       kong.String("second"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			HMACAuth: kong.HMACAuth{
				Username: kong.String("my-username13"),
				ID:       kong.String("third"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			HMACAuth: kong.HMACAuth{
				Username: kong.String("my-username21"),
				ID:       kong.String("fourth"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer2-id"),
					Username: kong.String("consumer2-name"),
				},
			},
		},
		{
			HMACAuth: kong.HMACAuth{
				Username: kong.String("my-username22"),
				ID:       kong.String("fifth"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer2-id"),
					Username: kong.String("consumer2-name"),
				},
			},
		},
	}

	for _, k := range hmacAuths {
		err := collection.Add(k)
		require.NoError(t, err)
	}
}
