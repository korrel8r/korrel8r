// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique

import "encoding/json"

// JSONList is like List but uses JSON strings as the equality test.
type JSONList[T any] struct {
	seen Set[string]
	List []T
}

func NewJSONList[T any]() JSONList[T] { return JSONList[T]{seen: Set[string]{}} }

func JSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (l *JSONList[T]) Has(v T) bool {
	_, ok := l.seen[JSONString(v)]
	return ok
}

func (l *JSONList[T]) Add(v T) bool {
	j := JSONString(v)
	seen := l.seen.Has(j)
	if !seen {
		l.seen.Add(j)
		l.List = append(l.List, v)
	}
	return !seen
}

func (l *JSONList[T]) Append(values ...T) {
	for _, v := range values {
		l.Add(v)
	}
}
