// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package mcp provides an MCP server and argument structures for MCP client calls.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/korrel8r/korrel8r/pkg/auth"
	"github.com/korrel8r/korrel8r/pkg/session"
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

type GetDomainDocParams struct {
	Domain string `json:"domain,omitempty" jsonschema:"Name of the domain. If omitted, returns documentation for all domains."`
}

type GetDomainDocResult struct {
	Documentation string `json:"documentation" jsonschema:"Domain documentation including query syntax and examples"`
}

type ListDomainClassesResult struct {
	Domain  string   `json:"domain" jsonschema:"Domain name"`
	Classes []string `json:"classes" jsonschema:"List of classes in the domain"`
}

type NeighborParams = rest.Neighbors
type GoalParams = rest.Goals
type ShowInConsoleParams = rest.Console

const instructions = `
Korrel8r finds correlations between observability signals and resources in a Kubernetes cluster.
It connects data from different domains (logs, metrics, alerts, traces, Kubernetes resources, etc.)
by following correlation rules to build a graph of related objects.

## Workflow

1. Use list_domains to discover available domains.
2. Use get_domain_doc to get documentation about domains, and the syntax of their classes, queries.
3. Search for correlated data:
   - Use create_goals_graph when the user asks about a specific signal type
     (e.g. "find logs for this pod", "what alerts fired for this deployment?").
   - Use create_neighbors_graph for open-ended exploration
     (e.g. "what is related to this pod?", "show me everything connected to these traces").
4. If the user refers to a console, use get_console to find out what the user
   is looking at, and show_in_console to update their display with new results.
   Console tools return an error if no console is connected — use search tools instead in that case.

## Tool groups

- Search tools (list_domains, list_domain_classes, get_domain_doc, create_neighbors_graph, create_goals_graph):
  Discover domains and classes, and search for correlated signals.
- Console tools (get_console, show_in_console):
  Only work when connected to a graphical console (e.g. OpenShift web console).
  Bridge between the AI agent and the user's console display.
`

const (
	ListDomains          = "list_domains"
	ListDomainClasses    = "list_domain_classes"
	GetDomainDoc         = "get_domain_doc"
	CreateGoalsGraph     = "create_goals_graph"
	CreateNeighborsGraph = "create_neighbors_graph"
	// Console tools, only work in sessions with a connected console.
	GetConsole    = "get_console"
	ShowInConsole = "show_in_console"
)

type Server struct {
	*mcp.Server
	sessions session.Manager
}

// NewServer creates a new MCP server.
// Tool handlers resolve the session per-request using the auth token from the context.
func NewServer(sessions session.Manager) *Server {
	s := &Server{
		Server: mcp.NewServer(
			&mcp.Implementation{Name: "korrel8r", Title: "Korrel8r MCP Server", Version: build.Version},
			&mcp.ServerOptions{Instructions: instructions}),
		sessions: sessions,
	}
	s.addTools()
	timeout := func(handler mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, err
			}
			ctx, cancel := ss.Engine.WithTimeout(ctx, 0)
			defer cancel()
			return handler(ctx, method, req)
		}
	}
	s.AddReceivingMiddleware(s.logger, timeout)
	return s
}

