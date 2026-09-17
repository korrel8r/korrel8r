// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
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
	benchmarkNode      *graph.Node
)

func BenchmarkDataNodeFor(b *testing.B) {
	e := makeEngine(b)
	start := benchmarkStart(b, e)
	data := e.GraphData()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkNode = data.NodeFor(start.Class)
	}
}

func BenchmarkDataLinesFrom(b *testing.B) {
	e := makeEngine(b)
	start := benchmarkStart(b, e)
	data := e.GraphData()
	node := data.NodeFor(start.Class)
	require.NotNil(b, node)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		count := 0
		data.EachLineFromID(node.ID(), func(*graph.Line) { count++ })
		benchmarkLineCount = count
	}
}

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

func BenchmarkRuleGraph(b *testing.B) {
	rules := makeEngine(b).Rules()

	b.Run("Data", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = graph.NewData(rules...)
		}
	})
	b.Run("SharedGraph", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			graph.NewData(rules...).SharedGraph()
		}
	})
}

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
