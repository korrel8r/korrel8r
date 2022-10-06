// package unique provides functions and types to remove duplicate values.
package unique

// Set copies comparable values from into a single slice without duplicates.
// Duplicates are values where Key(v) is equal.
//
type Set[K, V any] struct {
	seen map[any]struct{}
	list []V       // Unique values
	key  func(V) K // Key function must be set.
}

// NewSet using key function. Use Same if key and value are the same.
func NewSet[K, V any](key func(V) K, values ...V) *Set[K, V] {
	s := &Set[K, V]{seen: map[any]struct{}{}, key: key}
	s.Append(values...)
	return s
}

func (s *Set[K, V]) List() []V { return s.list }

// Same returns the value as a key, use when the value is the key.
func Same[V any](v V) V { return v }

// Append each value to c.Values if a value with the same key is not already present.
func (s *Set[K, V]) Append(values ...V) {
	for _, v := range values {
		k := s.key(v)
		if _, ok := s.seen[k]; !ok { // New value
			s.list = append(s.list, v)
			if s.seen == nil {
				s.seen = map[any]struct{}{}
			}
			s.seen[k] = struct{}{}
		}
	}
}

// Copy returns a copy of values where values with duplicate keys have been removed.
func Copy[K, V any](values []V, key func(V) K) []V {
	c := NewSet(key)
	for _, v := range values {
		c.Append(v)
	}
	return c.List()
}

// InPlace returns a de-duplicated slice using the same storage as the original.
func InPlace[K, V any](values []V, key func(V) K) []V {
	m := map[any]struct{}{}
	i := 0
	for i < len(values) {
		v := values[i]
		k := key(v)
		if _, ok := m[k]; ok { // Already seen, delete
			values = append((values)[:i], (values)[i+1:]...)
		} else {
			m[k] = struct{}{}
			i++
		}
	}
	return values
}