func (s *Server) addTools() {
	mcp.AddTool(s.Server, &mcp.Tool{
		Name: ListDomains,
		Description: `
Returns a list of Korrel8r domains with descriptions.
A domain contains observable signals or resources that use the same query syntax and data store.
Use this first to discover available domains, then use list_domain_classes to explore a domain.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (_ *mcp.CallToolResult, out ListDomainsResult, err error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, ListDomainsResult{}, err
			}
			return nil, ListDomainsResult{Domains: rest.ListDomains(ss.Engine)}, nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: ListDomainClasses,
		Description: `
List the classes in a domain.
A class represents objects with a specific structure within a domain.
Some domains have a single class (e.g. metric:metric), others like k8s have many classes.
Use get_domain_doc to get more details about a domain and its classes and queries.

Class names are used in queries and as goal parameters. The full class name is "domain:class".
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input DomainParams) (*mcp.CallToolResult, *ListDomainClassesResult, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err
			}
			d, err := ss.Engine.Domain(input.Domain)
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
		Name: GetDomainDoc,
		Description: `
Get documentation for one or all domains, including class and query syntax.
Set the domain parameter to get documentation for a single domain.
Omit the domain parameter to get documentation for all domains.

Class strings have the form "domain:class", where the legal values of "class" depend on the domain.

Query strings have the form "domain:class:selector".
The "domain:class" part indicates the class of data returned by the query.
The "selector" part is a domain-specific query string.

Use this tool to learn how to construct valid class names and queries for a domain before using tools that have class or query parameters.
For example: create_neighbors_graph, create_goals_graph, get_console or show_in_console.

`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input GetDomainDocParams) (*mcp.CallToolResult, *GetDomainDocResult, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err
			}
			var b strings.Builder
			var domains []korrel8r.Domain
			if input.Domain != "" {
				d, err := ss.Engine.Domain(input.Domain)
				if err != nil {
					return nil, nil, err
				}
				domains = []korrel8r.Domain{d}
			} else {
				domains = ss.Engine.Domains()
			}
			for _, d := range domains {
				_, detail := d.Description()
				fmt.Fprintf(&b, "%s\n\n", detail)
			}
			return nil, &GetDomainDocResult{Documentation: b.String()}, nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: CreateNeighborsGraph,
		Description: `
Search for correlated observability signals and resources starting from known objects.
Follows correlation rules outward from the start objects up to the specified depth.

Returns a graph where nodes represent classes (each with queries and result counts)
and edges represent correlation rules that were applied.

Use this for open-ended exploration: "what is related to this pod?" or "what resources are related to these traces?"

The start parameter requires queries in the format "domain:class:selector".
Use get_domain_doc to learn the class and query syntax for each domain.
Depth controls how many correlation steps to follow (1 = direct correlations only).
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input NeighborParams) (*mcp.CallToolResult, *rest.Graph, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err
			}
			start, err := rest.TraverseStart(ss.Engine, input.Start)
			if err != nil {
				return nil, nil, err
			}
			g, err := traverse.Neighbors(ctx, ss.Engine, start, input.Depth)
			if err != nil {
				return nil, nil, err
			}
			return nil, rest.NewGraph(g, nil), nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: CreateGoalsGraph,
		Description: `
Search for correlations between start objects and specific goal classes.
Only follows paths from the start objects that lead to one of the specified goal classes.

Returns a graph where nodes represent classes (each with queries and result counts)
and edges represent correlation rules that were applied.

Use this for targeted investigation: "find logs related to this pod" or "what alerts fired for this deployment?"

The start parameter uses queries in "domain:class:selector" format.
Use get_domain_doc to learn the class and query syntax for each domain.
Goals are full class names, e.g. ["log:application"], ["alert:alert", "metric:metric"].

Example: to find logs for a crashing pod, use:
  start: {"queries": ["k8s:Pod:{\"namespace\":\"myapp\",\"name\":\"web-0\"}"]}
  goals: ["log:application"]
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input GoalParams) (*mcp.CallToolResult, *rest.Graph, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err
			}
			start, err := rest.TraverseStart(ss.Engine, input.Start)
			if err != nil {
				return nil, nil, err
			}
			goals, err := ss.Engine.Classes(input.Goals)
			if err != nil {
				return nil, nil, err
			}
			g, err := traverse.Goals(ctx, ss.Engine, start, goals)
			if err != nil {
				return nil, nil, err
			}
			return nil, rest.NewGraph(g, nil), nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: GetConsole,
		Description: `
Returns the current state of the user's graphical console (e.g. OpenShift web console).
The result includes:
- view: a query that selects the data displayed in the main console view. Not set if the console is displaying a view that does not support queries.
- search: parameters for the correlation search displayed in the troubleshooting panel. Not set if the troubleshooting panel is not open.

Use view and search to understand what the user is looking at,
and include it as context for further planning or actions.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, *rest.Console, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err
			}
			if ss.Console == nil {
				return nil, nil, errors.New("not connected to console")
			}
			c := &rest.Console{}
			if err := ss.Console.Get(c); err != nil {
				return nil, nil, err
			}
			return nil, c, nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: ShowInConsole,
		Description: `
Update the user's graphical console to display new data to the user.

- view: setting this field to a query updates the main view of the console to display the results of the query.
- search: setting this field displays a correlation graph in the console troubleshooting panel.

Use get_domain_doc to learn the class and query syntax for each domain.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input ShowInConsoleParams) (*mcp.CallToolResult, any, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err
			}
			if err := rest.ConsoleOK(ss.Engine, &input); err != nil {
				return nil, nil, err
			}
			if err := ss.Console.Send(&input); err != nil {
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
	return withAuthContext(mcp.NewStreamableHTTPHandler(s.handler, &mcp.StreamableHTTPOptions{}))
}

// SSEHandler returns a handler for the SSE MCP protocol.
func (s *Server) SSEHandler() http.Handler {
	return withAuthContext(mcp.NewSSEHandler(s.handler, &mcp.SSEOptions{}))
}

// withAuthContext wraps an HTTP handler to extract the Authorization header into the request context.
func withAuthContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(auth.Context(r)))
	})
}

func (s *Server) session(ctx context.Context) (*session.Session, error) {
	return s.sessions.Get(s.sessions.Key(ctx))
}

// handler returns the shared server for all requests.
// Per-session state is resolved per-request by tool handlers via sessions.Get.
func (s *Server) handler(*http.Request) *mcp.Server {
	return s.Server
}

// logger is middleware to do debug logging of MCP methods
func (s *Server) logger(handler mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, tool string, req mcp.Request) (result mcp.Result, err error) {
		start := time.Now()
		log := logging.Log()
		if log.V(3).Enabled() {
			defer func() {
				latency := time.Since(start)
				log = log.WithValues(
					"tool", tool,
					"latency", latency,
					"parameters", logging.JSON(req.GetParams()))
				if ss, err := s.session(ctx); err == nil {
					log = log.WithValues("session", ss.ID)
				}
				if err != nil {
					log.V(3).Info("MCP call failed", "error", err)
				} else {
					log.V(3).Info("MCP call OK", "result", logging.JSON(result))
				}
			}()
		}
		return handler(ctx, tool, req)
	}
}
