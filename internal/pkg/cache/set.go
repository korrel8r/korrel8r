// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package cache

import (
	gosync "sync"
)

// Set is a goroutine-safe set.
type Set[T comparable] struct {
	m gosync.Map
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{m: gosync.Map{}}
}

// Has returns true if v is in the set.
func (s *Set[T]) Has(v T) bool {
	_, ok := s.m.Load(v)
	return ok
}

// Add a value, return true if the value was added, false if already present.
func (s *Set[T]) Add(v T) bool {
	_, loaded := s.m.Swap(v, nil)
	return !loaded
}
