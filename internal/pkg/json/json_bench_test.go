// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package json_test

import (
	stdjson "encoding/json"
	"strings"
	"testing"

	sonicjson "github.com/korrel8r/korrel8r/internal/pkg/json"
)

type alertObject struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Status      string            `json:"status"`
	StartsAt    string            `json:"startsAt"`
	Value       string            `json:"value"`
	Expression  string            `json:"expression"`
	Fingerprint string            `json:"fingerprint"`
	EndsAt      string            `json:"endsAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

type graphResponse struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

type graphNode struct {
	Class   string            `json:"class"`
	Count   int               `json:"count"`
	Queries []graphQueryCount `json:"queries"`
}

type graphEdge struct {
	Start string      `json:"start"`
	Goal  string      `json:"goal"`
	Rules []graphRule `json:"rules"`
}

type graphRule struct {
	Name    string            `json:"name"`
	Queries []graphQueryCount `json:"queries"`
}

type graphQueryCount struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

var testAlert = alertObject{
	Labels: map[string]string{
		"alertname": "KubeDeploymentReplicasMismatch",
		"namespace": "openshift-monitoring",
		"severity":  "warning",
		"service":   "prometheus-k8s",
		"pod":       "prometheus-k8s-0",
		"container": "prometheus",
	},
	Annotations: map[string]string{
		"summary":     "Deployment has not matched the expected number of replicas.",
		"description": "Deployment openshift-monitoring/prometheus-k8s has not matched the expected number of replicas for longer than 15 minutes.",
		"runbook_url": "https://runbooks.prometheus-operator.dev/runbooks/kubernetes/kubedeploymentreplicasmismatch",
	},
	Status:      "firing",
	StartsAt:    "2024-01-15T10:30:00Z",
	Value:       "1",
	Expression:  "kube_deployment_status_replicas_available != kube_deployment_spec_replicas",
	Fingerprint: "abc123def456",
	EndsAt:      "0001-01-01T00:00:00Z",
	UpdatedAt:   "2024-01-15T10:35:00Z",
}

func makeGraph(n int) graphResponse {
	g := graphResponse{}
	for i := range n {
		g.Nodes = append(g.Nodes, graphNode{
			Class: "log:application",
			Count: i * 10,
			Queries: []graphQueryCount{
				{Query: "log:application:{kubernetes_namespace_name=\"test\"}", Count: i * 10},
			},
		})
		if i > 0 {
			g.Edges = append(g.Edges, graphEdge{
				Start: "k8s:Pod",
				Goal:  "log:application",
				Rules: []graphRule{{Name: "k8sPodToLokiLog", Queries: []graphQueryCount{
					{Query: "log:application:{kubernetes_namespace_name=\"test\"}", Count: i * 5},
				}}},
			})
		}
	}
	return g
}

var alertJSON []byte
var graphSmallJSON, graphLargeJSON []byte

func init() {
	alertJSON, _ = stdjson.Marshal(testAlert) //nolint:errcheck
	g := makeGraph(5)
	graphSmallJSON, _ = stdjson.Marshal(g) //nolint:errcheck
	g = makeGraph(50)
	graphLargeJSON, _ = stdjson.Marshal(g) //nolint:errcheck
}

func BenchmarkMarshalAlert(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for b.Loop() {
			_, _ = stdjson.Marshal(testAlert)
		}
	})
	b.Run("sonic", func(b *testing.B) {
		for b.Loop() {
			_, _ = sonicjson.Marshal(testAlert)
		}
	})
}

func BenchmarkUnmarshalAlert(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for b.Loop() {
			var o alertObject
			_ = stdjson.Unmarshal(alertJSON, &o)
		}
	})
	b.Run("sonic", func(b *testing.B) {
		for b.Loop() {
			var o alertObject
			_ = sonicjson.Unmarshal(alertJSON, &o)
		}
	})
}

func BenchmarkMarshalGraphSmall(b *testing.B) {
	g := makeGraph(5)
	b.Run("std", func(b *testing.B) {
		for b.Loop() {
			_, _ = stdjson.Marshal(g)
		}
	})
	b.Run("sonic", func(b *testing.B) {
		for b.Loop() {
			_, _ = sonicjson.Marshal(g)
		}
	})
}

func BenchmarkMarshalGraphLarge(b *testing.B) {
	g := makeGraph(50)
	b.Run("std", func(b *testing.B) {
		for b.Loop() {
			_, _ = stdjson.Marshal(g)
		}
	})
	b.Run("sonic", func(b *testing.B) {
		for b.Loop() {
			_, _ = sonicjson.Marshal(g)
		}
	})
}

func BenchmarkUnmarshalGraphSmall(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for b.Loop() {
			var g graphResponse
			_ = stdjson.Unmarshal(graphSmallJSON, &g)
		}
	})
	b.Run("sonic", func(b *testing.B) {
		for b.Loop() {
			var g graphResponse
			_ = sonicjson.Unmarshal(graphSmallJSON, &g)
		}
	})
}

func BenchmarkUnmarshalGraphLarge(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for b.Loop() {
			var g graphResponse
			_ = stdjson.Unmarshal(graphLargeJSON, &g)
		}
	})
	b.Run("sonic", func(b *testing.B) {
		for b.Loop() {
			var g graphResponse
			_ = sonicjson.Unmarshal(graphLargeJSON, &g)
		}
	})
}

func BenchmarkDecoder(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		for b.Loop() {
			r := strings.NewReader(string(graphLargeJSON))
			var g graphResponse
			_ = stdjson.NewDecoder(r).Decode(&g)
		}
	})
	b.Run("sonic", func(b *testing.B) {
		for b.Loop() {
			r := strings.NewReader(string(graphLargeJSON))
			var g graphResponse
			_ = sonicjson.NewDecoder(r).Decode(&g)
		}
	})
}
