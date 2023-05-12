// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
)

func TestJSONList(t *testing.T) {
	l := unique.NewJSONList[[]int]()
	l.Append([]int{1}, []int{2}, []int{1}, []int{4}, []int{3})
	assert.Equal(t, [][]int{{1}, {2}, {4}, {3}}, l.List)
}
