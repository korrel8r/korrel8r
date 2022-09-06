package k8s

import (
	"context"
	"net/url"
	"testing"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestClassOf(t *testing.T) {
	c := ClassOf(&corev1.Pod{})
	assert.True(t, c.Contains(&corev1.Pod{TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"}}))
	assert.False(t, c.Contains(&corev1.Service{}))
}

func TestParseURIRegexp(t *testing.T) {
	for _, path := range [][]string{
		{"/api/v1/namespaces/default/pods/foo", "", "v1", "default", "pods", "foo"},
		{"/api/v1/namespaces/default/pods", "", "v1", "default", "pods", ""},
		{"/api/v1/namespaces/foo", "", "v1", "", "namespaces", "foo"},
		{"/api/v1/namespaces", "", "v1", "", "namespaces", ""},
		{"/apis/GROUP/VERSION/RESOURCETYPE", "GROUP", "VERSION", "", "RESOURCETYPE", ""},
		{"/apis/GROUP/VERSION/RESOURCETYPE/NAME", "GROUP", "VERSION", "", "RESOURCETYPE", "NAME"},
		{"/apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE", "GROUP", "VERSION", "NAMESPACE", "RESOURCETYPE", ""},
		{"/apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME", "GROUP", "VERSION", "NAMESPACE", "RESOURCETYPE", "NAME"},
	} {
		t.Run(path[0], func(t *testing.T) {
			assert.Equal(t, path, k8sPathRegex.FindStringSubmatch(path[0]))
		})
	}
}

// FIXME need https://pkg.go.dev/sigs.k8s.io/controller-runtime/tools/setup-envtest#section-readme

func TestStore_ParseURI(t *testing.T) {
	apiextensionsv1.AddToScheme(scheme.Scheme)
	s, err := NewStore(fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build())
	require.NoError(t, err)
	for _, x := range []struct {
		uri    string
		gvk    schema.GroupVersionKind
		nsName types.NamespacedName
	}{
		{"/api/v1/namespaces/default/pods/foo", schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}, types.NamespacedName{Namespace: "default", Name: "foo"}},
		{"/api/v1/namespaces/default/pods", schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}, types.NamespacedName{Namespace: "default", Name: ""}},
		{"/api/v1/namespaces/foo", schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}, types.NamespacedName{Namespace: "", Name: "foo"}},
		{"/api/v1/namespaces", schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}, types.NamespacedName{Namespace: "", Name: ""}},
		{"/apis/apiextensions.k8s.io/v1/customresourcedefinitions", schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}, types.NamespacedName{Namespace: "", Name: ""}},
		{"/apis/apiextensions.k8s.io/v1/namespaces/foo/customresourcedefinitions/bar", schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}, types.NamespacedName{Namespace: "foo", Name: "bar"}},
	} {
		t.Run(x.uri, func(t *testing.T) {
			u, err := url.Parse(x.uri)
			gvk, nsName, err := s.parseAPIPath(u)
			require.NoError(t, err)
			assert.Equal(t, x.gvk, gvk)
			assert.Equal(t, x.nsName, nsName)
		})
	}
}

func TestStore_Execute(t *testing.T) {
	c := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		WithObjects(
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "fred", Namespace: "x", Labels: map[string]string{"app": "foo"}},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "barney", Namespace: "x", Labels: map[string]string{"app": "bad"}},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "wilma", Namespace: "y", Labels: map[string]string{"app": "foo"}},
			},
		).Build()
	store, err := NewStore(c)
	require.NoError(t, err)
	for _, x := range []struct {
		query string
		want  []types.NamespacedName
	}{
		{"/api/v1/namespaces/x/pods/fred", []types.NamespacedName{{"x", "fred"}}},
		{"/api/v1/namespaces/x/pods", []types.NamespacedName{{"x", "fred"}, {"x", "barney"}}},
		{"/api/v1/pods", []types.NamespacedName{{"x", "fred"}, {"x", "barney"}, {"y", "wilma"}}},
		{"/api/v1/pods?labelSelector=app%3Dfoo", []types.NamespacedName{{"x", "fred"}, {"y", "wilma"}}},
		// Field selectors are not supported by the fake client.
		//		{"/api/v1/pods?fieldSelector=metadata.name%3D", []types.NamespacedName{{"y", "wilma"}}},
	} {
		t.Run(x.query, func(t *testing.T) {
			result, err := store.Execute(context.Background(), korrel8.Query(x.query))
			require.NoError(t, err)
			var got []types.NamespacedName
			for _, v := range result {
				o := v.(*v1.Pod)
				got = append(got, types.NamespacedName{Namespace: o.Namespace, Name: o.Name})
			}
			assert.ElementsMatch(t, x.want, got)
		})
	}
	// Need to validate labels and all get variations on fake client or env test...
}
