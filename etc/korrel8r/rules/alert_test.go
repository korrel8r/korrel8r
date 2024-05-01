// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/stretchr/testify/assert"
)

func Test_AlertToDeployment(t *testing.T) {
	e := setup()
	for _, x := range []struct {
		rule  string
		start alert.Object
		want  string
	}{
		{
			rule:  "AlertToDeployment",
			start: alert.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}},
			want:  `k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToPod",
			start: alert.Object{Labels: map[string]string{"namespace": "foo", "pod": "bar"}},
			want:  `k8s:Pod.v1.:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToDaemonSet",
			start: alert.Object{Labels: map[string]string{"namespace": "foo", "daemonset": "bar"}},
			want:  `k8s:DaemonSet.v1.apps:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToStatefulSet",
			start: alert.Object{Labels: map[string]string{"namespace": "foo", "statefulset": "bar"}},
			want:  `k8s:StatefulSet.v1.apps:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "AlertToMetric",
			start: alert.Object{Expression: "this is an expression"},
			want:  `metric:metric:this is an expression`,
		},
	} {
		t.Run(x.rule, func(t *testing.T) {
			tested(x.rule)
			got, err := e.Rule(x.rule).Apply(x.start)
			assert.NoError(t, err)
			assert.Equal(t, x.want, got.String())
		})
	}
}
