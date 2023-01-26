package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGet_Alert(t *testing.T) {
	// Dubious test, assumes there is an alert on the cluster.
	test.SkipIfNoCluster(t)
	var exitCode int
	stdout, stderr := test.FakeMain([]string{"", "get", "alert/alert", "", "-o=json"}, func() { exitCode = Execute() })
	require.Equal(t, 0, exitCode, "\n%v", stderr)

	decoder := json.NewDecoder(strings.NewReader(stdout))
	a := alert.Domain.Class("alert").New()
	require.NoError(t, decoder.Decode(&a), "invalid alert in: %v", stdout)
}

func TestCorrelate_Pods(t *testing.T) {
	test.SkipIfNoCluster(t)
	c := test.K8sClient
	ns := test.TempNamespace(t, c)

	// Deployment
	labels := map[string]string{"test": "testme"}
	d := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "testme", Namespace: ns},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:    "testme",
						Image:   "quay.io/quay/busybox",
						Command: []string{"sh", "-c", "while true; do echo $(date) hello world; sleep 1; done"},
					}}}}}}
	require.NoError(t, c.Create(ctx, d))

	// Wait for pod.
	w, err := c.Watch(ctx, &corev1.PodList{}, client.InNamespace(d.Namespace), client.MatchingLabels(labels))
	require.NoError(t, err)
	var pod *corev1.Pod
	test.Watch(t, w, time.Minute, func(e watch.Event) bool {
		pod = e.Object.(*corev1.Pod)
		return pod.Status.Phase == corev1.PodRunning
	})

	var exitCode int
	stdout, stderr := test.FakeMainStdin(test.JSONString(d), []string{"", "correlate", "k8s/Deployment", "logs/application", "--panic"}, func() {
		exitCode = Execute()
	})
	require.Equal(t, 0, exitCode, stderr)
	want := fmt.Sprintf(`{"LogQL":"{kubernetes_namespace_name=\"%v\"} | json | kubernetes_labels_test=\"testme\"","LogType":"application"}`, pod.Namespace)
	require.JSONEq(t, want, stdout, "got: %v", stdout)
}

func TestList_Classes(t *testing.T) {
	test.SkipIfNoCluster(t)
	// List all k8s classes
	var exitCode int
	stdout, stderr := test.FakeMain([]string{"", "list", "k8s"}, func() {
		exitCode = Execute()
	})
	require.Equal(t, 0, exitCode, stderr)
	for _, x := range []string{"Deployment.v1.apps", "Pod.v1.", "EventList.v1.events.k8s.io"} {
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
	for _, d := range newEngine().Domains() {
		want = append(want, d.String())
	}
	assert.ElementsMatch(t, want, got)
}
