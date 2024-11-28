// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"slices"
	"strconv"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

var Domain = mock.Domain("graphmock")

type rule = korrel8r.Rule

func c(i int) korrel8r.Class { return Domain.Class(strconv.Itoa(i)) }

func nodesToInts(nodes []*Node) (ret []int) {
	for _, n := range nodes {
		ret = append(ret, nodeToInt(n))
	}
	return ret
}

func nodeToInt(n *Node) int {
	i, _ := strconv.Atoi(n.Class.Name())
	return i
}

func testGraph(rules []korrel8r.Rule) *Graph {
	d := NewData()
	for _, r := range rules {
		d.addRule(r)
	}
	return d.FullGraph()
}

func graphRules(g *Graph) (rules []korrel8r.Rule) {
	g.EachLine(func(l *Line) { rules = append(rules, l.Rule) })
	mock.SortRules(rules)
	return rules
}

// assertComponentOrder components is an ordered list of unordered sets of rules.
// Asserts that the rules list is in an order that is compatible with components
func assertComponentOrder(t *testing.T, components [][]string, rules []string) bool {
	msg := "out of order\nrules:      %v\ncomponents: %v\n"
	t.Helper()
	j := 0 // rules index
	for i, c := range components {
		if !assert.LessOrEqual(t, j+len(c), len(rules), "rule[%v], component[%v] len %v\n"+msg, j, i, len(c), rules, components) {
			return false
		}
		slices.Sort(c)
		sub := rules[j : j+len(c)]
		slices.Sort(sub)
		if !assert.Equal(t, c, sub, msg, rules, components) {
			return false
		}
		j += len(c)
		if j >= len(rules) {
			break
		}
	}
	return true
}
