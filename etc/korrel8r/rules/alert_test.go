// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
)

func TestAlertTo(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "AlertToPod",
			start: &alert.Object{Labels: map[string]string{"namespace": "foo", "pod": "bar"}},
			query: `k8s:Pod.v1:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToDeployment",
			start: &alert.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}},
			query: `k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToDaemonSet",
			start: &alert.Object{Labels: map[string]string{"namespace": "foo", "daemonset": "bar"}},
			query: `k8s:DaemonSet.v1.apps:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToStatefulSet",
			start: &alert.Object{Labels: map[string]string{"namespace": "foo", "statefulset": "bar"}},
			query: `k8s:StatefulSet.v1.apps:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToPodDisruptionBudget",
			start: &alert.Object{Labels: map[string]string{"namespace": "foo", "poddisruptionbudget": "bar"}},
			query: `k8s:PodDisruptionBudget.v1.policy:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToMetric",
			start: &alert.Object{Expression: "this is an expression"},
			query: `metric:metric:this is an expression`,
		},
	} {
		x.Run(t)
	}
}
