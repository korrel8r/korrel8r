// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

var domain = mock.Domain("mock")

func c(i int) korrel8r.Class   { return domain.Class(fmt.Sprintf("%03v", i)) }
func r(i, j int) korrel8r.Rule { return mock.NewRule(c(i), c(j)) }

type rule = korrel8r.Rule

func testGraph(rules []rule) *Graph {
	d := NewData()
	for _, r := range rules {
		d.AddRule(r)
	}
	return d.NewGraph()
}

func graphRules(g *Graph) []rule {
	var rules []rule
	g.EachLine(func(l *Line) { rules = append(rules, l.Rule) })
	mock.SortRules(rules)
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
		if !assert.Equal(t, mock.SortRules(c), mock.SortRules(rules[j:j+len(c)]), msg, rules, components) {
			return false
		}
		j += len(c)
		if j >= len(rules) {
			break
		}
	}
	return true
}
