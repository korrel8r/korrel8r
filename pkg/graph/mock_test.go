package graph

import (
	"fmt"
	"strconv"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"golang.org/x/exp/slices"
	"gonum.org/v1/gonum/graph"
)

// Mock graphs for tests.

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
func r(u, v class) rule { return rule{u, v} }

func testGraph(mockRules []rule) *Graph {
	rules := make([]korrel8r.Rule, len(mockRules))
	for i := range mockRules {
		rules[i] = mockRules[i]
	}
	return New(rules)
}

func rules(g *Graph) []rule {
	var rules []rule
	g.EachLine(func(l *Line) { rules = append(rules, l.Rule.(rule)) })
	slices.SortFunc(rules, ruleLess)
	return rules
}

func ruleLess(a, b rule) bool       { return a.u < b.u || (a.u == b.u && a.v < b.v) }
func nodeLess(a, b graph.Node) bool { return a.(*Node).Class.(class) < b.(*Node).Class.(class) }
func testOrder(nodes []graph.Node)  { slices.SortFunc(nodes, nodeLess) }
