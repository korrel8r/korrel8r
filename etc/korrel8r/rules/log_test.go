// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
)

func TestSelectorToLogsRules(t *testing.T) {
	e := setup()
	// Verify rules selected the correct set of start classes
	classes := unique.NewList[korrel8r.Class]()
	for _, r := range e.Rules() {
		if r.Name() == "SelectorToLogs" {
			classes.Append(r.Start()...)
		}
	}
	want := []korrel8r.Class{
		k8s.Class{Group: "", Version: "v1", Kind: "ReplicationController"},
		k8s.Class{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
		k8s.Class{Group: "", Version: "v1", Kind: "Service"},
		k8s.Class{Group: "apps", Version: "v1", Kind: "Deployment"},
		k8s.Class{Group: "batch", Version: "v1", Kind: "Job"},
		k8s.Class{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"},
		k8s.Class{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		k8s.Class{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		k8s.Class{Group: "apps", Version: "v1", Kind: "ReplicaSet"}}
	assert.ElementsMatch(t, want, classes.List, "%#v", classes.List)
}

func TestLogRules(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule: "SelectorToLogs",
			start: k8s.Object{
				"metadata": k8s.Object{"namespace": "ns", "name": "x"},
				"spec": k8s.Object{
					"selector": k8s.Object{"matchLabels": k8s.Object{"a.b/c": "x"}},
				}},
			query: `log:application:{kubernetes_namespace_name="ns"}|json|kubernetes_labels_a_b_c="x"`,
		},
		{
			rule:  "PodToLogs",
			start: newK8s("Pod", "project", "application"),
			query: `log:application:{kubernetes_namespace_name="project",kubernetes_pod_name="application"}`,
		},
		{
			rule:  "PodToLogs",
			start: newK8s("Pod", "kube-something", "infrastructure"),
			query: `log:infrastructure:{kubernetes_namespace_name="kube-something",kubernetes_pod_name="infrastructure"}`,
		},
	} {
		x.Run(t)
	}
}
