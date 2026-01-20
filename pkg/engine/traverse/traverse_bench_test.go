// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse_test

import (
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/require"
)

const delay = time.Millisecond

func makeEngine(b *testing.B) *engine.Engine {
	b.Helper()
	// Load configuration that uses mock store for all domains with proper rules
	configs, err := config.Load("testdata/use_mock_store.yaml")
	require.NoError(b, err)

	// Create engine with all domains and rules
	e, err := engine.Build().
		Domains(domains.All...).
		Config(configs).
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

func BenchmarkNeighbors(b *testing.B) {
	e := makeEngine(b)
	const depth = 3
	query, err := e.Query(`k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}`)
	require.NoError(b, err)
	start := traverse.Start{Class: query.Class(), Queries: []korrel8r.Query{query}}
	b.ResetTimer()
	for b.Loop() {
		_, err := traverse.Neighbors(b.Context(), e, start, depth)
		require.NoError(b, err)
	}
}

func BenchmarkGoals(b *testing.B) {
	e := makeEngine(b)
	goals := []korrel8r.Class{log.Infrastructure}
	query, err := e.Query(`k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}`)
	require.NoError(b, err)
	start := traverse.Start{Class: query.Class(), Queries: []korrel8r.Query{query}}
	b.ResetTimer()
	for b.Loop() {
		_, err := traverse.Goals(b.Context(), e, start, goals)
		require.NoError(b, err)
	}
}
