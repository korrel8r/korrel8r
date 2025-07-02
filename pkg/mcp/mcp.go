// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package mcp Provides an MCP server and argument structures for MCP client calls.
package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/text"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	StreamablePath = "/mcp"
	SSEPath        = "/sse"
)

type DomainParams struct {
	Domain string `json:"domain" jsonschema:"Name of the domain to list"`
}

type NeighbourParams = rest.Neighbours
type GoalParams = rest.Goals
type Start = rest.Start
type Graph = rest.Graph

const (
	ListDomains           = "list_domains"
	ListDomainClasses     = "list_domain_classes"
	CreateGoalsGraph      = "create_goals_graph"
	CreateNeighboursGraph = "create_neighbours_graph"
)

type Server struct {
	*mcp.Server
	Engine *engine.Engine
}

func NewServer(e *engine.Engine) *Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "korrle8r", Title: "Korrle8r MCP Server", Version: build.Version}, nil)
	addTools(e, s)
	// TODO register prompts, instructions, resources.
	return &Server{Server: s, Engine: e}
}

func addTools(e *engine.Engine, s *mcp.Server) {

	mcp.AddTool(s, &mcp.Tool{
		Name: ListDomains,
		Description: `
Returns a list of Korrel8r domains.
A domain contains obeservable signals or resources that use the same data model,
storage technology, and query syntax.
`,
	},
		func(ctx context.Context, ss *mcp.ServerSession, p *mcp.CallToolParamsFor[struct{}]) (*mcp.CallToolResult, error) {
			return textResult(text.WriteString(text.NewPrinter(e).ListDomains)), nil
		})

	mcp.AddTool(s, &mcp.Tool{
		Name: ListDomainClasses,
		Description: `
Returns a list of classes in a domain.
A domain contains one or more classes, representing objects with different structures.
Some domains have only a single class, others (like the 'k8s' domain) have many.
`,
	},
		func(ctx context.Context, ss *mcp.ServerSession, p *mcp.CallToolParamsFor[DomainParams]) (*mcp.CallToolResult, error) {
			d, err := e.Domain(p.Arguments.Domain)
			if err != nil {
				return errorResult(err), nil
			}
			text := text.WriteString(func(w io.Writer) {
				text.NewPrinter(e).ListClasses(w, d)
			})
			return textResult(text), nil
		})

	mcp.AddTool(s, &mcp.Tool{
		Name: CreateNeighboursGraph,
		Description: `
Returns a JSON graph of correlated objects.
From a set of start objects, follow correlation rules to find related objects up to the specified depth.`,
	},
		func(ctx context.Context, ss *mcp.ServerSession, p *mcp.CallToolParamsFor[NeighbourParams]) (*mcp.CallToolResultFor[*Graph], error) {
			args := p.Arguments
			start, err := rest.TraverseStart(e, args.Start)
			if err != nil {
				return errorResultFor[*Graph](err), nil
			}
			return graphResult(traverse.New(e, e.Graph()).Neighbours(ctx, start, args.Depth)), nil
		})

	mcp.AddTool(s, &mcp.Tool{
		Name: CreateGoalsGraph,
		Description: `
Returns a JSON graph of correlated objects.
From a set of start objects, follow all paths leading to one of the goal classes.`,
	},
		func(ctx context.Context, ss *mcp.ServerSession, p *mcp.CallToolParamsFor[GoalParams]) (*mcp.CallToolResultFor[*Graph], error) {
			args := p.Arguments
			start, err := rest.TraverseStart(e, args.Start)
			if err != nil {
				return errorResultFor[*Graph](err), nil
			}
			goals, err := e.Classes(args.Goals)
			if err != nil {
				return errorResultFor[*Graph](err), nil
			}
			return graphResult(traverse.New(e, e.Graph()).Goals(ctx, start, goals)), nil
		})
}

// ServeStdio runs an MCP server, it returns when the client disconnects or the context is canceled.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.Run(ctx, mcp.NewStdioTransport())
}

// HTTPHandler  a handler for the Streaming MCP protocol.
func (s *Server) HTTPHandler() http.Handler {
	// Use the same server for all requests. Server and Engine are concurrent-safe.
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s.Server }, nil)
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func graphResult(g *graph.Graph, err error) *mcp.CallToolResultFor[*Graph] {
	if err != nil {
		return errorResultFor[*Graph](err)
	}
	return &mcp.CallToolResultFor[*Graph]{StructuredContent: rest.NewGraph(g, false)}
}

func errorResultFor[T any](err error) *mcp.CallToolResultFor[T] {
	return &mcp.CallToolResultFor[T]{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
		IsError: true,
	}
}

var errorResult = errorResultFor[any]
