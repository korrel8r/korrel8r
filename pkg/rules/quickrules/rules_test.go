// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package quickrules_test

import (
	"fmt"
	"maps"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules/quickrules"
	slices2 "github.com/korrel8r/korrel8r/pkg/slices"
	"github.com/korrel8r/korrel8r/pkg/status"
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

// Add custom resources used by tests to this list, otherwise tests will fail wit "class not found"
var customResourcesForTests = []*metav1.APIResourceList{
	{
		GroupVersion: "operators.coreos.com/v1alpha1",
		APIResources: []metav1.APIResource{
			{Kind: "ClusterServiceVersion", Namespaced: false},
			{Kind: "Subscription", Namespaced: false},
			{Kind: "InstallPlan", Namespaced: true},
			{Kind: "CatalogSource", Namespaced: true},
		},
	},
	{
		GroupVersion: "operators.coreos.com/v1",
		APIResources: []metav1.APIResource{
			{Kind: "Operator", Namespaced: false},
		},
	},
	{
		GroupVersion: "apiextensions.k8s.io/v1",
		APIResources: []metav1.APIResource{
			{Kind: "CustomResourceDefinition", Namespaced: false},
		},
	},
	{
		GroupVersion: "kubevirt.io/v1",
		APIResources: []metav1.APIResource{
			{Name: "virtualmachineinstances", Kind: "VirtualMachineInstance", Namespaced: true},
			{Name: "virtualmachines", Kind: "VirtualMachine", Namespaced: true},
			{Name: "virtualmachineinstancemigrations", Kind: "VirtualMachineInstanceMigration", Namespaced: true},
		},
	},
	{
		GroupVersion: "cdi.kubevirt.io/v1beta1",
		APIResources: []metav1.APIResource{
			{Kind: "DataVolume", Namespaced: true},
		},
	},
	{
		GroupVersion: "snapshot.kubevirt.io/v1beta1",
		APIResources: []metav1.APIResource{
			{Kind: "VirtualMachineSnapshot", Namespaced: true},
			{Kind: "VirtualMachineRestore", Namespaced: true},
		},
	},
	{
		GroupVersion: "export.kubevirt.io/v1beta1",
		APIResources: []metav1.APIResource{
			{Kind: "VirtualMachineExport", Namespaced: true},
		},
	},
	{
		GroupVersion: "k8s.cni.cncf.io/v1",
		APIResources: []metav1.APIResource{
			{Kind: "NetworkAttachmentDefinition", Namespaced: true},
		},
	},
	{
		GroupVersion: "instancetype.kubevirt.io/v1beta1",
		APIResources: []metav1.APIResource{
			{Kind: "VirtualMachineInstancetype", Namespaced: true},
			{Kind: "VirtualMachineClusterInstancetype", Namespaced: false},
			{Kind: "VirtualMachinePreference", Namespaced: true},
			{Kind: "VirtualMachineClusterPreference", Namespaced: false},
		},
	},
	{
		GroupVersion: "config.openshift.io/v1",
		APIResources: []metav1.APIResource{
			{Kind: "Node", Namespaced: false},
		},
	},
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
	{
		GroupVersion: "networking.k8s.io/v1",
		APIResources: []metav1.APIResource{
			{Kind: "Ingress", Namespaced: true},
			{Kind: "NetworkPolicy", Namespaced: true},
		},
	},
	{
		GroupVersion: "route.openshift.io/v1",
		APIResources: []metav1.APIResource{
			{Kind: "Route", Namespaced: true},
		},
	},
	{
		GroupVersion: "rbac.authorization.k8s.io/v1",
		APIResources: []metav1.APIResource{
			{Kind: "Role", Namespaced: true},
			{Kind: "RoleBinding", Namespaced: true},
			{Kind: "ClusterRole", Namespaced: false},
			{Kind: "ClusterRoleBinding", Namespaced: false},
		},
	},
	{
		GroupVersion: "autoscaling/v2",
		APIResources: []metav1.APIResource{
			{Kind: "HorizontalPodAutoscaler", Namespaced: true},
		},
	},
}

func setup() *engine.Engine {
	c := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		Build()
	s, err := k8s.Domain.NewStoreWithDiscovery(
		c,
		&rest.Config{},
		&fakediscovery.FakeDiscovery{
			Fake: &clienttesting.Fake{Resources: customResourcesForTests},
		})
	if err != nil {
		panic(err)
	}
	b := engine.Build().Domains(domains.All...)
	b.Stores(s)
	b.Rules(quickrules.Rules(b.GetDomains())...)
	b.StatusRules(quickrules.StatusRules(b.GetDomains())...)
	e, err := b.Engine()
	if err != nil {
		panic(err)
	}
	return e
}

func TestMain(m *testing.M) {
	e := setup()
	for _, r := range e.Rules() {
		untestedRules.Add(r.Name())
	}
	m.Run()
	if len(untestedRules) < len(e.Rules()) && len(untestedRules) > 0 {
		fmt.Printf("FAIL: %v rules not tested:\n- %v\n", len(untestedRules), strings.Join(slices.Collect(maps.Keys(untestedRules)), "\n- "))
		os.Exit(1)
	}
}

func tested(ruleName string) { untestedRules.Remove(ruleName) }

var untestedRules = unique.Set[string]{}

type ruleTest struct {
	rule  string
	start korrel8r.Object
	want  []string
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
				assert.Equal(t, x.want, slices2.Strings(got))
			}
		}
		tested(x.rule)
	})
}

