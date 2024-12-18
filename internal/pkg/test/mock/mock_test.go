// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mock_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_LoadFile(t *testing.T) {
	d := mock.Domain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)
	require.NoError(t, s.LoadFile("testdata/test_store.yaml"))

	q := mock.NewQuery(c, "query1")
	r := &mock.ListResult{}
	require.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []any{"x", "y"}, r.List())

	r = &mock.ListResult{}
	require.NoError(t, s.Get(context.Background(), mock.NewQuery(c, "query2"), nil, r))
	assert.Equal(t, []any{"a", "b", "c"}, r.List())
}

func TestStore_NewQuery(t *testing.T) {
	d := mock.Domain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)

	q1 := s.NewQuery(c, 1, 2)
	q2 := s.NewQuery(c, 3, 4)
	r := &mock.ListResult{}
	assert.NoError(t, s.Get(context.Background(), q1, nil, r))
	assert.Equal(t, []korrel8r.Object{1, 2}, r.List())
	r = &mock.ListResult{}
	assert.NoError(t, s.Get(context.Background(), q2, nil, r))
	assert.Equal(t, []korrel8r.Object{3, 4}, r.List())

	r = &mock.ListResult{}
	q3 := mock.NewQuery(c, "foo", 1, 2, 3)
	assert.NoError(t, s.Get(context.Background(), q3, nil, r))
	assert.Equal(t, []korrel8r.Object{1, 2, 3}, r.List())
}

func TestStore_NewResult(t *testing.T) {
	d := mock.Domain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)
	q := mock.NewQuery(c, "query")
	s.AddQuery(q, []korrel8r.Object{"a", "b"})
	r := &mock.ListResult{}
	require.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []korrel8r.Object{"a", "b"}, r.List())
}

func list[T any](x ...T) []T { return x }

func ClassesFunc(d korrel8r.Domain) func(names ...string) []korrel8r.Class {
	return func(names ...string) []korrel8r.Class {
		classes := make([]korrel8r.Class, len(names))
		for i, name := range names {
			classes[i] = d.Class(name)
		}
		return classes
	}
}

func TestRule_Apply(t *testing.T) {
	d := mock.Domain("foo")
	s := mock.NewStore(d)
	c := d.Class
	cx, cy := c("x"), c("y")

	for _, x := range []struct {
		result []any
		want   string
	}{
		{
			result: []any{mock.NewQuery(cx, "?baz")},
			want:   "foo:x:?baz",
		},
		{
			result: []any{func(o korrel8r.Object) (korrel8r.Query, error) {
				return mock.NewQuery(cy, fmt.Sprintf("?%v", o)), nil
			}},
			want: "foo:y:?0",
		},
		{
			result: []any{s, cy, 1, 2, 3},
			want:   "foo:y:[1,2,3]",
		},
	} {
		t.Run(x.want, func(t *testing.T) {
			q, err := mock.NewRule("A", list(cx), list(cy), x.result...).Apply(0)
			require.NoError(t, err)
			assert.Equal(t, x.want, q.String())
		})
	}
}

func TestFileStore(t *testing.T) {
	d := mock.Domain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)
	s.AddLookup(mock.QueryDir("testdata/filestore").Get)
	q := mock.NewQuery(c, "query1")
	r := &mock.ListResult{}
	require.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []any{"x", "y"}, r.List())

	r = &mock.ListResult{}
	require.NoError(t, s.Get(context.Background(), mock.NewQuery(c, "query2"), nil, r))
	assert.Equal(t, []any{"a", "b", "c"}, r.List())
}
