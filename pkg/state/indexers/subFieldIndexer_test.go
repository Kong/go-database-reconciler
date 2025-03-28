package indexers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubFieldIndexer(t *testing.T) {
	type Foo struct {
		Bar *string
	}

	type Baz struct {
		A *Foo
	}

	in := &SubFieldIndexer{
		Fields: []Field{
			{
				Struct: "A",
				Sub:    "Bar",
			},
		},
	}
	s := "fubar"
	b := Baz{
		A: &Foo{
			Bar: &s,
		},
	}

	ok, val, err := in.FromObject(b)
	assert := assert.New(t)
	assert.True(ok)
	require.NoError(t, err)
	assert.Equal("fubar\x00", string(val))

	ok, val, err = in.FromObject(Baz{})
	assert.False(ok)
	require.NoError(t, err)
	assert.Empty(val)

	s = ""
	ok, val, err = in.FromObject(Baz{
		A: &Foo{
			Bar: &s,
		},
	})
	assert.False(ok)
	require.NoError(t, err)
	assert.Empty(val)

	val, err = in.FromArgs("fubar")
	require.NoError(t, err)
	assert.Equal("fubar\x00", string(val))

	val, err = in.FromArgs(2)
	assert.Nil(val)
	require.Error(t, err)

	val, err = in.FromArgs("1", "2")
	assert.Equal([]byte("12\x00"), val)
	require.NoError(t, err)
}

func TestSubFieldIndexerPointer(t *testing.T) {
	type Foo struct {
		Bar *string
	}

	type Baz struct {
		A *Foo
	}

	in := &SubFieldIndexer{
		Fields: []Field{
			{
				Struct: "A",
				Sub:    "Bar",
			},
		},
	}
	s := "fubar"
	b := Baz{
		A: &Foo{
			Bar: &s,
		},
	}

	ok, val, err := in.FromObject(b)
	assert := assert.New(t)
	assert.True(ok)
	require.NoError(t, err)
	assert.Equal("fubar\x00", string(val))

	val, err = in.FromArgs("fubar")
	require.NoError(t, err)
	assert.Equal("fubar\x00", string(val))
}
