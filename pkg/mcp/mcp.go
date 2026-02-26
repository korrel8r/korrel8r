// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package mcp Provides an MCP server and argument structures for MCP client calls.
package mcp

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	StreamablePath = "/mcp"
	SSEPath        = "/mcp/sse"
)

type ListDomainsResult struct {
	Domains []rest.Domain `json:"domains" jsonschema:"List of domains"`
}

type DomainParams struct {
	Domain string `json:"domain" jsonschema:"Name of the domain to list"`
}

type ListDomainClassesResult struct {
	Domain  string   `json:"domain" jsonschema:"Domain name"`
	Classes []string `json:"classes" jsonschema:"List of classes in the domain"`
}

type NeighborParams = rest.Neighbors
type GoalParams = rest.Goals
type ShowInConsoleParams = rest.Console

const (
	ListDomains          = "list_domains"
	ListDomainClasses    = "list_domain_classes"
	CreateGoalsGraph     = "create_goals_graph"
	CreateNeighborsGraph = "create_neighbors_graph"
	GetConsole           = "get_console"
	ShowInConsole        = "show_in_console"
)

type Server struct {
	*mcp.Server
	Engine *engine.Engine
	API    *rest.API
}

// NewServer creates a new MCP server.
// If api is not nil, session operations are enabled in conjunction with the REST API.
func NewServer(e *engine.Engine, api *rest.API) *Server {
	timeout := func(handler mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			ctx, cancel := e.WithTimeout(ctx, 0)
			defer cancel()
			return handler(ctx, method, req)
		}
	}
	s := &Server{
		Server: mcp.NewServer(&mcp.Implementation{Name: "korrel8r", Title: "Korrel8r MCP Server", Version: build.Version}, nil),
		Engine: e,
		API:    api,
	}
	s.addTools(e, api)
	s.AddReceivingMiddleware(logger, timeout)
	return s
}

func (s *Server) addTools(e *engine.Engine, api *rest.API) {

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: ListDomains,
		Description: `
Returns a list of Korrel8r domains with descriptions.
A domain contains observable signals or resources that use the same data model,
storage technology, and query syntax.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (_ *mcp.CallToolResult, out ListDomainsResult, err error) {
			return nil, ListDomainsResult{Domains: rest.ListDomains(e)}, nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: ListDomainClasses,
		Description: `
List the classes in a domain.
A domain contains one or more classes, representing objects with different structures.
Some domains have only a single class, others (like the 'k8s' domain) have many.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input DomainParams) (*mcp.CallToolResult, *ListDomainClassesResult, error) {
			d, err := e.Domain(input.Domain)
			if err != nil {
				return nil, nil, err
			}
			out := &ListDomainClassesResult{Domain: d.Name()}
			for _, c := range d.Classes() {
				out.Classes = append(out.Classes, c.Name())
			}
			return nil, out, nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: CreateNeighborsGraph,
		Description: `
Returns a JSON graph of correlated objects.
From a set of start objects, follow correlation rules to find related objects up to the specified depth.`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input NeighborParams) (*mcp.CallToolResult, *rest.Graph, error) {
			start, err := rest.TraverseStart(e, input.Start)
			if err != nil {
				return nil, nil, err
			}
			g, err := traverse.Neighbors(ctx, e, start, input.Depth)
			if err != nil {
				return nil, nil, err
			}
			return nil, rest.NewGraph(g, nil), nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: CreateGoalsGraph,
		Description: `
Returns a JSON graph of correlated objects.
From a set of start objects, follow all paths leading to one of the goal classes.`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input GoalParams) (*mcp.CallToolResult, *rest.Graph, error) {
			start, err := rest.TraverseStart(e, input.Start)
			if err != nil {
				return nil, nil, err
			}
			goals, err := e.Classes(input.Goals)
			if err != nil {
				return nil, nil, err
			}
			g, err := traverse.Goals(ctx, e, start, goals)
			if err != nil {
				return nil, nil, err
			}
			return nil, rest.NewGraph(g, nil), nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: GetConsole,
		Description: `
Returns the current state of the console display, representing what the user is currently looking at.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, *rest.Console, error) {
			if err := s.apiCheck(); err != nil {
				return nil, nil, err
			}
			return nil, api.ConsoleState.Get(), nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: ShowInConsole,
		Description: `
Send updated display parameters to the user's console, to allow the user to visualize the data in
a rich graphical environment.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input ShowInConsoleParams) (*mcp.CallToolResult, any, error) {
			if err := s.apiCheck(); err != nil {
				return nil, nil, err
			}
			if err := rest.ConsoleOK(e, &input); err != nil {
				return nil, nil, err
			}
			if err := api.ConsoleState.Send(&input); err != nil {
				return nil, nil, err
			}
			return nil, nil, nil
		})
}

// ServeStdio runs an MCP server, it returns when the client disconnects or the context is canceled.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.Run(ctx, &mcp.StdioTransport{})
}

// HTTPHandler  a handler for the HTTP Streamable MCP protocol.
func (s *Server) HTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(s.handler, &mcp.StreamableHTTPOptions{})
}

// SSEHandler returns a handler for the SSE MCP protocol.
func (s *Server) SSEHandler() http.Handler {
	return mcp.NewSSEHandler(s.handler, &mcp.SSEOptions{})
}

// handler returns the same server for all requests. Server and Engine are concurrent-safe.
func (s *Server) handler(*http.Request) *mcp.Server { return s.Server }

func (s *Server) apiCheck() error {
	if s.API == nil {
		return errors.New("not connected to console")
	}
	return nil
}

// logger is middleware to do debug logging of MCP methods
func logger(handler mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
		log := logging.Log()
		if log.V(3).Enabled() {
			start := time.Now()
			defer func() {
				latency := time.Since(start)
				log = log.WithValues(
					"method", method,
					"latency", latency,
					"params", req.GetParams())
				if err != nil {
					log = log.WithValues("error", err)
				} else {
					log = log.WithValues("result", result)
				}
				log.V(3).Info("MCP")
			}()
		}
		return handler(ctx, method, req)
	}
}
