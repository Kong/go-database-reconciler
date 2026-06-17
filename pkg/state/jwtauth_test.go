package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jwtAuthsCollection() *JWTAuthsCollection {
	return state().JWTAuths
}

func TestJWTAuthInsert(t *testing.T) {
	collection := jwtAuthsCollection()

	var jwtAuth JWTAuth
	jwtAuth.Key = new("my-key")
	jwtAuth.ID = new("first")
	err := collection.Add(jwtAuth)
	require.Error(t, err)

	var jwtAuth2 JWTAuth
	jwtAuth2.Key = new("my-key")
	jwtAuth2.ID = new("first")
	jwtAuth2.Consumer = &kong.Consumer{
		ID:       new("consumer-id"),
		Username: new("my-username"),
	}
	err = collection.Add(jwtAuth2)
	require.NoError(t, err)
}

func TestJWTAuthGet(t *testing.T) {
	assert := assert.New(t)
	collection := jwtAuthsCollection()

	var jwtAuth JWTAuth
	jwtAuth.Key = new("my-key")
	jwtAuth.ID = new("first")
	jwtAuth.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}

	err := collection.Add(jwtAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-key", *res.Key)

	res, err = collection.Get("my-key")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("first", *res.ID)
	assert.Equal("consumer1-id", *res.Consumer.ID)

	res, err = collection.Get("does-not-exist")
	require.Error(t, err)
	assert.Nil(res)
}

func TestJWTAuthUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := jwtAuthsCollection()

	var jwtAuth JWTAuth
	jwtAuth.Key = new("my-key")
	jwtAuth.ID = new("first")
	jwtAuth.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}

	err := collection.Add(jwtAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-key", *res.Key)

	res.Key = new("my-key2")
	err = collection.Update(*res)
	require.NoError(t, err)

	res, err = collection.Get("my-key")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.Get("my-key2")
	require.NoError(t, err)
	assert.Equal("first", *res.ID)
}

func TestJWTAuthDelete(t *testing.T) {
	assert := assert.New(t)
	collection := jwtAuthsCollection()

	var jwtAuth JWTAuth
	jwtAuth.Key = new("my-key1")
	jwtAuth.ID = new("first")
	jwtAuth.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}
	err := collection.Add(jwtAuth)
	require.NoError(t, err)

	res, err := collection.Get("my-key1")
	require.NoError(t, err)
	assert.NotNil(res)

	err = collection.Delete(*res.ID)
	require.NoError(t, err)

	res, err = collection.Get("my-key1")
	require.Error(t, err)
	assert.Nil(res)

	// delete a non-existing one
	err = collection.Delete("first")
	require.Error(t, err)

	err = collection.Delete("my-key1")
	require.Error(t, err)
}

func TestJWTAuthGetAll(t *testing.T) {
	collection := jwtAuthsCollection()

	populateWithJWTAuthFixtures(t, collection)

	jwtAuths, err := collection.GetAll()
	require.NoError(t, err)
	require.Len(t, jwtAuths, 5)
}

func TestJWTAuthGetByConsumer(t *testing.T) {
	collection := jwtAuthsCollection()

	populateWithJWTAuthFixtures(t, collection)

	jwtAuths, err := collection.GetAllByConsumerID("consumer1-id")
	require.NoError(t, err)
	require.Len(t, jwtAuths, 3)
}

func populateWithJWTAuthFixtures(
	t *testing.T,
	collection *JWTAuthsCollection,
) {
	jwtAuths := []JWTAuth{
		{
			JWTAuth: kong.JWTAuth{
				Key: new("my-key11"),
				ID:  new("first"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			JWTAuth: kong.JWTAuth{
				Key: new("my-key12"),
				ID:  new("second"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			JWTAuth: kong.JWTAuth{
				Key: new("my-key13"),
				ID:  new("third"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			JWTAuth: kong.JWTAuth{
				Key: new("my-key21"),
				ID:  new("fourth"),
				Consumer: &kong.Consumer{
					ID:       new("consumer2-id"),
					Username: new("consumer2-name"),
				},
			},
		},
		{
			JWTAuth: kong.JWTAuth{
				Key: new("my-key22"),
				ID:  new("fifth"),
				Consumer: &kong.Consumer{
					ID:       new("consumer2-id"),
					Username: new("consumer2-name"),
				},
			},
		},
	}

	for _, k := range jwtAuths {
		err := collection.Add(k)
		require.NoError(t, err)
	}
}
