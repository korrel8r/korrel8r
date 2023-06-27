// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func command(args ...string) *exec.Cmd {
	return exec.Command("go", append([]string{"run", ".", "-c", "testdata/korrel8r.yaml"}, args...)...)
}

func TestMain_list(t *testing.T) {
	out, err := command("list").Output()
	require.NoError(t, test.ExecError(err))
	assert.Equal(t, "k8s\nlog\nalert\nmetric\nmock\n", string(out))
}

func TestMain_list_domain(t *testing.T) {
	out, err := command("list", "log").Output()
	require.NoError(t, test.ExecError(err))
	assert.Equal(t, "application\ninfrastructure\naudit\n", string(out))
}

func TestMain_get(t *testing.T) {
	out, err := command("get", "-o", "json", "mock", `{"class":"foo", "results":["hello"]}`).Output()
	require.NoError(t, test.ExecError(err))
	assert.Equal(t, "\"hello\"\n", string(out))
}

func start(t *testing.T) string {
	t.Helper()
	port, err := test.ListenPort()
	require.NoError(t, err)
	addr := fmt.Sprintf(":%v", port)
	cmd := command("web", "--http", addr)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	return addr
}

func get(t *testing.T, addr, urlPath string) string {
	var (
		err error
		r   *http.Response
	)
	require.Eventually(t, func() bool {
		r, err = http.Get("http://" + path.Join(addr, "/api/v1alpha1", urlPath))
		return err == nil
	}, 10*time.Second, time.Second/10)
	require.NoError(t, err)
	b, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	return string(b)
}

func TestMain_web_api(t *testing.T) {
	addr := start(t)
	assert.Equal(t, `["k8s","log","alert","metric","mock"]`, get(t, addr, "/domains"))
	assert.Equal(t, `[{"domain":"mock"}]`, get(t, addr, "/stores"))
	q := url.QueryEscape(`{"class":"foo", "results":["hello"]}`)
	assert.Equal(t, `[{"class":{"domain":"mock","class":"foo"}}]`,
		get(t, addr, "/goals?start=mock+foo&query="+q+"&goal=mock+foo"))
}
