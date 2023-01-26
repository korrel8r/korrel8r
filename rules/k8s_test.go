// package rules is a test-only package to verify the rules.yaml files give expected results.
//
// Note these tests only verify that the engine generates the expected references.
// It does not verify that the references yield expected results.
// For end-to-end tests see /home/aconway/src/korrel8r/korrel8r/cmd/korrel8r/cmd/cmd_test.go
package rules

import (
	"context"
	"fmt"
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
		require.NoError(t, templaterule.AddRules(d, e), "decoding file %v", name)
	}
	return c, e
}

func testFollow(t *testing.T, e *engine.Engine, start, goal korrel8r.Class, objs []korrel8r.Object, want korrel8r.Query) {
	t.Helper()
	paths, err := e.Graph().ShortestPaths(start, goal)
	require.NoError(t, err)
	var results engine.Results
	err = e.FollowAll(ctx, objs, nil, paths, &results)
	require.NoError(t, err)
	qs := results.Get(goal).Queries.List
	assert.Len(t, qs, 1)
	assert.Equal(t, want, qs[0])
}

func TestPodToLogs(t *testing.T) {
	c, e := setup(t, "k8s.yaml")
	for _, x := range []struct {
		pod  *corev1.Pod
		want []korrel8r.Query
	}{
		{pod: k8s.New[corev1.Pod]("project", "application")},
		{pod: k8s.New[corev1.Pod]("kube-something", "infrastructure")},
	} {
		ns, name := x.pod.Namespace, x.pod.Name
		t.Run(name, func(t *testing.T) {
			require.NoError(t, k8s.Create(c, x.pod))
			want := &loki.Query{
				Tenant: name,
				LogQL:  fmt.Sprintf(`{kubernetes_namespace_name="%v",kubernetes_pod_name="%v"} | json`, ns, name),
			}
			testFollow(t, e, k8s.ClassOf(x.pod), loki.Domain.Class(name), []korrel8r.Object{x.pod}, want)
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
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"a.b/c": "x"}},
	}
	require.NoError(t, k8s.Create(c, d))
	want := &loki.Query{
		LogQL:  `{kubernetes_namespace_name="ns"} | json | kubernetes_labels_a_b_c="x"`,
		Tenant: "application",
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
	class := k8s.ClassOf(podx)
	testFollow(t, e, k8s.ClassOf(d), class, []korrel8r.Object{d},
		k8s.NewQuery(class, "ns", "", client.MatchingLabels{"test": "testme"}, nil))
}

func TestK8sEvent(t *testing.T) {
	c, e := setup(t, "k8s.yaml")
	pod := k8s.New[corev1.Pod]("aNamespace", "foo")
	event := k8s.EventFor(pod, "a")
	require.NoError(t, k8s.Create(c, pod, event))

	t.Run("PodToEvent", func(t *testing.T) {
		testFollow(t, e, k8s.ClassOf(pod), k8s.ClassOf(event), []korrel8r.Object{pod},
			k8s.NewQuery(k8s.ClassOf(&corev1.Event{}), "", "",
				nil,
				client.MatchingFields{"involvedObject.name": "foo", "involvedObject.namespace": "aNamespace"}))
	})
	t.Run("EventToPod", func(t *testing.T) {
		testFollow(t, e, k8s.ClassOf(event), k8s.ClassOf(pod), []korrel8r.Object{event},
			k8s.NewQuery(k8s.ClassOf(pod), "aNamespace", "foo", nil, nil))

	})
}

var ctx = context.Background()
