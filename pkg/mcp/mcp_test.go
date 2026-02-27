// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTools(t *testing.T) {
	cs := newClient(t, newEngine(t))
	tools, err := cs.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	assert.ElementsMatch(t, names,
		[]string{CreateNeighborsGraph,
			CreateGoalsGraph,
			ListDomainClasses,
			ListDomains,
			GetConsole,
			ShowInConsole})
}

func TestListDomains(t *testing.T) {
	cs := newClient(t, newEngine(t))
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_domains"})
	require.NoError(t, err)
	want := map[string]any{
		"domains": []any{
			map[string]any{
				"description": "Mock domain.",
				"name":        "mock",
			},
		},
	}
	assert.Equal(t, want, r.StructuredContent, r)
}

func TestListDomainClasses(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_domain_classes",
		Arguments: DomainParams{Domain: "mock"},
	})
	require.NoError(t, err)
	want := map[string]any{
		"domain":  "mock",
		"classes": []any{"a", "b"},
	}

	assert.Equal(t, want, r.StructuredContent)
}

func TestCreateNeighborsGraph(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      CreateNeighborsGraph,
		Arguments: NeighborParams{Depth: 5, Start: rest.Start{Queries: []string{"mock:a:x"}}},
	})
	require.NoError(t, err)
	got := graphContent(t, r)
	want := `{"edges":[{"goal":"mock:b","start":"mock:a"}],"nodes":[{"class":"mock:a","count":1,"queries":[{"count":1,"query":"mock:a:x"}]},{"class":"mock:b","count":1,"queries":[{"count":1,"query":"mock:b:y"}]}]}`
	assert.Equal(t, want, got)
}

func TestCreateGoalsGraph(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name: CreateGoalsGraph,
		Arguments: GoalParams{
			Goals: []string{"mock:b"},
			Start: rest.Start{Queries: []string{"mock:a:x"}},
		},
	})
	require.NoError(t, err)
	got := graphContent(t, r)
	want := `{"edges":[{"goal":"mock:b","start":"mock:a"}],"nodes":[{"class":"mock:a","count":1,"queries":[{"count":1,"query":"mock:a:x"}]},{"class":"mock:b","count":1,"queries":[{"count":1,"query":"mock:b:y"}]}]}`
	require.Equal(t, want, got)
}

func newEngine(t *testing.T) *engine.Engine {
	t.Helper()
	d := mock.NewDomain("mock", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	s := mock.NewStore(d)
	s.AddQuery("mock:a:x", "ax")
	s.AddQuery("mock:b:y", "by")
	r := mock.NewRule("a-b", list(a), list(b), mock.NewQuery(b, "y"))
	e, err := engine.Build().Domains(d).Stores(s).Rules(r).Engine()
	require.NoError(t, err)
	return e
}

func newClient(t *testing.T, e *engine.Engine) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	s := NewServer(e, nil) // No API for basic functionality
	ct, st := mcp.NewInMemoryTransports()

	ss, err := s.Connect(ctx, st, nil)
	require.NoError(t, err)

	c := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)
	cs, err := c.Connect(ctx, ct, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = cs.Close(); _ = ss.Wait() })
	return cs
}

func list[T any](x ...T) []T { return x }

func TestListDomainClasses_invalid(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      ListDomainClasses,
		Arguments: DomainParams{Domain: "nosuch"},
	})
	require.NoError(t, err) // MCP call succeeds, error is in result
	assert.True(t, r.IsError)
}

func TestConsoleToolsNoAPI(t *testing.T) {
	// With no API, console tools return errors
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: GetConsole})
	require.NoError(t, err)
	assert.True(t, r.IsError)

	r, err = client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      ShowInConsole,
		Arguments: rest.Console{},
	})
	require.NoError(t, err)
	assert.True(t, r.IsError)
}

func TestCreateNeighborsGraph_invalidQuery(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      CreateNeighborsGraph,
		Arguments: NeighborParams{Depth: 1, Start: rest.Start{Queries: []string{"bad:query"}}},
	})
	require.NoError(t, err)
	assert.True(t, r.IsError)
}

func TestCreateGoalsGraph_invalidGoal(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name: CreateGoalsGraph,
		Arguments: GoalParams{
			Goals: []string{"bad:class"},
			Start: rest.Start{Queries: []string{"mock:a:x"}},
		},
	})
	require.NoError(t, err)
	assert.True(t, r.IsError)
}

// newAPIAndClient creates REST and MCP API and an MCP client.
func newAPIAndClient(t *testing.T, e *engine.Engine) (*rest.API, *gin.Engine, *mcp.ClientSession) {
	t.Helper()
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.TestMode)
	}
	router := gin.New()
	api, err := rest.New(e, nil, router)
	require.NoError(t, err)

	ctx := context.Background()
	mcpSrv := NewServer(e, api)
	ct, st := mcp.NewInMemoryTransports()
	ss, err := mcpSrv.Connect(ctx, st, nil)
	require.NoError(t, err)
	c := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)
	cs, err := c.Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close(); _ = ss.Wait() })
	return api, router, cs
}

func TestSetConsoleToGetConsole(t *testing.T) {
	_, router, cs := newAPIAndClient(t, newEngine(t))

	// Set console state via REST SetConsole (POST /api/v1alpha1/console)
	view := "mock:a:x"
	body, _ := json.Marshal(rest.Console{View: view})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1alpha1/console", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Read via MCP GetConsole — should see the state set by REST
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: GetConsole})
	require.NoError(t, err)
	assert.False(t, r.IsError)
	got := r.StructuredContent.(map[string]any)
	assert.Equal(t, "mock:a:x", got["view"])
}

func TestConsoleShowInConsoleToSSE(t *testing.T) {
	_, router, cs := newAPIAndClient(t, newEngine(t))
	srv := httptest.NewServer(router)

	// Connect to SSE endpoint
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/v1alpha1/console/updates", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Receive events in background
	events := make(chan string)
	go func() {
		defer func() { close(events) }()
		s := bufio.NewScanner(resp.Body)
		for s.Scan() {
			if data, ok := strings.CutPrefix(s.Text(), "data:"); ok {
				events <- data
			}
		}
	}()
	// Give SSE handler time to start reading from channel
	time.Sleep(100 * time.Millisecond)

	// send/receive helper functions
	send := func(c rest.Console) {
		t.Helper()
		r, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: ShowInConsole, Arguments: c})
		require.NoError(t, err)
		require.False(t, r.IsError, test.JSONString(r))
	}
	recv := func() (c rest.Console) {
		t.Helper()
		data, ok := <-events
		require.True(t, ok)
		require.NoError(t, json.Unmarshal([]byte(data), &c))
		return c
	}

	// Send update via MCP ShowInConsole
	want := rest.Console{
		View: "mock:a:x",
		Search: &rest.Search{
			Goals: &rest.Goals{Start: rest.Start{Class: "mock:b"}},
		},
	}
	send(want)
	got := recv()
	assert.Equal(t, want, got)
}

func graphContent(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	g := &rest.Graph{}
	require.NoError(t, json.Unmarshal([]byte(test.JSONString(r.StructuredContent)), g))
	rest.Normalize(g)
	b, _ := json.Marshal(g)
	return string(b)
}
