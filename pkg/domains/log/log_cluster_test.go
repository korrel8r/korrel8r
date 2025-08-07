// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newQuery(t *testing.T, q string, args ...any) *Query {
	t.Helper()
	query, err := NewQuery(fmt.Sprintf(q, args...))
	require.NoError(t, err)
	return query
}

func routeHost(t *testing.T, c client.Client, namespace, name string) string {
	route := unstructured.Unstructured{Object: map[string]any{"apiVersion": "route.openshift.io/v1", "kind": "Route"}}
	err := c.Get(t.Context(), client.ObjectKey{Namespace: namespace, Name: name}, &route)
	if err != nil {
		t.Skipf("no lokistack route in %v/%v: %v", namespace, name, err)
	}
	host, ok, err := unstructured.NestedString(route.Object, "spec", "host")
	require.NoError(t, err)
	require.True(t, ok)
	return host
}

// TestPodQueries tests that pod-style queries work for direct and loki stores.
func TestPodQueries(t *testing.T) {
	// Set up pods to create logs.
	c := test.SkipIfNoCluster(t)
	const n = 5
	namespace := test.TempNamespace(t, c, "podlog-").Name
	ctx := t.Context()
	require.NoError(t, c.Create(ctx, logger(namespace, "foo", "hello", 1, n, "box")))
	require.NoError(t, c.Create(ctx, logger(namespace, "bar", "goodbye", 1, n, "box", "bag")))
	test.WaitForPodReady(t, c, namespace, "foo")
	test.WaitForPodReady(t, c, namespace, "bar")

	for _, storeType := range []string{"lokiStack", "direct"} {
		t.Run(storeType, func(t *testing.T) {
			var storeConfig config.Store
			switch storeType {
			case "direct":
				storeConfig = config.Store{"direct": "true"}
			case "lokiStack":
				storeConfig = config.Store{"lokiStack": "https://" + routeHost(t, c, "openshift-logging", "logging-loki")}
			}
			s, err := Domain.Store(storeConfig)
			require.NoError(t, err)

			t.Run("simple", func(t *testing.T) {
				got := getLogs(t, s, newQuery(t, "log:application:{name: foo, namespace: %v}", namespace), nil, n)
				if assert.Equal(t, wantBodies("box: hello", 1, n), bodies(got)) {
					m := got[0].(Object)
					assert.Equal(t, "box: hello 1", m["body"])
					assert.Equal(t, "box", m["k8s_container_name"], m)
					assert.Equal(t, "foo", m["kubernetes_pod_name"])
					assert.Equal(t, namespace, m["k8s_namespace_name"])
				}
			})

			t.Run("multipod", func(t *testing.T) {
				got := getLogs(t, s, newQuery(t, `log:application:{namespace: %v}`, namespace), nil, n)
				want := wantBodies("box: hello", 1, n)
				want = append(want, wantBodies("box: goodbye", 1, n)...)
				want = append(want, wantBodies("bag: goodbye", 1, n)...)
				assert.ElementsMatch(t, want, bodies(got))
				// Make sure results are ordered by timestamp
				times := fields("timestamp", got)
				assert.True(t, sort.StringsAreSorted(times), times)
			})

			t.Run("container", func(t *testing.T) {
				got := getLogs(t, s, newQuery(t, `log:application:{containers: [bag], namespace: %v}`, namespace), nil, n)
				want := wantBodies("bag: goodbye", 1, n)
				assert.ElementsMatch(t, want, bodies(got))
			})

			t.Run("containers", func(t *testing.T) {
				got := getLogs(t, s, newQuery(t, `log:application:{containers: ["bar","box","bag"], namespace: %v}`, namespace), nil, n)
				want := wantBodies("box: hello", 1, n)
				want = append(want, wantBodies("box: goodbye", 1, n)...)
				want = append(want, wantBodies("bag: goodbye", 1, n)...)
				assert.ElementsMatch(t, want, bodies(got))
				// Make sure results are ordered by timestamp
				times := fields("timestamp", got)
				assert.True(t, sort.StringsAreSorted(times), times)
			})

			t.Run("timestamps", func(t *testing.T) {
				q := newQuery(t, `log:application:{name: foo, namespace: %v}`, namespace)
				all := getLogs(t, s, q, nil, n)
				require.Len(t, all, n)
				// Make a constraint that excludes the first and last logs by timestamp.
				start, err := all[0].(Object).SortTime()
				require.NoError(t, err)
				end, err := all[n-1].(Object).SortTime()
				require.NoError(t, err)
				constraint := &korrel8r.Constraint{Start: ptr.To(start.Add(1)), End: ptr.To(end.Add(-1))}

				got := getLogs(t, s, q, constraint, n-2)
				want := wantBodies("box: hello", 2, n-1)
				assert.Equal(t, want, bodies(got))
			})

			t.Run("limit", func(t *testing.T) {
				constraint := &korrel8r.Constraint{Limit: ptr.To(n - 2)}
				got := getLogs(t, s, newQuery(t, `log:application:{name: foo, namespace: %v}`, namespace), constraint, n-2)
				// Limit returns the last N logs, not the first N.
				assert.Equal(t, wantBodies("box: hello", 1, 3), bodies(got))
			})

			t.Run("timeout", func(t *testing.T) {
				q := newQuery(t, `log:application:{namespace: %v}`, namespace)
				constraint := &korrel8r.Constraint{Timeout: ptr.To(time.Nanosecond)}
				err := s.Get(t.Context(), q, constraint, result.New(q.Class()))
				t.Logf("(%T)%v", err, err)
				assert.Error(t, err)
			})
		})
	}
}

func getLogs(t testing.TB, s korrel8r.Store, q *Query, constraint *korrel8r.Constraint, min int) (logs []korrel8r.Object) {
	t.Helper()
	var err error
	assert.Eventually(t, func() bool {
		r := result.New(q.Class())
		err = s.Get(t.Context(), q, constraint, r)
		logs = r.List()
		ok := err == nil && len(logs) >= min
		if !ok {
			t.Logf("waiting for logs, want %v got %v: %v", min, len(logs), q.String())
		}
		return ok
	}, time.Minute, time.Second, "query %v, want %v logs got %v", q.String(), min, len(logs))
	assert.NoError(t, err)
	return logs
}

func bodies(logs []korrel8r.Object) []string { return fields("body", logs) }

func fields(field string, logs []korrel8r.Object) []string {
	var result []string
	for _, l := range logs {
		result = append(result, l.(Object)[field])
	}
	return result
}

func wantBodies(text string, first, last int) []string {
	var want []string
	for i := first; i <= last; i++ {
		want = append(want, fmt.Sprintf("%v %v", text, i))
	}
	return want
}

func logger(namespace, name, text string, first, last int, containers ...string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: name},
		Spec:       corev1.PodSpec{},
	}
	for _, c := range containers {
		pod.Spec.Containers = append(
			pod.Spec.Containers,
			corev1.Container{
				Name:  c,
				Image: "quay.io/quay/busybox",
				Command: []string{
					"sh", "-c",
					fmt.Sprintf(`for i in $(seq %v %v); do echo "%v: %v $i"; sleep .001; done; sleep 1h`, first, last, c, text),
				},
				SecurityContext: test.DefaultSecurityContext,
			})
	}
	return pod
}
