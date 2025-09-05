// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package domain

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (f *Fixture) Test(t *testing.T) {
	t.Helper()
	f.Init(t)
	for _, f := range []func(*testing.T){f.TestGet, f.TestMarshalUnmashal, f.TestGetCluster} {
		t.Run(funcName(f), f)
	}
}

// TestGet tests only the domain Get overhead using mock in-memory stores.
func (f *Fixture) TestGet(t *testing.T) {
	t.Helper()
	r := result.New(f.Query.Class())
	require.NoError(t, f.MockEngine.Get(context.Background(), f.Query, &korrel8r.Constraint{}, r))
	if assert.Equal(t, BatchLen, len(r.List()), "wrong number of results: %v", f.Query) {
		if _, ok := f.Query.Class().(korrel8r.IDer); ok { // Only test de-duplication for classes with ID.
			t.Run("TestGet_dedup", func(t *testing.T) {
				require.NoError(t, f.MockEngine.Get(context.Background(), f.Query, &korrel8r.Constraint{}, r))
				assert.Equal(t, BatchLen, len(r.List()), "de-duplication failed: %v", f.Query)
			})
		}
	}
}

func (f *Fixture) TestMarshalUnmashal(t *testing.T) {
	t.Helper()
	c := f.Query.Class()
	r := result.New(c)
	require.NoError(t, f.MockEngine.Get(context.Background(), f.Query, &korrel8r.Constraint{Limit: ptr.To(1)}, r))
	require.GreaterOrEqual(t, len(r.List()), 1)
	o := r.List()[0]
	bytes, err := json.Marshal(o)
	require.NoError(t, err)
	o2, err := c.Unmarshal(bytes)
	require.NoError(t, err)
	require.Equal(t, o, o2)
}

// TestGetCluster that query constraints work with real stores.
// Requires a cluster with the relevant signal stores available.
func (f *Fixture) TestGetCluster(t *testing.T) {
	t.Helper()
	e := f.ClusterEngine(t)
	limit := 3
	constraint := &korrel8r.Constraint{Limit: &limit}
	r := result.New(f.Query.Class())
	require.NoError(t, e.Get(context.Background(), f.Query, constraint, r), f.Query)
	assert.Equal(t, limit, len(r.List()), f.Query)
}
