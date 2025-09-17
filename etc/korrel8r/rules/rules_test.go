// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package rules is a test-only package to unit test YAML rules.
package rules_test

// Test use of rules in graph traversal.

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setup() *engine.Engine {
	configs, err := config.Load("all.yaml")
	if err != nil {
		panic(err)
	}
	for i := range configs {
		configs[i].Stores = nil // Use fake stores, not configured defaults.
	}
	c := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build()
	// Create a fake discovery with custom resources needed by the test.
	fd := &fakediscovery.FakeDiscovery{
		Fake: &clienttesting.Fake{
			Resources: []*metav1.APIResourceList{
				{GroupVersion: "kubevirt.io", APIResources: []metav1.APIResource{
					{Kind: "VirtualMachineInstance", Namespaced: true},
				}},
			},
		},
	}
	s, err := k8s.Domain.NewStoreWithDiscovery(c, &rest.Config{}, fd)
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
	e := setup()
	for _, r := range e.Rules() {
		rules.Add(r.Name())
	}
	m.Run()
	if len(rules) > 0 {
		fmt.Printf("FAIL: %v rules not tested:\n- %v\n", len(rules), strings.Join(slices.Collect(maps.Keys(rules)), "\n- "))
		os.Exit(1)
	}
}

// tested marks a rule as having been tested.
func tested(ruleName string) { rules.Remove(ruleName) }

var rules = unique.Set[string]{}

type ruleTest struct {
	rule  string
	start korrel8r.Object
	query string
}

func (x ruleTest) Run(t *testing.T) {
	t.Helper()
	t.Run(fmt.Sprintf("%v(%v)", x.rule, test.JSONString(x.start)), func(t *testing.T) {
		t.Helper()
		e := setup()
		r := e.Rule(x.rule)
		if assert.NotNil(t, r, "missing rule: "+x.rule) {
			got, err := r.Apply(x.start)
			if assert.NoError(t, err, x.rule) && assert.NotNil(t, got) {
				assert.Equal(t, x.query, got.String())
			}
		}
		tested(x.rule)
	})
}

func newK8s(class, namespace, name string, object k8s.Object) k8s.Object {
	if object == nil {
		object = k8s.Object{}
	}
	u := k8s.ToUnstructured(object)
	c := k8s.Domain.Class(class).(k8s.Class)
	u.GetObjectKind().SetGroupVersionKind(c.GVK())
	u.SetNamespace(namespace)
	u.SetName(name)
	return k8s.FromUnstructured(u)
}

func k8sEvent(o k8s.Object, name string) k8s.Object {
	u := k8s.ToUnstructured(o)
	gvk := u.GetObjectKind().GroupVersionKind()
	e := newK8s("Event.v1", name, u.GetNamespace(), k8s.Object{
		"involvedObject": k8s.Object{
			"kind":       gvk.Kind,
			"namespace":  u.GetNamespace(),
			"name":       u.GetName(),
			"apiVersion": gvk.GroupVersion().String(),
		}})
	return e
}

func k8sEvent2(o k8s.Object, name string) k8s.Object {
	u := k8s.ToUnstructured(o)
	gvk := u.GetObjectKind().GroupVersionKind()
	e := newK8s("Event.v1.events.k8s.io", name, u.GetNamespace(), k8s.Object{
		"regarding": k8s.Object{
			"kind":       gvk.Kind,
			"namespace":  u.GetNamespace(),
			"name":       u.GetName(),
			"apiVersion": gvk.GroupVersion().String(),
		}})
	return e
}
