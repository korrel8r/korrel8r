package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ctx = context.Background()

func TestStore_Get_PlainLoki(t *testing.T) {
	t.Parallel()
	l := test.RequireLokiServer(t)
	lines := []string{"hello", "there", "mr. frog"}
	err := l.Push(map[string]string{"test": "loki"}, lines...)
	require.NoError(t, err)
	s, err := NewPlainLokiStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)

	var want []korrel8r.Object
	for _, l := range lines {
		want = append(want, Object(l))
	}
	q := &Query{LogQL: `{test="loki"}`}
	result := korrel8r.NewListResult()
	require.NoError(t, s.Get(ctx, q, result))
	assert.Equal(t, want, result.List())
}

func TestLokiStackStore_Get(t *testing.T) {
	t.Parallel()
	test.SkipIfNoCluster(t)
	c := test.K8sClient
	ns := test.TempNamespace(t, c)
	want := []string{"hello", "there", "mr.", "frog"}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "logger", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "logger",
				Image:   "quay.io/quay/busybox",
				Command: []string{"sh", "-c", fmt.Sprintf("echo %v; sleep infinity", strings.Join(want, "; echo "))}}}},
	}
	require.NoError(t, c.Create(ctx, &pod))
	s, err := NewOpenshiftLokiStackStore(ctx, c, test.RESTConfig)
	require.NoError(t, err)
	logQL := fmt.Sprintf(`{kubernetes_pod_name="%v", kubernetes_namespace_name="%v"}`, pod.Name, pod.Namespace)
	q := &Query{LogQL: logQL, Tenant: "application"}
	var result korrel8r.ListResult
	assert.Eventually(t, func() bool {
		result = nil
		err = s.Get(ctx, q, &result)
		require.NoError(t, err)
		t.Logf("waiting for 4 logs, got %v. %v%v", len(result), s, q)
		return len(result) >= 3
	}, time.Minute, 5*time.Second)
	var got []string
	for _, obj := range result {
		var m map[string]any
		line := string(obj.(Object))
		assert.NoError(t, json.Unmarshal([]byte(line), &m), line)
		got = append(got, m["message"].(string))
	}
	assert.Equal(t, want, got)
}

func TestStoreGet_Constraint(t *testing.T) {
	t.Skip("TODO re-enable when constraints are implemented properly")
	t.Parallel()
	l := test.RequireLokiServer(t)

	err := l.Push(map[string]string{"test": "loki"}, "much", "too", "early")
	require.NoError(t, err)

	t1 := time.Now()
	err = l.Push(map[string]string{"test": "loki"}, "right", "on", "time")
	require.NoError(t, err)
	t2 := time.Now()

	err = l.Push(map[string]string{"test": "loki"}, "much", "too", "late")
	require.NoError(t, err)
	s, err := NewPlainLokiStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)

	for n, x := range []struct {
		q    korrel8r.Query
		c    *korrel8r.Constraint
		want []korrel8r.Object
	}{
		{
			q:    &Query{LogQL: `{test="loki"}`},
			c:    &korrel8r.Constraint{End: &t1},
			want: []korrel8r.Object{Object("much"), Object("too"), Object("early")},
		},
		{
			q:    &Query{LogQL: `{test="loki"}`},
			c:    &korrel8r.Constraint{Start: &t1, End: &t2},
			want: []korrel8r.Object{Object("right"), Object("on"), Object("time")},
		},
	} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			var result korrel8r.ListResult
			assert.NoError(t, s.Get(ctx, x.q, &result))
			assert.Equal(t, x.want, result.List())
		})
	}
}
