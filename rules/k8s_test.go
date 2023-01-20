// package rules is a test-only package to verify the rules.yaml files give expected results.
//
// Note these tests only verify that the engine generates the expected references.
// It does not verify that the references yield expected results.
// For end-to-end tests see /home/aconway/src/korrel8r/korrel8r/cmd/korrel8r/cmd/cmd_test.go
package rules

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/decoder"
	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/alert"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/loki"
	"github.com/korrel8r/korrel8r/pkg/templaterule"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/korrel8r/korrel8r/pkg/uri"
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

func setup(t *testing.T, ruleFiles ...string) (client.Client, *engine.Engine) {
	t.Helper()
	c := fake.NewClientBuilder().WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(k8s.Scheme)).Build()
	e := engine.New()
	e.AddDomain(k8s.Domain, test.Must(k8s.NewStore(c, &rest.Config{})))
	e.AddDomain(loki.Domain, nil)
	e.AddDomain(alert.Domain, nil)
	for _, name := range ruleFiles {
		f, err := os.Open(name)
		require.NoError(t, err)
		defer f.Close()
		d := decoder.New(f)
		require.NoError(t, templaterule.AddRules(d, e), name)
	}
	return c, e
}

func testFollow(t *testing.T, e *engine.Engine, start, goal korrel8r.Class, objs []korrel8r.Object, want []uri.Reference) {
	t.Helper()
	paths, err := e.Graph().ShortestPaths(start, goal)
	require.NoError(t, err)
	results := engine.NewResults()
	err = e.FollowAll(ctx, objs, nil, paths, results)
	require.NoError(t, err)
	assert.Equal(t, want, results.Get(goal).References.List)
}

func TestPodToLogs(t *testing.T) {
	c, e := setup(t, "k8s.yaml")
	for _, x := range []struct {
		pod  *corev1.Pod
		want []uri.Reference
	}{
		{pod: k8s.New[corev1.Pod]("project", "application")},
		{pod: k8s.New[corev1.Pod]("kube-something", "infrastructure")},
	} {
		ns, name := x.pod.Namespace, x.pod.Name
		t.Run(name, func(t *testing.T) {
			require.NoError(t, k8s.Create(c, x.pod))
			testFollow(t, e, k8s.ClassOf(x.pod), loki.Domain.Class(name), []korrel8r.Object{x.pod},
				[]uri.Reference{uri.Make(
					fmt.Sprintf("/api/logs/v1/%v/loki/api/v1/query_range", name),
					"query", fmt.Sprintf(`{kubernetes_namespace_name="%v",kubernetes_pod_name="%v"} | json`, ns, name)),
				})
		})
	}
}

func TestSelectorToLogsRules(t *testing.T) {
	_, e := setup(t, "k8s.yaml")
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
		k8s.Class{Group: "extensions", Version: "v1beta1", Kind: "DaemonSet"},
		k8s.Class{Group: "extensions", Version: "v1beta1", Kind: "Deployment"},
		k8s.Class{Group: "apps", Version: "v1beta1", Kind: "StatefulSet"},
		k8s.Class{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
		k8s.Class{Group: "", Version: "v1", Kind: "Service"},
		k8s.Class{Group: "policy", Version: "v1beta1", Kind: "PodDisruptionBudget"},
		k8s.Class{Group: "apps", Version: "v1", Kind: "Deployment"},
		k8s.Class{Group: "apps", Version: "v1beta2", Kind: "StatefulSet"},
		k8s.Class{Group: "batch", Version: "v1", Kind: "Job"},
		k8s.Class{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"},
		k8s.Class{Group: "apps", Version: "v1beta1", Kind: "Deployment"},
		k8s.Class{Group: "apps", Version: "v1beta2", Kind: "ReplicaSet"},
		k8s.Class{Group: "extensions", Version: "v1beta1", Kind: "ReplicaSet"},
		k8s.Class{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		k8s.Class{Group: "apps", Version: "v1beta2", Kind: "DaemonSet"},
		k8s.Class{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		k8s.Class{Group: "apps", Version: "v1beta2", Kind: "Deployment"},
		k8s.Class{Group: "apps", Version: "v1", Kind: "ReplicaSet"}}
	assert.ElementsMatch(t, want, classes.List, "%#v", classes.List)
}

func TestSelectorToLogs(t *testing.T) {
	c, e := setup(t, "k8s.yaml")
	d := k8s.New[appsv1.Deployment]("ns", "x")
	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"?a.b": "x"}},
	}
	require.NoError(t, k8s.Create(c, d))
	want := []uri.Reference{uri.Make(
		"/api/logs/v1/application/loki/api/v1/query_range",
		"query", `{kubernetes_namespace_name="ns"} | json | kubernetes_label__a_b="x"`),
	}
	testFollow(t, e, k8s.ClassOf(d), loki.Domain.Class("application"), []korrel8r.Object{d}, want)
}

func TestSelectorToPods(t *testing.T) {
	c, e := setup(t, "k8s.yaml")

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
	require.NoError(t, k8s.Create(c, d, podx, pody))

	testFollow(t, e, k8s.ClassOf(d), k8s.ClassOf(podx), []korrel8r.Object{d},
		[]uri.Reference{uri.Make("/api/v1/namespaces/ns/pods", "labelSelector", "test=testme")})
}

func TestK8sEvent(t *testing.T) {
	c, e := setup(t, "k8s.yaml")
	pod := k8s.New[corev1.Pod]("aNamespace", "foo")
	event := k8s.EventFor(pod, "a")
	require.NoError(t, k8s.Create(c, pod, event))

	t.Run("PodToEvent", func(t *testing.T) {
		testFollow(t, e, k8s.ClassOf(pod), k8s.ClassOf(event), []korrel8r.Object{pod},
			[]uri.Reference{{
				Path:     "/api/v1/events",
				RawQuery: url.Values{"fieldSelector": []string{"involvedObject.name=foo,involvedObject.namespace=aNamespace"}}.Encode()}})
	})

	t.Run("EventToPod", func(t *testing.T) {
		testFollow(t, e, k8s.ClassOf(event), k8s.ClassOf(pod), []korrel8r.Object{event},
			[]uri.Reference{{Path: "/api/v1/namespaces/aNamespace/pods/foo"}})
	})
}

var ctx = context.Background()
