package indexers

import (
	"crypto/md5"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMD5FieldsIndexer(t *testing.T) {
	assert := assert.New(t)

	type Foo struct {
		Bar *string
		Baz *string
	}

	in := &MD5FieldsIndexer{
		Fields: []string{"Bar", "Baz"},
	}
	s1 := "yolo"
	s2 := "oloy"
	b := Foo{
		Bar: &s1,
		Baz: &s2,
	}

	ok, val, err := in.FromObject(b)
	assert.True(ok)
	require.NoError(t, err)
	sum := md5.Sum([]byte(s1 + s2))
	assert.Equal(sum[:], val)

	val, err = in.FromArgs(s1, s2)
	require.NoError(t, err)
	assert.Equal(sum[:], val)

	ok, val, err = in.FromObject(Foo{})
	assert.False(ok)
	require.Error(t, err)
	assert.Empty(val)

	s1 = ""
	s2 = ""
	ok, val, err = in.FromObject(Foo{
		Bar: &s1,
		Baz: &s2,
	})
	assert.False(ok)
	require.NoError(t, err)
	assert.Empty(val)

	val, err = in.FromArgs("")
	require.Error(t, err)
	assert.Nil(val)

	val, err = in.FromArgs(2)
	require.Error(t, err)
	assert.Nil(val)
}
