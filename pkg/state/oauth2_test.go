package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func oauth2CredsCollection() *Oauth2CredsCollection {
	return state().Oauth2Creds
}

func TestOauth2CredInsert(t *testing.T) {
	collection := oauth2CredsCollection()

	var oauth2Cred Oauth2Credential
	oauth2Cred.ClientID = new("client-id")
	oauth2Cred.ID = new("first")
	err := collection.Add(oauth2Cred)
	require.Error(t, err)

	oauth2Cred.Consumer = &kong.Consumer{
		ID:       new("consumer-id"),
		Username: new("my-username"),
	}
	err = collection.Add(oauth2Cred)
	require.NoError(t, err)
}

func TestOauth2CredentialGet(t *testing.T) {
	assert := assert.New(t)
	collection := oauth2CredsCollection()

	var oauth2Cred Oauth2Credential
	oauth2Cred.ClientID = new("my-clientid")
	oauth2Cred.ID = new("first")
	oauth2Cred.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}

	err := collection.Add(oauth2Cred)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-clientid", *res.ClientID)

	res, err = collection.Get("my-clientid")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("first", *res.ID)
	assert.Equal("consumer1-id", *res.Consumer.ID)

	res, err = collection.Get("does-not-exist")
	require.Error(t, err)
	assert.Nil(res)
}

func TestOauth2CredentialUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := oauth2CredsCollection()

	var oauth2Cred Oauth2Credential
	oauth2Cred.ClientID = new("my-clientid")
	oauth2Cred.ID = new("first")
	oauth2Cred.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}

	err := collection.Add(oauth2Cred)
	require.NoError(t, err)

	res, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(res)
	assert.Equal("my-clientid", *res.ClientID)

	res.ClientID = new("my-clientid2")
	err = collection.Update(*res)
	require.NoError(t, err)

	res, err = collection.Get("my-clientid")
	require.Error(t, err)
	assert.Nil(res)

	res, err = collection.Get("my-clientid2")
	require.NoError(t, err)
	assert.Equal("first", *res.ID)
}

func TestOauth2CredentialDelete(t *testing.T) {
	assert := assert.New(t)
	collection := oauth2CredsCollection()

	var oauth2Cred Oauth2Credential
	oauth2Cred.ClientID = new("my-clientid1")
	oauth2Cred.ID = new("first")
	oauth2Cred.Consumer = &kong.Consumer{
		ID:       new("consumer1-id"),
		Username: new("consumer1-name"),
	}
	err := collection.Add(oauth2Cred)
	require.NoError(t, err)

	res, err := collection.Get("my-clientid1")
	require.NoError(t, err)
	assert.NotNil(res)

	err = collection.Delete(*res.ID)
	require.NoError(t, err)

	res, err = collection.Get("my-clientid1")
	require.Error(t, err)
	assert.Nil(res)

	// delete a non-existing one
	err = collection.Delete("first")
	require.Error(t, err)

	err = collection.Delete("my-clientid1")
	require.Error(t, err)
}

func TestOauth2CredentialGetAll(t *testing.T) {
	collection := oauth2CredsCollection()

	populateWithOauth2CredentialFixtures(t, collection)

	oauth2Creds, err := collection.GetAll()
	require.NoError(t, err)
	require.Len(t, oauth2Creds, 5)
}

func TestOauth2CredentialGetByConsumer(t *testing.T) {
	collection := oauth2CredsCollection()

	populateWithOauth2CredentialFixtures(t, collection)

	oauth2Creds, err := collection.GetAllByConsumerID("consumer1-id")
	require.NoError(t, err)
	require.Len(t, oauth2Creds, 3)
}

func populateWithOauth2CredentialFixtures(
	t *testing.T,
	collection *Oauth2CredsCollection,
) {
	oauth2Creds := []Oauth2Credential{
		{
			Oauth2Credential: kong.Oauth2Credential{
				ClientID: new("my-clientid11"),
				ID:       new("first"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			Oauth2Credential: kong.Oauth2Credential{
				ClientID: new("my-clientid12"),
				ID:       new("second"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			Oauth2Credential: kong.Oauth2Credential{
				ClientID: new("my-clientid13"),
				ID:       new("third"),
				Consumer: &kong.Consumer{
					ID:       new("consumer1-id"),
					Username: new("consumer1-name"),
				},
			},
		},
		{
			Oauth2Credential: kong.Oauth2Credential{
				ClientID: new("my-clientid21"),
				ID:       new("fourth"),
				Consumer: &kong.Consumer{
					ID:       new("consumer2-id"),
					Username: new("consumer2-name"),
				},
			},
		},
		{
			Oauth2Credential: kong.Oauth2Credential{
				ClientID: new("my-clientid22"),
				ID:       new("fifth"),
				Consumer: &kong.Consumer{
					ID:       new("consumer2-id"),
					Username: new("consumer2-name"),
				},
			},
		},
	}

	for _, k := range oauth2Creds {
		err := collection.Add(k)
		require.NoError(t, err)
	}
}
