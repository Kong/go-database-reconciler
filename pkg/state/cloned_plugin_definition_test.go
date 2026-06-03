package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clonedPluginDefinitionsCollection() *ClonedPluginDefinitionsCollection {
	return state().ClonedPluginDefinitions
}

func createTestClonedPluginDefinition(id string, name string) ClonedPluginDefinition {
	cpd := ClonedPluginDefinition{}
	cpd.ID = kong.String(id)
	if name != "" {
		cpd.Name = kong.String(name)
	}
	return cpd
}

func TestClonedPluginDefinitionsCollectionAdd(t *testing.T) {
	collection := clonedPluginDefinitionsCollection()

	tests := []struct {
		name    string
		cpd     ClonedPluginDefinition
		wantErr bool
	}{
		{
			name:    "add cloned plugin definition with ID",
			cpd:     createTestClonedPluginDefinition("cpd-id", "cpd-name"),
			wantErr: false,
		},
		{
			name:    "add cloned plugin definition without ID",
			cpd:     ClonedPluginDefinition{},
			wantErr: true,
		},
		{
			name:    "add cloned plugin definition without name",
			cpd:     createTestClonedPluginDefinition("cpd-id-2", ""),
			wantErr: false,
		},
		{
			name:    "add duplicate cloned plugin definition by ID",
			cpd:     createTestClonedPluginDefinition("cpd-id", "cpd-name"),
			wantErr: true,
		},
		{
			name:    "add duplicate cloned plugin definition by name",
			cpd:     createTestClonedPluginDefinition("cpd-id-new", "cpd-name"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collection.Add(tt.cpd); (err != nil) != tt.wantErr {
				t.Errorf("ClonedPluginDefinitionsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClonedPluginDefinitionsCollectionGet(t *testing.T) {
	collection := clonedPluginDefinitionsCollection()

	cpdID := "cpd-id"
	cpdName := "cpd-name"
	err := collection.Add(createTestClonedPluginDefinition(cpdID, cpdName))
	require.NoError(t, err, "error adding cloned plugin definition")

	t.Run("get cloned plugin definition by ID", func(t *testing.T) {
		res, err := collection.Get(cpdID)
		require.NoError(t, err, "error getting cloned plugin definition by ID")
		require.NotNil(t, res)
		assert.Equal(t, cpdID, *res.ID)
		assert.Equal(t, cpdName, *res.Name)
	})

	t.Run("get cloned plugin definition by name", func(t *testing.T) {
		res, err := collection.Get(cpdName)
		require.NoError(t, err, "error getting cloned plugin definition by Name")
		require.NotNil(t, res)
		assert.Equal(t, cpdID, *res.ID)
		assert.Equal(t, cpdName, *res.Name)
	})

	t.Run("get non-existent cloned plugin definition", func(t *testing.T) {
		res, err := collection.Get("non-existent")
		require.Error(t, err)
		require.Nil(t, res)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("get with empty ID", func(t *testing.T) {
		res, err := collection.Get("")
		require.Error(t, err)
		require.Nil(t, res)
		assert.Equal(t, errIDRequired, err)
	})
}

func TestClonedPluginDefinitionsCollectionUpdate(t *testing.T) {
	collection := clonedPluginDefinitionsCollection()

	t.Run("update existing cloned plugin definition", func(t *testing.T) {
		cpdID := "cpd-id"
		err := collection.Add(createTestClonedPluginDefinition(cpdID, "cpd-name"))
		require.NoError(t, err, "error adding cloned plugin definition")

		newName := "new-cpd-name"
		updated := createTestClonedPluginDefinition(cpdID, newName)
		err = collection.Update(updated)
		require.NoError(t, err, "error updating cloned plugin definition")

		res, err := collection.Get(cpdID)
		require.NoError(t, err, "error getting cloned plugin definition")
		require.NotNil(t, res)
		assert.Equal(t, cpdID, *res.ID)
		assert.Equal(t, newName, *res.Name)
	})

	t.Run("update non-existent cloned plugin definition", func(t *testing.T) {
		cpd := createTestClonedPluginDefinition("non-existent", "cpd-name")
		err := collection.Update(cpd)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("update with empty ID", func(t *testing.T) {
		cpd := ClonedPluginDefinition{}
		err := collection.Update(cpd)
		require.Error(t, err)
		assert.Equal(t, errIDRequired, err)
	})
}

func TestClonedPluginDefinitionsCollectionDelete(t *testing.T) {
	collection := clonedPluginDefinitionsCollection()

	t.Run("delete cloned plugin definition by ID", func(t *testing.T) {
		cpdID := "cpd-id"
		cpdName := "cpd-name"
		err := collection.Add(createTestClonedPluginDefinition(cpdID, cpdName))
		require.NoError(t, err, "error adding cloned plugin definition")

		res, err := collection.Get(cpdID)
		require.NoError(t, err)
		require.NotNil(t, res)

		err = collection.Delete(cpdID)
		require.NoError(t, err, "error deleting cloned plugin definition")

		res, err = collection.Get(cpdID)
		require.Error(t, err)
		require.Nil(t, res)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete cloned plugin definition by name", func(t *testing.T) {
		cpdID := "cpd-id2"
		cpdName := "cpd-name2"
		err := collection.Add(createTestClonedPluginDefinition(cpdID, cpdName))
		require.NoError(t, err, "error adding cloned plugin definition")

		res, err := collection.Get(cpdName)
		require.NoError(t, err)
		require.NotNil(t, res)

		err = collection.Delete(cpdName)
		require.NoError(t, err, "error deleting cloned plugin definition by name")

		res, err = collection.Get(cpdID)
		require.Error(t, err)
		require.Nil(t, res)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete non-existent cloned plugin definition", func(t *testing.T) {
		err := collection.Delete("non-existent")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete with empty ID", func(t *testing.T) {
		err := collection.Delete("")
		require.Error(t, err)
		assert.Equal(t, errIDRequired, err)
	})
}

func TestClonedPluginDefinitionsCollection_GetAll(t *testing.T) {
	collection := clonedPluginDefinitionsCollection()

	t.Run("get all from empty collection", func(t *testing.T) {
		res, err := collection.GetAll()
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("get all from non-empty collection", func(t *testing.T) {
		cpds := []ClonedPluginDefinition{
			createTestClonedPluginDefinition("cpd-id-1", "cpd-name-1"),
			createTestClonedPluginDefinition("cpd-id-2", "cpd-name-2"),
			createTestClonedPluginDefinition("cpd-id-3", "cpd-name-3"),
		}

		for _, cpd := range cpds {
			err := collection.Add(cpd)
			require.NoError(t, err, "error adding cloned plugin definition")
		}

		res, err := collection.GetAll()
		require.NoError(t, err, "error in getting all cloned plugin definitions")
		assert.Len(t, res, len(cpds))
		assert.IsType(t, []*ClonedPluginDefinition{}, res)

		cpdMap := make(map[string]bool)
		for _, c := range res {
			cpdMap[*c.ID] = true
		}
		for _, cpd := range cpds {
			assert.True(t, cpdMap[*cpd.ID], "ClonedPluginDefinition with ID %s not found in results", *cpd.ID)
		}
	})
}
