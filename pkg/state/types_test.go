package state

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
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
	assert.False(p1.EqualWithOpts(&p2, false, false, false, gjson.Result{}))

	p2.Name = kong.String("bar")
	assert.True(p1.Equal(&p2))
	assert.True(p1.EqualWithOpts(&p2, false, false, false, gjson.Result{}))
	p1.Tags = getTags(true)
	p2.Tags = getTags(false)
	assert.True(p1.EqualWithOpts(&p2, false, false, false, gjson.Result{}))

	// Verify that plugins are equal even if protocols are out of order
	p1.Protocols = getProtocols(true)
	p2.Protocols = getProtocols(false)
	assert.True(p1.EqualWithOpts(&p2, false, false, false, gjson.Result{}))

	p1.ID = kong.String("fuu")
	assert.False(p1.EqualWithOpts(&p2, false, false, false, gjson.Result{}))
	assert.True(p1.EqualWithOpts(&p2, true, false, false, gjson.Result{}))
	timestamp := 1
	p2.CreatedAt = &timestamp
	assert.False(p1.EqualWithOpts(&p2, false, false, false, gjson.Result{}))
	assert.False(p1.EqualWithOpts(&p2, false, true, false, gjson.Result{}))
	p1.Service = &kong.Service{ID: kong.String("1")}
	p2.Service = &kong.Service{ID: kong.String("2")}
	assert.False(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))
	assert.True(p1.EqualWithOpts(&p2, true, true, true, gjson.Result{}))
	p1.Service = &kong.Service{ID: kong.String("2")}
	assert.True(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))

	p1.Config = kong.Configuration{"foo": "bar"}
	p2.Config = kong.Configuration{"foo": "bar"}
	assert.True(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))
	p2.Config = kong.Configuration{"foo": "baz"}
	assert.False(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))

	p1.Config = kong.Configuration{"foo": []interface{}{"b", "a", "c"}, "bar": "baz"}
	p2.Config = kong.Configuration{"foo": []interface{}{"a", "b", "c"}, "bar": "baz"}
	assert.True(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))
	p2.Config = kong.Configuration{"foo": []interface{}{"a", "c", "b"}, "bar": "baz"}
	assert.True(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))

	p2.Config = kong.Configuration{"foo": []interface{}{"a", "c", "b"}, "bar": "baz"}
	assert.True(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))

	p2.Config = kong.Configuration{"foo": []interface{}{"a", "c", "b"}, "bar": "bar"}
	assert.False(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))

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
	assert.True(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))

	p2.Config = kong.Configuration{
		"foo": []interface{}{"a", "c", "c"},
		"bar": "baz",
		"nested": map[string]interface{}{
			"key1": []interface{}{"a", "b", "c"},
		},
	}
	assert.False(p1.EqualWithOpts(&p2, true, true, false, gjson.Result{}))
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
			result := sortNestedArraysBasedOnSchema(tt.input, gjson.Result{})
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

	sortedMap1 := sortNestedArraysBasedOnSchema(map1, gjson.Result{})
	sortedMap2 := sortNestedArraysBasedOnSchema(map2, gjson.Result{})

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

