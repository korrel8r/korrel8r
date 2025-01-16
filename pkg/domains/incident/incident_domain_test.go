// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package incident_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/incident"
)

var fixture = domain.Fixture{Query: incident.Query{}, SkipCluster: true}

func TestAlertDomain(t *testing.T)      { fixture.Test(t) }
func BenchmarkAlertDomain(b *testing.B) { fixture.Benchmark(b) }
