package rules

import (
	"context"
	"net/http"
	"testing"

	"github.com/alanconway/korrel8/internal/pkg/test"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/loki"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nsPodName(ns, name string) map[string]string {
	return map[string]string{"kubernetes_pod_name": name, "kubernetes_namespace_name": ns}
}

func TestRule_PodToLokiLogs(t *testing.T) {
	t.Parallel()
	l := test.RequireLokiServer(t)
	s := loki.NewStore(l.URL(), http.DefaultClient)
	for _, args := range [][]string{
		{"foo", "bar", "info: foo/bar 1", "info: foo/bar 2"},
		{"foo", "x", "info: foo/x"},
		{"y", "x", "info: y/x"},
		{"foo", "bar", "info: foo/bar 3"},
	} {
		require.NoError(t, l.Push(nsPodName(args[0], args[1]), args[2:]...))
	}
	rule := K8sToLoki()[0]
	queries, err := rule.Apply(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "bar"}}, nil)
	require.NoError(t, err)
	wantQuery := string(`query_range?direction=forward&query=%7Bkubernetes_namespace_name%3D%22foo%22%2Ckubernetes_pod_name%3D%22bar%22%7D`)
	require.Equal(t, []string{wantQuery}, queries)
	result := korrel8.NewListResult()
	for _, q := range queries {
		require.NoError(t, s.Get(context.Background(), q, result))
	}
	want := []korrel8.Object{loki.Object("info: foo/bar 1"), loki.Object("info: foo/bar 2"), loki.Object("info: foo/bar 3")}
	require.Equal(t, want, result.List())
}
