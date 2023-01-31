package unique

type empty = struct{}

type Set[T comparable] map[T]empty

func (s Set[T]) Has(v T) bool { _, ok := s[v]; return ok }
func (s Set[T]) Add(v T)      { s[v] = empty{} }
func (s Set[T]) Remove(v T)   { delete(s, v) }
