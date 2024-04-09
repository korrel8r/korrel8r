// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/stretchr/testify/assert"
)

func TestQueries(t *testing.T) {
	d := mock.Domain("x")
	q := mock.NewQuery(d.Class("a"))
	qs := Queries{}
	assert.False(t, qs.Has(q))
	assert.Equal(t, -1, qs.Get(q))
	qs.Set(q, 3)
	assert.True(t, qs.Has(q))
	assert.Equal(t, 3, qs.Get(q))
}
