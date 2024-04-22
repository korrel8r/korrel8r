// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package openshift

import (
	"net/url"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestConsole_URLFromQuery_k8s(t *testing.T) {
	client := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build()
	c := NewConsole(&url.URL{Scheme: "https:", Host: "foobar"}, client)
	for _, x := range []struct {
		q korrel8r.Query
		p string
	}{
		{k8s.NewQuery(k8s.ClassOf(&corev1.Pod{}), "default", "foo", nil, nil), "/k8s/ns/default/pods/foo"},
		{k8s.NewQuery(k8s.ClassOf(&corev1.Pod{}), "default", "", nil, nil), "/k8s/ns/default/pods"},
		{k8s.NewQuery(k8s.ClassOf(&corev1.Namespace{}), "", "foo", nil, nil), "/k8s/cluster/namespaces/foo"},
		{k8s.NewQuery(k8s.ClassOf(&corev1.Namespace{}), "", "", nil, nil), "/k8s/cluster/namespaces"},
		{k8s.NewQuery(k8s.ClassOf(&appv1.Deployment{}), "", "", nil, nil), "/k8s/cluster/deployments"},
		{k8s.NewQuery(k8s.ClassOf(&appv1.Deployment{}), "NAMESPACE", "", nil, nil), "/k8s/ns/NAMESPACE/deployments"},
		{k8s.NewQuery(k8s.ClassOf(&appv1.Deployment{}), "NAMESPACE", "NAME", nil, nil), "/k8s/ns/NAMESPACE/deployments/NAME"},
	} {
		t.Run(x.p, func(t *testing.T) {
			u, err := c.URLFromQuery(x.q)
			if assert.NoError(t, err) {
				assert.Equal(t, x.p, u.Path)
			}
		})
	}
}

func TestConsole_QueryFromURL_k8s(t *testing.T) {
	client := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build()
	c := NewConsole(&url.URL{Scheme: "https:", Host: "foobar"}, client)
	for _, x := range []struct {
		p string
		q korrel8r.Query
	}{
		{"/k8s/ns/default/pods/foo", k8s.NewQuery(k8s.ClassOf(&corev1.Pod{}), "default", "foo", nil, nil)},
		{"/k8s/ns/default/pods", k8s.NewQuery(k8s.ClassOf(&corev1.Pod{}), "default", "", nil, nil)},
		{"/k8s/cluster/namespaces/foo", k8s.NewQuery(k8s.ClassOf(&corev1.Namespace{}), "", "foo", nil, nil)},
		{"/k8s/cluster/projects/foo", k8s.NewQuery(k8s.ClassOf(&corev1.Namespace{}), "", "foo", nil, nil)},
		{"/k8s/ns/default?kind=Pod&q=name%3Dfoo%2Capp%3dbar", k8s.NewQuery(k8s.ClassOf(&corev1.Pod{}), "default", "", map[string]string{"name": "foo", "app": "bar"}, nil)},
	} {
		t.Run(x.p, func(t *testing.T) {
			u, _ := url.Parse(x.p)
			q, err := c.QueryFromURL(u)
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
