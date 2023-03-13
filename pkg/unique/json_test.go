package unique

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

func TestJSONMap(t *testing.T) {
	m := unique.JSONMap[int, int]{}
	assert.Equal(t, 0, m.Len())

	m.Set(1, 11)
	i, ok := m.Get(1)
	assert.Equal(t, 1, m.Len())
	assert.True(t, ok)
	assert.Equal(t, 11, i)

	_, ok = m.Get(2)
	assert.False(t, ok)

	m.Set(2, 22)
	assert.Equal(t, 2, m.Len())

	var keys, values []int
	m.Range(func(k, v int) {
		keys = append(keys, k)
		values = append(values, v)
	})
	assert.Equal(t, []int{1, 2}, keys)
	assert.Equal(t, []int{11, 22}, values)

	m.Delete(1)
	assert.Equal(t, 1, m.Len())
	_, ok = m.Get(1)
	assert.False(t, ok)
}
