// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Functional tests for the korrel8r REST API.

func startServer(t *testing.T, h *http.Client, scheme string, args ...string) *url.URL {
	t.Helper()
	port, err := test.ListenPort()
	require.NoError(t, err)
	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	cmd := command(t, append([]string{"web", "--html=false", "--" + scheme, addr}, args...)...)
	require.NoError(t, cmd.Start())
	// Wait till server is available.
	require.Eventually(t, func() bool {
		_, err = h.Get(fmt.Sprintf("%v://%v", scheme, addr))
		return err == nil
	}, 10*time.Second, time.Second/10, "timeout error: %v", err)
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
	})
	return &url.URL{Scheme: scheme, Host: addr, Path: rest.BasePath}
}

func request(t *testing.T, h *http.Client, method, url, body string) (string, error) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	res, err := h.Do(req)
	if err != nil {
		return "", err
	}
	if res.Status[0] != '2' {
		return "", fmt.Errorf("bad stauts: %v", res.Status)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func assertDo(t *testing.T, h *http.Client, want, method, url, body string) {
	t.Helper()
	got, err := request(t, h, method, url, body)
	require.NoError(t, err)
	assert.JSONEq(t, want, got)
}

func TestMain_server_insecure(t *testing.T) {
	u := startServer(t, http.DefaultClient, "http", "-c", "testdata/korrel8r.yaml").String() + "/domains"
	assertDo(t, http.DefaultClient, `[
{"name":"alert"},
{"name":"k8s"},
{"name":"log"},
{"name":"metric"},
{"name":"mock","stores":[{"domain":"mock"}]},
{"name":"netflow"}
]`, "GET", u, "")
}

func TestMain_server_secure(t *testing.T) {
	_, clientTLS := certSetup(tmpDir)
	h := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLS}}
	u := startServer(t, h, "https", "--cert", filepath.Join(tmpDir, "tls.crt"), "--key", filepath.Join(tmpDir, "tls.key"), "-c", "testdata/korrel8r.yaml").String() + "/domains"
	assertDo(t, h, `[
{"name":"alert"},
{"name":"k8s"},
{"name":"log"},
{"name":"metric"},
{"name":"mock","stores":[{"domain":"mock"}]},
{"name":"netflow"}
]`,
		"GET", u, "")
}

func TestMain_server_graph(t *testing.T) {
	test.SkipIfNoCluster(t)
	u := startServer(t, http.DefaultClient, "http", "-c", "../../etc/korrel8r/korrel8r.yaml").String()
	got, err := request(t, http.DefaultClient, "POST", u+"/graphs/neighbours", `{
  "depth": 1,
  "start": {
    "class": "k8s:Deployment",
    "queries": [ "k8s:Deployment:{namespace: korrel8r}" ]
  }
}`)
	require.NoError(t, err)
	var g rest.Graph
	require.NoError(t, json.Unmarshal([]byte(got), &g))
	require.NotEmpty(t, g.Nodes)
	require.NotEmpty(t, g.Edges)
	// FIXME better tests.
}

func TestMain_concurrent_requests(t *testing.T) {
	test.SkipIfNoCluster(t)
	u := startServer(t, http.DefaultClient, "http").String()
	const n = 10
	results := make(chan string, n)
	for i := 0; i < n; i++ {
		go func() {
			resp, err := request(t, http.DefaultClient, "POST", u+"/graphs/neighbours", `{
  "depth": 1,
  "start": {
    "class": "k8s:Deployment",
    "queries": [ "k8s:Deployment:{namespace: korrel8r}" ]
  }
}`)
			if err != nil {
				resp = err.Error()
			}
			results <- resp
		}()
	}
	var prev rest.Graph
	for i := 0; i < n; i++ {
		resp := <-results
		var got rest.Graph
		if assert.NoError(t, json.Unmarshal([]byte(resp), &got)) && i > 0 {
			assert.ElementsMatch(t, prev.Edges, got.Edges)
			assert.ElementsMatch(t, prev.Nodes, got.Nodes)
		}
		prev = got
	}
}
