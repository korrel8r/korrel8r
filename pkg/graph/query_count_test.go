// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/stretchr/testify/assert"
)

func TestQueryCounts(t *testing.T) {
	d := mock.Domain("x")
	q := mock.NewQuery(d.Class("a"))
	qcs := QueryCounts{}
	qcs.Put(q, 3)
	qc, ok := qcs.Get(q)
	assert.True(t, ok)
	assert.Equal(t, QueryCount{Query: q, Count: 3}, qc)
}
