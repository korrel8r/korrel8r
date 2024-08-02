// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
)

var fixture = domain.Fixture{Query: log.NewQuery(log.Infrastructure, `{kubernetes_namespace_name=~".+"}`)}

func TestLogDomain(t *testing.T)      { fixture.Test(t) }
func BenchmarLogkDomain(b *testing.B) { fixture.Benchmark(b) }
