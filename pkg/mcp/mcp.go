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
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/korrel8r/korrel8r/pkg/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const StreamablePath = "/mcp"

type ListDomainsResult struct {
	Domains []api.Domain `json:"domains" jsonschema:"List of domains"`
}

type DomainParams struct {
	Domain string `json:"domain" jsonschema:"Name of the domain to list"`
}

type HelpParams struct {
	Domain string `json:"domain,omitempty" jsonschema:"If specified, get help for this domain only."`
}

type HelpResult struct {
	Documentation string `json:"documentation" jsonschema:"Domain documentation including query syntax and examples"`
}

type ListDomainClassesResult struct {
	Domain  string   `json:"domain" jsonschema:"Domain name"`
	Classes []string `json:"classes" jsonschema:"List of classes in the domain"`
}

type NeighborParams = api.Neighbors
type GoalParams = api.Goals
type ShowInConsoleParams = api.Console

type ObjectsParams struct {
	Query      string          `json:"query" jsonschema:"Query string in the form 'domain:class:selector'. Use 'help' to learn query syntax for each domain."`
	Constraint *api.Constraint `json:"constraint,omitempty" jsonschema:"Optional constraint to limit results by time range and/or count."`
}

type ObjectsResult struct {
	Objects []any `json:"objects" jsonschema:"List of objects matching the query"`
}

const instructions = `
Korrel8r finds correlations between observability signals and resources in a Kubernetes cluster.
It connects data from different domains (logs, metrics, alerts, traces, Kubernetes resources, etc.)
by following correlation rules to build a graph of related objects.

## Search tools

1. Use list_domains to discover available domains.
2. Use 'help' to get examples of classes and query syntax for a domain, or for all domains.
3. Search for correlated data:
   - Use create_goals_graph when the user asks about a specific signal type
     (e.g. "find logs for this pod", "what alerts fired for this deployment?").
   - Use create_neighbors_graph for open-ended exploration
     (e.g. "what is related to this pod?", "show me everything connected to these traces").

## Console tools

The user may have a graphical console that displays cluster data.
If the user refers to a console:
- Use get_console to find out what data the user is looking at, in the form of a korrel8r query.
- Use show_in_console to display results in the console. Express the results as a korrel8r query.

Console tools return an error if no console is connected, you can still use search tools.
`

const (
	Help                 = "help"
	ListDomains          = "list_domains"
	ListDomainClasses    = "list_domain_classes"
	CreateGoalsGraph     = "create_goals_graph"
	CreateNeighborsGraph = "create_neighbors_graph"
	GetObjects           = "get_objects"
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
	s.AddReceivingMiddleware(s.logger)
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
Use 'help' to get more details about a domain and its classes and queries.

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
		Name: Help,
		Description: `
Get help about korrel8r domains, classes, and query syntax.
Omitting the domain parameter returns help about all domains.

Class strings have the form "domain:class", where the legal values of "class" depend on the domain.

Query strings have the form "domain:class:selector".
The "domain:class" part indicates the class of data returned by the query.
The "selector" part is a domain-specific query string.

Use this tool to learn how to construct valid class names and queries for a domain before using tools that have class or query parameters.
For example: create_neighbors_graph, create_goals_graph, get_console or show_in_console.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input HelpParams) (*mcp.CallToolResult, *HelpResult, error) {
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
				fmt.Fprintf(&b, "%s\n\n", d.Description())
			}
			return nil, &HelpResult{Documentation: b.String()}, nil
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
Use 'help' to learn the class and query syntax for each domain.
Depth controls how many correlation steps to follow (1 = direct correlations only).
Higher depths cast a wider net: depth 1 finds directly correlated objects,
depth 2-3 typically reaches related signals like logs, metrics, and alerts.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input NeighborParams) (*mcp.CallToolResult, *api.Graph, error) {
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
Use 'help' to learn the class and query syntax for each domain.
Goals are full class names, e.g. ["log:application"], ["alert:alert", "metric:metric"].

Example: to find logs for a crashing pod, use:
  start: {"queries": ["k8s:Pod:{\"namespace\":\"myapp\",\"name\":\"web-0\"}"]}
  goals: ["log:application"]
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input GoalParams) (*mcp.CallToolResult, *api.Graph, error) {
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
		Name: GetObjects,
		Description: `
Execute a query and return matching objects as complete JSON.
The query must be in the format "domain:class:selector".
Use 'help' to learn the query syntax for each domain.

The returned objects are self-contained: all relevant labels and fields are included in each object.
This differs from direct back-end APIs (e.g. Loki, Tempo) which use a compact "stream" format
where common labels are sent once per stream. The complete format is more verbose
but each object can be processed independently.

Use the optional constraint parameter to control result size:
- limit: maximum number of objects to return.
- start/end: time range (RFC 3339) to restrict results by timestamp.
Use constraints to avoid excessively large results, especially for
high-volume domains like logs, metrics, and traces.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input ObjectsParams) (*mcp.CallToolResult, *ObjectsResult, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err
			}
			query, err := ss.Engine.Query(input.Query)
			if err != nil {
				return nil, nil, err
			}
			var constraint *korrel8r.Constraint
			if c := input.Constraint; c != nil {
				constraint = &korrel8r.Constraint{Limit: c.Limit, Start: c.Start, End: c.End}
			}
			r := result.New(query.Class())
			if err := ss.Engine.Get(ctx, query, constraint, r); err != nil {
				return nil, nil, err
			}
			objects := []any(r.List())
			if objects == nil {
				objects = []any{}
			}
			return nil, &ObjectsResult{Objects: objects}, nil
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
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, *api.Console, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err
			}
			state := ss.ConsoleState.Get()
			if state == nil {
				return nil, nil, errors.New("not connected to console")
			}
			return nil, state, nil
		})

	mcp.AddTool(s.Server, &mcp.Tool{
		Name: ShowInConsole,
		Description: `
Update the user's graphical console to display new data to the user.

- view: setting this field to a query updates the main view of the console to display the results of the query.
- search: setting this field displays a correlation graph in the console troubleshooting panel.

Use 'help' to learn the class and query syntax for each domain.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input ShowInConsoleParams) (*mcp.CallToolResult, any, error) {
			ss, err := s.session(ctx)
			if err != nil {
				return nil, nil, err // This is an internal error
			}
			if err := rest.ConsoleOK(ss.Engine, &input); err != nil {
				return errorResult(err), nil, err
			}
			ss.ConsoleRequest.Set(&input)
			return nil, nil, nil
		})
}

// ServeStdio runs an MCP server, it returns when the client disconnects or the context is canceled.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.Run(ctx, &mcp.StdioTransport{})
}

// HTTPHandler returns a handler for the HTTP Streamable MCP protocol.
func (s *Server) HTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(s.handler, &mcp.StreamableHTTPOptions{})
}

func (s *Server) session(ctx context.Context) (*session.Session, error) {
	// First check for session already set on context by transport.
	if s := session.FromContext(ctx); s != nil {
		return s, nil
	}
	return s.sessions.Get(ctx)
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
				if sn, err := s.session(ctx); err == nil {
					log = log.WithValues("session", sn.ID)
				}
				if err != nil {
					log.V(3).Info("MCP call failed", "error", err)
				} else {
					log.V(3).Info("MCP call", "result", logging.JSON(result))
				}
			}()
		}
		return handler(ctx, tool, req)
	}
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		IsError: true,
	}
}
