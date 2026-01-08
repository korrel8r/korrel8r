// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package unique provides functions and types to remove duplicate values.
package unique

// Deduplicator keeps track of comparable keys that identify unique values.
type Deduplicator[K comparable, V any] struct {
	key  func(V) K
	seen Set[any]
}

// NewDeduplicator uses func key to extract keys from values.
func NewDeduplicator[K comparable, V any](key func(V) K) *Deduplicator[K, V] {
	return &Deduplicator[K, V]{key: key}
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

func (d *Deduplicator[K, V]) List(values ...V) *DedupList[K, V] {
	l := &DedupList[K, V]{Deduplicator: d}
	l.Append(values...)
	return l
}

type DedupList[K comparable, V any] struct {
	*Deduplicator[K, V]
	List []V
}

// Add a value if not already present.
func (l *DedupList[K, V]) Add(v V) {
	if l.Unique(v) {
		l.List = append(l.List, v)
	}
}

func (l *DedupList[K, V]) Append(values ...V) {
	for _, v := range values {
		l.Add(v)
	}
}

func (l *DedupList[K, V]) Clear() { l.List = l.List[:0] }
