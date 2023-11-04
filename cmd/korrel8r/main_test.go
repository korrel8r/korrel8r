// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	out, err := command(t, "get", "-o", "json", `mock:foo:["hello"]`).Output()
	require.NoError(t, test.ExecError(err))
	assert.Equal(t, "\"hello\"\n", string(out))
}

func TestMain_describe_domain(t *testing.T) {
	out, err := command(t, "describe", "log").Output()
	require.NoError(t, test.ExecError(err))
	want := "Records from container and node logs."
	assert.Equal(t, want, strings.TrimSpace(string(out)))
}

func TestMain_describe_class(t *testing.T) {
	out, err := command(t, "describe", "log:audit").Output()
	require.NoError(t, test.ExecError(err))
	want := "Audit logs from the node operating system (/var/log/audit) and the cluster API servers"
	assert.Equal(t, want, strings.TrimSpace(string(out)))
}

func TestMain_rules(t *testing.T) {
	for _, x := range []struct {
		args []string
		want string
	}{
		{
			args: []string{"rules"},
			want: "foobar(mock:foo,mock:bar)\nbarfoo(mock:bar,mock:foo)",
		},
		{
			args: []string{"rules", "--start", "mock:foo"},
			want: "foobar(mock:foo,mock:bar)",
		},
		{
			args: []string{"rules", "--goal", "mock:foo"},
			want: "barfoo(mock:bar,mock:foo)",
		},
	} {
		t.Run(strings.Join(x.args, " "), func(t *testing.T) {
			out, err := command(t, x.args...).Output()
			require.NoError(t, test.ExecError(err))
			assert.Equal(t, x.want, strings.TrimSpace(string(out)))
		})
	}
}

func startServer(t *testing.T, h *http.Client, proto string, args ...string) *url.URL {
	t.Helper()
	port, err := test.ListenPort()
	require.NoError(t, err)
	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	cmd := command(t, append([]string{"web", "--" + proto, addr}, args...)...)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	// Wait till server is available.
	require.Eventually(t, func() bool {
		_, err = h.Get(proto + "://" + addr)
		return err == nil
	}, 10*time.Second, time.Second/10, "timeout error: %v", err)
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
	return &url.URL{Scheme: proto, Host: addr, Path: api.BasePath}
}

func assertDo(t *testing.T, h *http.Client, want, method, url, body string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	res, err := h.Do(req)
	require.NoError(t, err)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.JSONEq(t, want, string(b))
}

func TestMain_server_insecure(t *testing.T) {
	test.SkipIfNoCluster(t)
	t.Run("insecure", func(t *testing.T) {
		u := startServer(t, http.DefaultClient, "http").String() + "/domains"
		assertDo(t, http.DefaultClient, `[{"name":"k8s"},{"name":"log"},{"name":"alert"},{"name":"metric"},{"name":"mock","stores":[{"domain":"mock"}]}]`, "GET", u, "")
	})
}

func TestMain_server_secure(t *testing.T) {
	test.SkipIfNoCluster(t)
	_, clientTLS := certSetup(tmpDir)
	h := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLS}}
	t.Run("secure", func(t *testing.T) {
		u := startServer(t, h, "https", "--cert", filepath.Join(tmpDir, "tls.crt"), "--key", filepath.Join(tmpDir, "tls.key")).String() + "/domains"
		assertDo(t, h, `[{"name":"k8s"},{"name":"log"},{"name":"alert"},{"name":"metric"},{"name":"mock","stores":[{"domain":"mock"}]}]`, "GET", u, "")
	})
}

func TestMain(m *testing.M) {
	// Build korrel8r once to run in tests, much faster than using 'go run' for each test.
	tmpDir = test.Must(os.MkdirTemp("", "korrel8r_test"))
	defer func() { _ = os.RemoveAll(tmpDir) }()
	cmd := exec.Command("go", "build", "-o", tmpDir)
	cmd.Stderr = os.Stderr
	test.PanicErr(cmd.Run())
	os.Exit(m.Run())
}

var tmpDir string

func command(t *testing.T, args ...string) *exec.Cmd {
	commonArgs := []string{"-v9", "-c", "testdata/korrel8r.yaml", "--panic"}
	cmd := exec.Command(filepath.Join(tmpDir, "korrel8r"), append(commonArgs, args...)...)
	cmd.Stderr = os.Stderr
	return cmd
}
