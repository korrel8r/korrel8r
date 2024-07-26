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
	"sync"
	"testing"
	"time"

	"sync/atomic"

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
	if test.RESTConfig != nil && test.RESTConfig.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+test.RESTConfig.BearerToken)
	}
	if err != nil {
		return "", err
	}
	res, err := h.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("%v: %v", res.Status, string(b))
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
{"name":"mock","stores":[{"domain":"mock", "mockData":"testdata/mock_store.yaml"}]},
{"name":"netflow"}
]`, "GET", u, "")
}

func TestMain_server_secure(t *testing.T) {
	_, clientTLS := certSetup(t, tmpDir)
	h := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLS}}
	u := startServer(t, h, "https", "--cert", filepath.Join(tmpDir, "tls.crt"), "--key", filepath.Join(tmpDir, "tls.key"), "-c", "testdata/korrel8r.yaml").String() + "/domains"
	assertDo(t, h, `[
{"name":"alert"},
{"name":"k8s"},
{"name":"log"},
{"name":"metric"},
{"name":"mock","stores":[{"domain":"mock", "mockData":"testdata/mock_store.yaml"}]},
{"name":"netflow"}
]`,
		"GET", u, "")
}

const testRequest = `{  "depth": 1, "start": { "queries": [ "mock:foo:x" ] }}`

var testResponse = rest.Normalize(rest.Graph{
	Nodes: rest.Array[rest.Node]{
		rest.Node{Class: "mock:foo", Queries: rest.Array[rest.QueryCount]{rest.QueryCount{Query: "mock:foo:x", Count: 1}}, Count: 1},
		rest.Node{Class: "mock:bar", Queries: rest.Array[rest.QueryCount]{rest.QueryCount{Query: "mock:bar:y", Count: 1}}, Count: 1}},
	Edges: rest.Array[rest.Edge]{rest.Edge{Start: "mock:foo", Goal: "mock:bar", Rules: rest.Array[rest.Rule](nil)}},
})

func TestMain_server_graph(t *testing.T) {
	u := startServer(t, http.DefaultClient, "http", "-c", "testdata/korrel8r.yaml").String()
	resp, err := request(t, http.DefaultClient, "POST", u+"/graphs/neighbours", testRequest)
	require.NoError(t, err)
	var got rest.Graph
	assert.NoError(t, json.Unmarshal([]byte(resp), &got))
	require.Equal(t, testResponse, rest.Normalize(got))
}

func TestMain_concurrent_requests(t *testing.T) {
	u := startServer(t, http.DefaultClient, "http", "-c", "testdata/korrel8r.yaml").String()
	workers := sync.WaitGroup{}
	failed := atomic.Uint32{}
	for i := 0; i < 10; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			resp, err := request(t, http.DefaultClient, "POST", u+"/graphs/neighbours", testRequest)
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
