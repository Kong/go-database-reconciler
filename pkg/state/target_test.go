package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func targetsCollection() *TargetsCollection {
	return state().Targets
}

func TestTargetInsert(t *testing.T) {
	collection := targetsCollection()

	var t0 Target
	t0.Target.Target = new("my-target")
	err := collection.Add(t0)
	require.Error(t, err)

	t0.ID = new("first")
	err = collection.Add(t0)
	require.Error(t, err)

	var t1 Target
	t1.Target.Target = new("my-target")
	t1.ID = new("first")
	t1.Upstream = &kong.Upstream{
		ID: new("upstream1-id"),
	}
	err = collection.Add(t1)
	require.NoError(t, err)

	var t2 Target
	t2.Target.Target = new("my-target")
	t2.ID = new("second")
	t2.Upstream = &kong.Upstream{
		ID: new("upstream1-id"),
	}
	err = collection.Add(t2)
	require.Error(t, err)

	var t3 Target
	t3.Target.Target = new("my-target")
	t3.ID = new("third")
	t3.Upstream = &kong.Upstream{
		Name: new("upstream1-id"),
	}
	err = collection.Add(t3)
	require.Error(t, err)
}

func TestTargetGetUpdate(t *testing.T) {
	assert := assert.New(t)
	collection := targetsCollection()

	var target Target
	target.Target.Target = new("my-target")
	target.ID = new("first")
	target.Upstream = &kong.Upstream{
		ID: new("upstream1-id"),
	}
	assert.NotNil(target.Upstream)
	err := collection.Add(target)
	require.NoError(t, err)

	re, err := collection.Get("upstream1-id", "first")
	require.NoError(t, err)
	assert.NotNil(re)
	assert.Equal("my-target", *re.Target.Target)

	re.ID = nil
	re.Upstream.ID = nil
	require.Error(t, collection.Update(*re))

	re.ID = new("does-not-exist")
	require.Error(t, collection.Update(*re))

	re.ID = new("first")
	require.Error(t, collection.Update(*re))

	re.Upstream.ID = new("upstream1-id")
	require.NoError(t, collection.Update(*re))

	re.Upstream.ID = new("upstream2-id")
	require.NoError(t, collection.Update(*re))
}

// Regression test
// to ensure that the memory reference of the pointer returned by Get()
// is different from the one stored in MemDB.
func TestTargetGetMemoryReference(t *testing.T) {
	assert := assert.New(t)
	collection := targetsCollection()

	var target Target
	target.Target.Target = new("my-target")
	target.ID = new("first")
	target.Upstream = &kong.Upstream{
		ID: new("upstream1-id"),
	}
	err := collection.Add(target)
	require.NoError(t, err)

	re, err := collection.Get("upstream1-id", "first")
	require.NoError(t, err)
	assert.NotNil(re)
	assert.Equal("my-target", *re.Target.Target)

	re.Weight = new(1)

	re, err = collection.Get("upstream1-id", "my-target")
	require.NoError(t, err)
	assert.NotNil(re)
	assert.Nil(re.Weight)
}

func TestTargetsInvalidType(t *testing.T) {
	assert := assert.New(t)

	collection := targetsCollection()

	type badTarget struct {
		kong.Target
		Meta
	}

	target := badTarget{
		Target: kong.Target{
			ID:     new("id"),
			Target: new("target"),
			Upstream: &kong.Upstream{
				ID: new("upstream-id"),
			},
		},
	}

	txn := collection.db.Txn(true)
	err := txn.Insert(targetTableName, &target)
	require.NoError(t, err)
	txn.Commit()

	assert.Panics(func() {
		collection.Get("upstream-id", "id")
	})

	assert.Panics(func() {
		collection.GetAll()
	})
}

func TestTargetDelete(t *testing.T) {
	assert := assert.New(t)
	collection := targetsCollection()

	var target Target
	target.Target.Target = new("my-target")
	target.ID = new("first")
	target.Upstream = &kong.Upstream{
		ID: new("upstream1-id"),
	}
	err := collection.Add(target)
	require.NoError(t, err)

	re, err := collection.Get("upstream1-id", "my-target")
	require.NoError(t, err)
	assert.NotNil(re)

	err = collection.Delete("upstream1-id", *re.ID)
	require.NoError(t, err)

	err = collection.Delete("upstream1-id", *re.ID)
	require.Error(t, err)

	err = collection.Delete("", "first")
	require.Error(t, err)

	err = collection.Delete("foo", "")
	require.Error(t, err)
}

func TestTargetGetAll(t *testing.T) {
	assert := assert.New(t)
	collection := targetsCollection()

	var target Target
	target.Target.Target = new("my-target1")
	target.ID = new("first")
	target.Upstream = &kong.Upstream{
		ID: new("upstream1-id"),
	}
	err := collection.Add(target)
	require.NoError(t, err)

	var target2 Target
	target2.Target.Target = new("my-target2")
	target2.ID = new("second")
	target2.Upstream = &kong.Upstream{
		ID: new("upstream1-id"),
	}
	err = collection.Add(target2)
	require.NoError(t, err)

	targets, err := collection.GetAll()

	require.NoError(t, err)
	assert.Len(targets, 2)
}

func TestTargetGetAllByUpstreamName(t *testing.T) {
	assert := assert.New(t)
	collection := targetsCollection()

	targets := []*Target{
		{
			Target: kong.Target{
				ID:     new("target1-id"),
				Target: new("target1-name"),
				Upstream: &kong.Upstream{
					ID: new("upstream1-id"),
				},
			},
		},
		{
			Target: kong.Target{
				ID:     new("target2-id"),
				Target: new("target2-name"),
				Upstream: &kong.Upstream{
					ID: new("upstream1-id"),
				},
			},
		},
		{
			Target: kong.Target{
				ID:     new("target3-id"),
				Target: new("target3-name"),
				Upstream: &kong.Upstream{
					ID: new("upstream2-id"),
				},
			},
		},
		{
			Target: kong.Target{
				ID:     new("target4-id"),
				Target: new("target4-name"),
				Upstream: &kong.Upstream{
					ID: new("upstream2-id"),
				},
			},
		},
	}

	for _, target := range targets {
		err := collection.Add(*target)
		require.NoError(t, err)
	}

	targets, err := collection.GetAllByUpstreamID("upstream1-id")
	require.NoError(t, err)
	assert.Len(targets, 2)
}