func TestSortNestedArraysBasedOnSchema(t *testing.T) {
	filePath := "./fixtures/test-plugin-config.json"
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}

	gjsonRes := gjson.ParseBytes(fileBytes)
	config := gjsonRes.Get("fields.#(config).config")

	// Create a plugin with all field types from the schema
	original := Plugin{
		Plugin: kong.Plugin{
			Name: kong.String("request-transformer"),
			Config: kong.Configuration{
				"primitive_str": "test-value",
				"record_of_array_of_str": map[string]interface{}{
					"headers": []interface{}{
						"header-z",
						"header-a",
						"header-m",
					},
				},
				"map_type": map[string]interface{}{
					"key": "value",
				},
				"array_of_record": []interface{}{
					map[string]interface{}{
						"host": "host2.example.com",
						"port": float64(6380),
					},
					map[string]interface{}{
						"host": "host1.example.com",
						"port": float64(6379),
					},
				},
				"nested_array_of_array_of_str": []interface{}{
					[]interface{}{"z", "a", "m"},
					[]interface{}{"b", "y", "c"},
				},
				"nested_array_of_set_of_str": []interface{}{
					[]interface{}{"z", "a", "m"},
					[]interface{}{"b", "y", "c"},
				},
				"nested_set_of_array_of_str": []interface{}{
					[]interface{}{"z", "a", "m"},
					[]interface{}{"b", "y", "c"},
				},
				"nested_set_of_set_of_str": []interface{}{
					[]interface{}{"z", "a", "m"},
					[]interface{}{"b", "y", "c"},
				},
				"nested_array_of_record_of_array": []interface{}{
					[]interface{}{
						map[string]interface{}{"ports": []interface{}{float64(8086), float64(8081), float64(8082)}},
						map[string]interface{}{"ports": []interface{}{float64(9096), float64(9091), float64(9092)}},
					},
				},
				"nested_array_of_record_of_set": []interface{}{
					[]interface{}{
						map[string]interface{}{"ports": []interface{}{float64(8086), float64(8081), float64(8082)}},
						map[string]interface{}{"ports": []interface{}{float64(9096), float64(9091), float64(9092)}},
					},
				},
				"shorthand_record_of_set_array": map[string]interface{}{
					"set_hosts": []interface{}{
						"example.com",
						"abcgefgh.com",
						"zzz.com",
					},
					"array_hosts": []interface{}{
						"example.com",
						"abcgefgh.com",
						"zzz.com",
					},
				},
			},
		},
	}

	// Clone the plugin
	clonedPlugin := original.DeepCopy()

	// Sort the cloned config
	sortedConfig := sortNestedArraysBasedOnSchema(clonedPlugin.Config, config)

	// Verify primitive string remains unchanged
	assert.Equal(t, original.Config["primitive_str"], sortedConfig["primitive_str"])

	// Verify map_type keys remain unchanged
	mapType := sortedConfig["map_type"].(map[string]interface{})
	assert.Equal(t, original.Config["map_type"].(map[string]interface{})["key"], mapType["key"])

	// Verify array fields remain in ORIGINAL order
	recordOfArray := sortedConfig["record_of_array_of_str"].(map[string]interface{})
	headers := recordOfArray["headers"].([]interface{})
	originalRecordOfArray := original.Config["record_of_array_of_str"].(map[string]interface{})
	originalHeaders := originalRecordOfArray["headers"].([]interface{})
	assert.Equal(t, originalHeaders, headers)

	// Verify array_of_record remains in ORIGINAL order
	arrayOfRecord := sortedConfig["array_of_record"].([]interface{})
	assert.Equal(t, len(original.Config["array_of_record"].([]interface{})), len(arrayOfRecord))
	firstRecord := arrayOfRecord[0].(map[string]interface{})
	originalFirstRecord := original.Config["array_of_record"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, originalFirstRecord["host"], firstRecord["host"], "array of records should NOT be sorted")

	// Verify nested_array_of_array_of_str remains in ORIGINAL order
	nestedArrayOfArray := sortedConfig["nested_array_of_array_of_str"].([]interface{})
	originalNestedArrayOfArray := original.Config["nested_array_of_array_of_str"].([]interface{})
	assert.Equal(t, originalNestedArrayOfArray[0], nestedArrayOfArray[0], "nested arrays should NOT be sorted")
	assert.Equal(t, originalNestedArrayOfArray[1], nestedArrayOfArray[1], "nested arrays should NOT be sorted")

	// Verify nested_array_of_set_of_str - outer array NOT sorted, inner sets SHOULD be sorted
	nestedArrayOfSet := sortedConfig["nested_array_of_set_of_str"].([]interface{})
	originalNestedArrayOfSet := original.Config["nested_array_of_set_of_str"].([]interface{})
	sort.Sort(EmptyInterfaceUsingUnderlyingType(originalNestedArrayOfSet[0].([]interface{})))
	sort.Sort(EmptyInterfaceUsingUnderlyingType(originalNestedArrayOfSet[1].([]interface{})))
	assert.Equal(t, originalNestedArrayOfSet[0], nestedArrayOfSet[0], "inner sets should be sorted")
	assert.Equal(t, originalNestedArrayOfSet[1], nestedArrayOfSet[1], "inner sets should be sorted")

	// Verify nested_set_of_array_of_str - outer set SHOULD be sorted, inner arrays NOT sorted
	nestedSetOfArray := sortedConfig["nested_set_of_array_of_str"].([]interface{})
	// Outer set should be sorted, but elements are arrays which maintain order
	originalNestedSetOfArray := original.Config["nested_set_of_array_of_str"].([]interface{})
	// Sort the outer set for comparison
	sort.Sort(EmptyInterfaceUsingUnderlyingType(originalNestedSetOfArray))
	assert.Equal(t, originalNestedSetOfArray, nestedSetOfArray, "outer set should be sorted")

	// Verify nested_set_of_set_of_str - both outer and inner sets SHOULD be sorted
	nestedSetOfSet := sortedConfig["nested_set_of_set_of_str"].([]interface{})
	innerSet1 := nestedSetOfSet[0].([]interface{})
	innerSet2 := nestedSetOfSet[1].([]interface{})
	originalNestedSetOfSet := original.Config["nested_set_of_set_of_str"].([]interface{})
	originalInnerSet1 := originalNestedSetOfSet[0].([]interface{})
	originalInnerSet2 := originalNestedSetOfSet[1].([]interface{})
	sort.Sort(EmptyInterfaceUsingUnderlyingType(originalInnerSet1))
	sort.Sort(EmptyInterfaceUsingUnderlyingType(originalInnerSet2))
	sort.Sort(EmptyInterfaceUsingUnderlyingType(originalNestedSetOfSet))
	assert.Equal(t, originalInnerSet1, innerSet1, "inner sets should be sorted")
	assert.Equal(t, originalInnerSet2, innerSet2, "inner sets should be sorted")
	assert.Equal(t, originalNestedSetOfSet, nestedSetOfSet, "outer set should be sorted")

	// Verify nested_array_of_record_of_array - arrays NOT sorted
	// Verify nested_array_of_record_of_set - outer array NOT sorted, inner sets SHOULD be sorted
	nestedArrayOfRecordArray1 := sortedConfig["nested_array_of_record_of_array"].([]interface{})
	outerArray1 := nestedArrayOfRecordArray1[0].([]interface{})
	record1 := outerArray1[0].(map[string]interface{})
	ports1 := record1["ports"].([]interface{})
	originalNestedArrayOfRecordArray := original.Config["nested_array_of_record_of_array"].([]interface{})
	originalOuterArray1 := originalNestedArrayOfRecordArray[0].([]interface{})
	originalRecord1 := originalOuterArray1[0].(map[string]interface{})
	originalPorts1 := originalRecord1["ports"].([]interface{})
	assert.Equal(t, originalPorts1, ports1)

	// Verify nested_array_of_record_of_set - outer array NOT sorted, inner sets SHOULD be sorted
	nestedArrayOfRecordArray2 := sortedConfig["nested_array_of_record_of_set"].([]interface{})
	outerArray2 := nestedArrayOfRecordArray2[0].([]interface{})
	record2 := outerArray2[0].(map[string]interface{})
	ports2 := record2["ports"].([]interface{})
	originalNestedArrayOfRecordSet := original.Config["nested_array_of_record_of_set"].([]interface{})
	originalOuterArray2 := originalNestedArrayOfRecordSet[0].([]interface{})
	originalRecord2 := originalOuterArray2[0].(map[string]interface{})
	originalPorts2 := originalRecord2["ports"].([]interface{})
	sort.Sort(EmptyInterfaceUsingUnderlyingType(originalPorts2))
	assert.Equal(t, originalPorts2, ports2)

	// Verify shorthand fields (shorthand_record_of_set_array)
	recordType := sortedConfig["shorthand_record_of_set_array"].(map[string]interface{})
	originalRecordType := original.Config["shorthand_record_of_set_array"].(map[string]interface{})

	// set_hosts should be SORTED
	setHosts := recordType["set_hosts"].([]interface{})
	originalSetHosts := originalRecordType["set_hosts"].([]interface{})
	sort.Sort(EmptyInterfaceUsingUnderlyingType(originalSetHosts))
	assert.Equal(t, originalSetHosts, setHosts, "set fields should be sorted")

	// set_hosts should be SORTED
	arrayHosts := recordType["array_hosts"].([]interface{})
	originalArrayHosts := originalRecordType["array_hosts"].([]interface{})
	assert.Equal(t, originalArrayHosts, arrayHosts, "array fields should NOT be sorted")
}
