package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mtlsAuthsCollection() *MTLSAuthsCollection {
	return state().MTLSAuths
}

func TestMTLSAuthInsert(t *testing.T) {
	collection := mtlsAuthsCollection()

	var mtlsAuth MTLSAuth
	mtlsAuth.ID = kong.String("first")
	err := collection.Add(mtlsAuth)
	require.Error(t, err)

	mtlsAuth.SubjectName = kong.String("test@example.com")
	err = collection.Add(mtlsAuth)
	require.Error(t, err)

	var mtlsAuth2 MTLSAuth
	mtlsAuth2.SubjectName = kong.String("test@example.com")
	mtlsAuth2.ID = kong.String("first")
	mtlsAuth2.Consumer = &kong.Consumer{
		ID:       kong.String("consumer-id"),
		Username: kong.String("my-username"),
	}
	err = collection.Add(mtlsAuth2)
	require.NoError(t, err)
}

func TestMTLSAuthGet(t *testing.T) {
	assert := assert.New(t)
	collection := mtlsAuthsCollection()

	var mtlsAuth MTLSAuth
	mtlsAuth.SubjectName = kong.String("test@example.com")
	mtlsAuth.ID = kong.String("first")
	mtlsAuth.Consumer = &kong.Consumer{
		ID:       kong.String("consumer1-id"),
		Username: kong.String("consumer1-name"),
	}

	err := collection.Add(mtlsAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("test@example.com", *res.SubjectName)

	res, err = collection.Get("does-not-exist")
	require.Error(t, err)
	assert.Nil(res)
}

func TestMTLSAuthUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := mtlsAuthsCollection()

	var mtlsAuth MTLSAuth
	mtlsAuth.SubjectName = kong.String("test@example.com")
	mtlsAuth.ID = kong.String("first")
	mtlsAuth.Consumer = &kong.Consumer{
		ID:       kong.String("consumer1-id"),
		Username: kong.String("consumer1-name"),
	}

	err := collection.Add(mtlsAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("test@example.com", *res.SubjectName)

	res.SubjectName = kong.String("test2@example.com")
	err = collection.Update(*res)
	require.NoError(t, err)

	res, err = collection.Get("first")
	require.NoError(t, err)
	assert.Equal("test2@example.com", *res.SubjectName)
}

func TestMTLSAuthDelete(t *testing.T) {
	assert := assert.New(t)
	collection := mtlsAuthsCollection()

	var mtlsAuth MTLSAuth
	mtlsAuth.SubjectName = kong.String("test@example.com")
	mtlsAuth.ID = kong.String("first")
	mtlsAuth.Consumer = &kong.Consumer{
		ID:       kong.String("consumer1-id"),
		Username: kong.String("consumer1-name"),
	}
	err := collection.Add(mtlsAuth)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)

	err = collection.Delete(*res.ID)
	require.NoError(t, err)

	res, err = collection.Get("first")
	require.Error(t, err)
	assert.Nil(res)

	// delete a non-existing one
	err = collection.Delete("first")
	require.Error(t, err)
}

func TestMTLSAuthGetAll(t *testing.T) {
	collection := mtlsAuthsCollection()

	populateWithMTLSAuthFixtures(t, collection)

	mtlsAuths, err := collection.GetAll()
	require.NoError(t, err)
	require.Len(t, mtlsAuths, 5)
}

func TestMTLSAuthGetByConsumer(t *testing.T) {
	collection := mtlsAuthsCollection()

	populateWithMTLSAuthFixtures(t, collection)

	mtlsAuths, err := collection.GetAllByConsumerID("consumer1-id")
	require.NoError(t, err)
	require.Len(t, mtlsAuths, 3)
}

func populateWithMTLSAuthFixtures(
	t *testing.T,
	collection *MTLSAuthsCollection,
) {
	mtlsAuths := []MTLSAuth{
		{
			MTLSAuth: kong.MTLSAuth{
				SubjectName: kong.String("test11@example.com"),
				ID:          kong.String("first"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			MTLSAuth: kong.MTLSAuth{
				SubjectName: kong.String("test12@example.com"),
				ID:          kong.String("second"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			MTLSAuth: kong.MTLSAuth{
				SubjectName: kong.String("test13@example.com"),
				ID:          kong.String("third"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer1-id"),
					Username: kong.String("consumer1-name"),
				},
			},
		},
		{
			MTLSAuth: kong.MTLSAuth{
				SubjectName: kong.String("test21@example.com"),
				ID:          kong.String("fourth"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer2-id"),
					Username: kong.String("consumer2-name"),
				},
			},
		},
		{
			MTLSAuth: kong.MTLSAuth{
				SubjectName: kong.String("test22@example.com"),
				ID:          kong.String("fifth"),
				Consumer: &kong.Consumer{
					ID:       kong.String("consumer2-id"),
					Username: kong.String("consumer2-name"),
				},
			},
		},
	}

	for _, k := range mtlsAuths {
		err := collection.Add(k)
		require.NoError(t, err)
	}
}
