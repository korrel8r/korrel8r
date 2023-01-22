package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDomain_Class(t *testing.T) {
	require.NoError(t, appsv1.AddToScheme(scheme.Scheme))
	for _, x := range []struct {
		name string
		want korrel8r.Class
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
	store, err := NewStore(c, &rest.Config{})
	require.NoError(t, err)
	var (
		fred   = types.NamespacedName{Namespace: "x", Name: "fred"}
		barney = types.NamespacedName{Namespace: "x", Name: "barney"}
		wilma  = types.NamespacedName{Namespace: "y", Name: "wilma"}
	)
	podGVK := ClassOf(&corev1.Pod{}).GVK()
	for _, x := range []struct {
		q    Query
		want []types.NamespacedName
	}{
		{Query{GroupVersionKind: podGVK, NamespacedName: fred}, []types.NamespacedName{fred}},
		{Query{GroupVersionKind: podGVK, NamespacedName: types.NamespacedName{Namespace: "x"}}, []types.NamespacedName{fred, barney}},
		{Query{GroupVersionKind: podGVK, Labels: client.MatchingLabels{"app": "foo"}}, []types.NamespacedName{fred, wilma}},
		// Field matches are not supported by the fake client.
	} {
		t.Run(fmt.Sprintf("%#v", x.q), func(t *testing.T) {
			var result korrel8r.ListResult
			err = store.Get(context.Background(), &x.q, &result)
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

func TestStore_QueryToConsoleURL(t *testing.T) {
	s, err := NewStore(fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build(), &rest.Config{})
	for _, x := range []struct {
		q Query
		p string
	}{
		{query(ClassOf(&corev1.Pod{}), "default", "foo"), "k8s/ns/default/pods/foo"},
		{query(ClassOf(&corev1.Pod{}), "default", ""), "k8s/ns/default/pods"},
		{query(ClassOf(&corev1.Namespace{}), "", "foo"), "k8s/cluster/namespaces/foo"},
		{query(ClassOf(&corev1.Namespace{}), "", ""), "k8s/cluster/namespaces"},
		{query(ClassOf(&appv1.Deployment{}), "", ""), "k8s/cluster/deployments"},
		{query(ClassOf(&appv1.Deployment{}), "NAMESPACE", ""), "k8s/ns/NAMESPACE/deployments"},
		{query(ClassOf(&appv1.Deployment{}), "NAMESPACE", "NAME"), "k8s/ns/NAMESPACE/deployments/NAME"},
	} {
		t.Run(x.p, func(t *testing.T) {
			u, err := s.QueryToConsoleURL(&x.q)
			if assert.NoError(t, err) {
				assert.Equal(t, x.p, u.Path)
			}
		})
	}
	require.NoError(t, err)
}

func query(c Class, namespace, name string) Query {
	return Query{GroupVersionKind: c.GVK(), NamespacedName: types.NamespacedName{Namespace: namespace, Name: name}}
}

func TestStore_ConsoleURLToQuery(t *testing.T) {
	s := must.Must1(NewStore(fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build(), &rest.Config{}))
	for _, x := range []struct {
		p string
		q Query
	}{
		{"/k8s/ns/default/pods/foo", query(ClassOf(&corev1.Pod{}), "default", "foo")},
		{"/k8s/ns/default/pods", query(ClassOf(&corev1.Pod{}), "default", "")},
		{"/k8s/cluster/namespaces/foo", query(ClassOf(&corev1.Namespace{}), "", "foo")},
		{"/k8s/cluster/projects/foo", query(ClassOf(&corev1.Namespace{}), "", "foo")},
	} {
		t.Run(x.p, func(t *testing.T) {
			u, _ := url.Parse(x.p)
			q, err := s.ConsoleURLToQuery(u)
			if assert.NoError(t, err) {
				assert.Equal(t, &x.q, q)
			}
		})
	}
}

func TestQuery_Marshal(t *testing.T) {
	class := ClassOf(&corev1.Pod{})
	q := NewQuery(class, "NAMESPACE", "NAME",
		client.MatchingLabels{"label": "foo"}, client.MatchingFields{"field": "bar"})
	b, err := json.Marshal(q)
	require.NoError(t, err)
	want := `{"Group":"","Version":"v1","Kind":"Pod","Namespace":"NAMESPACE","Name":"NAME","Labels":{"label":"foo"},"Fields":{"field":"bar"}}`
	assert.Equal(t, want, string(b))
	q2 := Domain.Query(nil)
	err = json.Unmarshal(b, q2)
	require.NoError(t, err)
	assert.Equal(t, q, q2)
}
