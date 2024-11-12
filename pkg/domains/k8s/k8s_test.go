// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

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
		{"Namespace", ClassOf(&corev1.Namespace{})},           // Kind only
		{"Namespace.", ClassOf(&corev1.Namespace{})},          // Kind and version
		{"Namespace.v1.", ClassOf(&corev1.Namespace{})},       // Kind, version and group
		{"Pod", ClassOf(&corev1.Pod{})},                       // Kind only
		{"Pod.", ClassOf(&corev1.Pod{})},                      // Kind and group (core group is named "")
		{"Pod.v1", ClassOf(&corev1.Pod{})},                    // Kind, version, implied core group.
		{"Pod.v1.", ClassOf(&corev1.Pod{})},                   // Kind, version, ""
		{"Deployment", ClassOf(&appsv1.Deployment{})},         // Kind only
		{"Deployment.apps", ClassOf(&appsv1.Deployment{})},    // Kind and group
		{"Deployment.v1.apps", ClassOf(&appsv1.Deployment{})}, // Kind, version and group
	} {
		t.Run(x.name, func(t *testing.T) {
			assert.NotNil(t, x.want)
			got := Domain.Class(x.name)
			require.NotNil(t, got)
			assert.Equal(t, x.want.Name(), got.Name())

			// Round trip for String()
			name := got.Name()
			got2 := Domain.Class(name)
			require.NotNil(t, got2)
			assert.Equal(t, name, got2.Name())
		})
	}
}

func TestDomain_Query(t *testing.T) {
	for _, x := range []struct {
		s    string
		want korrel8r.Query
	}{
		{`k8s:Namespace:{"name":"foo"}`, NewQuery(ClassOf(&corev1.Namespace{}), "", "foo", nil, nil)},
		{`k8s:Namespace:{name: foo}`, NewQuery(ClassOf(&corev1.Namespace{}), "", "foo", nil, nil)},
		{`k8s:Pod:{namespace: foo, name: bar}`, NewQuery(ClassOf(&corev1.Pod{}), "foo", "bar", nil, nil)},
		{`k8s:Pod:{namespace: foo, name: bar, labels: { a: b }, fields: { c: d }}`,
			NewQuery(ClassOf(&corev1.Pod{}), "foo", "bar", map[string]string{"a": "b"}, map[string]string{"c": "d"})},
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
	store, err := NewStore(c, &rest.Config{})
	require.NoError(t, err)
	var (
		fred   = types.NamespacedName{Namespace: "x", Name: "fred"}
		barney = types.NamespacedName{Namespace: "x", Name: "barney"}
		wilma  = types.NamespacedName{Namespace: "y", Name: "wilma"}
	)
	podGVK := ClassOf(&corev1.Pod{}).GVK()
	for _, x := range []struct {
		q    korrel8r.Query
		want []types.NamespacedName
	}{
		{NewQuery(Class(podGVK), "x", "fred", nil, nil), []types.NamespacedName{fred}},
		{NewQuery(Class(podGVK), "x", "", nil, nil), []types.NamespacedName{fred, barney}},
		{NewQuery(Class(podGVK), "", "", client.MatchingLabels{"app": "foo"}, nil), []types.NamespacedName{fred, wilma}},
	} {
		t.Run(fmt.Sprintf("%#v", x.q), func(t *testing.T) {
			var result korrel8r.ListResult
			err = store.Get(context.Background(), x.q, nil, &result)
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
	store, err := NewStore(c, &rest.Config{})
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
			var result korrel8r.ListResult
			err = store.Get(context.Background(), NewQuery(ClassOf(&corev1.Pod{}), "test", "", nil, nil), x.constraint, &result)
			require.NoError(t, err)
			var got []string
			for _, v := range result {
				got = append(got, v.(Object).(*corev1.Pod).GetName())
			}
			assert.ElementsMatch(t, x.want, got, "%v != %v", x.want, got)
		})
	}
	// Need to validate labels and all get variations on fake client or env test...
}

func TestDescription(t *testing.T) {
	for _, x := range []struct {
		class       Class
		name        string
		description string
	}{
		{ClassOf(&corev1.Pod{}), "Pod.v1.", "Pod is a collection of containers that can run on a host. This resource is created by clients and scheduled onto hosts."},
		{ClassOf(&appv1.Deployment{}), "Deployment.v1.apps", "Deployment enables declarative updates for Pods and ReplicaSets."},
	} {
		t.Run(x.class.Name(), func(t *testing.T) {
			assert.Equal(t, x.name, x.class.Name())
			assert.Equal(t, x.description, x.class.Description())
		})
	}
}
