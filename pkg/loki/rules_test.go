package loki

import (
	"context"
	"net/http"
	"testing"

	"github.com/alanconway/korrel8/internal/pkg/test"
	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nsPodName(ns, name string) map[string]string {
	return map[string]string{"kubernetes_pod_name": name, "kubernetes_namespace_name": ns}
}

func TestRule_PodLogs(t *testing.T) {
	t.Parallel()
	l := test.RequireLokiServer(t)
	s, err := NewStore(l.URL(), http.DefaultClient)
	for _, args := range [][]string{
		{"foo", "bar", "info: foo/bar 1", "info: foo/bar 2"},
		{"foo", "x", "info: foo/x"},
		{"y", "x", "info: y/x"},
		{"foo", "bar", "info: foo/bar 3"},
	} {
		require.NoError(t, l.Push(nsPodName(args[0], args[1]), args[2:]...))
	}
	rule := Rules[0]
	result, err := rule.Apply(k8s.Object{Object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}}, nil)
	require.NoError(t, err)
	require.Equal(t, korrel8.Queries{`{kubernetes_namespace_name="foo",kubernetes_pod_name="bar"}`}, result)
	want := []korrel8.Object{Object("info: foo/bar 1"), Object("info: foo/bar 2"), Object("info: foo/bar 3")}
	got, err := s.Query(context.Background(), result[0])
	require.Equal(t, want, got)
}
