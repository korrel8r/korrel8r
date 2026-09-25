// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustNewQuery(t *testing.T, s string) *Query {
	t.Helper()
	q, err := NewQuery(s)
	assert.NoError(t, err)
	return q
}

func mockLokiServer(t *testing.T, viaqFails bool, otelFails bool) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var queries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("query")
		mu.Lock()
		queries = append(queries, q)
		mu.Unlock()
		if viaqFails && strings.Contains(q, "kubernetes_namespace_name") {
			http.Error(w, "viaq store down", http.StatusInternalServerError)
			return
		}
		if otelFails && strings.Contains(q, "k8s_namespace_name") {
			http.Error(w, "otel store down", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data":   map[string]any{"resultType": "streams", "result": []any{}},
		})
	}))
	return server, &queries
}

func TestLokiStoreGet_ContainerSelectorWithPod(t *testing.T) {
	server, queries := mockLokiServer(t, false, false)
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	store := NewLokiStore(baseURL, server.Client())

	q := mustNewQuery(t, `log:application:{"namespace":"myns","name":"mypod"}`)
	require.NotNil(t, q.direct)

	err := store.Get(context.Background(), q, &korrel8r.Constraint{}, korrel8r.AppenderFunc(func(o ...korrel8r.Object) {}))
	require.NoError(t, err)

	assert.Len(t, *queries, 2)
	assert.Contains(t, (*queries)[0], "kubernetes_namespace_name")
	assert.Contains(t, (*queries)[1], "k8s_namespace_name")
}

func TestLokiStoreGet_ContainerSelectorNamespaceOnly(t *testing.T) {
	server, queries := mockLokiServer(t, false, false)
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	store := NewLokiStore(baseURL, server.Client())

	q := mustNewQuery(t, `log:application:{"namespace":"myns","labels":{"app":"web"}}`)
	require.NotNil(t, q.direct)

	err := store.Get(context.Background(), q, &korrel8r.Constraint{}, korrel8r.AppenderFunc(func(o ...korrel8r.Object) {}))
	require.NoError(t, err)

	assert.Len(t, *queries, 1)
	assert.Contains(t, (*queries)[0], "kubernetes_namespace_name")
}

func TestLokiStoreGet_RawLogQL(t *testing.T) {
	server, queries := mockLokiServer(t, false, false)
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	store := NewLokiStore(baseURL, server.Client())

	q := mustNewQuery(t, `log:application:{kubernetes_namespace_name="myns"}|json`)
	require.Nil(t, q.direct)

	err := store.Get(context.Background(), q, &korrel8r.Constraint{}, korrel8r.AppenderFunc(func(o ...korrel8r.Object) {}))
	require.NoError(t, err)

	assert.Len(t, *queries, 1)
	assert.Contains(t, (*queries)[0], "kubernetes_namespace_name")
}

func TestLokiStoreGet_ViaqFailsOTELSucceeds(t *testing.T) {
	server, queries := mockLokiServer(t, true, false)
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	store := NewLokiStore(baseURL, server.Client())

	q := mustNewQuery(t, `log:application:{"namespace":"myns","name":"mypod"}`)
	require.NotNil(t, q.direct)

	err := store.Get(context.Background(), q, &korrel8r.Constraint{}, korrel8r.AppenderFunc(func(o ...korrel8r.Object) {}))
	assert.NoError(t, err)

	assert.Len(t, *queries, 2)
	assert.Contains(t, (*queries)[0], "kubernetes_namespace_name")
	assert.Contains(t, (*queries)[1], "k8s_namespace_name")
}

func TestLokiStoreGet_ViaqSucceedsOTELFails(t *testing.T) {
	server, queries := mockLokiServer(t, false, true)
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	store := NewLokiStore(baseURL, server.Client())

	q := mustNewQuery(t, `log:application:{"namespace":"myns","name":"mypod"}`)
	require.NotNil(t, q.direct)

	err := store.Get(context.Background(), q, &korrel8r.Constraint{}, korrel8r.AppenderFunc(func(o ...korrel8r.Object) {}))
	assert.NoError(t, err)

	assert.Len(t, *queries, 2)
	assert.Contains(t, (*queries)[0], "kubernetes_namespace_name")
	assert.Contains(t, (*queries)[1], "k8s_namespace_name")
}

func TestLokiStoreGet_BothFail(t *testing.T) {
	server, queries := mockLokiServer(t, true, true)
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	store := NewLokiStore(baseURL, server.Client())

	q := mustNewQuery(t, `log:application:{"namespace":"myns","name":"mypod"}`)
	require.NotNil(t, q.direct)

	err := store.Get(context.Background(), q, &korrel8r.Constraint{}, korrel8r.AppenderFunc(func(o ...korrel8r.Object) {}))
	assert.Error(t, err)

	assert.Len(t, *queries, 2)
	assert.Contains(t, (*queries)[0], "kubernetes_namespace_name")
	assert.Contains(t, (*queries)[1], "k8s_namespace_name")
}