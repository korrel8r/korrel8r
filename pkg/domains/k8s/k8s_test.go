// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	namespace  = must.Must1(ParseClass("Namespace"))
	pod        = must.Must1(ParseClass("Pod"))
	deployment = must.Must1(ParseClass("Deployment.apps"))
)

func newQuery(c Class, namespace, name string, labels, fields map[string]string) *Query {
	return NewQuery(c, Selector{Namespace: namespace, Name: name, Labels: labels, Fields: fields})
}

func TestDomain_Query(t *testing.T) {
	for _, x := range []struct {
		s    string
		want korrel8r.Query
	}{
		{`k8s:Namespace:{"name":"foo"}`, newQuery(namespace, "", "foo", nil, nil)},
		{`k8s:Namespace:{name: foo}`, newQuery(namespace, "", "foo", nil, nil)},
		{`k8s:Pod:{namespace: foo, name: bar}`, newQuery(pod, "foo", "bar", nil, nil)},
		{`k8s:Pod:{namespace: foo, name: bar, labels: { a: b }, fields: { c: d }}`,
			newQuery(pod, "foo", "bar", map[string]string{"a": "b"}, map[string]string{"c": "d"})},
	} {
		t.Run(x.s, func(t *testing.T) {
			got, err := Domain.Query(x.s)
			if assert.NoError(t, err) {
				assert.Equal(t, x.want, got)
			}
		})
	}

}

func TestDomain_Query_error(t *testing.T) {
	for _, x := range []struct {
		s   string
		err string
	}{
		// Detect common error: yaml map with missing space interpreted as key containing '"'
		{`k8s:Namespace:{name:"foo"}`, "unknown field"},
	} {
		t.Run(x.s, func(t *testing.T) {
			_, err := Domain.Query(x.s)
			assert.ErrorContains(t, err, x.err)
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
	store, err := Domain.NewStore(c, &rest.Config{})
	require.NoError(t, err)
	var (
		fred   = types.NamespacedName{Namespace: "x", Name: "fred"}
		barney = types.NamespacedName{Namespace: "x", Name: "barney"}
		wilma  = types.NamespacedName{Namespace: "y", Name: "wilma"}
	)
	for _, x := range []struct {
		q    korrel8r.Query
		want []types.NamespacedName
	}{
		{newQuery(pod, "x", "fred", nil, nil), []types.NamespacedName{fred}},
		{newQuery(pod, "x", "", nil, nil), []types.NamespacedName{fred, barney}},
		{newQuery(pod, "", "", client.MatchingLabels{"app": "foo"}, nil), []types.NamespacedName{fred, wilma}},
	} {
		t.Run(fmt.Sprintf("%#v", x.q), func(t *testing.T) {
			var result mock.Result
			err = store.Get(context.Background(), x.q, nil, &result)
			require.NoError(t, err)
			var got []types.NamespacedName
			for _, v := range result {
				u := ToUnstructured(v.(Object))
				got = append(got, types.NamespacedName{Namespace: u.GetNamespace(), Name: u.GetName()})
			}
			assert.ElementsMatch(t, x.want, got)
		})
	}
	// Need to validate labels and all get variations on fake client or env test...
}

func TestStore_Get_Constraint(t *testing.T) {
	// Time range [start,end] and some time points.
	start := time.Now()
	end := start.Add(time.Minute)
	testPod := func(name string, t time.Time) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test", CreationTimestamp: metav1.Time{Time: t}},
		}
	}
	early, ontime, late := testPod("early", start.Add(-time.Second)), testPod("ontime", start.Add(time.Second)), testPod("late", end.Add(time.Second))
	c := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		WithObjects(early, ontime, late).Build()
	store, err := Domain.NewStore(c, &rest.Config{})
	require.NoError(t, err)

	for _, x := range []struct {
		constraint *korrel8r.Constraint
		want       []string
	}{
		{&korrel8r.Constraint{Start: &start, End: &end}, []string{"early", "ontime"}},
		{&korrel8r.Constraint{Start: &start}, []string{"early", "ontime", "late"}},
		{&korrel8r.Constraint{End: &end}, []string{"early", "ontime"}},
		{nil, []string{"early", "ontime", "late"}},
	} {
		t.Run(fmt.Sprintf("%+v", x.constraint), func(t *testing.T) {
			var result mock.Result
			err = store.Get(context.Background(), newQuery(pod, "test", "", nil, nil), x.constraint, &result)
			require.NoError(t, err)
			var got []string
			for _, v := range result {
				got = append(got, ToUnstructured(v.(Object)).GetName())
			}
			assert.ElementsMatch(t, x.want, got, "%v != %v", x.want, got)
		})
	}
	// Need to validate labels and all get variations on fake client or env test...
}

func fakeStore(t *testing.T) *Store {
	fc := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build()
	fd := &fakediscovery.FakeDiscovery{
		Fake: &clienttesting.Fake{
			Resources: []*metav1.APIResourceList{
				{GroupVersion: "v1", APIResources: []metav1.APIResource{
					{Kind: "Namespace", Namespaced: false},
					{Kind: "Pod", Namespaced: true},
				}},
				{GroupVersion: "apps/v1", APIResources: []metav1.APIResource{
					{Kind: "Deployment", Namespaced: true},
				}},
			},
		},
	}
	store, err := Domain.NewStoreWithDiscovery(fc, &rest.Config{}, fd)
	require.NoError(t, err)
	return store
}

func TestDomain_Classes(t *testing.T) {
	fakeStore(t) // Called for side effect of creating classes in domain
	want := []korrel8r.Class{deployment, namespace, pod}
	assert.Subset(t, Domain.Classes(), want)
}

func TestClass_IsNamespaceed(t *testing.T) {
	fakeStore(t) // Called for side effect of creating classes in domain
	assert.False(t, namespace.IsNamespaced())
	assert.True(t, deployment.IsNamespaced())
	assert.True(t, pod.IsNamespaced())
}
