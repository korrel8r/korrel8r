// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mcp

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// testArgs are valid arguments for each tool, for use with mockAPI.
var testArgs = map[string]any{
	InstructionsTool:     map[string]any{},
	ListDomains:          map[string]any{},
	ListDomainClasses:    map[string]any{"domain": "k8s"},
	Help:                 map[string]any{"domain": "k8s"},
	CreateNeighborsGraph: map[string]any{"start": map[string]any{"queries": []string{"k8s:Pod:{}"}}, "depth": 2},
	CreateGoalsGraph:     map[string]any{"start": map[string]any{"queries": []string{"k8s:Pod:{}"}}, "goals": []string{"log:application"}},
	GetObjects:           map[string]any{"query": "k8s:Pod:{}"},
	GetConsole:           map[string]any{},
	ShowInConsole:        map[string]any{"view": "k8s:Pod.v1:{}"},
}

// asJSON converts v to a generic JSON value, for comparison with values decoded from the wire.
func asJSON(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var result any
	require.NoError(t, json.Unmarshal(b, &result))
	return result
}

func callTool(t *testing.T, r Registration, args any) *mcp.CallToolResult {
	t.Helper()
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	result, err := r.Handler(t.Context(), &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: raw}})
	require.NoError(t, err)
	return result
}

// resultValues are the parts of a tool result that a handler controls.
// The server session adds its own metadata.
func resultValues(t *testing.T, r *mcp.CallToolResult) any {
	t.Helper()
	return asJSON(t, map[string]any{"content": r.Content, "structured": r.StructuredContent, "isError": r.IsError})
}

// Registrations must advertise and behave exactly like the tools of the standalone server.
func TestRegistrations(t *testing.T) {
	c, _ := testClient(t)
	registrations := Registrations(c)
	names := make([]string, len(registrations))
	for i, r := range registrations {
		names[i] = r.Tool.Name
	}
	require.ElementsMatch(t, slices.Collect(maps.Keys(testArgs)), names)

	// Connect an MCP client to the standalone server.
	ctx := t.Context()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	_, err := NewServer(c, "test", logr.Discard()).Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = cs.Close() }()
	list, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, list.Tools, len(registrations))
	listed := map[string]*mcp.Tool{}
	for _, tool := range list.Tools {
		listed[tool.Name] = tool
	}

	for _, r := range registrations {
		t.Run(r.Tool.Name, func(t *testing.T) {
			require.Contains(t, listed, r.Tool.Name, "tool not advertised by server")
			assert.Equal(t, asJSON(t, r.Tool), asJSON(t, listed[r.Tool.Name]), "tool definition")

			args := testArgs[r.Tool.Name]
			viaServer, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: r.Tool.Name, Arguments: args})
			require.NoError(t, err)
			assert.Equal(t, resultValues(t, viaServer), resultValues(t, callTool(t, r, args)), "tool result")
		})
	}
}

func TestRegistrations_instructions(t *testing.T) {
	var registration *Registration
	for _, r := range Registrations(nil) {
		if r.Tool.Name == InstructionsTool {
			registration = &r
			break
		}
	}
	require.NotNil(t, registration)
	assert.Contains(t, registration.Tool.Description, "client does not expose or use the instructions")
	for _, name := range []string{GetConsole, ShowInConsole, ListDomains, Help, CreateGoalsGraph, CreateNeighborsGraph} {
		assert.Contains(t, Instructions, "'"+name+"'")
	}

	// No REST client is needed to retrieve the instructions.
	direct := callTool(t, *registration, map[string]any{})
	require.False(t, direct.IsError)
	assert.Equal(t, InstructionsResult{Instructions: Instructions}, direct.StructuredContent)

	ctx := t.Context()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	_, err := NewServer(nil, "test", logr.Discard()).Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer func() { _ = cs.Close() }()

	result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: InstructionsTool, Arguments: map[string]any{}})
	require.NoError(t, err)
	require.False(t, result.IsError)
	want := cs.InitializeResult().Instructions
	require.Equal(t, Instructions, want)
	assert.Equal(t, map[string]any{"instructions": want}, asJSON(t, result.StructuredContent))
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	var body InstructionsResult
	require.NoError(t, json.Unmarshal([]byte(text.Text), &body))
	assert.Equal(t, want, body.Instructions)
}

