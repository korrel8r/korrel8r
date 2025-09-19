// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraverserGoals(t *testing.T) {
	b := mock.NewBuilder("d")
	e, err := engine.Build().Rules(b.Rules([][]any{
		// Return 2 results, must follow both
		{"ab", "d:a", "d:b", b.Query("d:b", "1", 1, 2)},
		// 2 rules, must follow both. Incorporate data from start object.
		{"bc1", "d:b", "d:c", func(start korrel8r.Object) (korrel8r.Query, error) {
			return b.Query("d:c", test.JSONString(start), start), nil
		}},
		{"bc2", "d:b", "d:c", func(start korrel8r.Object) (korrel8r.Query, error) {
			result := start.(int) + 10
			return b.Query("d:c", test.JSONString(result), result), nil
		}},
		{"dz", "d:c", "d:z", func(start korrel8r.Object) (korrel8r.Query, error) {
			return b.Query("d:z", test.JSONString(start), start), nil
		}},
	})...).Stores(b.Store("d", nil)).Engine()
	require.NoError(t, err)

	a := NewAsync(e, e.Graph())
	start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
	goal := b.Class("d:z")
	g, err := a.Goals(context.Background(), start, list(goal))
	if assert.NoError(t, err) {
		lines := []string{
			"\"ab\"(d:a->d:b)",
			"\"bc1\"(d:b->d:c)",
			"\"bc2\"(d:b->d:c)",
			"\"dz\"(d:c->d:z)",
		}
		assert.ElementsMatch(t, lines, lineStrings(g), "%#v", lineStrings(g))

		nodes := []string{
			"d:a [0]",
			"d:b [1,2]",
			"d:c [1,11,12,2]",
			"d:z [1,11,12,2]",
		}
		assert.ElementsMatch(t, nodes, nodeStrings(g), "%#v", nodeStrings(g))
	}
}

func TestTraverserNeighbours(t *testing.T) {
	b := mock.NewBuilder("d")
	e, err := engine.Build().Rules(b.Rules([][]any{
		{"ab", "d:a", "d:b", b.Query("d:b", "ab", 1)},
		{"ac", "d:a", "d:c", b.Query("d:c", "ac", 2)},
		{"bx", "d:b", "d:x", b.Query("d:x", "bx", 3)},
		{"cy", "d:c", "d:y", b.Query("d:y", "cy", 4)},
		{"yz", "d:y", "d:z", b.Query("d:z", "yz", 5)},
		{"zq", "d:z", "d:q", b.Query("d:q", "zq", 6)},
	})...).Stores(b.Store("d", nil)).Engine()
	require.NoError(t, err)

	for _, x := range []struct {
		depth int
		lines []string
		nodes []string
	}{
		{
			depth: 4,
			lines: []string{
				"\"ab\"(d:a->d:b)",
				"\"ac\"(d:a->d:c)",
				"\"bx\"(d:b->d:x)",
				"\"cy\"(d:c->d:y)",
				"\"yz\"(d:y->d:z)",
				"\"zq\"(d:z->d:q)",
			},
			nodes: []string{
				"d:a [0]",
				"d:b [1]",
				"d:c [2]",
				"d:x [3]",
				"d:y [4]",
				"d:z [5]",
				"d:q [6]",
			},
		},
		{
			depth: 3,
			lines: []string{
				"\"ab\"(d:a->d:b)",
				"\"ac\"(d:a->d:c)",
				"\"bx\"(d:b->d:x)",
				"\"cy\"(d:c->d:y)",
				"\"yz\"(d:y->d:z)",
			},
			nodes: []string{
				"d:a [0]",
				"d:b [1]",
				"d:c [2]",
				"d:x [3]",
				"d:y [4]",
				"d:z [5]",
			},
		},
		{
			depth: 2,
			lines: []string{
				"\"ab\"(d:a->d:b)",
				"\"ac\"(d:a->d:c)",
				"\"bx\"(d:b->d:x)",
				"\"cy\"(d:c->d:y)",
			},
			nodes: []string{
				"d:a [0]",
				"d:b [1]",
				"d:c [2]",
				"d:x [3]",
				"d:y [4]",
			},
		},
		{
			depth: 1, lines: []string{
				"\"ab\"(d:a->d:b)",
				"\"ac\"(d:a->d:c)",
			},
			nodes: []string{
				"d:a [0]",
				"d:b [1]",
				"d:c [2]",
			},
		},
		{
			depth: 0,
			lines: []string{},
			nodes: []string{
				"d:a [0]",
			},
		},
	} {
		t.Run(fmt.Sprintf("depth %v", x.depth), func(t *testing.T) {
			start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
			g, err := NewAsync(e, e.Graph()).Neighbours(context.Background(), start, x.depth)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.lines, lineStrings(g), "%#v", lineStrings(g))
				assert.ElementsMatch(t, x.nodes, nodeStrings(g), "%#v", nodeStrings(g))
			}
		})
	}
}

func list[T any](x ...T) []T {
	return x
}

func lineStrings(g *graph.Graph) (lines []string) {
	g.EachLine(func(l *graph.Line) { lines = append(lines, l.String()) })
	return lines
}

func nodeStrings(g *graph.Graph) (nodes []string) {
	g.EachNode(func(n *graph.Node) {
		nodes = append(nodes, fmt.Sprintf("%v %v", n.Class.String(), resultString(n.Result.List())))
	})
	return nodes
}

func resultString(list []any) string {
	slices.SortFunc(list, func(a, b any) int {
		ja, jb := test.JSONString(a), test.JSONString(b)
		return cmp.Compare(ja, jb)
	})
	return test.JSONString(list)
}
