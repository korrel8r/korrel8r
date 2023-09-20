// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

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

func makeLines(line string, n int) (lines []string, objects []korrel8r.Object) {
	for i := 0; i < n; i++ {
		line := fmt.Sprintf("%v: %v", i, line)
		lines = append(lines, line)
		objects = append(objects, NewObject(line))
	}
	return lines, objects
}

func TestPlainLokiStore_Get(t *testing.T) {
	t.Parallel()
	lines, want := makeLines(t.Name(), 10)
	l := test.RequireLokiServer(t)
	err := l.Push(map[string]string{"test": "log"}, lines...)
	require.NoError(t, err)
	s, err := NewPlainLokiStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)
	q := &Query{LogQL: `{test="log"}`}
	result := korrel8r.NewListResult()
	require.NoError(t, s.Get(ctx, q, result))
	assert.Equal(t, want, result.List())
}

func TestLokiStackStore_Get(t *testing.T) {
	t.Parallel()
	test.SkipIfNoCluster(t)
	lines, _ := makeLines(fmt.Sprintf("%v: %v", time.Now(), t.Name()), 10)
	c := test.K8sClient
	ns := test.TempNamespace(t, c)
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "logger", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "logger",
				Image:   "quay.io/quay/busybox",
				Command: []string{"sh", "-c", fmt.Sprintf("echo %v; sleep infinity", strings.Join(lines, "; echo "))}}}},
	}
	require.NoError(t, c.Create(ctx, &pod))
	s, err := NewOpenshiftLokiStackStore(ctx, c, test.RESTConfig)
	require.NoError(t, err)
	logQL := fmt.Sprintf(`{kubernetes_pod_name="%v", kubernetes_namespace_name="%v"}`, pod.Name, pod.Namespace)
	q := &Query{LogQL: logQL, LogType: "application"}
	var result korrel8r.ListResult
	assert.Eventually(t, func() bool {
		result = nil
		err = s.Get(ctx, q, &result)
		require.NoError(t, err)
		t.Logf("waiting for %v logs, got %v. %v%v", len(lines), len(result), s, q)
		return len(result) >= len(lines)
	}, 30*time.Second, 5*time.Second)
	var got []string
	for _, obj := range result {
		var m map[string]any
		line := obj.(Object).Entry
		assert.NoError(t, json.Unmarshal([]byte(line), &m), line)
		got = append(got, m["message"].(string))
	}
	assert.Equal(t, lines, got)
}

func TestStoreGet_Constraint(t *testing.T) {
	t.Skip("TODO re-enable when constraints are implemented properly")
	t.Parallel()
	l := test.RequireLokiServer(t)

	err := l.Push(map[string]string{"test": "log"}, "much", "too", "early")
	require.NoError(t, err)

	t1 := time.Now()
	err = l.Push(map[string]string{"test": "log"}, "right", "on", "time")
	require.NoError(t, err)
	t2 := time.Now()

	err = l.Push(map[string]string{"test": "log"}, "much", "too", "late")
	require.NoError(t, err)
	s, err := NewPlainLokiStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)

	for n, x := range []struct {
		q    korrel8r.Query
		c    *korrel8r.Constraint
		want []korrel8r.Object
	}{
		{
			q:    &Query{LogQL: `{test="log"}`},
			c:    &korrel8r.Constraint{End: &t1},
			want: []korrel8r.Object{NewObject("much"), NewObject("too"), NewObject("early")},
		},
		{
			q:    &Query{LogQL: `{test="log"}`},
			c:    &korrel8r.Constraint{Start: &t1, End: &t2},
			want: []korrel8r.Object{NewObject("right"), NewObject("on"), NewObject("time")},
		},
	} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			var result korrel8r.ListResult
			assert.NoError(t, s.Get(ctx, x.q, &result))
			assert.Equal(t, x.want, result.List())
		})
	}
}
