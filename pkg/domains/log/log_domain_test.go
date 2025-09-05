// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log_test

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/stretchr/testify/require"
)

func fixture(t testing.TB) *domain.Fixture {
	c := test.RequireCluster(t)
	const n = 10
	namespace := test.TempNamespace(t, c, "podlog-").Name
	ctx := t.Context()
	require.NoError(t, c.Create(ctx, logger(namespace, "foo", "hello", 1, n, "box")))
	test.WaitForPodReady(t, c, namespace, "foo")
	q, err := log.NewQuery(fmt.Sprintf(`log:application:{kubernetes_namespace_name="%v"}`, namespace))
	require.NoError(t, err)
	return &domain.Fixture{Query: q}
}

func TestLogDomain(t *testing.T)      { fixture(t).Test(t) }
func BenchmarkLogDomain(b *testing.B) { fixture(b).Benchmark(b) }
