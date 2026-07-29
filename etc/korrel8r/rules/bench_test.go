// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

func BenchmarkRuleApply(b *testing.B) {
	e := setup()
	type bench struct {
		rule  string
		start korrel8r.Object
	}
	benches := []bench{
		// alert templates
		{"AlertToDeployment", &alert.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}}},
		{"AlertToDaemonSet", &alert.Object{Labels: map[string]string{"namespace": "foo", "daemonset": "bar"}}},
		{"AlertToStatefulSet", &alert.Object{Labels: map[string]string{"namespace": "foo", "statefulset": "bar"}}},
		{"PodToAlert", newK8s("Pod", "ns", "pod1", nil)},
		{"DeploymentToAlert", newK8s("Deployment.apps", "ns", "dep1", nil)},

		// metric templates
		{"MetricToPod", metric.Object{Labels: map[string]string{"namespace": "foo", "pod": "bar"}, Fingerprint: "f"}},
		{"MetricToPod", metric.Object{Labels: map[string]string{"k8s_namespace_name": "foo", "k8s_pod_name": "bar"}, Fingerprint: "f"}},
		{"MetricToDeployment", metric.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}, Fingerprint: "f"}},
		{"MetricToNode", metric.Object{Labels: map[string]string{"node": "w1"}, Fingerprint: "f"}},
		{"AllToMetric", newK8s("Pod", "ns", "p1", nil)},

		// netflow templates
		{"NetflowToSrcK8s", map[string]any{"SrcK8S_Type": "Pod", "SrcK8S_Namespace": "ns", "SrcK8S_Name": "p1"}},
		{"NetflowToDstK8sOwner", map[string]any{"DstK8S_OwnerType": "Deployment", "DstK8S_Namespace": "ns", "DstK8S_OwnerName": "d1"}},

		// kubevirt templates
		{"AlertToVM", &alert.Object{Labels: map[string]string{"namespace": "ns", "name": "vm1"}}},
		{"VmToAlert", newK8s("VirtualMachine.kubevirt.io", "ns", "vm1", nil)},
		{"VmToSecret", newK8s("VirtualMachine.kubevirt.io", "ns", "vm1", k8s.Object{
			"spec": k8s.Object{"template": k8s.Object{"spec": k8s.Object{
				"volumes": []k8s.Object{{"secret": k8s.Object{"secretName": "s1"}}},
			}}},
		})},
		{"VmToConfigMap", newK8s("VirtualMachine.kubevirt.io", "ns", "vm1", k8s.Object{
			"spec": k8s.Object{"template": k8s.Object{"spec": k8s.Object{
				"volumes": []k8s.Object{{"configMap": k8s.Object{"name": "cm1"}}},
			}}},
		})},

		// k8s rules (no templates, baseline)
		{"SelectorToPods", newK8s("Deployment.apps", "ns", "dep1", k8s.Object{
			"spec": k8s.Object{"selector": k8s.Object{"matchLabels": k8s.Object{"app": "web"}}},
		})},
		{"DependentToOwner", newK8s("Pod", "ns", "pod1", k8s.Object{
			"metadata": k8s.Object{"ownerReferences": []k8s.Object{
				{"apiVersion": "apps/v1", "kind": "ReplicaSet", "name": "rs1"},
			}},
		})},
	}

	for _, bb := range benches {
		r := e.Rule(bb.rule)
		if r == nil {
			b.Fatalf("rule not found: %s", bb.rule)
		}
		b.Run(bb.rule, func(b *testing.B) {
			for b.Loop() {
				_, _ = r.Apply(bb.start)
			}
		})
	}
}
