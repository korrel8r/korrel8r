package unique_test

import (
	"testing"

	"github.com/korrel8/korrel8/pkg/unique"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, unique.Copy([]int{1, 2, 1, 3, 4, 5, 3, 6, 6}, unique.Same[int]))
	assert.Equal(t,
		[]string{"a1", "b1", "c2", "d2"},
		unique.Copy([]string{"a1", "b1", "a2", "c2", "d2", "d3"}, func(s string) string { return s[0:1] }))
	assert.Equal(t, []int(nil), unique.Copy(nil, unique.Same[int]))
	assert.Equal(t, []int(nil), unique.Copy([]int{}, unique.Same[int]))
}

func TestInPlace(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, unique.InPlace([]int{1, 2, 1, 3, 4, 5, 3, 6, 6}, unique.Same[int]))
	assert.Equal(t,
		[]string{"a1", "b1", "c2", "d2"},
		unique.InPlace([]string{"a1", "b1", "a2", "c2", "d2", "d3"}, func(s string) string { return s[0:1] }))
	assert.Equal(t, []int(nil), unique.InPlace(nil, unique.Same[int]))
	assert.Equal(t, []int{}, unique.InPlace([]int{}, unique.Same[int]))
}
