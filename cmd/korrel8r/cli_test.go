// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Functional tests for the korrel8r command line interface

func cliCommand(t *testing.T, args ...string) *exec.Cmd {
	return command(t, append(args, "-c", "testdata/korrel8r.yaml")...)
}

func TestMain_list(t *testing.T) {
	out, err := cliCommand(t, "list").Output()
	require.NoError(t, test.ExecError(err))
	want := `
alert     Alerts that metric values are out of bounds.
incident  Incidents group alerts into higher-level groups.
k8s       Resource objects in a Kubernetes API server
log       Records from container and node logs.
metric    Time-series of measured values
mock      Mock domain.
netflow   Network flows from source nodes to destination nodes.
otellog   Log Records in the otel format.
trace     Traces from Pods and Nodes.
`
	assert.Equal(t, strings.TrimSpace(want), strings.TrimSpace(string(out)))
}

func TestMain_list_domain(t *testing.T) {
	out, err := cliCommand(t, "list", "metric").Output()
	require.NoError(t, test.ExecError(err))
	want := "metric"
	assert.Equal(t, want, strings.TrimSpace(string(out)))
}

func TestMain_get(t *testing.T) {
	out, err := cliCommand(t, "get", "-o", "ndjson", `mock:foo:hello`).Output()
	require.NoError(t, test.ExecError(err))
	assert.Equal(t, "\"hello\"\n", string(out))
}

func TestMain_rules(t *testing.T) {
	for _, x := range []struct {
		args []string
		want string
	}{
		{
			args: []string{"rules"},
			want: "foobar\nbarfoo",
		},
		{
			args: []string{"rules", "--start", "mock:foo"},
			want: "foobar",
		},
		{
			args: []string{"rules", "--goal", "mock:foo"},
			want: "barfoo",
		},
	} {
		t.Run(strings.Join(x.args, " "), func(t *testing.T) {
			out, err := cliCommand(t, x.args...).Output()
			require.NoError(t, test.ExecError(err))
			assert.Equal(t, x.want, strings.TrimSpace(string(out)))
		})
	}
}
