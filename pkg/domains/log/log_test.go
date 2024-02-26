// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/openshift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

var ctx = context.Background()

func logs(lines []string) []korrel8r.Object {
	logs := make([]korrel8r.Object, len(lines))
	for i := range lines {
		logs[i] = NewObject(lines[i])
	}
	return logs
}

func numberedLines(line string, n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("%v: %v", i, line)
	}
	return lines
}

func TestPlainLokiStore_Get(t *testing.T) {
	test.SkipIfNoCluster(t)
	lines := numberedLines(t.Name(), 10)
	want := logs(lines)
	l := test.RequireLokiServer(t)
	err := l.Push(map[string]string{"test": "log"}, lines...)
	require.NoError(t, err)
	s, err := NewPlainLokiStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)
	q := Query{logQL: `{test="log"}`}
	result := korrel8r.NewListResult()
	require.NoError(t, s.Get(ctx, q, nil, result))
	assert.Equal(t, want, result.List())
}

// FIXME better test with kind?
func TestLokiStackStore_Get(t *testing.T) {
	test.SkipIfNoCluster(t)
	c := test.K8sClient
	ns := test.TempNamespace(t, c)
	lines := numberedLines(fmt.Sprintf("%v: %v - %v", time.Now(), t.Name(), ns), 10)
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "logger", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "logger",
				Image:   "quay.io/quay/busybox",
				Command: []string{"sh", "-c", fmt.Sprintf("echo %v; sleep infinity", strings.Join(lines, "; echo "))}}}},
	}

	require.NoError(t, c.Create(ctx, &pod))

	// Construct in-cluster lokistack store.
	host, err := openshift.RouteHost(ctx, c, openshift.LokiStackNSName)
	require.NoError(t, err)
	hc, err := rest.HTTPClientFor(test.RESTConfig)
	require.NoError(t, err)
	s, err := NewLokiStackStore(&url.URL{Scheme: "https", Host: host}, hc)
	require.NoError(t, err)

	logQL := fmt.Sprintf(`{kubernetes_pod_name="%v", kubernetes_namespace_name="%v"}`, pod.Name, pod.Namespace)
	q := Query{logQL: logQL, class: "application"}
	var result korrel8r.ListResult
	assert.Eventually(t, func() bool {
		result = nil
		err = s.Get(ctx, q, nil, &result)
		require.NoError(t, err)
		t.Logf("waiting for %v logs, got %v. %v%v", len(lines), len(result), s, q)
		return len(result) >= len(lines)
	}, 30*time.Second, time.Second)
	var got []string
	for _, o := range result {
		got = append(got, o.(Object).Properties()["message"].(string))
	}
	assert.Equal(t, lines, got)
}

func TestStoreGet_Constraint(t *testing.T) {
	test.SkipIfNoCluster(t)

	l := test.RequireLokiServer(t)

	before, during, after := []string{"too", "early"}, []string{"on", "time"}, []string{"too", "late"}
	labels := map[string]string{"test": "log"}

	require.NoError(t, l.Push(labels, before...))
	t1 := time.Now()
	require.NoError(t, l.Push(labels, during...))
	t2 := time.Now()
	require.NoError(t, l.Push(labels, after...))

	s, err := NewPlainLokiStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)
	for _, x := range []struct {
		c    *korrel8r.Constraint
		want []string
	}{
		{
			c:    &korrel8r.Constraint{End: &t1},
			want: before,
		},
		{
			c:    &korrel8r.Constraint{Start: &t1, End: &t2},
			want: during,
		},
		{
			c:    &korrel8r.Constraint{Start: &t1, End: &t2},
			want: after,
		},
	} {
		t.Run(fmt.Sprintf("%v", x.want), func(t *testing.T) {
			var result korrel8r.ListResult
			assert.NoError(t, s.Get(ctx, Query{logQL: `{test="log"}`}, x.c, &result))
			assert.Equal(t, logs(x.want), result.List())
		})
	}
}
