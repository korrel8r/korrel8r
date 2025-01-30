// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestK8sRules(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "LogToPod",
			start: log.NewObject(`{"kubernetes":{"namespace_name":"foo","pod_name":"bar"},"message":"hello"}`),
			query: `k8s:Pod.v1:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "LogToPod",
			start: log.NewObject(`{"kubernetes":{"namespace_name":"default","pod_name":"baz"},"message":"bye"}`),
			query: `k8s:Pod.v1:{"namespace":"default","name":"baz"}`,
		},
		{
			rule: "SelectorToPods",
			start: k8s.Object{
				"kind": "Deployment", "apiVersion": "v1",
				"metadata": k8s.Object{"name": "x", "namespace": "ns"},
				"spec": k8s.Object{
					"selector": k8s.Object{"matchLabels": k8s.Object{"test": "testme"}},
					"template": k8s.Object{"metadata": k8s.Object{"name": "x", "namespace": "ns"}}},
			},
			query: `k8s:Pod.v1:{"namespace":"ns","labels":{"test":"testme"}}`,
		},
		{
			rule:  "EventToAll",
			start: k8sEvent(newK8s("Pod", "aNamespace", "foo"), "a"),
			query: `k8s:Pod.v1:{"namespace":"aNamespace","name":"foo"}`,
		},
		{
			rule:  "AllToEvent",
			start: newK8s("Pod", "aNamespace", "foo"),
			query: `k8s:Event.v1:{"fields":{"involvedObject.apiVersion":"v1","involvedObject.kind":"Pod","involvedObject.name":"foo","involvedObject.namespace":"aNamespace"}}`,
		},
		{
			rule:  "AllToMetric",
			start: newK8s("Pod", "aNamespace", "foo"),
			query: `metric:metric:{namespace="aNamespace",pod="foo"}`,
		},
		{
			rule:  "PodToAlert",
			start: newK8s("Pod", "aNamespace", "foo"),
			query: `alert:alert:{"namespace":"aNamespace","pod":"foo"}`,
		},
	} {
		x.Run(t)
	}
}

func TestDomain_Classes(t *testing.T) {
	e := setup()
	// List of k8s classes used in the standard configuration, must be updated
	// if the configuration changes.
	want := []string{"k8s:Deployment.v1.apps",
		"k8s:Pod.v1",
		"k8s:PodDisruptionBudget.v1.policy",
		"k8s:DaemonSet.v1.apps",
		"k8s:StatefulSet.v1.apps",
		"k8s:PersistentVolumeClaim.v1",
		"k8s:ReplicationController.v1",
		"k8s:Service.v1",
		"k8s:ReplicaSet.v1.apps",
		"k8s:Job.v1.batch",
		"k8s:Event.v1",
		"k8s:Secret.v1",
		"k8s:ConfigMap.v1",
		"k8s:CronJob.v1.batch",
		"k8s:HorizontalPodAutoscaler.v1.autoscaling",
		"k8s:NetworkPolicy.v1.networking.k8s.io",
		"k8s:PersistentVolume.v1",
		"k8s:StorageClass.v1.storage.k8s.io",
		"k8s:VolumeAttachment.v1.storage.k8s.io",
		"k8s:ServiceAccount.v1",
		"k8s:Role.v1.rbac.authorization.k8s.io",
		"k8s:RoleBinding.v1.rbac.authorization.k8s.io",
		"k8s:ClusterRole.v1.rbac.authorization.k8s.io",
		"k8s:ClusterRoleBinding.v1.rbac.authorization.k8s.io", "k8s:Node.v1"}
	d, err := e.Domain("k8s")
	require.NoError(t, err)
	var names []string
	for _, c := range d.Classes() {
		names = append(names, c.String())
	}
	assert.ElementsMatch(t, want, names)
}
