package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keyAuthsCollection() *KeyAuthsCollection {
	return state().KeyAuths
}

func TestKeyAuthInsert(t *testing.T) {
	collection := keyAuthsCollection()

	var keyAuth KeyAuth
	keyAuth.Key = kong.String("my-secret-apikey")
	keyAuth.ID = kong.String("first")
	err := collection.Add(keyAuth)
	require.Error(t, err)

	var keyAuth2 KeyAuth
	keyAuth2.Key = kong.String("my-secret-apikey")
	keyAuth2.ID = kong.String("first")
	keyAuth2.Consumer = &kong.Consumer{
		ID: kong.String("consumer-id"),
	}
	err = collection.Add(keyAuth2)
	require.NoError(t, err)

	// same API key
	keyAuth2.Key = kong.String("my-secret-apikey")
	keyAuth2.ID = kong.String("second")
	keyAuth2.Consumer = &kong.Consumer{
		ID: kong.String("consumer-id"),
	}
	err = collection.Add(keyAuth2)
	require.Error(t, err)

	// re-insert
	err = collection.Add(keyAuth2)
	require.Error(t, err)
}

func TestKeyAuthGet(t *testing.T) {
	assert := assert.New(t)
	collection := keyAuthsCollection()

	var keyAuth KeyAuth
	keyAuth.Key = kong.String("my-apikey")
	keyAuth.ID = kong.String("first")
	keyAuth.Consumer = &kong.Consumer{
		ID: kong.String("consumer1-id"),
	}

	err := collection.Add(keyAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-apikey", *res.Key)

	res, err = collection.Get("my-apikey")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("first", *res.ID)
	assert.Equal("consumer1-id", *res.Consumer.ID)

	res, err = collection.Get("does-not-exist")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.Get("")
	require.Error(t, err)
	assert.Nil(res)
}

func TestKeyAuthUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := keyAuthsCollection()

	var keyAuth KeyAuth

	require.Error(t, collection.Add(keyAuth))

	keyAuth.Key = kong.String("my-apikey")
	keyAuth.ID = kong.String("first")
	keyAuth.Consumer = &kong.Consumer{
		ID: kong.String("consumer1-id"),
	}

	err := collection.Add(keyAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-apikey", *res.Key)

	res.Key = kong.String("my-apikey2")
	err = collection.Update(*res)
	require.NoError(t, err)

	res, err = collection.Get("first")
	require.NoError(t, err)
	assert.Equal("my-apikey2", *res.Key)

	res, err = collection.Get("my-apikey")
	require.Error(t, err)
	assert.Nil(res)
}

func TestKeyAuthDelete(t *testing.T) {
	assert := assert.New(t)
	collection := keyAuthsCollection()

	var keyAuth KeyAuth
	keyAuth.Key = kong.String("my-apikey1")
	keyAuth.ID = kong.String("first")
	keyAuth.Consumer = &kong.Consumer{
		ID: kong.String("consumer1-id"),
	}
	err := collection.Add(keyAuth)
	require.NoError(t, err)

	res, err := collection.Get("my-apikey1")
	require.NoError(t, err)
	assert.NotNil(res)

	err = collection.Delete(*res.ID)
	require.NoError(t, err)

	res, err = collection.Get("my-apikey1")
	require.Error(t, err)
	assert.Nil(res)

	// delete a non-existing one
	err = collection.Delete("first")
	require.Error(t, err)

	err = collection.Delete("my-apikey1")
	require.Error(t, err)

	err = collection.Delete("does-not-exist")
	require.Error(t, err)

	err = collection.Delete("")
	require.Error(t, err)
}

func TestKeyAuthGetAll(t *testing.T) {
	collection := keyAuthsCollection()

	populateWithKeyAuthFixtures(t, collection)

	keyAuths, err := collection.GetAll()
	require.NoError(t, err)
	require.Len(t, keyAuths, 5)
}

func TestKeyAuthGetByConsumer(t *testing.T) {
	collection := keyAuthsCollection()

	populateWithKeyAuthFixtures(t, collection)

	keyAuths, err := collection.GetAllByConsumerID("consumer1-id")
	require.NoError(t, err)
	require.Len(t, keyAuths, 3)
}

func populateWithKeyAuthFixtures(
	t *testing.T,
	collection *KeyAuthsCollection,
) {
	keyAuths := []KeyAuth{
		{
			KeyAuth: kong.KeyAuth{
				Key: kong.String("my-apikey11"),
				ID:  kong.String("first"),
				Consumer: &kong.Consumer{
					ID: kong.String("consumer1-id"),
				},
			},
		},
		{
			KeyAuth: kong.KeyAuth{
				Key: kong.String("my-apikey12"),
				ID:  kong.String("second"),
				Consumer: &kong.Consumer{
					ID: kong.String("consumer1-id"),
				},
			},
		},
		{
			KeyAuth: kong.KeyAuth{
				Key: kong.String("my-apikey13"),
				ID:  kong.String("third"),
				Consumer: &kong.Consumer{
					ID: kong.String("consumer1-id"),
				},
			},
		},
		{
			KeyAuth: kong.KeyAuth{
				Key: kong.String("my-apikey21"),
				ID:  kong.String("fourth"),
				Consumer: &kong.Consumer{
					ID: kong.String("consumer2-id"),
				},
			},
		},
		{
			KeyAuth: kong.KeyAuth{
				Key: kong.String("my-apikey22"),
				ID:  kong.String("fifth"),
				Consumer: &kong.Consumer{
					ID: kong.String("consumer2-id"),
				},
			},
		},
	}

	for _, k := range keyAuths {
		err := collection.Add(k)
		require.NoError(t, err)
	}
}

func TestKeyAuthInvalidType(t *testing.T) {
	assert := assert.New(t)
	collection := keyAuthsCollection()

	var hmacAuth HMACAuth
	hmacAuth.Username = kong.String("my-hmacAuth")
	hmacAuth.ID = kong.String("first")
	hmacAuth.Consumer = &kong.Consumer{
		ID: kong.String("consumer-id"),
	}
	txn := collection.db.Txn(true)
	require.NoError(t, txn.Insert("key-auth", &hmacAuth))
	txn.Commit()

	assert.Panics(func() {
		collection.Get("first")
	})
	assert.Panics(func() {
		collection.GetAll()
	})
	assert.Panics(func() {
		collection.GetAllByConsumerID("consumer-id")
	})
}