type statusRuleTest struct {
	rule   string
	class  string
	domain korrel8r.Domain
	start  korrel8r.Object
	want   []string
}

func (x statusRuleTest) Run(t *testing.T) {
	t.Helper()
	t.Run(fmt.Sprintf("%v(%v)", x.rule, test.JSONString(x.start)), func(t *testing.T) {
		t.Helper()
		e := setup()
		d := x.domain
		if d == nil {
			d = k8s.Domain
		}
		c := d.Class(x.class)
		if !assert.NotNil(t, c, "missing class: "+x.class) {
			return
		}
		var m status.Rule
		for _, mm := range e.StatusRulesFor(c) {
			if mm.Name() == x.rule {
				m = mm
				break
			}
		}
		if !assert.NotNil(t, m, "missing status rule: "+x.rule) {
			return
		}
		got, err := m.Apply(x.start)
		if assert.NoError(t, err, x.rule) {
			assert.Equal(t, x.want, got)
		}
	})
}

func newK8s(class, namespace, name string, object k8s.Object) k8s.Object {
	if object == nil {
		object = k8s.Object{}
	}
	u := k8s.ToUnstructured(object)
	kc := k8s.Domain.Class(class)
	if kc == nil {
		_, file, line, _ := runtime.Caller(0)
		panic(fmt.Errorf("class not found: k8s:%v. To add it, update customResourcesForTests. See %v:%v", class, file, line))
	}
	c := kc.(k8s.Class)
	u.GetObjectKind().SetGroupVersionKind(c.GVK())
	u.SetNamespace(namespace)
	u.SetName(name)
	return k8s.FromUnstructured(u)
}

func k8sEvent(o k8s.Object, name string) k8s.Object {
	u := k8s.ToUnstructured(o)
	gvk := u.GetObjectKind().GroupVersionKind()
	e := newK8s("Event.v1", u.GetNamespace(), name, k8s.Object{
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
	e := newK8s("Event.v1.events.k8s.io", u.GetNamespace(), name, k8s.Object{
		"regarding": k8s.Object{
			"kind":       gvk.Kind,
			"namespace":  u.GetNamespace(),
			"name":       u.GetName(),
			"apiVersion": gvk.GroupVersion().String(),
		}})
	return e
}
