// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gonum "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/path"
)

func nodeIDs(nodes gonum.Nodes) []int64 {
	ids := []int64{}
	for nodes.Next() {
		ids = append(ids, nodes.Node().ID())
	}
	slices.Sort(ids)
	return ids
}

func TestViewMatchesGraph(t *testing.T) {
	b := mock.NewBuilder("d")
	d := NewData(
		b.Rule("wide", "d:a", []string{"d:a", "d:b", "d:c"}, nil),
		b.Rule("ab", "d:a", "d:b", nil), // Parallel edge with lower spread.
		b.Rule("ba", "d:b", "d:a", nil),
		b.Rule("cc", "d:c", "d:c", nil),
		b.Rule("zz", "d:z", "d:z", nil), // Disconnected component.
	)
	view, reference := d.Graph(), d.FullGraph()
	assert.Equal(t, nodeIDs(reference.Nodes()), nodeIDs(view.Nodes()))
	for from := int64(-1); from <= int64(d.NodeCount()); from++ {
		assert.Equal(t, reference.Node(from) == nil, view.Node(from) == nil)
		assert.Equal(t, nodeIDs(reference.From(from)), nodeIDs(view.From(from)))
		assert.Equal(t, nodeIDs(reference.To(from)), nodeIDs(view.To(from)))
		for to := int64(-1); to <= int64(d.NodeCount()); to++ {
			assert.Equal(t, reference.HasEdgeFromTo(from, to), view.HasEdgeFromTo(from, to))
			assert.Equal(t, reference.HasEdgeBetween(from, to), view.HasEdgeBetween(from, to))
			want, wantOK := reference.Weight(from, to)
			got, gotOK := view.Weight(from, to)
			assert.Equal(t, want, got)
			assert.Equal(t, wantOK, gotOK)
			edge := view.WeightedEdge(from, to)
			if !reference.HasEdgeFromTo(from, to) {
				assert.Nil(t, edge)
				assert.Nil(t, view.Edge(from, to))
				continue
			}
			require.NotNil(t, edge)
			assert.Equal(t, from, edge.From().ID())
			assert.Equal(t, to, edge.To().ID())
			assert.Equal(t, want, edge.Weight())
			assert.Equal(t, to, edge.ReversedEdge().From().ID())
		}
	}
	assert.Empty(t, nodeIDs(NewData().Graph().Nodes()))
}

func TestViewIteratorsAndConcurrentReads(t *testing.T) {
	b := mock.NewBuilder("d")
	d := NewData(b.Rule("wide", "d:a", []string{"d:a", "d:b", "d:c"}, nil))
	view := d.Graph()
	for _, it := range []gonum.Nodes{view.Nodes(), view.From(0), view.To(0)} {
		assert.Nil(t, it.Node())
		count := it.Len()
		require.True(t, it.Next())
		assert.Equal(t, count-1, it.Len())
		it.Reset()
		assert.Nil(t, it.Node())
		assert.Equal(t, count, it.Len())
		assert.Len(t, nodeIDs(it), count)
		assert.Zero(t, it.Len())
		assert.False(t, it.Next())
		assert.Nil(t, it.Node())
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 20 {
				assert.Equal(t, []int64{0, 1, 2}, nodeIDs(view.From(0)))
				assert.Equal(t, []int64{0}, nodeIDs(view.To(1)))
				w, ok := view.Weight(0, 1)
				assert.True(t, ok)
				assert.Equal(t, 3.0, w)
			}
		})
	}
	wg.Wait()
	assert.Zero(t, testing.AllocsPerRun(100, func() { view = d.Graph() }))
}

// Compare complete near-shortest path sets, including equal-cost alternatives,
// rather than relying on Gonum's unspecified map iteration order.
func TestViewYenPaths(t *testing.T) {
	b := mock.NewBuilder("d")
	rng := rand.New(rand.NewPCG(1, 2))
	paths := func(g gonum.Graph, from, to int64) []string {
		var result []string
		for _, p := range path.YenKShortestPaths(g, math.MaxInt, 1, g.Node(from), g.Node(to)) {
			ids := make([]int64, len(p))
			for i, n := range p {
				ids[i] = n.ID()
			}
			result = append(result, fmt.Sprint(ids))
		}
		slices.Sort(result)
		return result
	}
	for trial := range 12 {
		rules := []korrel8r.Rule{b.Rule("spread", "d:n0", []string{"d:n1", "d:n2", "d:n3"}, nil)}
		for from := range 5 {
			start := fmt.Sprintf("d:n%d", from)
			rules = append(rules, b.Rule(start, start, start, nil)) // Include every node.
			for to := range 5 {
				if rng.IntN(3) == 0 {
					rules = append(rules, b.Rule(fmt.Sprintf("%d-%d", from, to), start, fmt.Sprintf("d:n%d", to), nil))
				}
			}
		}
		d := NewData(rules...)
		reference := d.FullGraph()
		for from := int64(0); from < 5; from++ {
			for to := int64(0); to < 5; to++ {
				assert.Equal(t, paths(reference, from, to), paths(d.Graph(), from, to), "trial=%d from=%d to=%d", trial, from, to)
			}
		}
	}
}
