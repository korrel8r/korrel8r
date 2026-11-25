// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	namespace  = Domain.Class("Namespace").(Class)
	pod        = Domain.Class("Pod").(Class)
	deployment = Domain.Class("Deployment.apps").(Class)
)

func newQuery(c Class, namespace, name string, labels, fields map[string]string) *Query {
	return NewQuery(c, Selector{Namespace: namespace, Name: name, Labels: labels, Fields: fields})
}

func TestDomain_Class(t *testing.T) {
	for _, x := range []struct {
		name    string
		group   string
		version string
		kind    string
	}{
		{"Pod", "", "v1", "Pod"},
		{"Pod.", "", "v1", "Pod"},
		{"Pod.v1", "", "v1", "Pod"},
		{"Deployment.apps", "apps", "v1", "Deployment"},
		{"Deployment.v1.apps", "apps", "v1", "Deployment"},
		{"StorageClass.storage.k8s.io", "storage.k8s.io", "v1", "StorageClass"},
		{"StorageClass.v1.storage.k8s.io", "storage.k8s.io", "v1", "StorageClass"},
	} {
		t.Run(x.name, func(t *testing.T) {
			kc := Domain.Class(x.name)
			assert.IsType(t, Class{}, kc, "%v", x.name)
			c := kc.(Class)
			assert.NotNil(t, c, x.name)
			assert.Equal(t, schema.GroupVersionKind{Group: x.group, Version: x.version, Kind: x.kind}, c.GVK())
		})
	}
}

func TestDomain_Class_invalid(t *testing.T) {
	for _, name := range []string{
		"NoSuchKind",
		"NoSuchKind.v1",
		"Pod.v99",
		"Deployment.v1.nosuchgroup",
	} {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, Domain.Class(name))
		})
	}
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

func TestStripUnused(t *testing.T) {
	for _, x := range []struct {
		name string
		o    Object
		want Object
	}{
		{
			name: "managedFields removed, rest untouched",
			o:    Object{"kind": "Pod", "metadata": map[string]any{"name": "fred", "managedFields": []any{"junk"}}},
			want: Object{"kind": "Pod", "metadata": map[string]any{"name": "fred"}},
		},
		{
			name: "no managedFields",
			o:    Object{"metadata": map[string]any{"name": "fred"}},
			want: Object{"metadata": map[string]any{"name": "fred"}},
		},
		{
			name: "no metadata",
			o:    pod.New(),
			want: pod.New(),
		},
		{
			name: "metadata is not a map",
			o:    Object{"metadata": "nonsense"},
			want: Object{"metadata": "nonsense"},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			assert.Equal(t, x.want, stripUnused(x.o))
		})
	}
}

// managedFields is large and korrel8r never reads it, so the store must not retain it.
func TestStore_Get_StripsManagedFields(t *testing.T) {
	managed := []metav1.ManagedFieldsEntry{{
		Manager:    "kubectl",
		Operation:  metav1.ManagedFieldsOperationApply,
		APIVersion: "v1",
		FieldsType: "FieldsV1",
		FieldsV1:   metav1.NewFieldsV1(`{"f:metadata":{"f:labels":{}}}`),
	}}
	c := fake.NewClientBuilder().
		WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme)).
		WithObjects(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "fred", Namespace: "x", ManagedFields: managed},
		}).Build()
	store, err := Domain.NewStore(c, &rest.Config{})
	require.NoError(t, err)

	// Cover both the single-object and the list paths.
	for _, q := range []korrel8r.Query{
		newQuery(pod, "x", "fred", nil, nil),
		newQuery(pod, "x", "", nil, nil),
	} {
		t.Run(fmt.Sprintf("%v", q), func(t *testing.T) {
			var result mock.Result
			require.NoError(t, store.Get(context.Background(), q, nil, &result))
			require.Len(t, result, 1)
			o := result[0].(Object)
			assert.Equal(t, "fred", ToUnstructured(o).GetName(), "wrong object")
			assert.NotContains(t, o["metadata"], "managedFields")
		})
	}
}

