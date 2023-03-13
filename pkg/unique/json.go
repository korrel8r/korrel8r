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

// JSONMap of uses the JSON form of the key type K as keys.
type JSONMap[K, V any] map[string]V

func (m JSONMap[K, V]) Get(k K) (v V, ok bool) { v, ok = m[JSONString(k)]; return v, ok }
func (m JSONMap[K, V]) Set(k K, v V)           { m[JSONString(k)] = v }
func (m JSONMap[K, V]) Len() int               { return len(m) }
func (m JSONMap[K, V]) Delete(k K)             { delete(m, JSONString(k)) }
func (m JSONMap[K, V]) Range(f func(K, V)) {
	for k, v := range m {
		f(JSONValue[K](k), v)
	}
}
func (m JSONMap[K, V]) Values(f func(V)) {
	for _, v := range m {
		f(v)
	}
}

func JSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func JSONValue[T any](s string) T {
	var v T
	err := json.Unmarshal([]byte(s), &v)
	if err != nil {
		panic(err)
	}
	return v
}
