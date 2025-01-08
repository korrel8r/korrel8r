// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	want := metric.Query("{namespace=\"aNamespace\",pod=\"foo\"}")
	testTraverse(t, e, k8s.ClassOf(pod), want.Class(), []korrel8r.Object{pod}, want)
}

func TestK8sPOdToAlert(t *testing.T) {
	e := setup()
	pod := k8s.New[corev1.Pod]("aNamespace", "foo")
	want := alert.Query{"namespace": "aNamespace", "pod": "foo"}
	testTraverse(t, e, k8s.ClassOf(pod), want.Class(), []korrel8r.Object{pod}, want)
}
