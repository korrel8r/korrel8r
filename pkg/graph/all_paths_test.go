package graph

import (
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

func newGraph(pairs ...int64) graph.Graph {
	g := simple.NewDirectedGraph()
	for i := 0; i < len(pairs); i += 2 {
		from, _ := g.NodeWithID(pairs[i])
		to, _ := g.NodeWithID(pairs[i+1])
		g.SetEdge(g.NewEdge(from, to))
	}
	return g
}

func TestAllPaths(t *testing.T) {
	for i, x := range []struct {
		g    graph.Graph
		want [][]int64
	}{
		{
			g:    newGraph(1, 2, 1, 3, 3, 11, 3, 12, 12, 13),
			want: [][]int64{{1, 3, 12, 13}},
		},
		{
			g: newGraph(1, 2, 1, 3, 3, 11, 2, 12, 3, 12, 12, 13),
			want: [][]int64{
				{1, 2, 12, 13},
				{1, 3, 12, 13},
			},
		},
		{
			g: newGraph(1, 2, 1, 3, 1, 13, 2, 12, 3, 12, 12, 13),
			want: [][]int64{
				{1, 2, 12, 13},
				{1, 3, 12, 13},
				{1, 13},
			},
		},
		{
			g:    newGraph(1, 2, 1, 3, 3, 11, 12, 13),
			want: [][]int64{},
		}} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			paths := AllPaths(x.g, 1, 13)
			got := make([][]int64, len(paths))
			for i, p := range paths {
				for _, n := range p {
					got[i] = append(got[i], n.ID())
				}
			}
			assert.ElementsMatch(t, x.want, got)
		})
	}
}
