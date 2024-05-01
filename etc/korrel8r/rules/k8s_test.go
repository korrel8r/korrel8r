// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestPodToLogs(t *testing.T) {
	e := setup()
	for _, pod := range []*corev1.Pod{
		k8s.New[corev1.Pod]("project", "application"),
		k8s.New[corev1.Pod]("kube-something", "infrastructure"),
	} {
		t.Run(pod.Name, func(t *testing.T) {
			want := log.NewQuery(log.Class(pod.Name), fmt.Sprintf(`{kubernetes_namespace_name="%v",kubernetes_pod_name="%v"}`, pod.Namespace, pod.Name))

			testTraverse(t, e, k8s.ClassOf(pod), log.Domain.Class(pod.Name), []korrel8r.Object{pod}, want)
		})
	}
}

func TestLogToPod(t *testing.T) {
	e := setup()
	for _, o := range []log.Object{
		log.NewObject(`{"kubernetes":{"namespace_name":"foo","pod_name":"bar"}, "message":"hello"}`),
		log.NewObject(`{"kubernetes":{"namespace_name":"default","pod_name":"baz"}, "message":"bye"}`),
	} {
		t.Run(log.Preview(o), func(t *testing.T) {
			k := o["kubernetes"].(map[string]any)
			namespace := k["namespace_name"].(string)
			name := k["pod_name"].(string)
			start := log.Application
			if log.Preview(o) == "default" {
				start = log.Infrastructure
			}
			want := k8s.NewQuery(k8s.ClassOf(&corev1.Pod{}), namespace, name, nil, nil)
			testTraverse(t, e, start, k8s.ClassOf(&corev1.Pod{}), []korrel8r.Object{o}, want)
		})
	}
}

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

func TestSelectorToLogs(t *testing.T) {
	e := setup()
	d := k8s.New[appsv1.Deployment]("ns", "x")
	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"a.b/c": "x"}},
	}
	want := log.NewQuery(log.Application, `{kubernetes_namespace_name="ns"}|toJson|kubernetes_labels_a_b_c="x"|line_format"{{__line__}}"`)
	testTraverse(t, e, k8s.ClassOf(d), log.Domain.Class("application"), []korrel8r.Object{d}, want)
}

func TestSelectorToPods(t *testing.T) {
	e := setup()

	// Deployment
	labels := map[string]string{"test": "testme"}

	podx := k8s.New[corev1.Pod]("ns", "x")
	podx.ObjectMeta.Labels = labels
	podx.Spec = corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:    "testme",
			Image:   "quay.io/quay/busybox",
			Command: []string{"sh", "-c", "while true; do echo $(date) hello world; sleep 1; done"},
		}}}

	pody := podx.DeepCopy()
	pody.ObjectMeta.Name = "y"

	d := k8s.New[appsv1.Deployment]("ns", "x")
	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: podx.ObjectMeta,
			Spec:       podx.Spec,
		}}
	class := k8s.ClassOf(podx)
	testTraverse(t, e, k8s.ClassOf(d), class, []korrel8r.Object{d},
		k8s.NewQuery(class, "ns", "", client.MatchingLabels{"test": "testme"}, nil))
}

func TestK8sEvent(t *testing.T) {
	e := setup()
	pod := k8s.New[corev1.Pod]("aNamespace", "foo")
	event := k8s.EventFor(pod, "a")

	t.Run("PodToEvent", func(t *testing.T) {
		want := k8s.NewQuery(
			k8s.ClassOf(&corev1.Event{}), "", "", nil,
			client.MatchingFields{
				"involvedObject.apiVersion": "v1", "involvedObject.kind": "Pod",
				"involvedObject.name": "foo", "involvedObject.namespace": "aNamespace"})
		testTraverse(t, e, k8s.ClassOf(pod), k8s.ClassOf(event), []korrel8r.Object{pod}, want)
	})
	t.Run("EventToPod", func(t *testing.T) {
		want := k8s.NewQuery(k8s.ClassOf(pod), "aNamespace", "foo", nil, nil)
		testTraverse(t, e, k8s.ClassOf(event), k8s.ClassOf(pod), []korrel8r.Object{event}, want)
	})
}

func TestK8sAllToMetric(t *testing.T) {
	e := setup()
	pod := k8s.New[corev1.Pod]("aNamespace", "foo")
	want := &metric.Query{PromQL: "{namespace=\"aNamespace\",pod=\"foo\"}"}
	testTraverse(t, e, k8s.ClassOf(pod), metric.Class{}, []korrel8r.Object{pod}, want)
}
