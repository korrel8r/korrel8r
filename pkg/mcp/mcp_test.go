// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAPI is a minimal HTTP handler that simulates the korrel8r REST API.
func mockAPI() http.Handler {
	mux := http.NewServeMux()
	prefix := api.BasePath

	mux.HandleFunc("GET "+prefix+"/domains", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []api.Domain{{Name: "k8s", Description: "Kubernetes resources"}})
	})
	mux.HandleFunc("GET "+prefix+"/domain/k8s/classes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []string{"Pod.v1", "Deployment.v1.apps"})
	})
	mux.HandleFunc("GET "+prefix+"/help", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, api.Help{Documentation: "all domains help"})
	})
	mux.HandleFunc("GET "+prefix+"/help/k8s", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, api.Help{Documentation: "k8s domain help"})
	})
	mux.HandleFunc("GET "+prefix+"/objects", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []json.RawMessage{json.RawMessage(`{"name":"pod1"}`)})
	})
	mux.HandleFunc("POST "+prefix+"/graphs/neighbors", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, api.Graph{Nodes: []api.Node{{Class: "k8s:Pod.v1", Count: intPtr(1)}}})
	})
	mux.HandleFunc("POST "+prefix+"/graphs/goals", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, api.Graph{Nodes: []api.Node{{Class: "log:application", Count: intPtr(5)}}})
	})
	mux.HandleFunc("GET "+prefix+"/console", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, api.Console{View: "k8s:Pod.v1:{}"})
	})
	mux.HandleFunc("PUT "+prefix+"/console/events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
	})
	return mux
}

func intPtr(i int) *int { return &i }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func testClient(t *testing.T) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mockAPI())
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, srv.Client()), srv
}

func TestClient_ListDomains(t *testing.T) {
	c, _ := testClient(t)
	domains, err := c.ListDomains(context.Background())
	require.NoError(t, err)
	require.Len(t, domains, 1)
	assert.Equal(t, "k8s", domains[0].Name)
}

func TestClient_ListDomainClasses(t *testing.T) {
	c, _ := testClient(t)
	classes, err := c.ListDomainClasses(context.Background(), "k8s")
	require.NoError(t, err)
	assert.Equal(t, []string{"Pod.v1", "Deployment.v1.apps"}, classes)
}

func TestClient_Help(t *testing.T) {
	c, _ := testClient(t)

	doc, err := c.Help(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "all domains help", doc)

	doc, err = c.Help(context.Background(), "k8s")
	require.NoError(t, err)
	assert.Equal(t, "k8s domain help", doc)
}

func TestClient_GetObjects(t *testing.T) {
	c, _ := testClient(t)
	objs, err := c.GetObjects(context.Background(), "k8s:Pod:{}", nil)
	require.NoError(t, err)
	require.Len(t, objs, 1)
	assert.Contains(t, string(objs[0]), "pod1")
}

func TestClient_GetObjects_WithConstraint(t *testing.T) {
	c, _ := testClient(t)
	limit := 10
	queryLimit := 5
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	objs, err := c.GetObjects(context.Background(), "k8s:Pod:{}", &api.Constraint{
		Limit:      &limit,
		QueryLimit: &queryLimit,
		Start:      &start,
		End:        &end,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, objs)
}

func TestClient_GraphNeighbors(t *testing.T) {
	c, _ := testClient(t)
	g, err := c.GraphNeighbors(context.Background(), api.Neighbors{
		Start: api.Start{Queries: []string{"k8s:Pod:{}"}},
		Depth: 2,
	})
	require.NoError(t, err)
	require.Len(t, g.Nodes, 1)
	assert.Equal(t, "k8s:Pod.v1", g.Nodes[0].Class)
}

func TestClient_GraphGoals(t *testing.T) {
	c, _ := testClient(t)
	g, err := c.GraphGoals(context.Background(), api.Goals{
		Start: api.Start{Queries: []string{"k8s:Pod:{}"}},
		Goals: []string{"log:application"},
	})
	require.NoError(t, err)
	require.Len(t, g.Nodes, 1)
	assert.Equal(t, "log:application", g.Nodes[0].Class)
}

func TestClient_Console(t *testing.T) {
	c, _ := testClient(t)
	console, err := c.GetConsole(context.Background())
	require.NoError(t, err)
	assert.Equal(t, api.Query("k8s:Pod.v1:{}"), console.View)

	err = c.ShowInConsole(context.Background(), &api.Console{View: "log:application:{}"})
	assert.NoError(t, err)
}

func TestClient_HTTPError(t *testing.T) {
	c, _ := testClient(t)
	_, err := c.ListDomainClasses(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestNewClientForHandler(t *testing.T) {
	c := NewClientForHandler(mockAPI())
	domains, err := c.ListDomains(context.Background())
	require.NoError(t, err)
	assert.Len(t, domains, 1)
}

func TestNewClient_DefaultHTTPClient(t *testing.T) {
	c := NewClient("http://localhost", nil)
	assert.NotNil(t, c.httpClient)
}

func TestAddTools(t *testing.T) {
	tools := AddTools(nil, nil)
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	assert.ElementsMatch(t, []string{
		ListDomains, ListDomainClasses, Help,
		CreateNeighborsGraph, CreateGoalsGraph, GetObjects,
		GetConsole, ShowInConsole,
	}, names)
}

func TestNewServer(t *testing.T) {
	c, _ := testClient(t)
	s := NewServer(c, "test-version", logr.Discard())
	assert.NotNil(t, s.Server)
	assert.Len(t, s.AllTools(), 8)
}

func TestJsonValue_MarshalLog(t *testing.T) {
	v := jsonValue{v: map[string]int{"a": 1}}
	result := v.MarshalLog()
	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), m["a"])
}

func TestJsonValue_MarshalLog_Unmarshable(t *testing.T) {
	v := jsonValue{v: make(chan int)}
	result := v.MarshalLog()
	_, ok := result.(string)
	assert.True(t, ok)
}
