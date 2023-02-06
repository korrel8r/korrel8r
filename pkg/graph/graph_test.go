package graph

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/iterator"
	"gonum.org/v1/gonum/graph/multi"
)

func TestSubGraph(t *testing.T) {
	for _, x := range []struct {
		name    string
		graph   []korrel8r.Rule
		include []int
		want    []korrel8r.Rule
	}{
		{
			name:    "one",
			graph:   []korrel8r.Rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			include: []int{1, 3, 12},
			want:    []korrel8r.Rule{r(1, 3), r(3, 12)},
		},
		{
			name:    "two",
			graph:   []korrel8r.Rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			include: []int{1},
			want:    nil,
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := New(x.graph...)
			var nodes []graph.Node
			for _, i := range x.include {
				nodes = append(nodes, g.NodeForClass(class(i)))
			}
			sub := g.SubGraph(nodes)
			assert.Equal(t, x.want, sub.rules.List)
		})
	}
}

var (
	_ korrel8r.Class = class(0)
	_ korrel8r.Rule  = rule{}
)

type class int64

func (class) Domain() korrel8r.Domain { return mock.Domain("test") }
func (c class) String() string        { return strconv.FormatInt(int64(c), 10) }

type rule struct{ u, v class }

func (l rule) Start() korrel8r.Class { return l.u }
func (l rule) Goal() korrel8r.Class  { return l.v }
func (l rule) String() string        { return fmt.Sprintf("(%v,%v)", l.u, l.v) }
func (l rule) Apply(start korrel8r.Object, c *korrel8r.Constraint) (korrel8r.Query, error) {
	return nil, nil
}
func r(u, v class) korrel8r.Rule { return rule{u, v} }

func visitSort(edge multi.Edge, visit func(l graph.Line)) {
	ls := edge.Lines.(graph.LineSlicer).LineSlice()
	slices.SortFunc(ls, lineLess)
	visitLines(iterator.NewOrderedLines(ls), visit)
}

func nodeInt(n graph.Node) int { return int(n.(*Node).Class.(class)) }

func nodesToInts(nodes []graph.Node) []int {
	var ints []int
	for _, n := range nodes {
		i := -1
		if n != nil {
			i = int(n.(*Node).Class.(class))
		}
		ints = append(ints, i)
	}
	return ints
}

func pathsToInts(paths [][]graph.Node) [][]int {
	var intPaths [][]int
	for _, p := range paths {
		intPaths = append(intPaths, nodesToInts(p))
	}
	slices.SortFunc(intPaths, func(a, b []int) bool { return slices.Compare(a, b) < 0 })
	return intPaths
}

func lineLess(a, b graph.Line) bool {
	return nodeInt(a.From()) < nodeInt(b.From()) || nodeInt(a.To()) < nodeInt(b.To())
}
