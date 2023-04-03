package graph

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

func TestNewData(t *testing.T) {
	rules := []korrel8r.Rule{r(1, 2), r(3, 4), r(1, 3), r(2, 4)}
	d := NewData(rules...)
	assert.Equal(t, rules, d.Rules())
	assert.Equal(t, []korrel8r.Class{class(1), class(2), class(3), class(4)}, d.Classes())
}

func TestData_Graph(t *testing.T) {
	rules := []korrel8r.Rule{r(1, 2), r(3, 4), r(1, 3), r(2, 4)}
	d := NewData(rules...)
	g := d.EmptyGraph()
	assert.Equal(t, d, g.Data)
	assert.Equal(t, 0, g.Nodes().Len())
	assert.Equal(t, 0, g.Edges().Len())
}
