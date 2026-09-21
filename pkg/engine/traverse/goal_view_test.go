// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"fmt"
	"math"
	"testing"

	logdomain "github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/path"
)

// Verify the full mock-store goal workload against Gonum's map-backed graph,
// including expansion of path edges to parallel lines and query accounting.
func TestGoalViewMatchesFullGraph(t *testing.T) {
	e := makeEngine(t)
	data := e.GraphData()
	q, err := e.Query(benchmarkQuery)
	require.NoError(t, err)
	start := Start{Class: q.Class(), Queries: []korrel8r.Query{q}}
	goals := []korrel8r.Class{logdomain.Infrastructure}
	reference := data.FullGraph()
	var wantScope []int
	for _, goal := range goals {
		for _, p := range path.YenKShortestPaths(reference, math.MaxInt, 1, reference.NodeFor(start.Class), reference.NodeFor(goal)) {
			for i := 1; i < len(p); i++ {
				lines := reference.Lines(p[i-1].ID(), p[i].ID())
				for lines.Next() {
					wantScope = append(wantScope, int(lines.Line().ID()))
				}
			}
		}
	}
	gotScope, err := goalScope(data, start.Class, goals)
	require.NoError(t, err)
	assert.ElementsMatch(t, wantScope, gotScope)
	want, err := newTraverser(e, data, wantScope, nil, -1).run(t.Context(), start)
	require.NoError(t, err)
	want.RemoveEmptyGoalPaths(goals)
	got, err := Goals(t.Context(), e, start, goals)
	require.NoError(t, err)
	assert.Equal(t, want.LineStrings(), got.LineStrings())
	assert.Equal(t, want.NodeStrings(true), got.NodeStrings(true))
	accounting := func(g *graph.Graph) []string {
		var counts []string
		g.EachLine(func(l *graph.Line) {
			for q, count := range l.Queries {
				counts = append(counts, fmt.Sprintf("%d/%s/%d", l.ID(), q.String(), count.Count))
			}
		})
		return counts
	}
	assert.ElementsMatch(t, accounting(want), accounting(got))
}
