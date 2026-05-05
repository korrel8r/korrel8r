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
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/auth"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/rest"
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
		[]string{
			GetConsole,
			ShowInConsole,
			CreateNeighborsGraph,
			CreateGoalsGraph,
			GetObjects,
			Help,
			ListDomainClasses,
			ListDomains})
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
		Arguments: NeighborParams{Depth: 5, Start: api.Start{Queries: []string{"mock:a:x"}}},
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
			Start: api.Start{Queries: []string{"mock:a:x"}},
		},
	})
	require.NoError(t, err)
	got := graphContent(t, r)
	want := `{"edges":[{"goal":"mock:b","start":"mock:a"}],"nodes":[{"class":"mock:a","count":1,"queries":[{"count":1,"query":"mock:a:x"}]},{"class":"mock:b","count":1,"queries":[{"count":1,"query":"mock:b:y"}]}]}`
	require.Equal(t, want, got)
}

func TestGetObjects(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      GetObjects,
		Arguments: ObjectsParams{Query: "mock:a:x"},
	})
	require.NoError(t, err)
	require.False(t, r.IsError)
	got := r.StructuredContent.(map[string]any)
	assert.Equal(t, []any{"ax"}, got["objects"])
}

func TestGetObjects_empty(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      GetObjects,
		Arguments: ObjectsParams{Query: "mock:a:none"},
	})
	require.NoError(t, err)
	require.False(t, r.IsError)
	got := r.StructuredContent.(map[string]any)
	assert.Equal(t, []any{}, got["objects"])
}

func TestGetObjects_invalidQuery(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      GetObjects,
		Arguments: ObjectsParams{Query: "bad:query"},
	})
	require.NoError(t, err)
	assert.True(t, r.IsError)
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
	s := NewServer(session.NewSingle(e))
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
		Arguments: NeighborParams{Depth: 1, Start: api.Start{Queries: []string{"bad:query"}}},
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
			Start: api.Start{Queries: []string{"mock:a:x"}},
		},
	})
	require.NoError(t, err)
	assert.True(t, r.IsError)
}

// --- REST ↔ MCP interop tests over HTTP ---

// interopFixture holds a REST router and an MCP client that share the same session manager.
type interopFixture struct {
	router *gin.Engine
	mcp    *mcp.ClientSession
}

func newInteropFixture(t *testing.T, e *engine.Engine) *interopFixture {
	t.Helper()
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.TestMode)
	}
	sessions := session.NewSingle(e)
	router := gin.New()
	_, err := rest.New(sessions, router)
	require.NoError(t, err)

	mcpSrv := NewServer(sessions)
	srv := httptest.NewServer(mcpSrv.HTTPHandler())
	t.Cleanup(srv.Close)

	transport := &mcp.StreamableClientTransport{
		Endpoint:             srv.URL,
		DisableStandaloneSSE: true,
	}
	c := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, err := c.Connect(context.Background(), transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })

	return &interopFixture{router: router, mcp: cs}
}

// restDo performs an HTTP request against the REST router and returns the response.
func (f *interopFixture) restDo(t *testing.T, method, url string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		require.NoError(t, err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, url, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	f.router.ServeHTTP(w, req)
	return w
}

func graphContent(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	b, err := json.Marshal(r.StructuredContent)
	require.NoError(t, err)
	g := &api.Graph{}
	require.NoError(t, json.Unmarshal(b, g))
	rest.Normalize(g)
	out, _ := json.Marshal(g)
	return string(out)
}

func TestInterop_ListDomains(t *testing.T) {
	f := newInteropFixture(t, newEngine(t))

	// REST: GET /api/v1alpha1/domains
	w := f.restDo(t, "GET", "/api/v1alpha1/domains", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var restDomains []api.Domain
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restDomains))

	// MCP: list_domains
	r, err := f.mcp.CallTool(context.Background(), &mcp.CallToolParams{Name: ListDomains})
	require.NoError(t, err)
	mcpResult := r.StructuredContent.(map[string]any)
	mcpDomains := mcpResult["domains"].([]any)

	assert.Equal(t, len(restDomains), len(mcpDomains))
	assert.Equal(t, restDomains[0].Name, mcpDomains[0].(map[string]any)["name"])
	assert.Equal(t, restDomains[0].Description, mcpDomains[0].(map[string]any)["description"])
}

func TestInterop_NeighborsGraph(t *testing.T) {
	f := newInteropFixture(t, newEngine(t))

	params := api.Neighbors{
		Start: api.Start{Queries: []string{"mock:a:x"}},
		Depth: 5,
	}

	// REST: POST /api/v1alpha1/graphs/neighbors
	w := f.restDo(t, "POST", "/api/v1alpha1/graphs/neighbors", params)
	require.Equal(t, http.StatusOK, w.Code)
	restGraph := &api.Graph{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), restGraph))
	rest.Normalize(restGraph)

	// MCP: create_neighbors_graph
	r, err := f.mcp.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      CreateNeighborsGraph,
		Arguments: NeighborParams(params),
	})
	require.NoError(t, err)
	mcpGraphJSON := graphContent(t, r)

	restGraphJSON, _ := json.Marshal(restGraph)
	assert.JSONEq(t, string(restGraphJSON), mcpGraphJSON)
}

