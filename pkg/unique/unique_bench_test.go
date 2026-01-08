// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique_test

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Benchmark data sizes
var (
	length = 1000
	width  = 100
)

func data(b *testing.B) []string {
	result := make([]string, length)
	for i := range length {
		result[i] = fmt.Sprintf("%0*v", width, i)
	}
	return result
}

func BenchmarkSetAdd(b *testing.B) {
	d := data(b)
	dups := append(d, d...)
	b.ResetTimer()
	for b.Loop() {
		_ = unique.NewSet(dups...)
	}
}

func BenchmarkSetHas(b *testing.B) {
	d := data(b)
	dups := append(d, d...)
	s := unique.NewSet(dups...)
	b.ResetTimer()
	for b.Loop() {
		for _, v := range d {
			_ = s.Has(v)
		}
	}
}
func BenchmarkListAdd(b *testing.B) {
	d := data(b)
	dups := append(d, d...)
	b.ResetTimer()
	for b.Loop() {
		_ = unique.NewList(dups...)
	}
}

func BenchmarkListHas(b *testing.B) {
	d := data(b)
	dups := append(d, d...)
	s := unique.NewList(dups...)
	b.ResetTimer()
	for b.Loop() {
		for _, v := range d {
			_ = s.Has(v)
		}
	}
}

func BenchmarkSetRemove(b *testing.B) {
	d := data(b)
	dups := append(d, d...)
	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		s := unique.NewSet(dups...)
		b.StartTimer()
		for _, v := range d {
			s.Remove(v)
		}
	}
}

func BenchmarkDeduplicatorUnique(b *testing.B) {
	d := data(b)
	dups := append(d, d...)
	dd := unique.NewDeduplicator(func(s string) string { return s })
	b.ResetTimer()
	for b.Loop() {
		for _, v := range dups {
			_ = dd.Unique(v)
		}
	}
}
