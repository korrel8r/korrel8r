// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	corev1 "k8s.io/api/core/v1"
)

var fixture = domain.Fixture{Query: k8s.NewQuery(k8s.ClassOf(&corev1.Pod{}), "", "", nil, nil)}

func TestLK8sDomain(t *testing.T)     { fixture.Test(t) }
func BenchmarkK8sDomain(b *testing.B) { fixture.Benchmark(b) }
