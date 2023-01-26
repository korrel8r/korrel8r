package cmd

import (
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
