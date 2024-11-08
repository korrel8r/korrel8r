// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package trace_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/trace"
)

// TODO tempo limits number of traces, not spans. Remove SkipCluster when fixed.
var fixture = domain.Fixture{Query: trace.NewQuery(`{}`), SkipCluster: true}

func TestTraceDomain(t *testing.T)     { fixture.Test(t) }
func BenchmarTraceDomain(b *testing.B) { fixture.Benchmark(b) }
