// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package slices

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransform(t *testing.T) {
	assert.Equal(t, []int{2, 4, 6}, Transform([]int{1, 2, 3}, func(x int) int { return x * 2 }))
	assert.Equal(t, []int{}, Transform([]int{}, func(x int) int { return x }))
}

func TestStrings(t *testing.T) {
	assert.Equal(t, []string{"1", "2", "3"}, Strings([]int{1, 2, 3}))
	assert.Equal(t, []string{"hello", "world"}, Strings([]string{"hello", "world"}))
}

func TestAnys(t *testing.T) {
	got := Anys([]int{1, 2})
	assert.Equal(t, []any{1, 2}, got)
}
