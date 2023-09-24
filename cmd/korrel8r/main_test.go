// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"bytes"
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

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func command(t *testing.T, args ...string) *exec.Cmd {
	cmd := exec.Command("go", append([]string{"run", ".", "-v9", "-c", "testdata/korrel8r.yaml", "--panic"}, args...)...)
	cmd.Stderr = os.Stderr
	return cmd
}

func TestMain_list(t *testing.T) {
	out, err := command(t, "list").Output()
	require.NoError(t, test.ExecError(err))
	assert.Equal(t, "k8s\nlog\nalert\nmetric\nmock\n", string(out))
}

func TestMain_list_domain(t *testing.T) {
	out, err := command(t, "list", "log").Output()
	require.NoError(t, test.ExecError(err))
	assert.Equal(t, "application\ninfrastructure\naudit\n", string(out))
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
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	t.Logf("%v", cmd)
	require.NoError(t, cmd.Start())
	// Wait till HTTP server is available.
	require.Eventually(t, func() bool {
		_, err = http.Get("http://" + addr)
		return err == nil
	}, 10*time.Second, time.Second/10, "timeout error: %v", err)
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		if t.Failed() {
			t.Logf("... stderr: %v\n%v\n---", cmd, stderr)
		}
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
	assert.Equal(t, want, string(b))
}

func TestMain_web_api(t *testing.T) {
	base := start(t)
	u := func(path string) string { return base.String() + path }
	assertDo(t, `[{"name":"k8s"},{"name":"log"},{"name":"alert"},{"name":"metric"},{"name":"mock","stores":[{"domain":"mock"}]}]`, "GET", u("/domains"), "")
	// FIXME result should not be empty?
	assertDo(t, `{"nodes":[]}`, "POST", u("/graphs/goals?withRules=true"), `{"goals":["bar.mock"],"start":{"class":"foo.mock","objects":["x"]}}`)
}
