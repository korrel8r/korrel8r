// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
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
			err = store.Get(context.Background(), x.q, &result)
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
		q korrel8r.Query
		p string
	}{
		{NewQuery(ClassOf(&corev1.Pod{}), "default", "foo", nil, nil), "k8s/ns/default/pods/foo"},
		{NewQuery(ClassOf(&corev1.Pod{}), "default", "", nil, nil), "k8s/ns/default/pods"},
		{NewQuery(ClassOf(&corev1.Namespace{}), "", "foo", nil, nil), "k8s/cluster/namespaces/foo"},
		{NewQuery(ClassOf(&corev1.Namespace{}), "", "", nil, nil), "k8s/cluster/namespaces"},
		{NewQuery(ClassOf(&appv1.Deployment{}), "", "", nil, nil), "k8s/cluster/deployments"},
		{NewQuery(ClassOf(&appv1.Deployment{}), "NAMESPACE", "", nil, nil), "k8s/ns/NAMESPACE/deployments"},
		{NewQuery(ClassOf(&appv1.Deployment{}), "NAMESPACE", "NAME", nil, nil), "k8s/ns/NAMESPACE/deployments/NAME"},
	} {
		t.Run(x.p, func(t *testing.T) {
			u, err := s.(*Store).QueryToConsoleURL(x.q)
			if assert.NoError(t, err) {
				assert.Equal(t, x.p, u.Path)
			}
		})
	}
	require.NoError(t, err)
}

func TestStore_ConsoleURLToQuery(t *testing.T) {
	s := must.Must1(NewStore(fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build(), &rest.Config{}))
	for _, x := range []struct {
		p string
		q korrel8r.Query
	}{
		{"/k8s/ns/default/pods/foo", NewQuery(ClassOf(&corev1.Pod{}), "default", "foo", nil, nil)},
		{"/k8s/ns/default/pods", NewQuery(ClassOf(&corev1.Pod{}), "default", "", nil, nil)},
		{"/k8s/cluster/namespaces/foo", NewQuery(ClassOf(&corev1.Namespace{}), "", "foo", nil, nil)},
		{"/k8s/cluster/projects/foo", NewQuery(ClassOf(&corev1.Namespace{}), "", "foo", nil, nil)},
		{"/k8s/ns/default?kind=Pod&q=name%3Dfoo%2Capp%3dbar", NewQuery(ClassOf(&corev1.Pod{}), "default", "", map[string]string{"name": "foo", "app": "bar"}, nil)},
	} {
		t.Run(x.p, func(t *testing.T) {
			u, _ := url.Parse(x.p)
			q, err := s.(*Store).ConsoleURLToQuery(u)
			if assert.NoError(t, err) {
				assert.Equal(t, x.q, q)
			}
		})
	}
}

func Test_parsePath(t *testing.T) {
	for _, x := range []struct {
		path      string
		match     []string
		tryEvents bool
	}{
		{`/k8s/ns/openshift-logging/operators.coreos.com~v1alpha1~ClusterServiceVersion/cluster-logging.v5.6.0`,
			[]string{"openshift-logging", "operators.coreos.com~v1alpha1~ClusterServiceVersion", "cluster-logging.v5.6.0"},
			true,
		},
		{`/k8s/ns/openshift-logging/pods/foo`,
			[]string{"openshift-logging", "pods", "foo"},
			true,
		},
		{`/k8s/all-namespaces/pods`,
			[]string{"", "pods", ""},
			false,
		},
		{`/k8s/cluster/nodes/oscar7`,
			[]string{"", "nodes", "oscar7"},
			false,
		},
		{`/search/ns/openshift-logging/pods`,
			[]string{"openshift-logging", "pods", ""},
			false,
		},
	} {

		var (
			got    = make([]string, 3)
			err    error
			events bool
		)
		t.Run(x.path, func(t *testing.T) {
			got[0], got[1], got[2], events, err = parsePath(&url.URL{Path: x.path})
			if assert.NoError(t, err) {
				assert.Equal(t, x.match, got)
				assert.False(t, events)
			}
		})
		if x.tryEvents {
			t.Run(x.path+"/events", func(t *testing.T) {
				got[0], got[1], got[2], events, err = parsePath(&url.URL{Path: x.path + "/events"})
				if assert.NoError(t, err) {
					assert.Equal(t, x.match, got)
					assert.True(t, events)
				}
			})
		}
	}
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
