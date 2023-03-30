package graph

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
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

func r(u, v class) rule             { return rule{u, v} }
func ruleLess(a, b rule) bool       { return a.u < b.u || (a.u == b.u && a.v < b.v) }
func sortRules(rules []rule) []rule { slices.SortFunc(rules, ruleLess); return rules }

func testGraph(rules []rule) *Graph {
	d := NewData()
	for _, r := range rules {
		d.AddRule(r)
	}
	return d.NewGraph()
}

func graphRules(g *Graph) []rule {
	var rules []rule
	g.EachLine(func(l *Line) { rules = append(rules, l.Rule.(rule)) })
	sortRules(rules)
	return rules
}

// assertComponentOrder components is an ordered list of unordered sets of rules.
// Asserts that the rules list is in an order that is compatible with components
func assertComponentOrder(t *testing.T, components [][]rule, rules []rule) bool {
	msg := "out of order\nrules:      %v\ncomponents: %v\n"
	t.Helper()
	j := 0 // rules index
	for i, c := range components {
		if !assert.LessOrEqual(t, j+len(c), len(rules), "rule[%v], component[%v] len %v\n"+msg, j, i, len(c), rules, components) {
			return false
		}
		if !assert.Equal(t, sortRules(c), sortRules(rules[j:j+len(c)]), msg, rules, components) {
			return false
		}
		j += len(c)
		if j >= len(rules) {
			break
		}
	}
	return true
}
