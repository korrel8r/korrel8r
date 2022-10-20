package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/alert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGet_Alert(t *testing.T) {
	// Dubious test, assumes there is an alert on the cluster.
	test.SkipIfNoCluster(t)
	var exitCode int
	stdout, stderr := test.FakeMain([]string{"", "get", "alert", "{}", "-o=json"}, func() { exitCode = Execute() })
	require.Equal(t, 0, exitCode, "%v", stderr)

	decoder := json.NewDecoder(strings.NewReader(stdout))
	var a alert.Alert
	require.NoError(t, decoder.Decode(&a), "invalid alert in: %v", stdout)
}

func TestCorrelate_Pods(t *testing.T) {
	test.SkipIfNoCluster(t)
	c := test.K8sClient
	ns := test.CreateUniqueNamespace(t, c)

	// Deployment
	d := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "testme", Namespace: ns},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "testme"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"test": "testme"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "testme",
							Image:   "quay.io/quay/busybox",
							Command: []string{"sh", "-c", "while true; do echo $(date) hello world; sleep 1; done"},
						}}}}}}
	require.NoError(t, c.Create(context.Background(), d))

	// Wait for it...
	w, err := c.Watch(context.Background(), &appv1.DeploymentList{}, client.InNamespace(d.Namespace))
	require.NoError(t, err)
	defer w.Stop()
	deadline := time.Now().Add(time.Minute)
	for d.Status.Replicas < 1 {
		select {
		case e, ok := <-w.ResultChan():
			if !ok {
				t.Fatal("watch closed")
			}
			d = e.Object.(*appv1.Deployment)
		case <-time.After(time.Until(deadline)):
			t.Fatal("timeout waiting")
		}
	}
	var exitCode int
	stdout, stderr := test.FakeMainStdin(test.JSONString(d), []string{"", "correlate", "k8s/Deployment", "k8s/Pod", "-v9"}, func() {
		exitCode = Execute()
	})
	require.Equal(t, 0, exitCode, stderr)
	t.Log(stderr)
	require.Equal(t, "resulting queries: [query_range?direction=forward&query=%7Bkubernetes_namespace_name%3D%22default%22%2Ckubernetes_pod_name%3D%22demo-5cf9f87c6-n9zzk%22%7D]", strings.TrimSpace(stdout))
}

func TestList_Classes(t *testing.T) {
	test.SkipIfNoCluster(t)
	// List all k8s classes
	var exitCode int
	stdout, stderr := test.FakeMain([]string{"", "list", "k8s"}, func() {
		exitCode = Execute()
	})
	require.Equal(t, 0, exitCode, stderr)
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
	require.Equal(t, 0, exitCode, stderr)
	got := strings.Split(strings.TrimSpace(stdout), "\n")
	var want []string
	for k := range newEngine().Domains {
		want = append(want, k)
	}
	assert.ElementsMatch(t, want, got)
}
