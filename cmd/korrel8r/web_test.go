// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"sync/atomic"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/rest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Functional tests for the korrel8r REST API.

func startServer(t *testing.T, h *http.Client, scheme string, args ...string) *url.URL {
	t.Helper()
	port, err := test.ListenPort()
	require.NoError(t, err)
	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	cmd := command(t, append([]string{"web", "--" + scheme, addr}, args...)...)
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
	require.NoError(t, err)
	if cfg, err := config.GetConfig(); err == nil && cfg != nil && cfg.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.BearerToken)
	}
	if err != nil {
		return "", err
	}
	res, err := h.Do(req)
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(res.Body)
	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("%v: %v", res.Status, string(b))
	}
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

const domains = `[
{"name":"alert"},
{"name":"incident"},
{"name":"k8s"},
{"name":"log"},
{"name":"metric"},
{"name":"mock","stores":[{"domain":"mock", "mockData":"testdata/mock_store.yaml"}]},
{"name":"netflow"},
{"name":"trace"}
]`

func TestMain_server_insecure(t *testing.T) {
	u := startServer(t, http.DefaultClient, "http", "-c", "testdata/korrel8r.yaml").String() + "/domains"
	assertDo(t, http.DefaultClient, domains, "GET", u, "")
}

func TestMain_server_secure(t *testing.T) {
	_, clientTLS := certSetup(t, tmpDir)
	h := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLS}}
	u := startServer(t, h, "https", "--cert", filepath.Join(tmpDir, "tls.crt"), "--key", filepath.Join(tmpDir, "tls.key"), "-c", "testdata/korrel8r.yaml").String() + "/domains"
	assertDo(t, h, `[
{"name":"alert"},
{"name":"incident"},
{"name":"k8s"},
{"name":"log"},
{"name":"metric"},
{"name":"mock","stores":[{"domain":"mock", "mockData":"testdata/mock_store.yaml"}]},
{"name":"netflow"},
{"name":"trace"}
]`,
		"GET", u, "")
}

const testRequest = `{  "depth": 1, "start": { "queries": [ "mock:foo:x" ] }}`

var testResponse = rest.Normalize(rest.Graph{
	Nodes: []rest.Node{
		{Class: "mock:foo", Queries: []rest.QueryCount{{Query: "mock:foo:x", Count: ptr.To(1)}}, Count: ptr.To(1)},
		{Class: "mock:bar", Queries: []rest.QueryCount{{Query: "mock:bar:y", Count: ptr.To(1)}}, Count: ptr.To(1)}},
	Edges: []rest.Edge{{Start: "mock:foo", Goal: "mock:bar", Rules: []rest.Rule(nil)}},
})

func TestMain_server_graph(t *testing.T) {
	u := startServer(t, http.DefaultClient, "http", "-c", "testdata/korrel8r.yaml").String()
	resp, err := request(t, http.DefaultClient, "POST", u+"/graphs/neighbors", testRequest)
	require.NoError(t, err)
	var got rest.Graph
	assert.NoError(t, json.Unmarshal([]byte(resp), &got))
	require.Equal(t, testResponse, rest.Normalize(got))
}

func TestMain_server_tls_min_version(t *testing.T) {
	dir := t.TempDir()
	_, clientTLS := certSetup(t, dir)
	certArgs := []string{
		"--cert", filepath.Join(dir, "tls.crt"),
		"--key", filepath.Join(dir, "tls.key"),
		"--tls-min-version", "VersionTLS13",
		"-c", "testdata/korrel8r.yaml",
	}

	// Start server requiring TLS 1.3
	port, err := test.ListenPort()
	require.NoError(t, err)
	addr := net.JoinHostPort("localhost", strconv.Itoa(port))
	cmd := command(t, append([]string{"web", "--https", addr}, certArgs...)...)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// Client with TLS 1.3 should succeed.
	tls13Client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    clientTLS.RootCAs,
			MinVersion: tls.VersionTLS13,
			MaxVersion: tls.VersionTLS13,
		},
	}}
	u := fmt.Sprintf("https://%v%v/domains", addr, rest.BasePath)
	require.Eventually(t, func() bool {
		_, err = tls13Client.Get(u)
		return err == nil
	}, 10*time.Second, time.Second/10, "timeout waiting for TLS 1.3 server")
	assertDo(t, tls13Client, domains, "GET", u, "")

	// Client limited to TLS 1.2 should fail.
	tls12Client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    clientTLS.RootCAs,
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS12,
		},
	}}
	_, err = tls12Client.Get(u)
	require.Error(t, err, "TLS 1.2 client should fail against TLS 1.3 server")
}

func TestMain_concurrent_requests(t *testing.T) {
	u := startServer(t, http.DefaultClient, "http", "-c", "testdata/korrel8r.yaml").String()
	workers := sync.WaitGroup{}
	failed := atomic.Uint32{}
	for range 10 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			resp, err := request(t, http.DefaultClient, "POST", u+"/graphs/neighbors", testRequest)
			var got rest.Graph
			ok := assert.NoError(t, err) &&
				assert.NoError(t, json.Unmarshal([]byte(resp), &got)) &&
				assert.Equal(t, rest.Normalize(testResponse), rest.Normalize(got))
			if !ok {
				failed.Add(1)
			}
		}()
	}
	workers.Wait()
	assert.Zero(t, failed.Load())
}
