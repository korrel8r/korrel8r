// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"fmt"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains"
	logdomain "github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules/quickrules"
	"github.com/stretchr/testify/require"
)

const delay = time.Microsecond

func makeEngine(b testing.TB) *engine.Engine {
	b.Helper()
	// Load configuration that uses mock store for all domains with proper rules
	configs, err := config.Load("testdata/use_mock_store.yaml")
	require.NoError(b, err)

	// Match the CLI engine: configuration rules plus compiled quickrules.
	builder := engine.Build()
	e, err := builder.
		Domains(domains.All...).
		Config(configs).
		Rules(quickrules.Rules(builder.GetDomains())...).
		StatusRules(quickrules.StatusRules(builder.GetDomains())...).
		Engine()
	require.NoError(b, err)

	// Set a delay on all mock stores
	for _, d := range e.Domains() {
		for _, s := range e.StoresFor(d) {
			if s, ok := s.(*mock.Store); ok {
				s.Delay = delay
			}
		}
	}
	return e
}

const (
	benchmarkDepth = 3
	benchmarkQuery = `k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}`
)

func benchmarkStart(b *testing.B, e *engine.Engine) Start {
	b.Helper()
	query, err := e.Query(benchmarkQuery)
	require.NoError(b, err)
	return Start{Class: query.Class(), Queries: []korrel8r.Query{query}}
}

func validateNeighborGraph(b *testing.B, g *graph.Graph) {
	b.Helper()
	nodes, edges, queries := 0, 0, 0
	g.EachNode(func(n *graph.Node) {
		nodes++
		queries += len(n.Queries)
	})
	g.EachEdge(func(*graph.Edge) { edges++ })
	require.Positive(b, nodes)
	require.Positive(b, edges)
	require.Positive(b, queries)
}

// BenchmarkNeighbors measures a complete depth-limited neighborhood traversal.
// The mock scenario is warmed up and checked for non-empty results before timing.
func BenchmarkNeighbors(b *testing.B) {
	e := makeEngine(b)
	start := benchmarkStart(b, e)

	// Validate the scenario before excluding setup and validation from measurements.
	g, err := Neighbors(b.Context(), e, start, benchmarkDepth)
	require.NoError(b, err)
	validateNeighborGraph(b, g)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := Neighbors(b.Context(), e, start, benchmarkDepth)
		require.NoError(b, err)
	}
}

// BenchmarkNeighborScope measures direct BFS selection of the depth-limited rule scope.
func BenchmarkNeighborScope(b *testing.B) {
	e := makeEngine(b)
	start := benchmarkStart(b, e)
	data := e.GraphData()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := neighborScope(data, start.Class, benchmarkDepth)
		require.NoError(b, err)
	}
}

var (
	benchmarkLineCount int
	benchmarkNodeID    int64
)

// BenchmarkDataNodeID measures ID lookup by stable class identity.
func BenchmarkDataNodeID(b *testing.B) {
	e := makeEngine(b)
	start := benchmarkStart(b, e)
	data := e.GraphData()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkNodeID, _ = data.NodeID(start.Class)
	}
}

// BenchmarkDataLinesFrom measures allocation-free iteration over outgoing adjacency.
func BenchmarkDataLinesFrom(b *testing.B) {
	e := makeEngine(b)
	start := benchmarkStart(b, e)
	data := e.GraphData()
	nodeID, ok := data.NodeID(start.Class)
	require.True(b, ok)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		count := 0
		data.EachLineIDFrom(nodeID, func(int) { count++ })
		benchmarkLineCount = count
	}
}

// BenchmarkNewTraverser measures construction of mutable state for a selected scope.
func BenchmarkNewTraverser(b *testing.B) {
	e := makeEngine(b)
	start := benchmarkStart(b, e)
	data := e.GraphData()
	scope, err := neighborScope(data, start.Class, benchmarkDepth)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = newTraverser(e, data, scope, nil, benchmarkDepth)
	}
}

// BenchmarkRuleGraph measures immutable rule topology construction. The Data subbenchmark
// excludes the Gonum adapter; View includes creation of its read-only view.
func BenchmarkRuleGraph(b *testing.B) {
	rules := makeEngine(b).Rules()

	b.Run("Data", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = graph.NewData(rules...)
		}
	})
	b.Run("View", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkView = graph.NewData(rules...).Graph()
		}
	})
}

var (
	benchmarkView        graph.View
	benchmarkDenseLines  []*graph.Line
	benchmarkSparseLines map[int]*graph.Line
)

// BenchmarkLineStorage compares complete mutable-line storage lifecycles with
// identical topology and active lines. Include zero, sparse and fully active
// searches; these numbers do not include scope selection or query execution.
func BenchmarkLineStorage(b *testing.B) {
	data := makeEngine(b).GraphData()
	nodes := make([]*graph.Node, data.NodeCount())
	for id := range nodes {
		nodes[id] = data.NewNode(int64(id))
	}
	newLine := func(id int) *graph.Line {
		from, to := data.Endpoints(id)
		return data.NewLine(id, nodes[from], nodes[to])
	}
	for _, active := range []int{0, 8, 64, data.LineCount()} {
		b.Run(fmt.Sprintf("active=%d", active), func(b *testing.B) {
			b.Run("Dense", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					lines := make([]*graph.Line, data.LineCount())
					for id := 0; id < active; id++ {
						lines[id] = newLine(id)
					}
					benchmarkDenseLines = lines
				}
				benchmarkDenseLines = nil
			})
			b.Run("Sparse", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var lines map[int]*graph.Line
					for id := 0; id < active; id++ {
						if lines == nil {
							lines = make(map[int]*graph.Line)
						}
						lines[id] = newLine(id)
					}
					benchmarkSparseLines = lines
				}
				benchmarkSparseLines = nil
			})
		})
	}
}

// BenchmarkGoals measures a complete goal-directed traversal to the infrastructure log class.
func BenchmarkGoals(b *testing.B) {
	e := makeEngine(b)
	goals := []korrel8r.Class{logdomain.Infrastructure}
	start := benchmarkStart(b, e)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := Goals(b.Context(), e, start, goals)
		require.NoError(b, err)
	}
}
