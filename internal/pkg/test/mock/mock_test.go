// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mock_test

import (
	"context"
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

	q := mock.NewQuery(c, "query 1")
	r := &korrel8r.ListResult{}
	require.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []any{"x", "y"}, r.List())

	r = &korrel8r.ListResult{}
	require.NoError(t, s.Get(context.Background(), mock.NewQuery(c, "query 2"), nil, r))
	assert.Equal(t, []any{"a", "b", "c"}, r.List())
}

func TestStore_NewQuery(t *testing.T) {
	d := mock.Domain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)

	q1 := s.NewQuery(c, 1, 2)
	q2 := s.NewQuery(c, 3, 4)
	r := &korrel8r.ListResult{}
	assert.NoError(t, s.Get(context.Background(), q1, nil, r))
	assert.Equal(t, []korrel8r.Object{1, 2}, r.List())
	r = &korrel8r.ListResult{}
	assert.NoError(t, s.Get(context.Background(), q2, nil, r))
	assert.Equal(t, []korrel8r.Object{3, 4}, r.List())
}

func TestStore_NewResult(t *testing.T) {
	d := mock.Domain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)
	q := mock.NewQuery(c, "query")
	s.Add(mock.QueryMap{q.String(): []korrel8r.Object{"a", "b"}})
	r := &korrel8r.ListResult{}
	require.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []korrel8r.Object{"a", "b"}, r.List())
}
