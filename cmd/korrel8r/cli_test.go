// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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
alert     Prometheus/AlertManager alerts.
incident  cluster health incidents.
k8s       Kubernetes resources.
log       application, infrastructure, and audit logs.
metric    Prometheus metrics.
mock      Mock domain.
netflow   network flow data.
trace     OpenTelemetry traces.
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
		args    []string
		want    []string
		exclude []string
	}{
		{
			args:    []string{"rules"},
			want:    []string{"foobar: [mock:foo] -> [mock:bar]", "barfoo: [mock:bar] -> [mock:foo]"},
			exclude: nil,
		},
		{
			args:    []string{"rules", "--start", "mock:foo"},
			want:    []string{"foobar: [mock:foo] -> [mock:bar]"},
			exclude: []string{"barfoo:"},
		},
		{
			args:    []string{"rules", "--goal", "mock:foo"},
			want:    []string{"barfoo: [mock:bar] -> [mock:foo]"},
			exclude: []string{"foobar:"},
		},
	} {
		t.Run(strings.Join(x.args, " "), func(t *testing.T) {
			out, err := cliCommand(t, x.args...).Output()
			require.NoError(t, test.ExecError(err))
			for _, w := range x.want {
				assert.Contains(t, string(out), w)
			}
			for _, e := range x.exclude {
				assert.NotContains(t, string(out), e)
			}
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

func TestMain_metric_file(t *testing.T) {
	f := filepath.Join(tmpDir, "metrics.json")
	_, err := cliCommand(t, "list", "--metric-file", f).Output()
	require.NoError(t, test.ExecError(err))
	data, err := os.ReadFile(f)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Contains(t, m, "Resource")
	assert.Contains(t, m, "ScopeMetrics")
}

func TestMain_metric_file_not_written(t *testing.T) {
	f := filepath.Join(tmpDir, "unused-metrics.json")
	_ = os.Remove(f)
	_, err := cliCommand(t, "list").Output()
	require.NoError(t, test.ExecError(err))
	_, err = os.Stat(f)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
