// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package openshift

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func init() {
	if err := test.HasCluster(); err != nil {
		panic(fmt.Errorf("No cluster available: %v", err))
	}
}

func logObjects(lines []string) []korrel8r.Object {
	logs := make([]korrel8r.Object, len(lines))
	for i := range lines {
		logs[i] = log.FromLine(lines[i])
	}
	return logs
}

func logLines(line string, n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("%v: %v", i, line)
	}
	return lines
}

func TestLogLokiStore_Get(t *testing.T) {
	lines := logLines(t.Name(), 10)
	want := logObjects(lines)
	l := test.RequireLokiServer(t)
	err := l.Push(map[string]string{"test": "log"}, lines...)
	require.NoError(t, err)
	s, err := log.NewPlainLokiStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)
	q := log.NewQuery("", `{test="log"}`)
	result := korrel8r.NewListResult()
	require.NoError(t, s.Get(context.Background(), q, nil, result))
	assert.Equal(t, want, result.List())
}

func TestLogLokiStackStore_Get(t *testing.T) {
	c := test.K8sClient
	ns := test.TempNamespace(t, c)
	lines := logLines(fmt.Sprintf("%v: %v - %v", time.Now(), t.Name(), ns), 10)
	falsev := false
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "logger", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "logger",
				Image:   "quay.io/quay/busybox",
				Command: []string{"sh", "-c", fmt.Sprintf("echo %v; sleep infinity", strings.Join(lines, "; echo "))},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: &falsev,
					Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
					SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				}}}},
	}

	require.NoError(t, c.Create(context.Background(), &pod))

	// Construct in-cluster lokistack store.
	host, err := RouteHost(context.Background(), c, LokiStackNSName)
	require.NoError(t, err)
	hc, err := rest.HTTPClientFor(test.RESTConfig)
	require.NoError(t, err)
	s, err := log.NewLokiStackStore(&url.URL{Scheme: "https", Host: host}, hc)
	require.NoError(t, err)

	logQL := fmt.Sprintf(`{kubernetes_pod_name="%v", kubernetes_namespace_name="%v"}`, pod.Name, pod.Namespace)
	q := log.NewQuery(log.Application, logQL)
	var result korrel8r.ListResult
	assert.Eventually(t, func() bool {
		err = s.Get(context.Background(), q, nil, &result)
		done := (err != nil || len(result) >= len(lines))
		if !done {
			t.Logf("waiting for %v logs, got %v. %v%v", len(lines), len(result), s, q)
		}
		return done
	}, 30*time.Second, time.Second)
	require.NoError(t, err)
	var got []string
	for _, o := range result {
		got = append(got, o.(log.Object)["message"].(string))
	}
	assert.Equal(t, lines, got)
}