func TestRegistrations_emptyLists(t *testing.T) {
	c := NewClientForHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, nil) // The REST API may return null for empty lists.
	}))
	for _, r := range Registrations(c) {
		switch r.Tool.Name {
		case ListDomains, ListDomainClasses:
			args := map[string]any{}
			if r.Tool.Name == ListDomainClasses {
				args["domain"] = "empty"
			}
			result := callTool(t, r, args)
			require.False(t, result.IsError, r.Tool.Name)
			var body map[string]any
			require.NoError(t, json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &body))
			key := "domains"
			if r.Tool.Name == ListDomainClasses {
				key = "classes"
			}
			assert.Equal(t, []any{}, body[key], r.Tool.Name)
		}
	}
}

type invalidOutput struct {
	Items string `json:"items"`
}

func (invalidOutput) MarshalJSON() ([]byte, error) { return []byte(`{"items":42}`), nil }

func TestNewTool_validatesOutput(t *testing.T) {
	r := newTool(&mcp.Tool{Name: "invalid_output"}, func(context.Context, struct{}) (invalidOutput, error) {
		return invalidOutput{}, nil
	})
	_, err := r.Handler(t.Context(), &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{}`)}})
	require.ErrorContains(t, err, "invalid invalid_output result")
}

func TestRegistrations_annotations(t *testing.T) {
	for _, r := range Registrations(nil) {
		if r.Tool.Name == ShowInConsole { // The only tool that changes state.
			require.NotNil(t, r.Tool.Annotations.DestructiveHint)
			assert.True(t, *r.Tool.Annotations.DestructiveHint)
		} else {
			assert.True(t, r.Tool.Annotations.ReadOnlyHint, r.Tool.Name)
		}
	}
}

func TestRegistrations_errors(t *testing.T) {
	c, _ := testClient(t)
	registrations := map[string]Registration{}
	for _, r := range Registrations(c) {
		registrations[r.Tool.Name] = r
	}
	for _, x := range []struct{ tool, args, want string }{
		{ListDomainClasses, `{"domain":"nonexistent"}`, "404"}, // Error from the REST API.
		{GetObjects, `{"constraint":{}}`, "invalid arguments"}, // Missing required argument.
		{GetObjects, `{"query":123}`, "invalid arguments"},     // Wrong argument type.
	} {
		t.Run(x.tool+x.args, func(t *testing.T) {
			result := callTool(t, registrations[x.tool], json.RawMessage(x.args))
			assert.True(t, result.IsError)
			assert.Contains(t, result.Content[0].(*mcp.TextContent).Text, x.want)
		})
	}
}

func TestClient_responseTooLarge(t *testing.T) {
	c, _ := testClient(t)
	c.MaxResponseBytes = 4
	_, err := c.ListDomains(t.Context())
	assert.ErrorIs(t, err, ErrResponseTooLarge)
}

func TestClient_cancel(t *testing.T) {
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(blocked)
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(t.Context())
	go func() { <-blocked; cancel() }()
	_, err := NewClient(srv.URL, srv.Client()).ListDomains(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

// Console calls go to the session selected by the identity of the client transport.
// Clients for different users must not see each other's console.
func TestClient_consolePerUser(t *testing.T) {
	consoles := map[string]api.Console{"alice": {View: "k8s:Pod.v1:alice"}, "bob": {View: "k8s:Pod.v1:bob"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		console, ok := consoles[r.Header.Get("X-User")]
		if !ok {
			http.Error(w, "no console is connected", http.StatusNotFound)
			return
		}
		if r.Method == http.MethodPut {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&console))
			consoles[r.Header.Get("X-User")] = console
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, console)
	}))
	defer srv.Close()
	clientFor := func(user string) *Client {
		return NewClient(srv.URL, &http.Client{Transport: userTransport(user)})
	}

	require.NoError(t, clientFor("alice").ShowInConsole(t.Context(), &api.Console{View: "k8s:Pod.v1:updated"}))
	alice, err := clientFor("alice").GetConsole(t.Context())
	require.NoError(t, err)
	assert.Equal(t, api.Query("k8s:Pod.v1:updated"), alice.View)
	bob, err := clientFor("bob").GetConsole(t.Context())
	require.NoError(t, err)
	assert.Equal(t, api.Query("k8s:Pod.v1:bob"), bob.View)
	_, err = clientFor("nobody").GetConsole(t.Context())
	assert.ErrorContains(t, err, "no console is connected")
}

// userTransport identifies the user to the test server, like an authorization header.
func userTransport(user string) roundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.Header.Set("X-User", user)
		return http.DefaultTransport.RoundTrip(r)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

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
