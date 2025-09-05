// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package domain_test

// Self-test for domain testing using a mock domain.

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
)

var fixture = domain.Fixture{
	Query:        mock.NewQuery(mock.NewDomain("mock").Class("thing"), "query"),
	ClusterSetup: func(testing.TB) bool { return false },
}

func TestMockDomain(t *testing.T)      { fixture.Test(t) }
func BenchmarkMockDomain(b *testing.B) { fixture.Benchmark(b) }
