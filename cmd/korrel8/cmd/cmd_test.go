package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alanconway/korrel8/internal/pkg/test"
	"github.com/alanconway/korrel8/pkg/alert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestGet_Alert(t *testing.T) {
	// Dubious test, assumes there is an alert on the cluster.
	test.SkipIfNoCluster(t)
	var exitCode int
	stdout, stderr := test.FakeMain([]string{"", "get", "alert", "{}", "-o=json"}, func() { exitCode = Execute() })
	t.Logf("stdout: %s", stdout)
	t.Logf("stderr: %s", stderr)
	require.Equal(t, 0, exitCode, "exitCode=%v", exitCode)

	decoder := json.NewDecoder(strings.NewReader(stdout))
	var a alert.Alert
	require.NoError(t, decoder.Decode(&a), "invalid alert in: %v", stdout)
}

func TestCorrelate_Pods(t *testing.T) {
	test.SkipIfNoCluster(t)
	c := test.K8sClient
	ns := test.CreateUniqueNamespace(t, c)

	// Pod for test deployment
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name(),
			Namespace: ns,
			Labels:    map[string]string{"test": "testme"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "testme",
					Image: "quay.io/quay/busybox",
					Command: []string{
						"sh", "-c",
						"for i in range 6 do; echo $(date) here we go $i; sleep 10; echo $(date) Oh dear, oh dear; exit 1",
					},
				},
			}}}
	// Watch for pod creation
	w, err := c.Watch(context.Background(), &corev1.PodList{Items: []corev1.Pod{pod}})
	require.NoError(t, err)
	defer w.Stop()

	// Deployment
	d := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "testme", Namespace: ns},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: pod.Labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: pod.ObjectMeta,
				Spec:       pod.Spec,
			},
		},
	}
	require.NoError(t, c.Create(context.Background(), &d))
	require.NoError(t, err)
	select {
	case e := <-w.ResultChan():
		assert.Equal(t, e.Type, watch.Added)
	case <-time.After(time.Second * 10):
		t.Fatal("timeout waiting for Pod")
	}
	var exitCode int
	stdout, stderr := test.FakeMain([]string{"", "correlate", "alert/alert", "loki/log", "testdata/kubeDeploymentAlert.json"}, func() {
		exitCode = Execute()
	})
	require.Equal(t, 0, exitCode, "exitCode=%v: %v", exitCode, stderr)
	require.Equal(t, "resulting queries: [query_range?direction=forward&query=%7Bkubernetes_namespace_name%3D%22default%22%2Ckubernetes_pod_name%3D%22demo-5cf9f87c6-n9zzk%22%7D]", strings.TrimSpace(stdout))
}

func TestList_Classes(t *testing.T) {
	test.SkipIfNoCluster(t)
	// List all k8s classes
	var exitCode int
	stdout, stderr := test.FakeMain([]string{"", "list", "k8s"}, func() {
		exitCode = Execute()
	})
	require.Equal(t, 0, exitCode, "exitCode=%v: %v", exitCode, stderr)
	for _, x := range []string{"Deployment.v1.apps", "Pod.v1", "EventList.v1.events.k8s.io"} {
		assert.Contains(t, stdout, "\n"+x+"\n")
	}
}

func TestList_Domains(t *testing.T) {
	test.SkipIfNoCluster(t)
	// List all k8s classes
	var exitCode int
	stdout, stderr := test.FakeMain([]string{"", "list"}, func() {
		exitCode = Execute()
	})
	require.Equal(t, 0, exitCode, "exitCode=%v: %v", exitCode, stderr)
	got := strings.Split(strings.TrimSpace(stdout), "\n")
	var want []string
	for k := range newEngine().Domains {
		want = append(want, k)
	}
	assert.ElementsMatch(t, want, got)
}
