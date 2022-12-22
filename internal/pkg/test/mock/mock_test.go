package mock

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomain(t *testing.T) {
	d := Domain("foo")
	assert.Equal(t, "foo", d.String())
	assert.Equal(t, Class("foo/x"), d.Class("x"))
	assert.Empty(t, d.Classes())

	d = Domain("foo a b c")
	assert.Equal(t, "foo", d.String())
	assert.Equal(t, Class("foo/a"), d.Class("a"))
	assert.Equal(t, nil, d.Class("x"))
	assert.Equal(t, []korrel8.Class{Class("foo/a"), Class("foo/b"), Class("foo/c")}, d.Classes())
}

func TestClass(t *testing.T) {
	c := Class("d/c")
	assert.Equal(t, Domain("d"), c.Domain())
	assert.Equal(t, "c", c.String())
	assert.Equal(t, Object("d/c:"), c.New())
	assert.Equal(t, Object("d/c:foo"), c.Key(Object("d/c:foo")))

	c = Class("c")
	assert.Equal(t, Domain(""), c.Domain())
	assert.Equal(t, Object("c:"), c.New())
	assert.Equal(t, Object("c:foo"), c.Key(Object("c:foo")))
}

func TestObject(t *testing.T) {
	o := Object("d/c:hello")
	assert.Equal(t, []any{Class("d/c"), "hello"}, []any{o.Class(), o.Data()})
}

func TestStore_Get(t *testing.T) {
	r := korrel8.NewListResult()
	s := Store{"test": Objects("X/foo:x", "Y/bar.y", "foo:a", "bar:b", ":u", ":v")}
	require.NoError(t, s.Get(context.Background(), uri.Reference{Path: "test"}, r))
	want := Objects("X/foo:x", "Y/bar.y", "foo:a", "bar:b", ":u", ":v")
	assert.Equal(t, want, r.List())
}

func TestStore_NewReference(t *testing.T) {
	r := korrel8.NewListResult()
	s := Store{}
	ref := s.NewReference("X/foo:x", "Y/bar.y", "foo:a", "bar:b", ":u", ":v")
	require.NoError(t, s.Get(context.Background(), ref, r))
	want := Objects("X/foo:x", "Y/bar.y", "foo:a", "bar:b", ":u", ":v")
	assert.Equal(t, want, r.List())
}
