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
	qs[q.String()] = 3
	n, ok := qs[q.String()]
	assert.True(t, ok)
	assert.Equal(t, 3, n)
}
