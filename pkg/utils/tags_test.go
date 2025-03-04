package utils

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeTags(t *testing.T) {
	type Foo struct {
		Tags []*string
	}
	type Bar struct{}

	assert := assert.New(t)

	a := "tag1"
	b := "tag2"
	c := "tag3"

	var f Foo
	err := MergeTags(f, []string{"tag1"})
	require.Error(t, err)

	assert.Panics(func() {
		MustMergeTags(f, []string{"tag1"})
	})

	var bar Bar
	err = MergeTags(&bar, []string{"tag1"})
	require.NoError(t, err)

	f = Foo{Tags: []*string{&a, &b}}
	require.NoError(t, MergeTags(&f, []string{"tag1", "tag2", "tag3"}))
	assert.True(equalArray([]*string{&a, &b, &c}, f.Tags))

	f = Foo{Tags: []*string{}}
	require.NoError(t, MergeTags(&f, []string{"tag1", "tag2", "tag3"}))
	assert.True(equalArray([]*string{&a, &b, &c}, f.Tags))

	f = Foo{Tags: []*string{&a, &b}}
	require.NoError(t, MergeTags(&f, nil))
	assert.True(equalArray([]*string{&a, &b}, f.Tags))
}

func equalArray(want, have []*string) bool {
	if len(want) != len(have) {
		return false
	}
	for i := 0; i < len(want); i++ {
		if *want[i] != *have[i] {
			return false
		}
	}
	return true
}

func TestRemoveTags(t *testing.T) {
	type Foo struct {
		Tags []*string
	}
	type Bar struct{}

	assert := assert.New(t)

	a := "tag1"
	b := "tag2"

	var f Foo
	err := RemoveTags(f, []string{"tag1"})
	require.Error(t, err)

	assert.Panics(func() {
		MustRemoveTags(f, []string{"tag1"})
	})

	var bar Bar
	err = RemoveTags(&bar, []string{"tag1"})
	require.NoError(t, err)

	f = Foo{Tags: []*string{&a, &b}}
	RemoveTags(&f, []string{"tag2", "tag3"})
	assert.True(equalArray([]*string{&a}, f.Tags))

	f = Foo{Tags: []*string{}}
	RemoveTags(&f, []string{"tag1", "tag2", "tag3"})
	assert.True(equalArray([]*string{}, f.Tags))

	f = Foo{Tags: []*string{&a, &b}}
	RemoveTags(&f, nil)
	assert.True(equalArray([]*string{&a, &b}, f.Tags))
}

func TestHasTags(t *testing.T) {
	assert := assert.New(t)

	assert.False(HasTags(&kong.Consumer{}, []string{"tag1"}))

	consumer := &kong.Consumer{
		Tags: []*string{
			kong.String("tag1"),
			kong.String("tag2"),
		},
	}
	assert.True(HasTags(consumer, []string{"tag1"}))
	assert.True(HasTags(consumer, []string{"tag1", "tag2"}))
	assert.True(HasTags(consumer, []string{"tag1", "tag2", "tag3"}))
	assert.False(HasTags(consumer, []string{"tag3"}))
}

func BenchmarkHasTags(b *testing.B) {
	consumer := &kong.Consumer{
		Tags: []*string{
			kong.String("tag1"),
			kong.String("tag2"),
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HasTags(consumer, []string{"tag1", "tag2", "tag3"})
	}
}
