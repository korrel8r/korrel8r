package k8s

import (
	"context"
	"testing"

	"net/url"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
			assert.Equal(t, path, apiPath.FindStringSubmatch(path[0]))
		})
	}
}

func TestDomain_Class(t *testing.T) {
	require.NoError(t, appsv1.AddToScheme(scheme.Scheme))
	for _, x := range []struct {
		name string
		want korrel8.Class
	}{
		{"Pod", ClassOf(&corev1.Pod{})},                       // Kind only
		{"Pod.", ClassOf(&corev1.Pod{})},                      // Kind and group (core group is named "")
		{"Pod.v1.", ClassOf(&corev1.Pod{})},                   // Kind, version gand roup.
		{"Deployment", ClassOf(&appsv1.Deployment{})},         // Kind only
		{"Deployment.apps", ClassOf(&appsv1.Deployment{})},    // Kind and group
		{"Deployment.v1.apps", ClassOf(&appsv1.Deployment{})}, // Kind, version and group
	} {
		t.Run(x.name, func(t *testing.T) {
			assert.NotNil(t, x.want)
			got := Domain.Class(x.name)
			require.NotNil(t, got)
			assert.Equal(t, x.want.String(), got.String())

			// Round trip for String()
			name := got.String()
			got2 := Domain.Class(name)
			require.NotNil(t, got2)
			assert.Equal(t, name, got2.String())
		})
	}
}

func TestStore_ParseURI(t *testing.T) {
	require.NoError(t, apiextensionsv1.AddToScheme(scheme.Scheme))
	s := NewStore(fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build())
	for _, x := range []struct {
		path   string
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
		t.Run(x.path, func(t *testing.T) {
			gvk, nsName, err := s.parsePath(x.path)
			require.NoError(t, err)
			assert.Equal(t, x.gvk, gvk)
			assert.Equal(t, x.nsName, nsName)
		})
	}
}

func TestStore_Get(t *testing.T) {
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
	store := NewStore(c)
	var (
		fred   = types.NamespacedName{Namespace: "x", Name: "fred"}
		barney = types.NamespacedName{Namespace: "x", Name: "barney"}
		wilma  = types.NamespacedName{Namespace: "y", Name: "wilma"}
	)
	for _, x := range []struct {
		s    string
		want []types.NamespacedName
	}{
		{"/api/v1/namespaces/x/pods/fred", []types.NamespacedName{fred}},
		{"/api/v1/namespaces/x/pods", []types.NamespacedName{fred, barney}},
		{"/api/v1/pods", []types.NamespacedName{fred, barney, wilma}},
		{"/api/v1/pods?labelSelector=" + url.QueryEscape("app=foo"), []types.NamespacedName{fred, wilma}},
		// Field matches are not supported by the fake client.
	} {
		t.Run(x.s, func(t *testing.T) {
			ref, err := uri.Parse(x.s)
			require.NoError(t, err)
			var result korrel8.ListResult
			err = store.Get(context.Background(), ref, &result)
			require.NoError(t, err)
			var got []types.NamespacedName
			for _, v := range result {
				o := v.(Object).(*corev1.Pod)
				got = append(got, types.NamespacedName{Namespace: o.Namespace, Name: o.Name})
			}
			assert.ElementsMatch(t, x.want, got)
		})
	}
	// Need to validate labels and all get variations on fake client or env test...
}

func TestStore_RefStoreToConsole(t *testing.T) {
	s := NewStore(fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build())
	for _, x := range [][2]string{
		{"api/v1/namespaces/default/pods/foo", "k8s/ns/default/pods/foo"},
		{"api/v1/namespaces/default/pods", "k8s/ns/default/pods"},
		{"api/v1/namespaces/foo", "k8s/cluster/namespaces/foo"},
		{"api/v1/namespaces", "k8s/cluster/namespaces"},
		{"apis/GROUP/VERSION/RESOURCETYPE", "k8s/cluster/RESOURCETYPE"},
		{"apis/GROUP/VERSION/RESOURCETYPE/NAME", "k8s/cluster/RESOURCETYPE/NAME"},
		{"apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE", "k8s/ns/NAMESPACE/RESOURCETYPE"},
		{"apis/GROUP/VERSION/namespaces/NAMESPACE/RESOURCETYPE/NAME", "k8s/ns/NAMESPACE/RESOURCETYPE/NAME"},
	} {
		t.Run(x[0], func(t *testing.T) {
			ref, err := s.RefStoreToConsole(uri.Reference{Path: x[0]})
			if assert.NoError(t, err) {
				assert.Equal(t, x[1], ref.String())
			}
		})
	}
}

func TestStore_RefConsoleToStore(t *testing.T) {
	s := NewStore(fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build())
	for _, x := range [][2]string{
		{"k8s/ns/default/pods/foo", "api/v1/namespaces/default/pods/foo"},
		{"k8s/ns/default/pods", "api/v1/namespaces/default/pods"},
		{"k8s/cluster/nodes/foo", "api/v1/nodes/foo"},
		{"k8s/cluster/namespaces/foo", "api/v1/namespaces/foo"},
		{"k8s/cluster/projects/foo", "api/v1/namespaces/foo"},
		{"k8s/cluster/namespaces", "api/v1/namespaces"},
		{"k8s/cluster/projects", "api/v1/namespaces"},
		{"k8s/cluster/customresourcedefinitions", "apis/apiextensions.k8s.io/v1/customresourcedefinitions"},
		{"k8s/cluster/customresourcedefinitions/NAME", "apis/apiextensions.k8s.io/v1/customresourcedefinitions/NAME"},
		{"k8s/ns/NAMESPACE/deployments", "apis/apps/v1/namespaces/NAMESPACE/deployments"},
		{"k8s/ns/NAMESPACE/deployments/NAME", "apis/apps/v1/namespaces/NAMESPACE/deployments/NAME"},
	} {
		t.Run(x[0], func(t *testing.T) {
			_, ref, err := s.RefConsoleToStore(uri.Reference{Path: x[0]})
			if assert.NoError(t, err) {
				assert.Equal(t, x[1], ref.String())
			}
		})
	}
}
