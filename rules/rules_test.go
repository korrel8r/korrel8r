// package rules is a test-only package to verify the rules.yaml files give expected results.
//
// Note these tests only verify that the engine generates the expected queries.
// It does not verify that the queries yield expected results.
package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/logs"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/templaterule"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setup(t *testing.T) *engine.Engine {
	t.Helper()
	e := engine.New()
	c := fake.NewClientBuilder().WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(k8s.Scheme)).Build()
	e.AddDomain(k8s.Domain, test.Must(k8s.NewStore(c, &rest.Config{})))
	for _, d := range []korrel8r.Domain{logs.Domain, alert.Domain, metric.Domain} {
		e.AddDomain(d, nil)
	}
	names, err := filepath.Glob("*.yaml")
	require.NoError(t, err)
	for _, name := range names {
		f, err := os.Open(name)
		require.NoError(t, err)
		defer f.Close()
		require.NoError(t, templaterule.Decode(f, e), "decoding file %v", name)
	}
	return e
}

func testTraverse(t *testing.T, e *engine.Engine, start, goal korrel8r.Class, starters []korrel8r.Object, wantQuery korrel8r.Query) {
	t.Helper()
	paths := e.Graph().AllPaths(start, goal)
	paths.NodeFor(start).Result.Append(starters...)
	f := e.Follower(context.Background())
	assert.NoError(t, paths.Traverse(f.Traverse))
	assert.NoError(t, f.Err)
	n := paths.NodeFor(goal)
	want := graph.QueryCounts{}
	want.Put(wantQuery, 0)
	assert.Equal(t, want, n.QueryCounts)
}

func TestPodToLogs(t *testing.T) {
	e := setup(t)
	for _, pod := range []*corev1.Pod{
		k8s.New[corev1.Pod]("project", "application"),
		k8s.New[corev1.Pod]("kube-something", "infrastructure"),
	} {
		t.Run(pod.Name, func(t *testing.T) {
			want := &logs.Query{
				LogType: pod.Name,
				LogQL:   fmt.Sprintf(`{kubernetes_namespace_name="%v",kubernetes_pod_name="%v"} | json`, pod.Namespace, pod.Name),
			}
			testTraverse(t, e, k8s.ClassOf(pod), logs.Domain.Class(pod.Name), []korrel8r.Object{pod}, want)
		})
	}
}

func TestSelectorToLogsRules(t *testing.T) {
	e := setup(t)
	// Verify rules selected the correct set of start classes
	classes := unique.NewList[korrel8r.Class]()
	for _, r := range e.Rules() {
		if r.String() == "SelectorToLogs" {
			classes.Append(r.Start())
		}
	}
	want := []korrel8r.Class{
		k8s.Class{Group: "", Version: "v1", Kind: "ReplicationController"},
		k8s.Class{Group: "apps.openshift.io", Version: "v1", Kind: "DeploymentConfig"},
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
	e := setup(t)
	d := k8s.New[appsv1.Deployment]("ns", "x")
	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"a.b/c": "x"}},
	}
	want := &logs.Query{
		LogQL:   `{kubernetes_namespace_name="ns"} | json | kubernetes_labels_a_b_c="x"`,
		LogType: "application",
	}
	testTraverse(t, e, k8s.ClassOf(d), logs.Domain.Class("application"), []korrel8r.Object{d}, want)
}

func TestSelectorToPods(t *testing.T) {
	e := setup(t)

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
	e := setup(t)
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
		testTraverse(t, e, k8s.ClassOf(event), k8s.ClassOf(pod), []korrel8r.Object{event},
			k8s.NewQuery(k8s.ClassOf(pod), "aNamespace", "foo", nil, nil))

	})
}

func TestK8sAllToMetric(t *testing.T) {
	e := setup(t)
	pod := k8s.New[corev1.Pod]("aNamespace", "foo")
	want := &metric.Query{PromQL: "{ namespace=\"aNamespace\", pod=\"foo\" }"}
	testTraverse(t, e, k8s.ClassOf(pod), metric.Class{}, []korrel8r.Object{pod}, want)
}

func TestNamespace(t *testing.T) {
	e := setup(t)
	ns := k8s.New[corev1.Namespace]("", "ns")
	poda := k8s.New[corev1.Pod]("ns", "a")

	// TODO this rule is disabled, see TODO comment in k8s.yaml
	// t.Run("PodToNamespace", func(t *testing.T) {
	// 	want := k8s.NewQuery(k8s.ClassOf(ns), "", "ns", nil, nil)
	// 	testTraverse(t, e, k8s.ClassOf(poda), k8s.ClassOf(ns), []korrel8r.Object{poda, podb}, want)
	// })

	t.Run("NamespaceToPods", func(t *testing.T) {
		want := k8s.NewQuery(k8s.ClassOf(poda), "ns", "", nil, nil)
		testTraverse(t, e, k8s.ClassOf(ns), k8s.ClassOf(poda), []korrel8r.Object{ns}, want)
	})
}
