package unique

// List of unique comparable values, maintains order.
type List[T any] struct {
	seen map[any]struct{}
	List []T
}

func NewList[T any]() List[T] { return List[T]{seen: map[any]struct{}{}} }

func (l *List[T]) Append(values ...T) {
	for _, v := range values {
		if _, ok := l.seen[v]; !ok {
			l.seen[v] = struct{}{}
			l.List = append(l.List, v)
		}
	}
}
