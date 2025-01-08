// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package domain

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func (f *Fixture) Benchmark(b *testing.B) {
	b.Helper()
	f.Init(b)
	b.ResetTimer()
	for name, bench := range map[string]func(*testing.B){
		"Get":              f.BenchmarkGet,
		"ParseQuery":       f.BenchmarkParseQuery,
		"MarshalUnmarshal": f.BenchmarkMarshalUnmashal,
	} {
		b.Run(name, bench)
	}
}

func (f *Fixture) BenchmarkGet(b *testing.B) {
	b.Helper()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := graph.NewResult(f.Query.Class())
		require.NoError(b, f.MockEngine.Get(context.Background(), f.Query, nil, r))
		require.Equal(b, BatchLen, len(r.List()))
	}
}

func (f *Fixture) BenchmarkParseQuery(b *testing.B) {
	b.Helper()
	q := f.Query.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := f.MockEngine.Query(q)
		require.NoError(b, err)
	}
}

func (f *Fixture) BenchmarkMarshalUnmashal(b *testing.B) {
	b.Helper()
	r := graph.NewResult(f.Query.Class())
	require.NoError(b, f.MockEngine.Get(context.Background(), f.Query, &korrel8r.Constraint{Limit: ptr.To(1)}, r))
	o := r.List()[0]
	c := f.Query.Class()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytes, _ := json.Marshal(o)
		o2, _ := c.Unmarshal(bytes)
		require.IsType(b, o, o2)
	}
}
