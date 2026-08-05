// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unused_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	slices2 "github.com/korrel8r/korrel8r/pkg/slices"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var customResources = []*metav1.APIResourceList{
	{
		GroupVersion: "discovery.k8s.io/v1",
		APIResources: []metav1.APIResource{
			{Kind: "EndpointSlice", Namespaced: true},
		},
	},
	{
		GroupVersion: "resource.k8s.io/v1",
		APIResources: []metav1.APIResource{
			{Kind: "ResourceSlice", Namespaced: false},
		},
	},
}

func setup() *engine.Engine {
	configs, err := config.Load("../all.yaml")
	if err != nil {
		panic(err)
	}
	nodeConfigs, err := config.Load("k8s_node.yaml")
	if err != nil {
		panic(err)
	}
	configs = append(configs, nodeConfigs...)
	for i := range configs {
		configs[i].Stores = nil
	}
	c := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build()
	s, err := k8s.Domain.NewStoreWithDiscovery(
		c,
		&rest.Config{},
		&fakediscovery.FakeDiscovery{
			Fake: &clienttesting.Fake{Resources: customResources},
		})
	if err != nil {
		panic(err)
	}
	e, err := engine.Build().
		Domains(domains.All...).
		Stores(s).
		Config(configs).
		Engine()
	if err != nil {
		panic(err)
	}
	return e
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func newK8s(class, namespace, name string, object k8s.Object) k8s.Object {
	if object == nil {
		object = k8s.Object{}
	}
	u := k8s.ToUnstructured(object)
	kc := k8s.Domain.Class(class)
	if kc == nil {
		_, file, line, _ := runtime.Caller(0)
		panic(fmt.Errorf("class not found: k8s:%v. To add it, update customResources. See %v:%v", class, file, line))
	}
	c := kc.(k8s.Class)
	u.GetObjectKind().SetGroupVersionKind(c.GVK())
	u.SetNamespace(namespace)
	u.SetName(name)
	return k8s.FromUnstructured(u)
}

type ruleTest struct {
	rule  string
	start korrel8r.Object
	want  []string
}

func (x ruleTest) Run(t *testing.T) {
	t.Helper()
	t.Run(fmt.Sprintf("%v(%v)", x.rule, x.start), func(t *testing.T) {
		t.Helper()
		e := setup()
		r := e.Rule(x.rule)
		if assert.NotNil(t, r, "missing rule: "+x.rule) {
			got, err := r.Apply(x.start)
			if assert.NoError(t, err, x.rule) && assert.NotNil(t, got) {
				assert.Equal(t, x.want, slices2.Strings(got))
			}
		}
	})
}

func TestK8sNodeRules(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule: "PodToNode",
			start: newK8s("Pod", "ns", "pod", k8s.Object{
				"spec": k8s.Object{
					"nodeName": "worker-1",
				},
			}),
			want: []string{`k8s:Node.v1:{"name":"worker-1"}`},
		},
		{
			rule: "VolumeAttachmentToNode",
			start: newK8s("VolumeAttachment.storage.k8s.io", "", "va-1", k8s.Object{
				"spec": k8s.Object{
					"nodeName": "worker-1",
				},
			}),
			want: []string{`k8s:Node.v1:{"name":"worker-1"}`},
		},
		{
			rule:  "NodeToResourceSlice",
			start: newK8s("Node", "", "worker-1", nil),
			want:  []string{`k8s:ResourceSlice.v1.resource.k8s.io:{"fields":{"spec.nodeName":"worker-1"}}`},
		},
		{
			rule: "EndpointSliceToNode",
			start: newK8s("EndpointSlice.discovery.k8s.io", "ns", "eps-1", k8s.Object{
				"endpoints": []k8s.Object{
					{"nodeName": "worker-1"},
					{"nodeName": "worker-2"},
				},
			}),
			want: []string{
				`k8s:Node.v1:{"name":"worker-1"}`,
				`k8s:Node.v1:{"name":"worker-2"}`,
			},
		},
		{
			rule: "ResourceSliceToNode",
			start: newK8s("ResourceSlice.resource.k8s.io", "", "rs-1", k8s.Object{
				"spec": k8s.Object{
					"nodeName": "worker-1",
				},
			}),
			want: []string{`k8s:Node.v1:{"name":"worker-1"}`},
		},
	} {
		x.Run(t)
	}
}