func TestInterop_GoalsGraph(t *testing.T) {
	f := newInteropFixture(t, newEngine(t))

	params := api.Goals{
		Start: api.Start{Queries: []string{"mock:a:x"}},
		Goals: []string{"mock:b"},
	}

	// REST: POST /api/v1alpha1/graphs/goals
	w := f.restDo(t, "POST", "/api/v1alpha1/graphs/goals", params)
	require.Equal(t, http.StatusOK, w.Code)
	restGraph := &api.Graph{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), restGraph))
	rest.Normalize(restGraph)

	// MCP: create_goals_graph
	r, err := f.mcp.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      CreateGoalsGraph,
		Arguments: GoalParams(params),
	})
	require.NoError(t, err)
	mcpGraphJSON := graphContent(t, r)

	restGraphJSON, _ := json.Marshal(restGraph)
	assert.JSONEq(t, string(restGraphJSON), mcpGraphJSON)
}

func TestInterop_GetObjects(t *testing.T) {
	f := newInteropFixture(t, newEngine(t))

	// REST: GET /api/v1alpha1/objects?query=mock:a:x
	w := f.restDo(t, "GET", "/api/v1alpha1/objects?query=mock:a:x", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var restObjects []any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restObjects))

	// MCP: get_objects
	r, err := f.mcp.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      GetObjects,
		Arguments: ObjectsParams{Query: "mock:a:x"},
	})
	require.NoError(t, err)
	require.False(t, r.IsError)
	mcpResult := r.StructuredContent.(map[string]any)
	mcpObjects := mcpResult["objects"].([]any)

	assert.Equal(t, restObjects, mcpObjects)
}

func TestInterop_SetConsoleViaREST_GetViaMCP(t *testing.T) {
	f := newInteropFixture(t, newEngine(t))

	// Set console state via REST
	w := f.restDo(t, "PUT", "/api/v1alpha1/console", api.Console{View: "mock:a:x"})
	require.Equal(t, http.StatusOK, w.Code)

	// Read via MCP get_console
	r, err := f.mcp.CallTool(context.Background(), &mcp.CallToolParams{Name: GetConsole})
	require.NoError(t, err)
	require.False(t, r.IsError, "get_console should succeed")
	got := r.StructuredContent.(map[string]any)
	assert.Equal(t, "mock:a:x", got["view"])
}

func TestInterop_ListDomainClasses(t *testing.T) {
	f := newInteropFixture(t, newEngine(t))

	// REST: GET /api/v1alpha1/domain/mock/classes
	w := f.restDo(t, "GET", "/api/v1alpha1/domain/mock/classes", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var restClasses []string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &restClasses))

	// MCP: list_domain_classes
	r, err := f.mcp.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      ListDomainClasses,
		Arguments: DomainParams{Domain: "mock"},
	})
	require.NoError(t, err)
	mcpResult := r.StructuredContent.(map[string]any)
	mcpClasses := mcpResult["classes"].([]any)

	require.Equal(t, len(restClasses), len(mcpClasses))
	for i, name := range restClasses {
		assert.Equal(t, name, mcpClasses[i])
	}
}

func TestInterop_ShowInConsoleToSSE(t *testing.T) {
	f := newInteropFixture(t, newEngine(t))

	// Start a real HTTP server for the REST router so we can stream SSE.
	restSrv := httptest.NewServer(f.router)
	t.Cleanup(restSrv.Close)

	// Connect to the SSE endpoint.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, "GET", restSrv.URL+"/api/v1alpha1/console/events", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	// Read SSE data events in background.
	events := make(chan string, 1)
	go func() {
		defer close(events)
		s := bufio.NewScanner(resp.Body)
		for s.Scan() {
			if data, ok := strings.CutPrefix(s.Text(), "data: "); ok {
				events <- data
			}
		}
	}()
	time.Sleep(100 * time.Millisecond) // let SSE handler start

	// Send console update via MCP show_in_console.
	want := api.Console{View: "mock:a:x"}
	r, err := f.mcp.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      ShowInConsole,
		Arguments: ShowInConsoleParams(want),
	})
	require.NoError(t, err)
	require.False(t, r.IsError, "show_in_console should succeed")

	// Verify the SSE event matches.
	select {
	case data := <-events:
		var got api.Console
		require.NoError(t, json.Unmarshal([]byte(data), &got))
		assert.Equal(t, want, got)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
}

// --- Multi-session tests ---

// multiSessionFixture provides REST and MCP servers sharing a pool session manager.
// Each MCP client and REST request carries a bearer token to select its session.
type multiSessionFixture struct {
	restRouter *gin.Engine
	sessions   session.Manager
	mcpSrvURL  string
}

