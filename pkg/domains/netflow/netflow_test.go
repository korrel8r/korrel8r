// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package netflow_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/internal/pkg/test/domain"
	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
)

var fixture = domain.Fixture{Query: netflow.NewQuery(`{DstK8S_Namespace="netobserv"}`)}

func TestNetflowDomain(t *testing.T)      { fixture.Test(t) }
func BenchmarkNetflowDomain(b *testing.B) { fixture.Benchmark(b) }

func TestNewObject(t *testing.T) {
	o := netflow.NewObject(&loki.Log{
		Body:   `{"SrcAddr":"10.0.0.1","DstPort":80,"Proto":6}`,
		Labels: map[string]string{"SrcAddr": "label-override", "stream": "network"},
	})
	if got := o["SrcAddr"]; got != "label-override" {
		t.Fatalf("SrcAddr: got %v, want label-override", got)
	}
	if got := o["DstPort"]; got != float64(80) {
		t.Fatalf("DstPort: got %v, want 80", got)
	}
	if got := o["stream"]; got != "network" {
		t.Fatalf("stream: got %v, want network", got)
	}
}

func BenchmarkNewObject(b *testing.B) {
	entry := &loki.Log{
		Body: `{"SrcAddr":"10.0.0.1","DstAddr":"10.0.0.2","SrcPort":45321,"DstPort":443,"Proto":6,"SrcK8S_Namespace":"source","DstK8S_Namespace":"destination","Bytes":12345,"Packets":42,"IfDirections":[0,1]}`,
		Labels: map[string]string{
			"job": "network-flows", "namespace": "netobserv", "stream": "network",
		},
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = netflow.NewObject(entry)
	}
}
