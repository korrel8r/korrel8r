// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique

type empty = struct{}

type Set[T comparable] map[T]empty

func NewSet[T comparable](v ...T) Set[T] {
	s := Set[T]{}
	s.Add(v...)
	return s
}
func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }

func (s Set[T]) Add(vs ...T) {
	for _, v := range vs {
		s[v] = empty{}
	}
}
func (s Set[T]) Remove(v T) { delete(s, v) }
