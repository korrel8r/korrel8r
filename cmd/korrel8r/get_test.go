// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"errors"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/stretchr/testify/assert"
)

func TestConstraintTotalFlags(t *testing.T) {
	oldLimit, oldQueryLimit := limit, queryLimit
	oldTotalLimit, oldTotalQueryLimit := totalLimit, totalQueryLimit
	t.Cleanup(func() {
		limit, queryLimit = oldLimit, oldQueryLimit
		totalLimit, totalQueryLimit = oldTotalLimit, oldTotalQueryLimit
	})

	limit, queryLimit, totalLimit, totalQueryLimit = 1, 2, 3, 4
	c := constraint()
	assert.Equal(t, 1, c.GetLimit())
	assert.Equal(t, 2, c.GetQueryLimit())
	assert.Equal(t, 3, c.GetTotalLimit())
	assert.Equal(t, 4, c.GetTotalQueryLimit())
}

func TestMustTraversalAcceptsLimitError(t *testing.T) {
	assert.NotPanics(t, func() {
		mustTraverse(&traverse.LimitError{Name: "totalLimit", Limit: 1})
	})
	assert.Panics(t, func() {
		mustTraverse(errors.New("failed"))
	})
}
