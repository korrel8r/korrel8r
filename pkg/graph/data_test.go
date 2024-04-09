// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

func TestNewData(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }
	rules := []korrel8r.Rule{r(1, 2), r(3, 4), r(1, 3), r(2, 4)}
	d := NewData(rules...)
	assert.Equal(t, rules, d.Rules())
	assert.Equal(t, []korrel8r.Class{c(1), c(2), c(3), c(4)}, d.Classes())
}

func TestData_Graph(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }
	rules := []korrel8r.Rule{r(1, 2), r(3, 4), r(1, 3), r(2, 4)}
	d := NewData(rules...)
	g := d.EmptyGraph()
	assert.Equal(t, d, g.Data)
	assert.Equal(t, 0, g.Nodes().Len())
	assert.Equal(t, 0, g.Edges().Len())
}
