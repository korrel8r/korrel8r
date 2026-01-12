// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/engine"
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

func BenchmarkNeighbours(b *testing.B) {
	e := makeEngine(b)
	for _, q := range []string{
		`k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}`,
	} {
		query, err := e.Query(q)
		require.NoError(b, err)
		// Set up start parameters for traversal
		start := Start{
			Class:   query.Class(),
			Queries: []korrel8r.Query{query},
		}
		ctx := context.Background()
		for i := range 3 {
			b.ResetTimer()
			b.Run(fmt.Sprintf("%v depth=%v", q, i+1),
				func(b *testing.B) {
					for b.Loop() {
						// Execute a simple neighbor traversal, depth3, to benchmark the core traverse functionality
						_, err := Neighbors(ctx, e, start, i+1)
						require.NoError(b, err)
					}
				})
		}
	}
}
