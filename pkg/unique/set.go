// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique

import (
	"maps"
	"slices"
)

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](vs ...T) Set[T] {
	s := Set[T]{}
	for _, v := range vs {
		s.Add(v)
	}
	return s
}

func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }
func (s Set[T]) Add(v T)      { s[v] = struct{}{} }
func (s Set[T]) Remove(v T)   { delete(s, v) }
func (s Set[T]) List() []T    { return slices.Collect(maps.Keys(s)) }
