package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func upstreamsCollection() *UpstreamsCollection {
	return state().Upstreams
}

func TestUpstreamInsert(t *testing.T) {
	collection := upstreamsCollection()

	// name is required
	var upstream Upstream
	upstream.ID = kong.String("first")
	err := collection.Add(upstream)
	require.Error(t, err)

	// happy path
	upstream.Name = kong.String("my-upstream")
	require.NoError(t, collection.Add(upstream))

	// ID is required
	var upstream2 Upstream
	upstream2.Name = kong.String("my-upstream")
	err = collection.Add(upstream2)
	require.Error(t, err)

	// re-insert
	upstream2.ID = kong.String("first")
	require.Error(t, collection.Add(upstream2))

	upstream2.ID = kong.String("same-name-but-different-id")
	require.Error(t, collection.Add(upstream2))
}

func TestUpstreamGetUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := upstreamsCollection()

	se, err := collection.Get("does-not-exist")
	require.Error(t, err)
	assert.Nil(se)

	se, err = collection.Get("")
	require.Error(t, err)
	assert.Nil(se)

	var upstream Upstream
	upstream.Name = kong.String("my-upstream")
	upstream.ID = kong.String("first")
	err = collection.Add(upstream)
	require.NoError(t, err)

	se, err = collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(se)

	se.Name = kong.String("my-updated-upstream")
	err = collection.Update(*se)
	require.NoError(t, err)

	se, err = collection.Get("my-updated-upstream")
	require.NoError(t, err)
	assert.NotNil(se)

	se.ID = nil
	err = collection.Update(*se)
	require.Error(t, err)

	se, err = collection.Get("my-upstream")
	assert.Equal(ErrNotFound, err)
	assert.Nil(se)
}

// Regression test
// to ensure that the memory reference of the pointer returned by Get()
// is different from the one stored in MemDB.
func TestUpstreamGetMemoryReference(t *testing.T) {
	assert := assert.New(t)
	collection := upstreamsCollection()

	var upstream Upstream
	upstream.Name = kong.String("my-upstream")
	upstream.ID = kong.String("first")
	err := collection.Add(upstream)
	require.NoError(t, err)

	se, err := collection.Get("first")
	require.NoError(t, err)
	assert.NotNil(se)
	se.Slots = kong.Int(1)

	se, err = collection.Get("my-upstream")
	require.NoError(t, err)
	assert.NotNil(se)
	assert.Nil(se.Slots)
}

func TestUpstreamsInvalidType(t *testing.T) {
	assert := assert.New(t)

	collection := upstreamsCollection()

	var route Route
	route.Name = kong.String("my-route")
	route.ID = kong.String("first")
	txn := collection.db.Txn(true)
	txn.Insert(upstreamTableName, &route)
	txn.Commit()

	assert.Panics(func() {
		collection.Get("my-route")
	})
	assert.Panics(func() {
		collection.GetAll()
	})
}

func TestUpstreamDelete(t *testing.T) {
	assert := assert.New(t)
	collection := upstreamsCollection()

	var upstream Upstream
	upstream.Name = kong.String("my-upstream")
	upstream.ID = kong.String("first")
	err := collection.Add(upstream)
	require.NoError(t, err)

	se, err := collection.Get("my-upstream")
	require.NoError(t, err)
	assert.NotNil(se)

	err = collection.Delete(*se.ID)
	require.NoError(t, err)

	err = collection.Delete("")
	require.Error(t, err)

	_, err = collection.Get("my-upstream")
	assert.Equal(ErrNotFound, err)

	err = collection.Delete(*se.ID)
	require.Error(t, err)
}

func TestUpstreamGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := upstreamsCollection()

	var upstream Upstream
	upstream.Name = kong.String("my-upstream1")
	upstream.ID = kong.String("first")
	err := collection.Add(upstream)
	require.NoError(t, err)

	var upstream2 Upstream
	upstream2.Name = kong.String("my-upstream2")
	upstream2.ID = kong.String("second")
	err = collection.Add(upstream2)
	require.NoError(t, err)

	upstreams, err := collection.GetAll()

	require.NoError(t, err)
	assert.Len(upstreams, 2)
}
