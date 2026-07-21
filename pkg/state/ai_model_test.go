package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func aiModelsCollection() *AIModelsCollection {
	return state().AIModels
}

func createTestAIModel(id string, name string) AIModel {
	amd := AIModel{}
	amd.ID = new(id)
	if name != "" {
		amd.Name = new(name)
	}
	return amd
}

func TestAIModelsCollectionAdd(t *testing.T) {
	collection := aiModelsCollection()

	tests := []struct {
		name    string
		amd     AIModel
		wantErr bool
	}{
		{
			name:    "add AI model with ID",
			amd:     createTestAIModel("ai-id", "ai-name"),
			wantErr: false,
		},
		{
			name:    "add AI model without ID",
			amd:     AIModel{},
			wantErr: true,
		},
		{
			name:    "add AI model without name",
			amd:     createTestAIModel("ai-id-2", ""),
			wantErr: false,
		},
		{
			name:    "add duplicate AI model by ID",
			amd:     createTestAIModel("ai-id", "ai-name"),
			wantErr: true,
		},
		{
			name:    "add duplicate AI model by name",
			amd:     createTestAIModel("ai-id-new", "ai-name"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collection.Add(tt.amd); (err != nil) != tt.wantErr {
				t.Errorf("AIModelsCollection.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAIModelsCollectionGet(t *testing.T) {
	collection := aiModelsCollection()

	aiID := "ai-id"
	aiName := "ai-name"
	err := collection.Add(createTestAIModel(aiID, aiName))
	require.NoError(t, err, "error adding AI model")

	t.Run("get AI model by ID", func(t *testing.T) {
		res, err := collection.Get(aiID)
		require.NoError(t, err, "error getting AI model by ID")
		require.NotNil(t, res)
		assert.Equal(t, aiID, *res.ID)
		assert.Equal(t, aiName, *res.Name)
	})

	t.Run("get AI model by name", func(t *testing.T) {
		res, err := collection.Get(aiName)
		require.NoError(t, err, "error getting AI model by name")
		require.NotNil(t, res)
		assert.Equal(t, aiID, *res.ID)
		assert.Equal(t, aiName, *res.Name)
	})

	t.Run("get non-existent AI model", func(t *testing.T) {
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

func TestAIModelsCollectionUpdate(t *testing.T) {
	collection := aiModelsCollection()

	t.Run("update existing AI model", func(t *testing.T) {
		aiID := "ai-id"
		err := collection.Add(createTestAIModel(aiID, "ai-name"))
		require.NoError(t, err, "error adding AI model")

		newName := "new-ai-name"
		updated := createTestAIModel(aiID, newName)
		err = collection.Update(updated)
		require.NoError(t, err, "error updating AI model")

		res, err := collection.Get(aiID)
		require.NoError(t, err, "error getting AI model")
		require.NotNil(t, res)
		assert.Equal(t, aiID, *res.ID)
		assert.Equal(t, newName, *res.Name)
	})

	t.Run("update non-existent AI model", func(t *testing.T) {
		amd := createTestAIModel("non-existent", "ai-name")
		err := collection.Update(amd)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("update with empty ID", func(t *testing.T) {
		amd := AIModel{}
		err := collection.Update(amd)
		require.Error(t, err)
		assert.Equal(t, errIDRequired, err)
	})
}

func TestAIModelsCollectionDelete(t *testing.T) {
	collection := aiModelsCollection()

	t.Run("delete AI model by ID", func(t *testing.T) {
		aiID := "ai-id"
		aiName := "ai-name"
		err := collection.Add(createTestAIModel(aiID, aiName))
		require.NoError(t, err, "error adding AI model")

		res, err := collection.Get(aiID)
		require.NoError(t, err)
		require.NotNil(t, res)

		err = collection.Delete(aiID)
		require.NoError(t, err, "error deleting AI model")

		res, err = collection.Get(aiID)
		require.Error(t, err)
		require.Nil(t, res)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete AI model by name", func(t *testing.T) {
		aiID := "ai-id2"
		aiName := "ai-name2"
		err := collection.Add(createTestAIModel(aiID, aiName))
		require.NoError(t, err, "error adding AI model")

		res, err := collection.Get(aiName)
		require.NoError(t, err)
		require.NotNil(t, res)

		err = collection.Delete(aiName)
		require.NoError(t, err, "error deleting AI model by name")

		res, err = collection.Get(aiID)
		require.Error(t, err)
		require.Nil(t, res)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete non-existent AI model", func(t *testing.T) {
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

func TestAIModelsCollection_GetAll(t *testing.T) {
	collection := aiModelsCollection()

	t.Run("get all from empty collection", func(t *testing.T) {
		res, err := collection.GetAll()
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("get all from non-empty collection", func(t *testing.T) {
		amds := []AIModel{
			createTestAIModel("ai-id-1", "ai-name-1"),
			createTestAIModel("ai-id-2", "ai-name-2"),
			createTestAIModel("ai-id-3", "ai-name-3"),
		}

		for _, amd := range amds {
			err := collection.Add(amd)
			require.NoError(t, err, "error adding AI model")
		}

		res, err := collection.GetAll()
		require.NoError(t, err, "error in getting all AI models")
		assert.Len(t, res, len(amds))
		assert.IsType(t, []*AIModel{}, res)

		amdMap := make(map[string]bool)
		for _, a := range res {
			amdMap[*a.ID] = true
		}
		for _, amd := range amds {
			assert.True(t, amdMap[*amd.ID], "AIModel with ID %s not found in results", *amd.ID)
		}
	})
}
