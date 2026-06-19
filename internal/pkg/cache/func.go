// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package cache

import (
	"sync"
	"weak"
)

// Func wraps function f in a function that calls f and caches the result,
// then returns the cached result subsequently.
func Func[K, V any](f func(K) V) func(K) V {
	m := sync.Map{}
	return func(k K) V {
		v, ok := m.Load(k)
		if !ok {
			v = f(k)
			m.Store(k, v)
		}
		return v.(V)
	}
}

// FuncWeakValue keeps a weak pointer to the return value and deletes cache entry
// if it is garbage collected.
func FuncWeakValue[K comparable, V any](f func(K) *V) func(K) *V {
	m := sync.Map{}
	return func(k K) *V {
		v, ok := m.Load(k)
		if !ok {
			v = weak.Make(f(k))
			m.Store(k, v)
		}
		return v.(*V)
	}
}

// FuncWeakKey keeps a weak pointer to the key argument and deletes cache entry
// if it is garbage collected.
func FuncWeakKey[K, V any](f func(*K) V) func(*K) V {
	m := sync.Map{}
	return func(k *K) V {
		wk := weak.Make(k)
		v, ok := m.Load(wk)
		if !ok {
			v = f(k)
			m.Store(wk, v)
		}
		return v.(V)
	}
}
