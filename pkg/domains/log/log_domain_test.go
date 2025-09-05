// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/stretchr/testify/require"
)

var fixture = &domain.Fixture{
	Query: must.Must1(log.NewQuery(`log:application:{log_type="application"}`)),
	ClusterSetup: func(t testing.TB) bool {
		c := test.RequireCluster(t)
		namespace := test.TempNamespace(t, c, "podlog-").Name
		// Generate application logs so there is something to find.
		require.NoError(t, c.Create(t.Context(), logger(namespace, "foo", "hello", 1, 10, "box")))
		test.WaitForPodReady(t, c, namespace, "foo")
		return true
	},
}

func TestLogDomain(t *testing.T)      { fixture.Test(t) }
func BenchmarkLogDomain(b *testing.B) { fixture.Benchmark(b) }
