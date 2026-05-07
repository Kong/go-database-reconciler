package utils

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTag1 = "tag1"
	testTag2 = "tag2"
	testTag3 = "tag3"
)

func TestMergeTags(t *testing.T) {
	type Foo struct {
		Tags []*string
	}
	type Bar struct{}

	assert := assert.New(t)

	a := testTag1
	b := testTag2
	c := testTag3

	var f Foo
	err := MergeTags(f, []string{testTag1})
	require.Error(t, err)

	assert.Panics(func() {
		MustMergeTags(f, []string{testTag1})
	})

	var bar Bar
	err = MergeTags(&bar, []string{testTag1})
	require.NoError(t, err)

	f = Foo{Tags: []*string{&a, &b}}
	require.NoError(t, MergeTags(&f, []string{testTag1, testTag2, testTag3}))
	assert.True(equalArray([]*string{&a, &b, &c}, f.Tags))

	f = Foo{Tags: []*string{}}
	require.NoError(t, MergeTags(&f, []string{testTag1, testTag2, testTag3}))
	assert.True(equalArray([]*string{&a, &b, &c}, f.Tags))

	f = Foo{Tags: []*string{&a, &b}}
	require.NoError(t, MergeTags(&f, nil))
	assert.True(equalArray([]*string{&a, &b}, f.Tags))
}

func equalArray(want, have []*string) bool {
	if len(want) != len(have) {
		return false
	}
	for i := range want {
		if want[i] == nil || have[i] == nil {
			if want[i] != have[i] {
				return false
			}
			continue
		}
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

	a := testTag1
	b := testTag2

	var f Foo
	err := RemoveTags(f, []string{testTag1})
	require.Error(t, err)

	assert.Panics(func() {
		MustRemoveTags(f, []string{testTag1})
	})

	var bar Bar
	err = RemoveTags(&bar, []string{testTag1})
	require.NoError(t, err)

	f = Foo{Tags: []*string{&a, &b}}
	RemoveTags(&f, []string{testTag2, testTag3})
	assert.True(equalArray([]*string{&a}, f.Tags))

	f = Foo{Tags: []*string{}}
	RemoveTags(&f, []string{testTag1, testTag2, testTag3})
	assert.True(equalArray([]*string{}, f.Tags))

	f = Foo{Tags: []*string{&a, &b}}
	RemoveTags(&f, nil)
	assert.True(equalArray([]*string{&a, &b}, f.Tags))
}

func TestHasTags(t *testing.T) {
	assert := assert.New(t)

	assert.False(HasTags(&kong.Consumer{}, []string{testTag1}))

	consumer := &kong.Consumer{
		Tags: []*string{
			kong.String(testTag1),
			kong.String(testTag2),
		},
	}
	assert.True(HasTags(consumer, []string{testTag1}))
	assert.True(HasTags(consumer, []string{testTag1, testTag2}))
	assert.True(HasTags(consumer, []string{testTag1, testTag2, testTag3}))
	assert.False(HasTags(consumer, []string{testTag3}))
}

func BenchmarkHasTags(b *testing.B) {
	consumer := &kong.Consumer{
		Tags: []*string{
			kong.String(testTag1),
			kong.String(testTag2),
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HasTags(consumer, []string{testTag1, testTag2, testTag3})
	}
}
