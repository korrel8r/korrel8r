package graph

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/topo"
)

func TestTraverse(t *testing.T) {
	for _, x := range []struct {
		name   string
		graph  []korrel8r.Rule
		sorted []int
		cycles [][]int
		want   []korrel8r.Rule
	}{
		{
			name:   "simple",
			graph:  []korrel8r.Rule{r(1, 2), r(2, 3), r(1, 3), r(3, 4), r(4, 13)},
			sorted: []int{1, 2, 3, 4, 13},
			cycles: nil,
			want:   []korrel8r.Rule{r(1, 2), r(2, 3), r(1, 3), r(3, 4), r(4, 13)},
		},
		{
			name:   "inner-cycle",
			graph:  []korrel8r.Rule{r(1, 2), r(2, 3), r(3, 4), r(4, 3), r(4, 13)},
			sorted: []int{1, 2, -1, 13},
			cycles: [][]int{{3, 4}},
			want:   []korrel8r.Rule{r(1, 2), r(2, 3), r(3, 4), r(4, 13)},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			// Run topo sort as part of the test for debugging.
			g := testGraph(x.graph)
			sorted, err := topo.Sort(g)
			sortedInt := nodesToInts(sorted)
			cycles, _ := err.(topo.Unorderable)
			if len(cycles) == 0 {
				assert.NoError(t, err)
			}
			var cyclesInt [][]int
			for _, cycle := range cycles {
				cyclesInt = append(cyclesInt, nodesToInts(cycle))
			}
			assert.Equal(t, x.sorted, sortedInt)
			assert.Equal(t, x.cycles, cyclesInt)

			var got []korrel8r.Rule
			err = Traverse(g, func(lines multi.Edge) {
				visitSort(lines, func(l graph.Line) { got = append(got, l.(*Line).Rule) })
			})
			assert.NoError(t, err)
			assert.Equal(t, x.want, got)
		})
	}
}

// testGraph ensures the node IDs sort in the same order as class values.
func testGraph(rules []korrel8r.Rule) *Graph {
	g := New()
	classes := unique.NewList[korrel8r.Class]()
	for _, r := range rules {
		classes.Append(r.Start(), r.Goal())
	}
	slices.SortFunc(classes.List, func(a, b korrel8r.Class) bool { return a.(class) < b.(class) })
	for _, c := range classes.List {
		g.NodeForClass(c)
	}
	for _, r := range rules {
		g.AddRule(r)
	}
	return g
}
