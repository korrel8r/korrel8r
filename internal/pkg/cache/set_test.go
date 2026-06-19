// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package cache

import (
	gosync "sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet_Add(t *testing.T) {
	s := NewSet[string]()
	assert.True(t, s.Add("a"), "first add should return true")
	assert.True(t, s.Add("b"), "first add of different value should return true")
	assert.False(t, s.Add("a"), "duplicate add should return false")
}

func TestSet_Has(t *testing.T) {
	s := NewSet[int]()
	assert.False(t, s.Has(1))
	s.Add(1)
	assert.True(t, s.Has(1))
	assert.False(t, s.Has(2))
}

func TestSet_ConcurrentAdd(t *testing.T) {
	s := NewSet[int]()
	var wg gosync.WaitGroup
	added := make([]bool, 100)
	for i := range 100 {
		wg.Go(func() {
			added[i] = s.Add(i)
		})
	}
	wg.Wait()
	for i, ok := range added {
		assert.True(t, ok, "first add of %d should return true", i)
	}
}

func TestSet_ConcurrentAddDuplicates(t *testing.T) {
	s := NewSet[int]()
	const goroutines = 50
	results := make([]bool, goroutines)
	var wg gosync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			results[i] = s.Add(42)
		})
	}
	wg.Wait()
	trueCount := 0
	for _, ok := range results {
		if ok {
			trueCount++
		}
	}
	assert.Equal(t, 1, trueCount, "exactly one goroutine should successfully add the value")
}
