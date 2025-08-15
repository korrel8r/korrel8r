// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
)

func TestK8sRules(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "LogToPod",
			start: log.Object{"kubernetes_namespace_name": "foo", "kubernetes_pod_name": "bar", "message": "hello"},
			query: `k8s:Pod.v1:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "LogToPod",
			start: log.Object{"kubernetes_namespace_name": "default", "kubernetes_pod_name": "baz", "message": "bye"},
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
			start: k8sEvent(newK8s("Pod", "aNamespace", "foo", nil), "a"),
			query: `k8s:Pod.v1:{"namespace":"aNamespace","name":"foo"}`,
		},
		{
			rule:  "AllToEvent",
			start: newK8s("Pod", "aNamespace", "foo", nil),
			query: `k8s:Event.v1:{"fields":{"involvedObject.apiVersion":"v1","involvedObject.kind":"Pod","involvedObject.name":"foo","involvedObject.namespace":"aNamespace"}}`,
		},
		{
			rule:  "AllToMetric",
			start: newK8s("Pod", "aNamespace", "foo", nil),
			query: `metric:metric:{namespace="aNamespace",pod="foo"}`,
		},
		{
			rule:  "PodToAlert",
			start: newK8s("Pod", "aNamespace", "foo", nil),
			query: `alert:alert:{"namespace":"aNamespace","pod":"foo"}`,
		},
		{
			rule: "DependentToOwner",
			start: newK8s("Pod", "aNamespace", "foo", k8s.Object{
				"metadata": k8s.Object{
					"ownerReferences": []k8s.Object{{
						"name":       "owner",
						"kind":       "Deployment",
						"apiVersion": "apps/v1",
					}}},
			}),
			query: `k8s:Deployment.v1.apps:{"namespace":"aNamespace","name":"owner"}`,
		},
	} {
		x.Run(t)
	}
}
