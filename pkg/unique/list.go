// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique

// List of unique comparable values, maintains order.
type List[T comparable] struct {
	List []T
	Set  Set[T]
}

func NewList[T comparable](values ...T) *List[T] {
	l := &List[T]{Set: Set[T]{}}
	l.Append(values...)
	return l
}

// Add a value if not already present, return true if the value was added.
func (l *List[T]) Add(v T) bool {
	has := l.Set.Has(v)
	if !has {
		l.Set.Add(v)
		l.List = append(l.List, v)
	}
	return !has
}

func (l *List[T]) Has(v T) bool { return l.Set.Has(v) }

func (l *List[T]) Append(values ...T) {
	for _, v := range values {
		_ = l.Add(v)
	}
}
