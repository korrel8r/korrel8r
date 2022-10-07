// package unique provides functions and types to remove duplicate values.
package unique

// Deduplicator keeps track of comparable keys that identify unique values.
type Deduplicator[K, V any] struct {
	key  func(V) K
	seen map[any]struct{}
}

// NewDeduplicator uses func key to extract keys from values.
func NewDeduplicator[K, V any](key func(V) K) Deduplicator[K, V] {
	return Deduplicator[K, V]{key: key}
}

// Unique returns true if  the key of v has not been seen before.
// Unique will return false for future values with the same key.
func (d *Deduplicator[K, V]) Unique(v V) bool {
	k := d.key(v)
	_, seen := d.seen[k]
	if !seen {
		if d.seen == nil {
			d.seen = map[any]struct{}{}
		}
		d.seen[k] = struct{}{}
	}
	return !seen
}

// Copy returns a copy of values where values with duplicate keys have been removed.
func Copy[K, V any](values []V, key func(V) K) []V {
	d := NewDeduplicator(key)
	var result []V
	for _, v := range values {
		if d.Unique(v) {
			result = append(result, v)
		}
	}
	return result
}

// InPlace returns a de-duplicated slice using the same storage as the original.
func InPlace[K, V any](values []V, key func(V) K) []V {
	d := NewDeduplicator(key)
	i := 0
	for i < len(values) {
		v := values[i]
		if d.Unique(v) {
			i++
		} else { // Not unique, delete from slice
			values = append((values)[:i], (values)[i+1:]...)
		}
	}
	return values
}

// Same returns the value as a key, use when the value is the key.
func Same[V any](v V) V { return v }
