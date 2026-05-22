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
alert     Domain alert is a korrel8r domain for Prometheus/AlertManager alerts.
incident  Domain incident is a korrel8r domain for cluster health incidents.
k8s       Domain k8s is a korrel8r domain for Kubernetes resources.
log       Domain log is a korrel8r domain for application, infrastructure, and audit logs.
metric    Domain metric is a korrel8r domain for Prometheus metrics.
mock      Mock domain.
netflow   Domain netflow is a korrel8r domain for network flow data.
trace     Domain trace is a korrel8r domain for OpenTelemetry traces.
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
			want: "foobar: [mock:foo] -> [mock:bar]\nbarfoo: [mock:bar] -> [mock:foo]",
		},
		{
			args: []string{"rules", "--start", "mock:foo"},
			want: "foobar: [mock:foo] -> [mock:bar]",
		},
		{
			args: []string{"rules", "--goal", "mock:foo"},
			want: "barfoo: [mock:bar] -> [mock:foo]",
		},
	} {
		t.Run(strings.Join(x.args, " "), func(t *testing.T) {
			out, err := cliCommand(t, x.args...).Output()
			require.NoError(t, test.ExecError(err))
			assert.Equal(t, x.want, strings.TrimSpace(string(out)))
		})
	}
}

func TestMain_stores(t *testing.T) {
	out, err := cliCommand(t, "stores").Output()
	require.NoError(t, test.ExecError(err))
	want := `{
  "alert": null,
  "incident": null,
  "k8s": null,
  "log": null,
  "metric": null,
  "mock": [
    {
      "domain": "mock",
      "mockData": "testdata/mock_store.yaml"
    }
  ],
  "netflow": null,
  "trace": null
}`
	assert.Equal(t, strings.TrimSpace(want), strings.TrimSpace(string(out)))
}

func TestMain_stores_selected(t *testing.T) {
	out, err := cliCommand(t, "stores", "k8s", "mock").Output()
	require.NoError(t, test.ExecError(err))
	want := `{
  "k8s": null,
  "mock": [
    {
      "domain": "mock",
      "mockData": "testdata/mock_store.yaml"
    }
  ]
}
`
	assert.Equal(t, strings.TrimSpace(want), strings.TrimSpace(string(out)))
}
