// package rules is a test-only package to verify the rules.yaml files give expected results.
package rules

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	tk "github.com/korrel8/korrel8/internal/pkg/test/k8s"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/loki"
	"github.com/korrel8/korrel8/pkg/templaterule"
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
	e := engine.New("test")
	s, err := k8s.NewStore(c, &rest.Config{})
	require.NoError(t, err)
	e.AddDomain(k8s.Domain, s)
	e.AddDomain(loki.Domain, nil)
	for _, name := range ruleFiles {
		f, err := os.Open(name)
		require.NoError(t, err)
		defer f.Close()
		d := decoder.New(f)
		require.NoError(t, templaterule.AddRules(d, e), "%v:%v", name, d.Line())
	}
	return c, e
}

func makeQuery(path string, keysAndValues ...string) korrel8.Query {
	v := url.Values{}
	for i := 0; i < len(keysAndValues); i += 2 {
		v.Set(keysAndValues[i], keysAndValues[i+1])
	}
	return korrel8.Query{Path: path, RawQuery: v.Encode()}
}

func testFollow(t *testing.T, e *engine.Engine, start, goal korrel8.Class, objs []korrel8.Object, want []korrel8.Query) {
	paths, err := e.Graph().ShortestPaths(start, goal)
	require.NoError(t, err)
	queries, err := e.FollowAll(ctx, objs, nil, paths)
	require.NoError(t, err)
	assert.Equal(t, want, queries)
}

func TestSelectorToPods(t *testing.T) {
	c, e := setup(t, "k8s.yaml")
	// Deployment
	labels := map[string]string{"test": "testme"}

	podx := tk.Build(&corev1.Pod{}).NSName("ns", "x").Object()
	podx.ObjectMeta.Labels = labels
	podx.Spec = corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:    "testme",
			Image:   "quay.io/quay/busybox",
			Command: []string{"sh", "-c", "while true; do echo $(date) hello world; sleep 1; done"},
		}}}

	pody := podx.DeepCopy()
	pody.ObjectMeta.Name = "y"

	d := tk.Build(&appsv1.Deployment{}).NSName("ns", "x").Object()
	d.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: podx.ObjectMeta,
			Spec:       podx.Spec,
		}}
	require.NoError(t, tk.Create(c, d, podx, pody))

	testFollow(t, e, k8s.ClassOf(d), k8s.ClassOf(podx), []korrel8.Object{d},
		[]korrel8.Query{makeQuery("/api/v1/namespaces/ns/pods", "labelSelector", "test=testme")})
}

func TestK8sEvent(t *testing.T) {
	c, e := setup(t, "k8s.yaml")
	pod := tk.Build(&corev1.Pod{}).NSName("aNamespace", "foo").Object()
	event := tk.EventFor(pod)
	require.NoError(t, tk.Create(c, pod, event))

	t.Run("PodToEvent", func(t *testing.T) {
		testFollow(t, e, k8s.ClassOf(pod), k8s.ClassOf(event), []korrel8.Object{pod},
			[]korrel8.Query{{
				Path:     "/api/v1/events",
				RawQuery: url.Values{"fieldSelector": []string{"involvedObject.name=foo,involvedObject.namespace=aNamespace"}}.Encode()}})
	})

	t.Run("EventToPod", func(t *testing.T) {
		testFollow(t, e, k8s.ClassOf(event), k8s.ClassOf(pod), []korrel8.Object{event},
			[]korrel8.Query{{Path: "/api/v1/namespaces/aNamespace/pods/foo"}})
	})
}

var ctx = context.Background()
