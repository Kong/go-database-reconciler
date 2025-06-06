package state

import (
	"reflect"
	"sort"
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

// getTags returns a slice of test tags. If reversed is true, the tags are backwards!
// backwards tag slices are useful for confirming that our equality checks ignore tag order
func getTags(reversed bool) []*string {
	fooString := "foo"
	barString := "bar"
	if reversed {
		return []*string{&barString, &fooString}
	}
	return []*string{&fooString, &barString}
}

// getProtocols returns a slice of test protocols. If reversed is true, the protocols are backwards!
// backwards protocol slices are useful for confirming that our equality checks ignore protocol order
func getProtocols(reversed bool) []*string {
	httpString := "http"
	httpsString := "https"
	if reversed {
		return []*string{&httpsString, &httpString}
	}
	return []*string{&httpString, &httpsString}
}

func getCACertificates(reversed bool) []*string {
	ca1 := "ca1"
	ca2 := "ca2"
	ca3 := "ca3"

	if reversed {
		return []*string{&ca3, &ca2, &ca1}
	}
	return []*string{&ca1, &ca2, &ca3}
}

func TestMeta(t *testing.T) {
	assert := assert.New(t)

	var m Meta

	m.AddMeta("foo", "bar")
	r := m.GetMeta("foo")
	res, ok := r.(string)
	assert.True(ok)
	assert.Equal("bar", res)
	// assert.Equal(reflect.TypeOf(r).String(), "string")

	s := "string-pointer"
	m.AddMeta("baz", &s)
	r = m.GetMeta("baz")
	res2, ok := r.(*string)
	assert.True(ok)
	assert.Equal("string-pointer", *res2)

	// can retrieve a previous value
	r = m.GetMeta("foo")
	res, ok = r.(string)
	assert.True(ok)
	assert.Equal("bar", res)
}

func TestServiceEqual(t *testing.T) {
	assert := assert.New(t)

	var s1, s2 Service
	s1.ID = kong.String("foo")
	s1.Name = kong.String("bar")

	s2.ID = kong.String("foo")
	s2.Name = kong.String("baz")

	assert.False(s1.Equal(&s2))
	assert.False(s1.EqualWithOpts(&s2, false, false))

	s2.Name = kong.String("bar")
	assert.True(s1.Equal(&s2))
	assert.True(s1.EqualWithOpts(&s2, false, false))
	s1.Tags = getTags(true)
	s2.Tags = getTags(false)
	assert.True(s1.EqualWithOpts(&s2, false, false))

	s1.ID = kong.String("fuu")
	assert.False(s1.EqualWithOpts(&s2, false, false))
	assert.True(s1.EqualWithOpts(&s2, true, false))

	s2.CreatedAt = kong.Int(1)
	s1.UpdatedAt = kong.Int(2)
	assert.False(s1.EqualWithOpts(&s2, false, false))
	assert.False(s1.EqualWithOpts(&s2, false, true))

	s1.CACertificates = getCACertificates(false)
	s2.CACertificates = getCACertificates(true)
	assert.True(s1.EqualWithOpts(&s2, true, true))
}

func TestRouteEqual(t *testing.T) {
	assert := assert.New(t)

	var r1, r2 Route
	r1.ID = kong.String("foo")
	r1.Name = kong.String("bar")

	r2.ID = kong.String("foo")
	r2.Name = kong.String("baz")

	assert.False(r1.Equal(&r2))
	assert.False(r1.EqualWithOpts(&r2, false, false, false))

	r2.Name = kong.String("bar")
	assert.True(r1.Equal(&r2))
	assert.True(r1.EqualWithOpts(&r2, false, false, false))
	r1.Tags = getTags(true)
	r2.Tags = getTags(false)
	assert.True(r1.EqualWithOpts(&r2, false, false, false))

	r1.ID = kong.String("fuu")
	assert.False(r1.EqualWithOpts(&r2, false, false, false))
	assert.True(r1.EqualWithOpts(&r2, true, false, false))

	r2.CreatedAt = kong.Int(1)
	r1.UpdatedAt = kong.Int(2)
	assert.False(r1.EqualWithOpts(&r2, false, false, false))
	assert.False(r1.EqualWithOpts(&r2, false, true, false))
	assert.True(r1.EqualWithOpts(&r2, true, true, false))

	r1.Hosts = kong.StringSlice("demo1.example.com", "demo2.example.com")

	// order matters
	r2.Hosts = kong.StringSlice("demo2.example.com", "demo1.example.com")
	assert.False(r1.EqualWithOpts(&r2, true, true, false))

	r2.Hosts = kong.StringSlice("demo1.example.com", "demo2.example.com")
	assert.True(r1.EqualWithOpts(&r2, true, true, false))

	r1.Service = &kong.Service{ID: kong.String("1")}
	r2.Service = &kong.Service{ID: kong.String("2")}
	assert.False(r1.EqualWithOpts(&r2, true, true, false))
	assert.True(r1.EqualWithOpts(&r2, true, true, true))

	r1.Service = &kong.Service{ID: kong.String("2")}
	assert.True(r1.EqualWithOpts(&r2, true, true, false))
}

func TestUpstreamEqual(t *testing.T) {
	assert := assert.New(t)

	var u1, u2 Upstream
	u1.ID = kong.String("foo")
	u1.Name = kong.String("bar")

	u2.ID = kong.String("foo")
	u2.Name = kong.String("baz")

	assert.False(u1.Equal(&u2))
	assert.False(u1.EqualWithOpts(&u2, false, false))

	u2.Name = kong.String("bar")
	assert.True(u1.Equal(&u2))
	assert.True(u1.EqualWithOpts(&u2, false, false))
	u1.Tags = getTags(true)
	u2.Tags = getTags(false)
	assert.True(u1.EqualWithOpts(&u2, false, false))

	u1.ID = kong.String("fuu")
	assert.False(u1.EqualWithOpts(&u2, false, false))
	assert.True(u1.EqualWithOpts(&u2, true, false))

	var timestamp int64 = 1
	u2.CreatedAt = &timestamp
	assert.False(u1.EqualWithOpts(&u2, false, false))
	assert.False(u1.EqualWithOpts(&u2, false, true))
}

func TestTargetEqual(t *testing.T) {
	assert := assert.New(t)

	var t1, t2 Target
	t1.ID = kong.String("foo")
	t1.Target.Target = kong.String("bar")

	t2.ID = kong.String("foo")
	t2.Target.Target = kong.String("baz")

	assert.False(t1.Equal(&t2))
	assert.False(t1.EqualWithOpts(&t2, false, false, false))

	t2.Target.Target = kong.String("bar")
	assert.True(t1.Equal(&t2))
	assert.True(t1.EqualWithOpts(&t2, false, false, false))
	t1.Tags = getTags(true)
	t2.Tags = getTags(false)
	assert.True(t1.EqualWithOpts(&t2, false, false, false))

	t1.ID = kong.String("fuu")
	assert.False(t1.EqualWithOpts(&t2, false, false, false))
	assert.True(t1.EqualWithOpts(&t2, true, false, false))

	var timestamp float64 = 1
	t2.CreatedAt = &timestamp
	assert.False(t1.EqualWithOpts(&t2, false, false, false))
	assert.False(t1.EqualWithOpts(&t2, false, true, false))

	t1.Upstream = &kong.Upstream{ID: kong.String("1")}
	t2.Upstream = &kong.Upstream{ID: kong.String("2")}
	assert.False(t1.EqualWithOpts(&t2, true, true, false))
	assert.True(t1.EqualWithOpts(&t2, true, true, true))

	t1.Upstream = &kong.Upstream{ID: kong.String("2")}
	assert.True(t1.EqualWithOpts(&t2, true, true, false))
}

func TestCertificateEqual(t *testing.T) {
	assert := assert.New(t)

	var c1, c2 Certificate
	c1.ID = kong.String("foo")
	c1.Cert = kong.String("certfoo")
	c1.Key = kong.String("keyfoo")

	c2.ID = kong.String("foo")
	c2.Cert = kong.String("certfoo")
	c2.Key = kong.String("keyfoo-unequal")

	assert.False(c1.Equal(&c2))
	assert.False(c1.EqualWithOpts(&c2, false, false))

	c2.Key = kong.String("keyfoo")
	assert.True(c1.Equal(&c2))
	assert.True(c1.EqualWithOpts(&c2, false, false))
	c1.Tags = getTags(true)
	c2.Tags = getTags(false)
	assert.True(c1.EqualWithOpts(&c2, false, false))

	c1.ID = kong.String("fuu")
	assert.False(c1.EqualWithOpts(&c2, false, false))
	assert.True(c1.EqualWithOpts(&c2, true, false))

	var timestamp int64 = 1
	c2.CreatedAt = &timestamp
	assert.False(c1.EqualWithOpts(&c2, false, false))
	assert.False(c1.EqualWithOpts(&c2, false, true))
}

func TestSNIEqual(t *testing.T) {
	assert := assert.New(t)

	var s1, s2 SNI
	s1.ID = kong.String("foo")
	s1.Name = kong.String("bar")

	s2.ID = kong.String("foo")
	s2.Name = kong.String("baz")

	assert.False(s1.Equal(&s2))
	assert.False(s1.EqualWithOpts(&s2, false, false, false))

	s2.Name = kong.String("bar")
	assert.True(s1.Equal(&s2))
	assert.True(s1.EqualWithOpts(&s2, false, false, false))
	s1.Tags = getTags(true)
	s2.Tags = getTags(false)
	assert.True(s1.EqualWithOpts(&s2, false, false, false))

	s1.ID = kong.String("fuu")
	assert.False(s1.EqualWithOpts(&s2, false, false, false))
	assert.True(s1.EqualWithOpts(&s2, true, false, false))

	var timestamp int64 = 1
	s2.CreatedAt = &timestamp
	assert.False(s1.EqualWithOpts(&s2, false, false, false))
	assert.False(s1.EqualWithOpts(&s2, false, true, false))

	s1.Certificate = &kong.Certificate{ID: kong.String("1")}
	s2.Certificate = &kong.Certificate{ID: kong.String("2")}
	assert.False(s1.EqualWithOpts(&s2, true, true, false))
	assert.True(s1.EqualWithOpts(&s2, true, true, true))

	s1.Certificate = &kong.Certificate{ID: kong.String("2")}
	assert.True(s1.EqualWithOpts(&s2, true, true, false))
}

func TestPluginEqual(t *testing.T) {
	assert := assert.New(t)

	var p1, p2 Plugin
	p1.ID = kong.String("foo")
	p1.Name = kong.String("bar")

	p2.ID = kong.String("foo")
	p2.Name = kong.String("baz")

	assert.False(p1.Equal(&p2))
	assert.False(p1.EqualWithOpts(&p2, false, false, false))

	p2.Name = kong.String("bar")
	assert.True(p1.Equal(&p2))
	assert.True(p1.EqualWithOpts(&p2, false, false, false))
	p1.Tags = getTags(true)
	p2.Tags = getTags(false)
	assert.True(p1.EqualWithOpts(&p2, false, false, false))

	// Verify that plugins are equal even if protocols are out of order
	p1.Protocols = getProtocols(true)
	p2.Protocols = getProtocols(false)
	assert.True(p1.EqualWithOpts(&p2, false, false, false))

	p1.ID = kong.String("fuu")
	assert.False(p1.EqualWithOpts(&p2, false, false, false))
	assert.True(p1.EqualWithOpts(&p2, true, false, false))

	timestamp := 1
	p2.CreatedAt = &timestamp
	assert.False(p1.EqualWithOpts(&p2, false, false, false))
	assert.False(p1.EqualWithOpts(&p2, false, true, false))

	p1.Service = &kong.Service{ID: kong.String("1")}
	p2.Service = &kong.Service{ID: kong.String("2")}
	assert.False(p1.EqualWithOpts(&p2, true, true, false))
	assert.True(p1.EqualWithOpts(&p2, true, true, true))

	p1.Service = &kong.Service{ID: kong.String("2")}
	assert.True(p1.EqualWithOpts(&p2, true, true, false))

	p1.Config = kong.Configuration{"foo": "bar"}
	p2.Config = kong.Configuration{"foo": "bar"}
	assert.True(p1.EqualWithOpts(&p2, true, true, false))

	p2.Config = kong.Configuration{"foo": "baz"}
	assert.False(p1.EqualWithOpts(&p2, true, true, false))

	p1.Config = kong.Configuration{"foo": []interface{}{"b", "a", "c"}, "bar": "baz"}
	p2.Config = kong.Configuration{"foo": []interface{}{"a", "b", "c"}, "bar": "baz"}
	assert.True(p1.EqualWithOpts(&p2, true, true, false))

	p2.Config = kong.Configuration{"foo": []interface{}{"a", "c", "b"}, "bar": "baz"}
	assert.True(p1.EqualWithOpts(&p2, true, true, false))

	p2.Config = kong.Configuration{"foo": []interface{}{"a", "c", "b"}, "bar": "baz"}
	assert.True(p1.EqualWithOpts(&p2, true, true, false))

	p2.Config = kong.Configuration{"foo": []interface{}{"a", "c", "b"}, "bar": "bar"}
	assert.False(p1.EqualWithOpts(&p2, true, true, false))

	p1.Config = kong.Configuration{
		"foo": []interface{}{"b", "a", "c"},
		"bar": "baz",
		"nested": map[string]interface{}{
			"key1": []interface{}{"b", "a", "c"},
		},
	}
	p2.Config = kong.Configuration{
		"foo": []interface{}{"a", "b", "c"},
		"bar": "baz",
		"nested": map[string]interface{}{
			"key1": []interface{}{"a", "b", "c"},
		},
	}
	assert.True(p1.EqualWithOpts(&p2, true, true, false))

	p2.Config = kong.Configuration{
		"foo": []interface{}{"a", "c", "c"},
		"bar": "baz",
		"nested": map[string]interface{}{
			"key1": []interface{}{"a", "b", "c"},
		},
	}
	assert.False(p1.EqualWithOpts(&p2, true, true, false))
}

func TestConsumerEqual(t *testing.T) {
	assert := assert.New(t)

	var c1, c2 Consumer
	c1.ID = kong.String("foo")
	c1.Username = kong.String("bar")

	c2.ID = kong.String("foo")
	c2.Username = kong.String("baz")

	assert.False(c1.Equal(&c2))
	assert.False(c1.EqualWithOpts(&c2, false, false))

	c2.Username = kong.String("bar")
	assert.True(c1.Equal(&c2))
	assert.True(c1.EqualWithOpts(&c2, false, false))
	c1.Tags = getTags(true)
	c2.Tags = getTags(false)
	assert.True(c1.EqualWithOpts(&c2, false, false))

	c1.ID = kong.String("fuu")
	assert.False(c1.EqualWithOpts(&c2, false, false))
	assert.True(c1.EqualWithOpts(&c2, true, false))

	var a int64 = 1
	c2.CreatedAt = &a
	assert.False(c1.EqualWithOpts(&c2, false, false))
	assert.False(c1.EqualWithOpts(&c2, false, true))
}

func TestKeyAuthEqual(t *testing.T) {
	assert := assert.New(t)

	var k1, k2 KeyAuth
	k1.ID = kong.String("foo")
	k1.Key = kong.String("bar")

	k2.ID = kong.String("foo")
	k2.Key = kong.String("baz")

	assert.False(k1.Equal(&k2))
	assert.False(k1.EqualWithOpts(&k2, false, false, false))

	k2.Key = kong.String("bar")
	assert.True(k1.Equal(&k2))
	assert.True(k1.EqualWithOpts(&k2, false, false, false))
	k1.Tags = getTags(true)
	k2.Tags = getTags(false)
	assert.True(k1.EqualWithOpts(&k2, false, false, false))

	k1.ID = kong.String("fuu")
	assert.False(k1.EqualWithOpts(&k2, false, false, false))
	assert.True(k1.EqualWithOpts(&k2, true, false, false))

	k2.CreatedAt = kong.Int(1)
	assert.False(k1.EqualWithOpts(&k2, false, false, false))
	assert.False(k1.EqualWithOpts(&k2, false, true, false))

	k2.Consumer = &kong.Consumer{Username: kong.String("u1")}
	assert.False(k1.EqualWithOpts(&k2, false, true, false))
	assert.False(k1.EqualWithOpts(&k2, false, false, true))
}

func TestHMACAuthEqual(t *testing.T) {
	assert := assert.New(t)

	var k1, k2 HMACAuth
	k1.ID = kong.String("foo")
	k1.Username = kong.String("bar")

	k2.ID = kong.String("foo")
	k2.Username = kong.String("baz")

	assert.False(k1.Equal(&k2))
	assert.False(k1.EqualWithOpts(&k2, false, false, false))

	k2.Username = kong.String("bar")
	assert.True(k1.Equal(&k2))
	assert.True(k1.EqualWithOpts(&k2, false, false, false))
	k1.Tags = getTags(true)
	k2.Tags = getTags(false)
	assert.True(k1.EqualWithOpts(&k2, false, false, false))

	k1.ID = kong.String("fuu")
	assert.False(k1.EqualWithOpts(&k2, false, false, false))
	assert.True(k1.EqualWithOpts(&k2, true, false, false))

	k2.CreatedAt = kong.Int(1)
	assert.False(k1.EqualWithOpts(&k2, false, false, false))
	assert.False(k1.EqualWithOpts(&k2, false, true, false))

	k2.Consumer = &kong.Consumer{Username: kong.String("u1")}
	assert.False(k1.EqualWithOpts(&k2, false, true, false))
	assert.False(k1.EqualWithOpts(&k2, false, false, true))
}

func TestJWTAuthEqual(t *testing.T) {
	assert := assert.New(t)

	var k1, k2 JWTAuth
	k1.ID = kong.String("foo")
	k1.Key = kong.String("bar")

	k2.ID = kong.String("foo")
	k2.Key = kong.String("baz")

	assert.False(k1.Equal(&k2))
	assert.False(k1.EqualWithOpts(&k2, false, false, false))

	k2.Key = kong.String("bar")
	assert.True(k1.Equal(&k2))
	assert.True(k1.EqualWithOpts(&k2, false, false, false))
	k1.Tags = getTags(true)
	k2.Tags = getTags(false)
	assert.True(k1.EqualWithOpts(&k2, false, false, false))

	k1.ID = kong.String("fuu")
	assert.False(k1.EqualWithOpts(&k2, false, false, false))
	assert.True(k1.EqualWithOpts(&k2, true, false, false))

	k2.CreatedAt = kong.Int(1)
	assert.False(k1.EqualWithOpts(&k2, false, false, false))
	assert.False(k1.EqualWithOpts(&k2, false, true, false))

	k2.Consumer = &kong.Consumer{Username: kong.String("u1")}
	assert.False(k1.EqualWithOpts(&k2, false, true, false))
	assert.False(k1.EqualWithOpts(&k2, false, false, true))
}

func TestBasicAuthEqual(t *testing.T) {
	assert := assert.New(t)

	var k1, k2 BasicAuth
	k1.ID = kong.String("foo")
	k1.Password = kong.String("bar")

	k2.ID = kong.String("foo")
	k2.Password = kong.String("baz")

	assert.False(k1.Equal(&k2))
	assert.False(k1.EqualWithOpts(&k2, false, false, false, false))

	k2.Password = kong.String("bar")
	assert.True(k1.Equal(&k2))
	assert.True(k1.EqualWithOpts(&k2, false, false, false, false))
	assert.True(k1.EqualWithOpts(&k2, false, false, false, true))
	k1.Tags = getTags(true)
	k2.Tags = getTags(false)
	assert.True(k1.EqualWithOpts(&k2, false, false, false, false))

	k1.ID = kong.String("fuu")
	assert.False(k1.EqualWithOpts(&k2, false, false, false, false))
	assert.True(k1.EqualWithOpts(&k2, true, false, false, false))

	k2.CreatedAt = kong.Int(1)
	assert.False(k1.EqualWithOpts(&k2, false, false, false, false))
	assert.False(k1.EqualWithOpts(&k2, false, true, false, false))

	k2.Consumer = &kong.Consumer{Username: kong.String("u1")}
	assert.False(k1.EqualWithOpts(&k2, false, true, false, false))
	assert.False(k1.EqualWithOpts(&k2, false, false, true, false))
}

func TestACLGroupEqual(t *testing.T) {
	assert := assert.New(t)

	var k1, k2 ACLGroup
	k1.ID = kong.String("foo")
	k1.Group = kong.String("bar")

	k2.ID = kong.String("foo")
	k2.Group = kong.String("baz")

	assert.False(k1.Equal(&k2))
	assert.False(k1.EqualWithOpts(&k2, false, false, false))

	k2.Group = kong.String("bar")
	assert.True(k1.Equal(&k2))
	assert.True(k1.EqualWithOpts(&k2, false, false, false))
	k1.Tags = getTags(true)
	k2.Tags = getTags(false)
	assert.True(k1.EqualWithOpts(&k2, false, false, false))

	k1.ID = kong.String("fuu")
	assert.False(k1.EqualWithOpts(&k2, false, false, false))
	assert.True(k1.EqualWithOpts(&k2, true, false, false))

	k2.CreatedAt = kong.Int(1)
	assert.False(k1.EqualWithOpts(&k2, false, false, false))
	assert.False(k1.EqualWithOpts(&k2, false, true, false))

	k2.Consumer = &kong.Consumer{Username: kong.String("u1")}
	assert.False(k1.EqualWithOpts(&k2, false, true, false))
	assert.False(k1.EqualWithOpts(&k2, false, false, true))
}

func TestCACertificateEqual(t *testing.T) {
	assert := assert.New(t)

	var c1, c2 CACertificate
	c1.ID = kong.String("cert")
	c1.Cert = kong.String("==== cert")

	c2.ID = kong.String("cert")
	c2.Cert = kong.String("==== cert")

	assert.True(c1.Equal(&c2))

	c2.Cert = kong.String("==== another cert")
	assert.False(c1.Equal(&c2), "should distinguish cert")
	c2.Cert = kong.String("==== cert")

	c2.CertDigest = kong.String("digest")
	assert.True(c1.Equal(&c2), "should ignore cert_digest")
	c2.CertDigest = nil

	c2.ID = kong.String("another")
	assert.False(c1.Equal(&c2), "should distinguish ID")
	assert.True(c1.EqualWithOpts(&c2, true, false), "should ignore ID when configured")

	c2.ID = kong.String("cert")
	c2.CreatedAt = lo.ToPtr(int64(1))
	assert.False(c1.Equal(&c2), "should distinguish createdAt")
	assert.True(c1.EqualWithOpts(&c2, false, true), "should ignore createdAt when configured")

	c2.CreatedAt = nil
	c2.Tags = lo.ToSlicePtr([]string{"tag1", "tag2"})
	assert.False(c1.Equal(&c2), "should distinguish tags")
}

func TestStripKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("hello", stripKey("hello"))
	assert.Equal("yolo", stripKey("yolo"))
	assert.Equal("world", stripKey("hello world"))
}

func TestSortInterfaceSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			name:     "integers",
			input:    []interface{}{3, 1, 2},
			expected: []interface{}{1, 2, 3},
		},
		{
			name:     "strings",
			input:    []interface{}{"b", "c", "a"},
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name:     "mixed types",
			input:    []interface{}{"b", 2, "a", 1},
			expected: []interface{}{1, 2, "a", "b"},
		},
		{
			name:     "floats",
			input:    []interface{}{2.2, 3.3, 1.1},
			expected: []interface{}{1.1, 2.2, 3.3},
		},
		{
			name:     "maps",
			input:    []interface{}{map[string]interface{}{"key": "b"}, map[string]interface{}{"key": "a"}},
			expected: []interface{}{map[string]interface{}{"key": "a"}, map[string]interface{}{"key": "b"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort.Sort(EmptyInterfaceUsingUnderlyingType(tt.input))
			if !reflect.DeepEqual(tt.input, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, tt.input)
			}
		})
	}
}

func TestSortNestedArrays(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "single level",
			input: map[string]interface{}{
				"key1": []interface{}{"b", "a", "c"},
			},
			expected: map[string]interface{}{
				"key1": []interface{}{"a", "b", "c"},
			},
		},
		{
			name: "nested map",
			input: map[string]interface{}{
				"key1": []interface{}{"b", "a", "c"},
				"key2": map[string]interface{}{
					"nestedKey1": []interface{}{3, 1, 2},
				},
				"key3": []map[string]interface{}{
					{"nestedKey1": map[string]interface{}{"key": "b", "key2": "a"}},
					{"nestedKey2": map[string]interface{}{"key": "a", "key2": "b"}},
				},
			},
			expected: map[string]interface{}{
				"key1": []interface{}{"a", "b", "c"},
				"key2": map[string]interface{}{
					"nestedKey1": []interface{}{1, 2, 3},
				},
				"key3": []map[string]interface{}{
					{"nestedKey1": map[string]interface{}{"key": "b", "key2": "a"}},
					{"nestedKey2": map[string]interface{}{"key": "a", "key2": "b"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortNestedArrays(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDeepEqualWithSorting(t *testing.T) {
	map1 := map[string]interface{}{
		"key1": []interface{}{"a", "c", "b"},
		"key2": map[string]interface{}{
			"nestedKey1": []interface{}{3, 1, 2},
		},
	}

	map2 := map[string]interface{}{
		"key1": []interface{}{"a", "b", "c"},
		"key2": map[string]interface{}{
			"nestedKey1": []interface{}{1, 2, 3},
		},
	}

	sortedMap1 := sortNestedArrays(map1)
	sortedMap2 := sortNestedArrays(map2)

	if !reflect.DeepEqual(sortedMap1, sortedMap2) {
		t.Errorf("expected maps to be equal, but they are not")
	}
}

func TestPluginConsole(t *testing.T) {
	tests := []struct {
		plugin   kong.Plugin
		name     string
		expected string
	}{
		{
			name:     "plugin default case",
			plugin:   kong.Plugin{},
			expected: "foo-plugin (global)",
		},
		{
			name: "plugin associated with service",
			plugin: kong.Plugin{
				Service: &kong.Service{ID: kong.String("bar")},
			},
			expected: "foo-plugin for service bar",
		},
		{
			name: "plugin associated with route",
			plugin: kong.Plugin{
				Route: &kong.Route{ID: kong.String("baz")},
			},
			expected: "foo-plugin for route baz",
		},
		{
			name: "plugin associated with consumer",
			plugin: kong.Plugin{
				Consumer: &kong.Consumer{ID: kong.String("demo")},
			},
			expected: "foo-plugin for consumer demo",
		},
		{
			name: "plugin associated with consumer group",
			plugin: kong.Plugin{
				ConsumerGroup: &kong.ConsumerGroup{ID: kong.String("demo-group")},
			},
			expected: "foo-plugin for consumer-group demo-group",
		},
		{
			name: "plugin associated with >1 entities",
			plugin: kong.Plugin{
				Service:       &kong.Service{ID: kong.String("bar")},
				Route:         &kong.Route{ID: kong.String("baz")},
				Consumer:      &kong.Consumer{ID: kong.String("demo")},
				ConsumerGroup: &kong.ConsumerGroup{ID: kong.String("demo-group")},
			},
			expected: "foo-plugin for service bar and route baz and consumer demo and consumer-group demo-group",
		},
	}
	for _, tt := range tests {
		var p1 Plugin
		p1.Plugin = tt.plugin
		p1.ID = kong.String("foo")
		p1.Name = kong.String("foo-plugin")

		t.Run(tt.name, func(t *testing.T) {
			actual := p1.Console()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
