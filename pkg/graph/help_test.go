// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/stretchr/testify/assert"
)

var Domain = domain{}

type domain struct{}

func (d domain) Name() string                                       { return "graphmock" }
func (d domain) String() string                                     { return d.Name() }
func (d domain) Description() string                                { return "" }
func (d domain) Class(name string) korrel8r.Class                   { panic("not implemented") }
func (d domain) Classes() (classes []korrel8r.Class)                { panic("not implemented") }
func (d domain) Query(s string) (korrel8r.Query, error)             { panic("not implemented") }
func (d domain) Store(korrel8r.StoreConfig) (korrel8r.Store, error) { panic("not implemented") }

type Class int

func c(i int) korrel8r.Class { return Class(i) }

func (c Class) Domain() korrel8r.Domain  { return Domain }
func (c Class) Name() string             { return fmt.Sprintf("%v", int(c)) }
func (c Class) String() string           { return impl.ClassString(c) }
func (c Class) Description() string      { return "" }
func (c Class) ID(o korrel8r.Object) any { return int(c) }
func (c Class) New() korrel8r.Object     { panic("not implemented") }

func r(i, j int) korrel8r.Rule { return mock.NewRule("", c(i), c(j)) }

func testGraph(rules []korrel8r.Rule) *Graph {
	d := NewData()
	for _, r := range rules {
		d.AddRule(r)
	}
	return d.NewGraph()
}

func graphRules(g *Graph) (rules []korrel8r.Rule) {
	g.EachLine(func(l *Line) { rules = append(rules, l.Rule) })
	mock.SortRules(rules)
	return rules
}

// assertComponentOrder components is an ordered list of unordered sets of rules.
// Asserts that the rules list is in an order that is compatible with components
func assertComponentOrder(t *testing.T, components [][]korrel8r.Rule, rules []korrel8r.Rule) bool {
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
