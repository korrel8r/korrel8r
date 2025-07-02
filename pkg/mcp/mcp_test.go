// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mcp

import (
	"context"
	"encoding/json"
	"testing"

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
		[]string{"create_neighbours_graph", "create_goals_graph", "list_domain_classes", "list_domains"})
}

func TestListDomains(t *testing.T) {
	cs := newClient(t, newEngine(t))
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "list_domains",
	})
	require.NoError(t, err)
	want := []mcp.Content{&mcp.TextContent{Text: "mock  Mock domain.\n"}}
	assert.Equal(t, want, r.Content, test.JSONPretty(r))
}

func TestListDomainClasses(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_domain_classes",
		Arguments: DomainParams{Domain: "mock"},
	})
	require.NoError(t, err)
	want := []mcp.Content{&mcp.TextContent{Text: "a\nb\n"}}
	assert.Equal(t, want, r.Content, test.JSONPretty(r))
}

func TestCreateNeighboursGraph(t *testing.T) {
	client := newClient(t, newEngine(t))
	r, err := client.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      CreateNeighboursGraph,
		Arguments: NeighbourParams{Depth: 5, Start: Start{Queries: []string{"mock:a:x"}}},
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
			Start: Start{Queries: []string{"mock:a:x"}},
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
	s := NewServer(e)
	ct, st := mcp.NewInMemoryTransports()

	ss, err := s.Connect(ctx, st)
	require.NoError(t, err)

	c := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)
	cs, err := c.Connect(ctx, ct)
	require.NoError(t, err)

	t.Cleanup(func() { _ = cs.Close(); _ = ss.Wait() })
	return cs
}

func list[T any](x ...T) []T { return x }

func graphContent(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	g := &rest.Graph{}
	require.NoError(t, json.Unmarshal([]byte(test.JSONString(r.StructuredContent)), g))
	rest.Normalize(g)
	b, _ := json.Marshal(g)
	return string(b)
}
