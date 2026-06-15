package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func customPluginDefinitionsCollection() *CustomPluginDefinitionsCollection {
	return state().CustomPluginDefinitions
}

func createTestCustomPluginDefinition(id string, name string) CustomPluginDefinition {
	cpd := CustomPluginDefinition{}
	cpd.ID = new(id)
	if name != "" {
		cpd.Name = new(name)
	}
	return cpd
}

func TestCustomPluginDefinitionsCollectionAdd(t *testing.T) {
	collection := customPluginDefinitionsCollection()

	tests := []struct {
		name    string
		cpd     CustomPluginDefinition
		wantErr bool
	}{
		{
			name:    "add custom plugin definition with ID",
			cpd:     createTestCustomPluginDefinition("cpd-id", "cpd-name"),
			wantErr: false,
		},
		{
			name:    "add custom plugin definition without ID",
			cpd:     CustomPluginDefinition{},
			wantErr: true,
		},
		{
			name:    "add custom plugin definition without name",
			cpd:     createTestCustomPluginDefinition("cpd-id-2", ""),
			wantErr: false,
		},
		{
			name:    "add duplicate custom plugin definition by ID",
			cpd:     createTestCustomPluginDefinition("cpd-id", "cpd-name"),
			wantErr: true,
		},
		{
			name:    "add duplicate custom plugin definition by name",
			cpd:     createTestCustomPluginDefinition("cpd-id-new", "cpd-name"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collection.Add(tt.cpd); (err != nil) != tt.wantErr {
				t.Errorf("CustomPluginDefinitionsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustomPluginDefinitionsCollectionGet(t *testing.T) {
	collection := customPluginDefinitionsCollection()

	cpdID := "cpd-id"
	cpdName := "cpd-name"
	err := collection.Add(createTestCustomPluginDefinition(cpdID, cpdName))
	require.NoError(t, err, "error adding custom plugin definition")

	t.Run("get custom plugin definition by ID", func(t *testing.T) {
		res, err := collection.Get(cpdID)
		require.NoError(t, err, "error getting custom plugin definition by ID")
		require.NotNil(t, res)
		assert.Equal(t, cpdID, *res.ID)
		assert.Equal(t, cpdName, *res.Name)
	})

	t.Run("get custom plugin definition by name", func(t *testing.T) {
		res, err := collection.Get(cpdName)
		require.NoError(t, err, "error getting custom plugin definition by Name")
		require.NotNil(t, res)
		assert.Equal(t, cpdID, *res.ID)
		assert.Equal(t, cpdName, *res.Name)
	})

	t.Run("get non-existent custom plugin definition", func(t *testing.T) {
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

func TestCustomPluginDefinitionsCollectionUpdate(t *testing.T) {
	collection := customPluginDefinitionsCollection()

	t.Run("update existing custom plugin definition", func(t *testing.T) {
		cpdID := "cpd-id"
		err := collection.Add(createTestCustomPluginDefinition(cpdID, "cpd-name"))
		require.NoError(t, err, "error adding custom plugin definition")

		newName := "new-cpd-name"
		updated := createTestCustomPluginDefinition(cpdID, newName)
		err = collection.Update(updated)
		require.NoError(t, err, "error updating custom plugin definition")

		res, err := collection.Get(cpdID)
		require.NoError(t, err, "error getting custom plugin definition")
		require.NotNil(t, res)
		assert.Equal(t, cpdID, *res.ID)
		assert.Equal(t, newName, *res.Name)
	})

	t.Run("update non-existent custom plugin definition", func(t *testing.T) {
		cpd := createTestCustomPluginDefinition("non-existent", "cpd-name")
		err := collection.Update(cpd)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("update with empty ID", func(t *testing.T) {
		cpd := CustomPluginDefinition{}
		err := collection.Update(cpd)
		require.Error(t, err)
		assert.Equal(t, errIDRequired, err)
	})
}

func TestCustomPluginDefinitionsCollectionDelete(t *testing.T) {
	collection := customPluginDefinitionsCollection()

	t.Run("delete custom plugin definition by ID", func(t *testing.T) {
		cpdID := "cpd-id"
		cpdName := "cpd-name"
		err := collection.Add(createTestCustomPluginDefinition(cpdID, cpdName))
		require.NoError(t, err, "error adding custom plugin definition")

		res, err := collection.Get(cpdID)
		require.NoError(t, err)
		require.NotNil(t, res)

		err = collection.Delete(cpdID)
		require.NoError(t, err, "error deleting custom plugin definition")

		res, err = collection.Get(cpdID)
		require.Error(t, err)
		require.Nil(t, res)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete custom plugin definition by name", func(t *testing.T) {
		cpdID := "cpd-id2"
		cpdName := "cpd-name2"
		err := collection.Add(createTestCustomPluginDefinition(cpdID, cpdName))
		require.NoError(t, err, "error adding custom plugin definition")

		res, err := collection.Get(cpdName)
		require.NoError(t, err)
		require.NotNil(t, res)

		err = collection.Delete(cpdName)
		require.NoError(t, err, "error deleting custom plugin definition by name")

		res, err = collection.Get(cpdID)
		require.Error(t, err)
		require.Nil(t, res)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete non-existent custom plugin definition", func(t *testing.T) {
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

func TestCustomPluginDefinitionsCollection_GetAll(t *testing.T) {
	collection := customPluginDefinitionsCollection()

	t.Run("get all from empty collection", func(t *testing.T) {
		res, err := collection.GetAll()
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("get all from non-empty collection", func(t *testing.T) {
		cpds := []CustomPluginDefinition{
			createTestCustomPluginDefinition("cpd-id-1", "cpd-name-1"),
			createTestCustomPluginDefinition("cpd-id-2", "cpd-name-2"),
			createTestCustomPluginDefinition("cpd-id-3", "cpd-name-3"),
		}

		for _, cpd := range cpds {
			err := collection.Add(cpd)
			require.NoError(t, err, "error adding custom plugin definition")
		}

		res, err := collection.GetAll()
		require.NoError(t, err, "error in getting all custom plugin definitions")
		assert.Len(t, res, len(cpds))
		assert.IsType(t, []*CustomPluginDefinition{}, res)

		cpdMap := make(map[string]bool)
		for _, c := range res {
			cpdMap[*c.ID] = true
		}
		for _, cpd := range cpds {
			assert.True(t, cpdMap[*cpd.ID], "CustomPluginDefinition with ID %s not found in results", *cpd.ID)
		}
	})
}
