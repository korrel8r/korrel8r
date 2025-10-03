// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/unique"

	"github.com/stretchr/testify/assert"
)

func TestDeduplicator(t *testing.T) {
	d := unique.NewDeduplicator(func(s string) string { return s })
	assert.True(t, d.Unique("a"))
	assert.False(t, d.Unique("a"))
	assert.True(t, d.Unique("b"))
	assert.False(t, d.Unique("b"))
}

func TestDeduplicatorKey(t *testing.T) {
	type item struct{ id int }
	d := unique.NewDeduplicator(func(i item) int { return i.id })
	assert.True(t, d.Unique(item{1}))
	assert.False(t, d.Unique(item{1}))
	assert.True(t, d.Unique(item{2}))
}

func TestDedupList(t *testing.T) {
	d := unique.NewDeduplicator(func(i int) int { return i })
	l := d.List(1, 2, 3, 1, 4, 3, 5)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, l.List)
}

func TestDedupListAdd(t *testing.T) {
	d := unique.NewDeduplicator(func(i int) int { return i })
	l := d.List()
	l.Add(1)
	l.Add(2)
	l.Add(1)
	l.Add(3)
	l.Add(2)
	l.Add(1)
	assert.Equal(t, []int{1, 2, 3}, l.List)
}

func TestDedupListClear(t *testing.T) {
	d := unique.NewDeduplicator(func(i int) int { return i })
	l := d.List()
	l.Add(1)
	l.Add(2)
	l.Add(1)
	l.Add(3)
	assert.Equal(t, []int{1, 2, 3}, l.List)
	l.Clear()
	assert.Empty(t, l.List)
	l.Add(3)
	assert.Empty(t, l.List)
	l.Add(5)
	assert.Equal(t, []int{5}, l.List)
}

func TestSet(t *testing.T) {
	s := unique.NewSet[string]()
	assert.False(t, s.Has("a"))
	s.Add("a")
	assert.True(t, s.Has("a"))
	s.Add("b")
	s.Add("a")
	assert.True(t, s.Has("a"))
	assert.True(t, s.Has("b"))
	s.Remove("a")
	assert.False(t, s.Has("a"))
	assert.True(t, s.Has("b"))
}

func TestSetList(t *testing.T) {
	s := unique.NewSet("c", "a", "b", "a", "c")
	list := s.List()
	assert.Len(t, list, 3)
	assert.Contains(t, list, "a")
	assert.Contains(t, list, "b")
	assert.Contains(t, list, "c")
}

func TestUniqueList(t *testing.T) {
	l := unique.NewList(1, 2, 3, 1, 4, 3, 5, 5, 5, 6)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, l.List)

	l2 := unique.NewList[int]()
	l2.Append(1, 2, 3, 1, 4, 3, 5, 5, 5, 6)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, l2.List)
}

func TestUniqueListAdd(t *testing.T) {
	l := unique.NewList[string]()
	l.Add("a")
	l.Add("b")
	l.Add("a")
	l.Add("c")
	l.Add("b")
	assert.Equal(t, []string{"a", "b", "c"}, l.List)
	assert.True(t, l.Has("a"))
	assert.True(t, l.Has("b"))
	assert.True(t, l.Has("c"))
	assert.False(t, l.Has("d"))
}

func TestList(t *testing.T) {
	l := unique.NewList[int]()
	l.Append(1, 2, 3, 1, 4, 3, 5, 5, 5, 6)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, l.List)
}
