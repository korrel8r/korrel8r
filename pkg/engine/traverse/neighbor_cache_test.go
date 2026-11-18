// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE
package traverse

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"weak"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeighborRoutingCache(t *testing.T) {
	b := mock.NewBuilder("d")
	e, err := engine.Build().Rules(b.Rule("ab", "d:a", "d:b", b.Query("d:b", "ab", 1))).
		Stores(b.Store("d", nil)).Engine()
	require.NoError(t, err)
	start := b.Class("d:a")
	var cache neighborRoutingCache
	routing, err := cache.get(e, start, 1)
	require.NoError(t, err)
	again, err := cache.get(e, start, 1)
	require.NoError(t, err)
	assert.Same(t, &routing[0], &again[0])

	// Concurrent misses publish one shared entry, not separate routing arrays.
	var concurrent neighborRoutingCache
	var wg sync.WaitGroup
	results := make([][]nodeRouting, 8)
	errs := make([]error, len(results))
	for i := range results {
		wg.Go(func() { results[i], errs[i] = concurrent.get(e, start, 1) })
	}
	wg.Wait()
	for i, r := range results {
		require.NoError(t, errs[i])
		assert.Same(t, &results[0][0], &r[0])
	}

	// Cached routing must be safe to share between concurrent searches.
	for range 8 {
		wg.Go(func() {
			g, err := Neighbors(context.Background(), e, Start{
				Class: start, Objects: []korrel8r.Object{0},
			}, 1)
			assert.NoError(t, err)
			assert.ElementsMatch(t, []string{"d:a[0]", "d:b[1]"}, g.NodeStrings(true))
		})
	}
	wg.Wait()

	zero, err := cache.get(e, start, 0)
	require.NoError(t, err)
	assert.Empty(t, zero[0].lines)
	_, err = cache.get(e, b.Class("d:missing"), 1)
	require.Error(t, err)
}

func TestNeighborRoutingCacheConcurrentMiss(t *testing.T) {
	e := &engine.Engine{}
	defer runtime.KeepAlive(e)
	var cache neighborRoutingCache
	key := neighborRoutingKey{engine: weak.Make(e), depth: 1}
	warmKey := neighborRoutingKey{engine: weak.Make(e), depth: 2}
	warm, err := cache.getOrBuild(e, warmKey, func() ([]nodeRouting, error) {
		return []nodeRouting{{}}, nil
	})
	require.NoError(t, err)

	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	var builds atomic.Int32
	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = cache.getOrBuild(e, key, func() ([]nodeRouting, error) {
			builds.Add(1)
			close(started)
			<-release
			return []nodeRouting{{}}, nil
		})
	})
	<-started

	// A cold build for one key must not block hits or misses for other keys.
	otherDone := make(chan struct{})
	var warmBuilds atomic.Int32
	wg.Go(func() {
		defer close(otherDone)
		got, err := cache.getOrBuild(e, warmKey, func() ([]nodeRouting, error) {
			warmBuilds.Add(1)
			return []nodeRouting{{}}, nil
		})
		if assert.NoError(t, err) && assert.NotEmpty(t, got) {
			assert.Same(t, &warm[0], &got[0])
		}
		_, err = cache.getOrBuild(e, neighborRoutingKey{engine: weak.Make(e), depth: 3}, func() ([]nodeRouting, error) {
			return []nodeRouting{{}}, nil
		})
		assert.NoError(t, err)
	})
	select {
	case <-otherDone:
	case <-time.After(2 * time.Second):
		t.Error("unrelated cache access blocked by cold build")
	}

	// Additional callers for the blocked key should share its one build.
	for range 8 {
		wg.Go(func() {
			_, err := cache.getOrBuild(e, key, func() ([]nodeRouting, error) {
				builds.Add(1)
				return []nodeRouting{{}}, nil
			})
			assert.NoError(t, err)
		})
	}
	unblock()
	wg.Wait()
	assert.EqualValues(t, 1, builds.Load())
	assert.Zero(t, warmBuilds.Load())
}

func TestNeighborRoutingCacheEviction(t *testing.T) {
	b := mock.NewBuilder("d")
	e, err := engine.Build().Rules(b.Rule("ab", "d:a", "d:b", nil)).Engine()
	require.NoError(t, err)
	var cache neighborRoutingCache
	start := b.Class("d:a")
	for depth := range maxNeighborRoutings + 1 {
		_, err := cache.get(e, start, depth)
		require.NoError(t, err)
	}
	require.Len(t, cache.entries, maxNeighborRoutings)
	assert.Equal(t, 1, cache.entries[0].key.depth)
	assert.Equal(t, maxNeighborRoutings, cache.entries[len(cache.entries)-1].key.depth)

	// Releasing one engine removes its scopes without disturbing another's.
	other, err := engine.Build().Rules(b.Rule("ab", "d:a", "d:b", nil)).Engine()
	require.NoError(t, err)
	_, err = cache.get(other, start, 0)
	require.NoError(t, err)
	engineCleanup{&cache, weak.Make(e)}.remove()
	require.Len(t, cache.entries, 1)
	assert.Equal(t, weak.Make(other), cache.entries[0].key.engine)
}
