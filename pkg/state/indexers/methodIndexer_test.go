package indexers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Foo struct {
	id string
}

func (f Foo) ID() string {
	return f.id
}

func (f Foo) BadID() (string, error) {
	return f.id, nil
}

type ID interface {
	ID() string
}

func TestMethodIndexer(t *testing.T) {
	assert := assert.New(t)

	in := &MethodIndexer{
		Method: "ID",
	}
	b := Foo{
		id: "id1",
	}

	ok, val, err := in.FromObject(b)
	assert.True(ok)
	require.NoError(t, err)
	assert.Equal([]byte("id1"), val)

	ok, val, err = in.FromObject(Foo{})
	assert.False(ok)
	require.NoError(t, err)
	assert.Empty(val)

	idInterface := (ID)(b)
	ok, val, err = in.FromObject(idInterface)
	assert.True(ok)
	require.NoError(t, err)
	assert.Equal([]byte("id1"), val)

	val, err = in.FromArgs("id1")
	require.NoError(t, err)
	assert.Equal([]byte("id1"), val)

	val, err = in.FromArgs("")
	require.Error(t, err)
	assert.Nil(val)

	val, err = in.FromArgs(42)
	require.Error(t, err)
	assert.Nil(val)

	in = &MethodIndexer{
		Method: "BadID",
	}

	ok, val, err = in.FromObject(Foo{id: "id1"})
	assert.False(ok)
	require.Error(t, err)
	assert.Empty(val)
}
