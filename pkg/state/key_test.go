package state

import (
	"errors"
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keysCollection() *KeysCollection {
	return state().Keys
}

func createTestKey(id string, name string) Key {
	key := Key{}
	key.ID = kong.String(id)
	if name != "" {
		key.Name = kong.String(name)
	}
	return key
}

func TestKeysCollectionAdd(t *testing.T) {
	collection := keysCollection()

	tests := []struct {
		name    string
		key     Key
		wantErr bool
	}{
		{
			name:    "add key with ID",
			key:     createTestKey("key-id", "key-name"),
			wantErr: false,
		},
		{
			name:    "add key without ID",
			key:     Key{},
			wantErr: true,
		},
		{
			name:    "add duplicate key by ID",
			key:     createTestKey("key-id", "key-name"),
			wantErr: true,
		},
		{
			name:    "add duplicate key by name",
			key:     createTestKey("key-id-new", "key-name"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collection.Add(tt.key); (err != nil) != tt.wantErr {
				t.Errorf("KeysCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeysCollectionGet(t *testing.T) {
	collection := keysCollection()
	assert := assert.New(t)
	require := require.New(t)

	keyID := "key-id"
	keyName := "key-name"
	err := collection.Add(createTestKey(keyID, keyName))
	require.NoError(err, "error adding key")
	t.Run("get key by ID", func(t *testing.T) {
		res, err := collection.Get(keyID)
		require.NoError(err, "error getting key by ID")
		require.NotNil(res)
		assert.Equal(keyID, *res.ID)
		assert.Equal(keyName, *res.Name)
	})

	t.Run("get key by name", func(t *testing.T) {
		res, err := collection.Get(keyName)
		require.NoError(err, "error getting key by Name")
		require.NotNil(res)
		assert.Equal(keyID, *res.ID)
		assert.Equal(keyName, *res.Name)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		res, err := collection.Get("non-existent-key")
		require.Error(err)
		require.Nil(res)
		assert.True(errors.Is(err, ErrNotFound))
	})

	t.Run("get with empty ID", func(t *testing.T) {
		res, err := collection.Get("")
		require.Error(err)
		require.Nil(res)
		assert.Equal(errIDRequired, err)
	})
}

func TestKeysCollectionUpdate(t *testing.T) {
	collection := keysCollection()
	assert := assert.New(t)
	require := require.New(t)
	t.Run("update existing key", func(t *testing.T) {
		keyID := "key-id"
		err := collection.Add(createTestKey(keyID, "key-name"))
		require.NoError(err, "error adding key")

		// Update the key
		newName := "new-key-name"
		updatedKey := createTestKey(keyID, newName)
		err = collection.Update(updatedKey)
		require.NoError(err, "error updating key")

		// Verify the key was updated
		res, err := collection.Get(keyID)
		require.NoError(err, "error getting key")
		require.NotNil(res)
		assert.Equal(keyID, *res.ID)
		assert.Equal(newName, *res.Name)
	})

	t.Run("update non-existent key", func(t *testing.T) {
		key := createTestKey("non-existent", "key-name")

		err := collection.Update(key)
		require.Error(err)
		assert.True(errors.Is(err, ErrNotFound))
	})

	t.Run("update with empty ID", func(t *testing.T) {
		key := Key{}

		err := collection.Update(key)
		require.Error(err)
		assert.Equal(errIDRequired, err)
	})
}

func TestKeysCollectionDelete(t *testing.T) {
	collection := keysCollection()
	require := require.New(t)
	assert := assert.New(t)

	t.Run("delete key by ID", func(t *testing.T) {
		// Add a key
		keyID := "key-id"
		keyName := "key-name"
		err := collection.Add(createTestKey(keyID, keyName))
		require.NoError(err, "error adding key")

		// Check if key exists
		res, err := collection.Get(keyID)
		require.NoError(err, "error getting key by ID")
		require.NotNil(res)

		// Delete key
		err = collection.Delete(keyID)
		require.NoError(err, "error in deleting key")

		// Verify the key was deleted
		res, err = collection.Get(keyID)
		require.Error(err)
		require.Nil(res)
		assert.True(errors.Is(err, ErrNotFound))
	})

	t.Run("delete key by name", func(t *testing.T) {
		// Add a key
		keyID := "key-id"
		keyName := "key-name"
		err := collection.Add(createTestKey(keyID, keyName))
		require.NoError(err, "error adding key")

		// Check if key exists
		res, err := collection.Get(keyName)
		require.NoError(err, "error getting key by name")
		require.NotNil(res)

		// Delete key
		err = collection.Delete(keyName)
		require.NoError(err, "error in deleting key")

		// Verify the key was deleted
		res, err = collection.Get(keyID)
		require.Error(err)
		require.Nil(res)
		assert.True(errors.Is(err, ErrNotFound))
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		err := collection.Delete("non-existent")
		require.Error(err)
		assert.True(errors.Is(err, ErrNotFound))
	})

	t.Run("delete with empty ID", func(t *testing.T) {
		err := collection.Delete("")
		require.Error(err)
		assert.Equal(errIDRequired, err)
	})
}

func TestKeysCollection_GetAll(t *testing.T) {
	collection := keysCollection()
	assert := assert.New(t)
	require := require.New(t)
	t.Run("get all keys from empty collection", func(t *testing.T) {
		res, err := collection.GetAll()
		require.NoError(err)
		assert.Empty(res)
	})

	t.Run("get all keys from non-empty collection", func(t *testing.T) {
		// Add multiple keys
		keys := []Key{
			createTestKey("key-id-1", "key-name-1"),
			createTestKey("key-id-2", "key-name-2"),
			createTestKey("key-id-3", "key-name-3"),
		}

		for _, key := range keys {
			err := collection.Add(key)
			require.NoError(err, "error adding key")
		}

		// Get all keys
		res, err := collection.GetAll()
		require.NoError(err, "error in getting all keys")
		assert.Len(res, len(keys))
		assert.IsType([]*Key{}, res)

		// Verify all keys are present
		keyMap := make(map[string]bool)
		for _, k := range res {
			keyMap[*k.ID] = true
		}

		for _, key := range keys {
			assert.True(keyMap[*key.ID], "Key with ID %s not found in results", *key.ID)
		}
	})
}
