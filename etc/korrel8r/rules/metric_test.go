// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/stretchr/testify/assert"
)

func TestMetricRules(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "AllToMetric",
			start: newK8s("Pod", "aNamespace", "foo", nil),
			want:  []string{`metric:metric:{namespace="aNamespace",pod="foo"}`},
		},

		// Standard Prometheus labels
		{
			rule:  "MetricToPod",
			start: metric.Object{Labels: map[string]string{"namespace": "foo", "pod": "bar"}, Fingerprint: "abc123"},
			want:  []string{`k8s:Pod.v1:{"namespace":"foo","name":"bar"}`},
		},
		// OTel labels
		{
			rule:  "MetricToPod",
			start: metric.Object{Labels: map[string]string{"k8s_namespace_name": "foo", "k8s_pod_name": "bar"}, Fingerprint: "abc123"},
			want:  []string{`k8s:Pod.v1:{"namespace":"foo","name":"bar"}`},
		},

		// Standard Prometheus labels
		{
			rule:  "MetricToDeployment",
			start: metric.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}, Fingerprint: "abc123"},
			want:  []string{`k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`},
		},
		// OTel labels
		{
			rule:  "MetricToDeployment",
			start: metric.Object{Labels: map[string]string{"k8s_namespace_name": "foo", "k8s_deployment_name": "bar"}, Fingerprint: "abc123"},
			want:  []string{`k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`},
		},

		// Standard Prometheus labels
		{
			rule:  "MetricToDaemonSet",
			start: metric.Object{Labels: map[string]string{"namespace": "foo", "daemonset": "bar"}, Fingerprint: "abc123"},
			want:  []string{`k8s:DaemonSet.v1.apps:{"namespace":"foo","name":"bar"}`},
		},
		// OTel labels
		{
			rule:  "MetricToDaemonSet",
			start: metric.Object{Labels: map[string]string{"k8s_namespace_name": "foo", "k8s_daemonset_name": "bar"}, Fingerprint: "abc123"},
			want:  []string{`k8s:DaemonSet.v1.apps:{"namespace":"foo","name":"bar"}`},
		},

		// Standard Prometheus labels
		{
			rule:  "MetricToStatefulSet",
			start: metric.Object{Labels: map[string]string{"namespace": "foo", "statefulset": "bar"}, Fingerprint: "abc123"},
			want:  []string{`k8s:StatefulSet.v1.apps:{"namespace":"foo","name":"bar"}`},
		},
		// OTel labels
		{
			rule:  "MetricToStatefulSet",
			start: metric.Object{Labels: map[string]string{"k8s_namespace_name": "foo", "k8s_statefulset_name": "bar"}, Fingerprint: "abc123"},
			want:  []string{`k8s:StatefulSet.v1.apps:{"namespace":"foo","name":"bar"}`},
		},

		// Standard Prometheus labels
		{
			rule:  "MetricToNode",
			start: metric.Object{Labels: map[string]string{"node": "worker-1"}, Fingerprint: "abc123"},
			want:  []string{`k8s:Node.v1:{"name":"worker-1"}`},
		},
		// OTel labels
		{
			rule:  "MetricToNode",
			start: metric.Object{Labels: map[string]string{"k8s_node_name": "worker-1"}, Fingerprint: "abc123"},
			want:  []string{`k8s:Node.v1:{"name":"worker-1"}`},
		},

	} {
		x.Run(t)
	}
}

func TestMetricRulesRequiredGuards(t *testing.T) {
	e := setup()
	for _, x := range []struct {
		rule  string
		start metric.Object
	}{
		{rule: "MetricToPod", start: metric.Object{Labels: map[string]string{"namespace": "foo"}}},
		{rule: "MetricToPod", start: metric.Object{Labels: map[string]string{"pod": "bar"}}},
		{rule: "MetricToDeployment", start: metric.Object{Labels: map[string]string{"namespace": "foo"}}},
		{rule: "MetricToDaemonSet", start: metric.Object{Labels: map[string]string{"namespace": "foo"}}},
		{rule: "MetricToStatefulSet", start: metric.Object{Labels: map[string]string{"namespace": "foo"}}},
		{rule: "MetricToNode", start: metric.Object{Labels: map[string]string{"namespace": "foo"}}},
	} {
		t.Run(fmt.Sprintf("%v_missing_label", x.rule), func(t *testing.T) {
			r := e.Rule(x.rule)
			if assert.NotNil(t, r) {
				_, err := r.Apply(x.start)
				assert.Error(t, err, "expected error for missing required label")
			}
		})
	}
}
