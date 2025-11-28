// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package mcp Provides an MCP server and argument structures for MCP client calls.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/text"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	StreamablePath = "/mcp"
)

type DomainParams struct {
	Domain string `json:"domain" jsonschema:"Name of the domain to list"`
}

type NeighborParams = rest.Neighbors

type GoalParams = rest.Goals

const (
	ListDomains          = "list_domains"
	ListDomainClasses    = "list_domain_classes"
	CreateGoalsGraph     = "create_goals_graph"
	CreateNeighborsGraph = "create_neighbors_graph"
)

type Server struct {
	*mcp.Server
	Engine *engine.Engine
}

func NewServer(e *engine.Engine) *Server {
	timeout := func(handler mcp.MethodHandler[*mcp.ServerSession]) mcp.MethodHandler[*mcp.ServerSession] {
		return func(ctx context.Context, ss *mcp.ServerSession, method string, params mcp.Params) (result mcp.Result, err error) {
			ctx, cancel := e.WithTimeout(ctx, 0)
			defer cancel()
			return handler(ctx, ss, method, params)
		}
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "korrle8r", Title: "Korrle8r MCP Server", Version: build.Version}, nil)
	addTools(e, s)
	s.AddReceivingMiddleware(logger, timeout)
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
		Name: CreateNeighborsGraph,
		Description: `
Returns a JSON graph of correlated objects.
From a set of start objects, follow correlation rules to find related objects up to the specified depth.`,
	},
		func(ctx context.Context, ss *mcp.ServerSession, p *mcp.CallToolParamsFor[NeighborParams]) (*mcp.CallToolResultFor[rest.Graph], error) {
			args := p.Arguments
			start, err := rest.TraverseStart(e, args.Start)
			if err != nil {
				return errorResultFor[rest.Graph](err), nil
			}
			g, err := traverse.Neighbors(ctx, e, start, args.Depth)
			if err != nil {
				return errorResultFor[rest.Graph](err), nil
			}
			return structuredResult(*rest.NewGraph(g, nil)), nil
		})

	mcp.AddTool(s, &mcp.Tool{
		Name: CreateGoalsGraph,
		Description: `
Returns a JSON graph of correlated objects.
From a set of start objects, follow all paths leading to one of the goal classes.`,
	},
		func(ctx context.Context, ss *mcp.ServerSession, p *mcp.CallToolParamsFor[GoalParams]) (*mcp.CallToolResultFor[rest.Graph], error) {
			args := p.Arguments
			start, err := rest.TraverseStart(e, args.Start)
			if err != nil {
				return errorResultFor[rest.Graph](err), nil
			}
			goals, err := e.Classes(args.Goals)
			if err != nil {
				return errorResultFor[rest.Graph](err), nil
			}
			g, err := traverse.Goals(ctx, e, start, goals)
			if err != nil {
				return errorResultFor[rest.Graph](err), nil
			}
			return structuredResult(*rest.NewGraph(g, nil)), nil
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

func errorResultFor[T any](err error) *mcp.CallToolResultFor[T] {
	return &mcp.CallToolResultFor[T]{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(`error: %v`, err.Error())}},
		IsError: true,
	}
}

var errorResult = errorResultFor[any]

// logger is middleware to do debug logging of MCP methods
func logger[S mcp.Session](handler mcp.MethodHandler[S]) mcp.MethodHandler[S] {
	return func(ctx context.Context, s S, method string, params mcp.Params) (result mcp.Result, err error) {
		log := logging.Log()
		if log.V(3).Enabled() {
			start := time.Now()
			defer func() {
				latency := time.Since(start)
				log.V(3).Info("MCP method",
					"method", method,
					"params", params,
					"result", result,
					"latency", latency)
			}()
		}
		return handler(ctx, s, method, params)
	}
}

func structuredResult[T any](value T) *mcp.CallToolResultFor[T] {
	// Also return result as JSON text content for older clients.
	text, _ := json.Marshal(value)
	return &mcp.CallToolResultFor[T]{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(text)}},
		StructuredContent: value,
	}
}