func TestClass_Unmarshal_StripsManagedFields(t *testing.T) {
	o, err := pod.Unmarshal([]byte(`{"kind":"Pod","metadata":{"name":"fred","managedFields":[{"manager":"kubectl"}]}}`))
	require.NoError(t, err)
	assert.NotContains(t, o.(Object)["metadata"], "managedFields")
	assert.Equal(t, "fred", ToUnstructured(o.(Object)).GetName())
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

func widgets(group string, kinds ...string) []*metav1.APIResourceList {
	l := &metav1.APIResourceList{GroupVersion: group + "/v1"}
	for _, k := range kinds {
		l.APIResources = append(l.APIResources, metav1.APIResource{Name: strings.ToLower(k) + "s", Kind: k, Namespaced: true})
	}
	return []*metav1.APIResourceList{l}
}

func TestRefreshUpdatesPreferredVersion(t *testing.T) {
	d := newDomain()
	d.addClasses(nil, widgets("examples.io", "Widget")) // Initially only v1 is known.
	assert.Equal(t, d.Class("Widget.v1.examples.io"), d.Class("Widget.examples.io"))

	preferred := metav1.GroupVersionForDiscovery{GroupVersion: "examples.io/v2", Version: "v2"}
	d.addClasses([]*metav1.APIGroup{{Name: "examples.io", PreferredVersion: preferred}},
		[]*metav1.APIResourceList{{GroupVersion: "examples.io/v2", APIResources: []metav1.APIResource{{Name: "widgets", Kind: "Widget"}}}})
	assert.Equal(t, d.Class("Widget.v2.examples.io"), d.Class("Widget.examples.io"))
	assert.NotNil(t, d.Class("Widget.v1.examples.io")) // Existing classes remain available.
}

func TestRefreshAddsOnlyNewClasses(t *testing.T) {
	d := newDomain()
	assert.Zero(t, d.ClassVersion()) // Built-in classes are not additions.
	fake := &clienttesting.Fake{Resources: widgets("examples.io", "Widget")}
	di := &fakediscovery.FakeDiscovery{Fake: fake}
	d.refresh(di)
	assert.Equal(t, 1, d.ClassVersion())
	assert.NotNil(t, d.Class("Widget.examples.io"))
	assert.True(t, d.resources[d.Class("Widget.examples.io")].Namespaced)

	d.refresh(di)
	assert.Equal(t, 1, d.ClassVersion())

	// A CRD recreated with the same GVK may have a different resource name or scope.
	fake.Resources = []*metav1.APIResourceList{{GroupVersion: "examples.io/v1", APIResources: []metav1.APIResource{{
		Name: "newwidgets", Kind: "Widget", Namespaced: false,
	}}}}
	d.refresh(di)
	assert.Equal(t, 1, d.ClassVersion())
	assert.Equal(t, "newwidgets.examples.io", d.CRDName("examples.io/v1", "Widget"))
	assert.False(t, d.resources[d.Class("Widget.examples.io")].Namespaced)

	fake.Resources = nil // Deletion does not unregister the old class.
	d.refresh(di)
	assert.Equal(t, 1, d.ClassVersion())
	assert.NotNil(t, d.Class("Widget.examples.io"))
}

// testWatchClient returns watchers from a channel, or an error if fail is set.
type testWatchClient struct {
	client.WithWatch
	watchers chan *watch.FakeWatcher
	fail     atomic.Bool
	calls    atomic.Int64
}

func newTestWatchClient() *testWatchClient {
	return &testWatchClient{watchers: make(chan *watch.FakeWatcher)}
}

func (c *testWatchClient) Watch(ctx context.Context, _ client.ObjectList, _ ...client.ListOption) (watch.Interface, error) {
	c.calls.Add(1)
	if c.fail.Load() {
		return nil, errors.New("watch failed")
	}
	select {
	case w := <-c.watchers:
		return w, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// fastWatchRetry shortens watch delays for a test.
func fastWatchRetry(t *testing.T) {
	r, lo, hi := refreshDelay, watchRetryMin, watchRetryMax
	t.Cleanup(func() { refreshDelay, watchRetryMin, watchRetryMax = r, lo, hi })
	refreshDelay, watchRetryMin, watchRetryMax = time.Millisecond, time.Millisecond, 4*time.Millisecond
}

// currentWatcher returns the domain's current CRD watcher, and stops it at the end of the test.
func currentWatcher(t *testing.T, d *domain) *crdWatcher {
	d.watchMu.Lock()
	defer d.watchMu.Unlock()
	w := d.watcher
	require.NotNil(t, w)
	t.Cleanup(func() { w.cancel(); <-w.done })
	return w
}

func TestWatchClasses(t *testing.T) {
	fastWatchRetry(t)
	d := newDomain()
	c := newTestWatchClient()
	fake := &clienttesting.Fake{Resources: widgets("watch.example.io", "Widget")}
	d.startWatch(c, &fakediscovery.FakeDiscovery{Fake: fake})
	currentWatcher(t, d)

	w := watch.NewFake()
	c.watchers <- w
	w.Add(&unstructured.Unstructured{})
	assert.Eventually(t, func() bool { return d.Class("Widget.v1.watch.example.io") != nil }, time.Second, time.Millisecond)

	// A CRD added while the watch is down is found on reconnect, even if
	// the new watch sends no events for existing CRDs.
	w.Stop()
	fake.Lock()
	fake.Resources = widgets("watch.example.io", "Widget", "Gadget")
	fake.Unlock()
	w = watch.NewFake()
	c.watchers <- w
	assert.Eventually(t, func() bool { return d.Class("Gadget.v1.watch.example.io") != nil }, time.Second, time.Millisecond)
	assert.Equal(t, 2, d.ClassVersion())
}

func TestWatchUpdatesResourceMetadata(t *testing.T) {
	fastWatchRetry(t)
	d := newDomain()
	c := newTestWatchClient()
	fake := &clienttesting.Fake{Resources: widgets("metadata.example.io", "Widget")}
	d.startWatch(c, &fakediscovery.FakeDiscovery{Fake: fake})
	currentWatcher(t, d)

	w := watch.NewFake()
	c.watchers <- w
	const apiVersion = "metadata.example.io/v1"
	assert.Eventually(t, func() bool {
		return d.CRDName(apiVersion, "Widget") == "widgets.metadata.example.io"
	}, time.Second, time.Millisecond)
	version := d.ClassVersion()
	class := d.Class("Widget.v1.metadata.example.io")
	require.NotNil(t, class)

	fake.Lock()
	fake.Resources = []*metav1.APIResourceList{{GroupVersion: apiVersion, APIResources: []metav1.APIResource{{
		Name: "renamedwidgets", Kind: "Widget", Namespaced: false,
	}}}}
	fake.Unlock()
	w.Modify(&unstructured.Unstructured{})
	assert.Eventually(t, func() bool {
		return d.CRDName(apiVersion, "Widget") == "renamedwidgets.metadata.example.io"
	}, time.Second, time.Millisecond)
	d.m.Lock()
	assert.False(t, d.resources[class].Namespaced)
	d.m.Unlock()
	assert.Equal(t, version, d.ClassVersion())
	assert.Equal(t, class, d.Class("Widget.v1.metadata.example.io"))
}

func TestWatchClassesRetriesFailedWatch(t *testing.T) {
	fastWatchRetry(t)
	d := newDomain()
	c := newTestWatchClient()
	c.fail.Store(true)
	fake := &clienttesting.Fake{Resources: widgets("retry.example.io", "Widget")}
	d.startWatch(c, &fakediscovery.FakeDiscovery{Fake: fake})
	cw := currentWatcher(t, d)
	assert.Eventually(t, func() bool { return cw.failing.Load() && c.calls.Load() >= 3 }, time.Second, time.Millisecond)

	// Recovers when the watch succeeds.
	c.fail.Store(false)
	c.watchers <- watch.NewFake()
	assert.Eventually(t, func() bool { return !cw.failing.Load() }, time.Second, time.Millisecond)
	assert.NotNil(t, d.Class("Widget.v1.retry.example.io"))
}

func TestNewStoreKeepsHealthyWatcher(t *testing.T) {
	fastWatchRetry(t)
	d := newDomain()
	di := &fakediscovery.FakeDiscovery{Fake: &clienttesting.Fake{Resources: widgets("keep.example.io", "Widget")}}
	first, second := newTestWatchClient(), newTestWatchClient()
	_, err := d.NewStoreWithDiscovery(first, &rest.Config{}, di)
	require.NoError(t, err)
	cw := currentWatcher(t, d)
	w1 := watch.NewFake()
	first.watchers <- w1

	// A re-created store does not disturb a working watch.
	_, err = d.NewStoreWithDiscovery(second, &rest.Config{}, di)
	require.NoError(t, err)
	assert.Same(t, cw, currentWatcher(t, d))
	assert.NoError(t, cw.ctx.Err())
	assert.False(t, w1.IsStopped())
	assert.Zero(t, second.calls.Load())
}

func TestNewStoreReplacesFailingWatcher(t *testing.T) {
	fastWatchRetry(t)
	d := newDomain()
	di := &fakediscovery.FakeDiscovery{Fake: &clienttesting.Fake{Resources: widgets("replace.example.io", "Widget")}}
	first, second := newTestWatchClient(), newTestWatchClient()
	first.fail.Store(true)
	_, err := d.NewStoreWithDiscovery(first, &rest.Config{}, di)
	require.NoError(t, err)
	old := currentWatcher(t, d)
	assert.Eventually(t, old.failing.Load, time.Second, time.Millisecond)

	_, err = d.NewStoreWithDiscovery(second, &rest.Config{}, di)
	require.NoError(t, err)
	cw := currentWatcher(t, d)
	assert.NotSame(t, old, cw)
	assert.Error(t, old.ctx.Err())
	w2 := watch.NewFake()
	select {
	case second.watchers <- w2:
	case <-time.After(time.Second):
		t.Fatal("new store did not start a CRD watch")
	}
	assert.Eventually(t, func() bool { return !cw.failing.Load() }, time.Second, time.Millisecond)
	assert.False(t, w2.IsStopped())
}

func TestDomain_DefaultClasses(t *testing.T) {
	want := []korrel8r.Class{deployment, namespace, pod}
	assert.Subset(t, Domain.Classes(), want)
}

func TestClass_DefaultNamespaceed(t *testing.T) {
	assert.False(t, namespace.Namespaced())
	assert.True(t, deployment.Namespaced())
	assert.True(t, pod.Namespaced())
}
