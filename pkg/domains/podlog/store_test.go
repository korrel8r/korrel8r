// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package podlog

import (
	"fmt"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Group cluster tests
func TestStoreGet(t *testing.T) {
	ctx := t.Context()
	test.SkipIfNoCluster(t)
	s, err := NewStore(nil, nil)
	require.NoError(t, err)
	c := s.K8sStore.Client()
	ns := test.TempNamespace(t, c, "podlog-")
	namespace := ns.Name

	// Start pods to create logs
	const n = 5
	require.NoError(t, c.Create(ctx, logger(namespace, "foo", "hello", 1, n, "box")))
	require.NoError(t, c.Create(ctx, logger(namespace, "bar", "goodbye", 1, n, "box", "bag")))
	// Wait for pods to be ready to avoid "cannot get logs from CRIO..." messages.
	test.WaitForPodReady(t, c, namespace, "foo")
	test.WaitForPodReady(t, c, namespace, "bar")

	makeQuery := func(name string, containers ...string) Query {
		return Query{
			Selector:   k8s.Selector{Namespace: namespace, Name: name},
			Containers: containers,
		}
	}

	t.Run("simple", func(t *testing.T) {
		got := getLogs(t, s, makeQuery("foo"), nil, n)
		assert.Equal(t, wantBodies("box: hello", 1, n), bodies(got))
		want := map[string]any{"k8s.container": "box", "k8s.pod.name": "foo", "k8s.pod.namespace.name": namespace}
		assert.Equal(t, want, got[0].(Object).Attributes)
		// FIXME test field values (Severity etc.)
	})

	t.Run("multipod", func(t *testing.T) {
		got := getLogs(t, s, makeQuery(""), nil, n)
		want := wantBodies("box: hello", 1, n)
		want = append(want, wantBodies("box: goodbye", 1, n)...)
		want = append(want, wantBodies("bag: goodbye", 1, n)...)
		assert.ElementsMatch(t, want, bodies(got))
	})

	t.Run("container", func(t *testing.T) {
		got := getLogs(t, s, makeQuery("", "bag"), nil, n)
		want := wantBodies("bag: goodbye", 1, n)
		assert.ElementsMatch(t, want, bodies(got))
	})

	t.Run("containers", func(t *testing.T) {
		got := getLogs(t, s, makeQuery("bar", "box", "bag"), nil, n)
		want := append(wantBodies("box: goodbye", 1, n), wantBodies("bag: goodbye", 1, n)...)
		assert.ElementsMatch(t, want, bodies(got))
	})

	t.Run("timestamps", func(t *testing.T) {
		q := makeQuery("foo")
		all := getLogs(t, s, q, nil, n)
		ts := func(i int) *time.Time { return ptr.To(all[i].(Object).Timestamp) }
		// Constraint excludes the first and last logs by timestamp.
		constraint := &korrel8r.Constraint{Start: ptr.To(ts(0).Add(time.Millisecond)), End: ptr.To(ts(n - 1).Add(-time.Millisecond))}
		got := getLogs(t, s, q, constraint, n-2)
		want := wantBodies("box: hello", 2, n-1)
		assert.Equal(t, want, bodies(got))
	})

	t.Run("limit", func(t *testing.T) {
		constraint := &korrel8r.Constraint{Limit: ptr.To(n - 2)}
		got := getLogs(t, s, makeQuery("foo"), constraint, n-2)
		// Limit returns the last N logs, not the first N.
		assert.Equal(t, wantBodies("box: hello", 3, n), bodies(got))
	})
}

func getLogs(t testing.TB, s korrel8r.Store, q Query, constraint *korrel8r.Constraint, min int) (logs []korrel8r.Object) {
	t.Helper()
	var err error
	assert.Eventually(t, func() bool {
		r := result.New(q.Class())
		err = s.Get(t.Context(), &q, constraint, r)
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

func bodies(logs []korrel8r.Object) []string {
	var bodies []string
	for _, l := range logs {
		bodies = append(bodies, l.(Object).Body.(string))
	}
	return bodies
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
