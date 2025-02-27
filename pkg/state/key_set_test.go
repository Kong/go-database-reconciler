package state

import (
	"errors"
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keySetsCollection() *KeySetsCollection {
	return state().KeySets
}

func createTestKeySet(id string, name string) KeySet {
	keySet := KeySet{}
	keySet.ID = kong.String(id)
	if name != "" {
		keySet.Name = kong.String(name)
	}
	return keySet
}

func TestKeySetCollectionAdd(t *testing.T) {
	collection := keySetsCollection()

	tests := []struct {
		name    string
		keySet  KeySet
		wantErr bool
	}{
		{
			name:    "add key-set with ID",
			keySet:  createTestKeySet("keyset-id", "keyset-name"),
			wantErr: false,
		},
		{
			name:    "add key-set without ID",
			keySet:  KeySet{},
			wantErr: true,
		},
		{
			name:    "add duplicate key-set by ID",
			keySet:  createTestKeySet("keyset-id", "keyset-name"),
			wantErr: true,
		},
		{
			name:    "add duplicate key-set by name",
			keySet:  createTestKeySet("keyset-id-new", "keyset-name"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collection.Add(tt.keySet); (err != nil) != tt.wantErr {
				t.Errorf("KeySetsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeySetCollectionGet(t *testing.T) {
	collection := keySetsCollection()
	assert := assert.New(t)
	require := require.New(t)

	keySetID := "keyset-id"
	keySetName := "keyset-name"
	err := collection.Add(createTestKeySet(keySetID, keySetName))
	require.NoError(err, "error adding key set")
	t.Run("get key-set by ID", func(t *testing.T) {
		res, err := collection.Get(keySetID)
		require.NoError(err, "error getting key-set by ID")
		require.NotNil(res)
		assert.Equal(keySetID, *res.ID)
		assert.Equal(keySetName, *res.Name)
	})

	t.Run("get key-set by name", func(t *testing.T) {
		res, err := collection.Get(keySetName)
		require.NoError(err, "error getting key-set by Name")
		require.NotNil(res)
		assert.Equal(keySetID, *res.ID)
		assert.Equal(keySetName, *res.Name)
	})

	t.Run("get non-existent key-set", func(t *testing.T) {
		res, err := collection.Get("non-existent-key-set")
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

func TestKeySetCollectionUpdate(t *testing.T) {
	collection := keySetsCollection()
	assert := assert.New(t)
	require := require.New(t)
	t.Run("update existing key-set", func(t *testing.T) {
		keySetID := "keyset-id"
		err := collection.Add(createTestKeySet(keySetID, "keyset-name"))
		require.NoError(err, "error adding key-set")

		// Update the key-set
		newName := "new-keyset-name"
		updatedKeySet := createTestKeySet(keySetID, newName)
		err = collection.Update(updatedKeySet)
		require.NoError(err, "error updating key-set")

		// Verify the key-set was updated
		res, err := collection.Get(keySetID)
		require.NoError(err, "error getting key-set")
		require.NotNil(res)
		assert.Equal(keySetID, *res.ID)
		assert.Equal(newName, *res.Name)
	})

	t.Run("update non-existent key-set", func(t *testing.T) {
		keySet := createTestKeySet("non-existent", "keyset-name")

		err := collection.Update(keySet)
		require.Error(err)
		assert.True(errors.Is(err, ErrNotFound))
	})

	t.Run("update with empty ID", func(t *testing.T) {
		err := collection.Update(KeySet{})
		require.Error(err)
		assert.Equal(errIDRequired, err)
	})
}

func TestKeySetCollectionDelete(t *testing.T) {
	collection := keySetsCollection()
	require := require.New(t)
	assert := assert.New(t)

	t.Run("delete key-set by ID", func(t *testing.T) {
		// Add a key-set
		keysetID := "keyset-id"
		keySetName := "keyset-name"
		err := collection.Add(createTestKeySet(keysetID, keySetName))
		require.NoError(err, "error adding key-set")

		// Check if key-set exists
		res, err := collection.Get(keysetID)
		require.NoError(err, "error getting key-set by ID")
		require.NotNil(res)

		// Delete key-set
		err = collection.Delete(keysetID)
		require.NoError(err, "error in deleting key-set")

		// Verify the key-set was deleted
		res, err = collection.Get(keysetID)
		require.Error(err)
		require.Nil(res)
		assert.True(errors.Is(err, ErrNotFound))
	})

	t.Run("delete key-set by name", func(t *testing.T) {
		// Add a key-set
		keysetID := "keyset-id"
		keySetName := "keyset-name"
		err := collection.Add(createTestKeySet(keysetID, keySetName))
		require.NoError(err, "error adding key")

		// Check if key-set exists
		res, err := collection.Get(keySetName)
		require.NoError(err, "error getting key by name")
		require.NotNil(res)

		// Delete key-set
		err = collection.Delete(keySetName)
		require.NoError(err, "error in deleting key")

		// Verify the key-set was deleted
		res, err = collection.Get(keysetID)
		require.Error(err)
		require.Nil(res)
		assert.True(errors.Is(err, ErrNotFound))
	})

	t.Run("delete non-existent key-set", func(t *testing.T) {
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

func TestKeySetCollectionGetAll(t *testing.T) {
	collection := keySetsCollection()
	assert := assert.New(t)
	require := require.New(t)
	t.Run("get all key-sets from empty collection", func(t *testing.T) {
		res, err := collection.GetAll()
		require.NoError(err)
		assert.Empty(res)
	})

	t.Run("get all key-sets from non-empty collection", func(t *testing.T) {
		// Add multiple key-sets
		keySets := []KeySet{
			createTestKeySet("keyset-id-1", "keyset-name-1"),
			createTestKeySet("keyset-id-2", "keyset-name-2"),
			createTestKeySet("keyset-id-3", "keyset-name-3"),
		}

		for _, keySet := range keySets {
			err := collection.Add(keySet)
			require.NoError(err, "error adding key-set")
		}

		// Get all key-sets
		res, err := collection.GetAll()
		require.NoError(err, "error in getting all key-sets")
		assert.Len(res, len(keySets))
		assert.IsType([]*KeySet{}, res)

		// Verify all keys are present
		keySetMap := make(map[string]bool)
		for _, k := range res {
			keySetMap[*k.ID] = true
		}

		for _, keySet := range keySets {
			assert.True(keySetMap[*keySet.ID], "KeySet with ID %s not found in results", *keySet.ID)
		}
	})
}
