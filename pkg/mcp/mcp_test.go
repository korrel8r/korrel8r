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
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/korrel8r/korrel8r/pkg/rest/auth"
	"github.com/korrel8r/korrel8r/pkg/session"
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
	s := NewServer(session.NewSingle(e, nil))
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
	sessions := session.NewSingle(e, nil)
	api, err := rest.New(sessions, router)
	require.NoError(t, err)

	ctx := context.Background()
	mcpSrv := NewServer(sessions)
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

func TestMultiSessionConsole(t *testing.T) {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.TestMode)
	}
	sessions := session.NewPool(time.Hour, func() (*engine.Engine, config.Configs, error) {
		return newEngine(t), nil, nil
	})
	defer sessions.Close()

	router := gin.New()
	// Auth middleware to extract Authorization header into context (as web.go does).
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.Context(c.Request))
		c.Next()
	})
	_, err := rest.New(sessions, router)
	require.NoError(t, err)

	setConsole := func(token, view string) {
		t.Helper()
		body, _ := json.Marshal(rest.Console{View: view})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/api/v1alpha1/console", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}

	getSession := func(token string) *session.Session {
		t.Helper()
		ctx := auth.WithToken(context.Background(), token)
		s, err := session.FromContext(ctx, sessions)
		require.NoError(t, err)
		return s
	}

	getConsoleState := func(token string) rest.Console {
		t.Helper()
		var c rest.Console
		require.NoError(t, getSession(token).Console.Get(&c))
		return c
	}

	// Set different console views for different auth tokens.
	setConsole("token-A", "mock:a:x")
	setConsole("token-B", "mock:b:y")

	// Verify each token's session has its own console state.
	assert.Equal(t, "mock:b:y", getConsoleState("token-B").View)
	assert.Equal(t, "mock:a:x", getConsoleState("token-A").View)

	// Verify send delivers to the matching session's Updates channel.
	sA := getSession("token-A")
	sB := getSession("token-B")

	// Send update to session A.
	go func() { _ = sA.Console.Send(&rest.Console{View: "update-A"}) }()

	select {
	case msg := <-sA.Console.Updates:
		var c rest.Console
		require.NoError(t, json.Unmarshal(msg, &c))
		assert.Equal(t, "update-A", c.View, "update should arrive on session A")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update on session A")
	}

	// Session B's channel should be empty.
	select {
	case msg := <-sB.Console.Updates:
		t.Fatalf("session B should not receive session A's update, got: %s", msg)
	default:
		// expected — nothing on B's channel
	}
}

// newMultiSessionRouter creates a gin router with auth middleware and a pool-based session manager.
func newMultiSessionRouter(t *testing.T) (*gin.Engine, session.Manager) {
	t.Helper()
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.TestMode)
	}
	sessions := session.NewPool(time.Hour, func() (*engine.Engine, config.Configs, error) {
		return newEngine(t), nil, nil
	})
	t.Cleanup(func() { sessions.Close() })

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(auth.Context(c.Request))
		c.Next()
	})
	_, err := rest.New(sessions, router)
	require.NoError(t, err)
	return router, sessions
}

// authRoundTripper injects an Authorization header into every request.
type authRoundTripper struct {
	token string
	next  http.RoundTripper
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", rt.token)
	return rt.next.RoundTrip(req)
}

// newMCPClientHTTP creates an MCP client that connects over HTTP with the given auth token.
func newMCPClientHTTP(t *testing.T, token string, sessions session.Manager) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	mcpSrv := NewServer(sessions)
	// Register MCP handler on a separate mux for this test.
	mux := http.NewServeMux()
	mux.Handle(StreamablePath, mcpSrv.HTTPHandler())
	// Wrap the mux with auth middleware.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(auth.Context(r))
		mux.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	transport := &mcp.StreamableClientTransport{
		Endpoint:             srv.URL + StreamablePath,
		DisableStandaloneSSE: true,
		HTTPClient: &http.Client{
			Transport: &authRoundTripper{token: token, next: http.DefaultTransport},
		},
	}
	c := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, err := c.Connect(ctx, transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestMultiSession_CrossProtocol(t *testing.T) {
	router, sessions := newMultiSessionRouter(t)

	// Set console state via REST with token-A.
	body, _ := json.Marshal(rest.Console{View: "view-A"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1alpha1/console", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token-A")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Set different console state via REST with token-B.
	body, _ = json.Marshal(rest.Console{View: "view-B"})
	w = httptest.NewRecorder()
	req = httptest.NewRequest("PUT", "/api/v1alpha1/console", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token-B")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// MCP client with token-A should see view-A.
	csA := newMCPClientHTTP(t, "token-A", sessions)
	r, err := csA.CallTool(context.Background(), &mcp.CallToolParams{Name: GetConsole})
	require.NoError(t, err)
	require.False(t, r.IsError, "MCP call should succeed")
	got := r.StructuredContent.(map[string]any)
	assert.Equal(t, "view-A", got["view"], "MCP client A should see view-A set by REST")

	// MCP client with token-B should see view-B.
	csB := newMCPClientHTTP(t, "token-B", sessions)
	r, err = csB.CallTool(context.Background(), &mcp.CallToolParams{Name: GetConsole})
	require.NoError(t, err)
	require.False(t, r.IsError, "MCP call should succeed")
	got = r.StructuredContent.(map[string]any)
	assert.Equal(t, "view-B", got["view"], "MCP client B should see view-B set by REST")
}

func TestMultiSession_SSEIsolation(t *testing.T) {
	router, sessions := newMultiSessionRouter(t)
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Connect two SSE clients with different auth tokens.
	connectSSE := func(token string) (<-chan string, context.CancelFunc) {
		ctx, cancel := context.WithCancel(t.Context())
		req, err := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/v1alpha1/console/updates", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", token)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		events := make(chan string, 10)
		go func() {
			defer func() { close(events); _ = resp.Body.Close() }()
			s := bufio.NewScanner(resp.Body)
			for s.Scan() {
				if data, ok := strings.CutPrefix(s.Text(), "data:"); ok {
					events <- data
				}
			}
		}()
		return events, cancel
	}

	eventsA, cancelA := connectSSE("token-A")
	defer cancelA()
	eventsB, cancelB := connectSSE("token-B")
	defer cancelB()

	// Give SSE handlers time to start reading.
	time.Sleep(100 * time.Millisecond)

	// Send update to session A only.
	ctx := auth.WithToken(context.Background(), "token-A")
	sA, err := session.FromContext(ctx, sessions)
	require.NoError(t, err)
	go func() { _ = sA.Console.Send(&rest.Console{View: "update-for-A"}) }()

	// Session A's SSE client should receive the update.
	select {
	case data := <-eventsA:
		var c rest.Console
		require.NoError(t, json.Unmarshal([]byte(data), &c))
		assert.Equal(t, "update-for-A", c.View)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE update on session A")
	}

	// Session B's SSE client should NOT receive anything.
	select {
	case data := <-eventsB:
		t.Fatalf("session B should not receive session A's update, got: %s", data)
	case <-time.After(200 * time.Millisecond):
		// expected — nothing on B's channel
	}
}

func graphContent(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	g := &rest.Graph{}
	require.NoError(t, json.Unmarshal([]byte(test.JSONString(r.StructuredContent)), g))
	rest.Normalize(g)
	b, _ := json.Marshal(g)
	return string(b)
}
