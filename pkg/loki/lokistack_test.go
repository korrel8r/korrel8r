package loki

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLogTypeTenants(t *testing.T) {
	for _, x := range []struct {
		query string
		want  []string
	}{
		{query: `{kubernetes_namespace_name="test",kubernetes_pod_name="testme"}`, want: []string{"application", "infrastructure", "audit"}},
		{query: `{  log_type =~ ".+"  ,  kubernetes_pod_name="testme"}`, want: []string{"application", "infrastructure", "audit"}},
		{query: `{kubernetes_pod_name="testme",log_type="audit"}`, want: []string{"audit"}},
		{query: `{kubernetes_pod_name="testme",log_type=~"application|audit"}`, want: []string{"application", "audit"}},
		{query: `{kubernetes_pod_name="testme",log_type="application|audit"}`, want: nil},
		{query: `{kubernetes_pod_name="testme",log_type=~"nosuch"}`, want: nil},
		{query: `{kubernetes_pod_name="testme",log_type="application"}`, want: []string{"application"}},
	} {
		t.Run(x.query, func(t *testing.T) {
			q := Query(x.query)
			assert.Equal(t, x.want, q.LogTypes())
		})
	}
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
				Command: []string{"echo", strings.Join(want, "\n")}}}},
	}
	require.NoError(t, c.Create(ctx, &pod))
	s, err := NewLokiStackStore(ctx, c, test.RESTConfig)
	require.NoError(t, err)

	query := Query(fmt.Sprintf(`{kubernetes_pod_name="%v", kubernetes_namespace_name="%v"}`, pod.Name, pod.Namespace))
	var result korrel8.ListResult
	assert.Eventually(t, func() bool {
		t.Log("waiting for pod")
		result = nil
		err = s.Get(ctx, &query, &result)
		require.NoError(t, err)
		return len(result) >= 3
	}, time.Minute, time.Second)
	var got []string
	for _, obj := range result {
		var m map[string]any
		line := string(obj.(Object))
		assert.NoError(t, json.Unmarshal([]byte(line), &m), line)
		got = append(got, m["message"].(string))
	}
	assert.Equal(t, want, got)
}
