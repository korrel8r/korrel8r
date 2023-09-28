// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"sync"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var buildOnce sync.Once

func command(t *testing.T, args ...string) *exec.Cmd {
	buildOnce.Do(func() {
		cmd := exec.Command("go", "build", "-o", "../../korrel8r")
		cmd.Stderr = os.Stderr
		test.Must(t, cmd.Run())
	})
	commonArgs := []string{"-v9", "-c", "testdata/korrel8r.yaml", "--panic"}
	cmd := exec.Command("../../korrel8r", append(commonArgs, args...)...)
	cmd.Stderr = os.Stderr
	return cmd
}

func TestMain_list(t *testing.T) {
	out, err := command(t, "list").Output()
	require.NoError(t, test.ExecError(err))
	want := `
k8s      Resource objects in a Kubernetes API server
log      Records from container and node logs.
alert    Alerts that metric values are out of bounds.
metric   Time-series of measured values
mock     Mock domain.
`
	assert.Equal(t, strings.TrimSpace(want), strings.TrimSpace(string(out)))
}

func TestMain_list_domain(t *testing.T) {
	out, err := command(t, "list", "metric").Output()
	require.NoError(t, test.ExecError(err))
	want := "metric   A set of label:value pairs identifying a time-series."
	assert.Equal(t, want, strings.TrimSpace(string(out)))
}

func TestMain_get(t *testing.T) {
	out, err := command(t, "get", "-o", "json", "mock", `{"class":"foo", "results":["hello"]}`).Output()
	require.NoError(t, test.ExecError(err))
	assert.Equal(t, "\"hello\"\n", string(out))
}

func start(t *testing.T) *url.URL {
	t.Helper()
	port, err := test.ListenPort()
	require.NoError(t, err)
	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	cmd := command(t, "web", "--http", addr)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	// Wait till HTTP server is available.
	require.Eventually(t, func() bool {
		_, err = http.Get("http://" + addr)
		return err == nil
	}, 10*time.Second, time.Second/10, "timeout error: %v", err)
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
	return &url.URL{Scheme: "http", Host: addr, Path: api.BasePath}
}

func assertDo(t *testing.T, want, method, url, body string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.JSONEq(t, want, string(b))
}

func TestMain_web_api(t *testing.T) {
	test.SkipIfNoCluster(t)
	base := start(t)
	u := func(path string) string { return base.String() + path }
	assertDo(t, `[{"name":"k8s"},{"name":"log"},{"name":"alert"},{"name":"metric"},{"name":"mock","stores":[{"domain":"mock"}]}]`, "GET", u("/domains"), "")
}