func newMultiSessionFixture(t *testing.T) *multiSessionFixture {
	t.Helper()
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.TestMode)
	}
	sessions := session.NewPool(time.Hour, func() (*engine.Engine, error) {
		return newEngine(t), nil
	})

	router := gin.New()
	_, err := rest.New(sessions, router)
	require.NoError(t, err)

	mcpSrv := NewServer(sessions)
	mcpHandler := mcpSrv.HTTPHandler()
	// Wrap the MCP HTTP handler with auth middleware so sessions.Get resolves
	// the bearer token from the Authorization header.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mcpHandler.ServeHTTP(w, auth.UpdateRequest(r))
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &multiSessionFixture{restRouter: router, sessions: sessions, mcpSrvURL: srv.URL}
}

// restSetConsole sets console state via REST PUT /console with the given bearer token.
func (f *multiSessionFixture) restSetConsole(t *testing.T, token string, console api.Console) {
	t.Helper()
	body, err := json.Marshal(console)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1alpha1/console", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	f.restRouter.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

// mcpClient creates an MCP client that authenticates with the given bearer token.
func (f *multiSessionFixture) mcpClient(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	transport := &mcp.StreamableClientTransport{
		Endpoint:             f.mcpSrvURL,
		DisableStandaloneSSE: true,
		HTTPClient: &http.Client{
			Transport: &tokenRoundTripper{token: "Bearer " + token, next: http.DefaultTransport},
		},
	}
	c := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, err := c.Connect(context.Background(), transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// mcpGetConsoleView calls MCP get_console and returns the view field.
func mcpGetConsoleView(t *testing.T, cs *mcp.ClientSession) string {
	t.Helper()
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: GetConsole})
	require.NoError(t, err)
	require.False(t, r.IsError, "get_console should succeed")
	return r.StructuredContent.(map[string]any)["view"].(string)
}

type tokenRoundTripper struct {
	token string
	next  http.RoundTripper
}

func (rt *tokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", rt.token)
	return rt.next.RoundTrip(req)
}

func TestMultiSession_ConsoleIsolation(t *testing.T) {
	f := newMultiSessionFixture(t)

	// Set different console views via REST for two different tokens.
	f.restSetConsole(t, "token-A", api.Console{View: "mock:a:x"})
	f.restSetConsole(t, "token-B", api.Console{View: "mock:b:y"})

	// Each MCP client should see only its own session's console state.
	csA := f.mcpClient(t, "token-A")
	csB := f.mcpClient(t, "token-B")

	assert.Equal(t, "mock:a:x", mcpGetConsoleView(t, csA), "token-A should see view set for A")
	assert.Equal(t, "mock:b:y", mcpGetConsoleView(t, csB), "token-B should see view set for B")
}

func TestMultiSession_ConsoleUpdate(t *testing.T) {
	f := newMultiSessionFixture(t)

	// Set initial view for token-A.
	f.restSetConsole(t, "token-A", api.Console{View: "mock:a:x"})
	csA := f.mcpClient(t, "token-A")
	assert.Equal(t, "mock:a:x", mcpGetConsoleView(t, csA))

	// Update the view for the same token.
	f.restSetConsole(t, "token-A", api.Console{View: "mock:b:y"})
	assert.Equal(t, "mock:b:y", mcpGetConsoleView(t, csA), "should reflect updated view")
}

func TestMultiSession_SSEIsolation(t *testing.T) {
	f := newMultiSessionFixture(t)

	restSrv := httptest.NewServer(f.restRouter)
	t.Cleanup(restSrv.Close)

	// Connect two SSE clients with different auth tokens.
	connectSSE := func(token string) <-chan string {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		req, err := http.NewRequestWithContext(ctx, "GET", restSrv.URL+"/api/v1alpha1/console/events", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })

		events := make(chan string, 10)
		go func() {
			defer close(events)
			s := bufio.NewScanner(resp.Body)
			for s.Scan() {
				if data, ok := strings.CutPrefix(s.Text(), "data: "); ok {
					events <- data
				}
			}
		}()
		return events
	}

	eventsA := connectSSE("token-A")
	eventsB := connectSSE("token-B")
	time.Sleep(100 * time.Millisecond) // let SSE handlers start

	// MCP client for token-A sends a console update.
	csA := f.mcpClient(t, "token-A")
	r, err := csA.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      ShowInConsole,
		Arguments: ShowInConsoleParams{View: "mock:a:x"},
	})
	require.NoError(t, err)
	require.False(t, r.IsError)

	// Session A's SSE client should receive the update.
	select {
	case data := <-eventsA:
		var got api.Console
		require.NoError(t, json.Unmarshal([]byte(data), &got))
		assert.Equal(t, "mock:a:x", got.View)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE event on session A")
	}

	// Session B's SSE client should NOT receive anything.
	select {
	case data := <-eventsB:
		t.Fatalf("session B should not receive session A's update, got: %s", data)
	case <-time.After(200 * time.Millisecond):
		// expected
	}
}
