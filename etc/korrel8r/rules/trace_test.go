// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/trace"
	"github.com/stretchr/testify/assert"
)

func Test_TraceToPod(t *testing.T) {
	e := setup()
	for _, x := range []struct {
		rule  string
		start *trace.Span
		want  string
	}{
		{
			rule: "TraceToPod",
			start: &trace.Span{
				Context:    trace.SpanContext{TraceID: "232323", SpanID: "3d48369744164bd0"},
				Attributes: map[string]any{"k8s.namespace.name": "tracing-app-k6", "k8s.pod.name": "bar"},
			},
			want: `[k8s:Pod.v1:{"namespace":"tracing-app-k6","name":"bar"}]`,
		},
	} {
		t.Run(x.rule, func(t *testing.T) {
			tested(x.rule)
			got, err := e.Rule(x.rule).Apply(x.start)
			assert.NoError(t, err)
			assert.Equal(t, x.want, fmt.Sprintf("%v", got))
		})
	}
}

func Test_TraceFromPod(t *testing.T) {
	e := setup()
	for _, x := range []struct {
		rule  string
		start k8s.Object
		want  string
	}{
		{
			rule:  "PodToTrace",
			start: newK8s("Pod", "bar", "foo", nil),
			want:  `trace:span:{resource.k8s.namespace.name="bar"&&resource.k8s.pod.name="foo"}`,
		},
	} {
		t.Run(x.rule, func(t *testing.T) {
			tested(x.rule)
			got, err := e.Rule(x.rule).Apply(x.start)
			if assert.NoError(t, err) {
				assert.Equal(t, x.want, got[0].String())
			}
		})
	}
}
