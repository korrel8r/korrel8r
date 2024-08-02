// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package netflow_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
)

var fixture = domain.Fixture{Query: netflow.NewQuery(`{DstK8S_Namespace=~".+"}`)}

func TestNetflowDomain(t *testing.T)      { fixture.Test(t) }
func BenchmarkNetflowDomain(b *testing.B) { fixture.Benchmark(b) }
