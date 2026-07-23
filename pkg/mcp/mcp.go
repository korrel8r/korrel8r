// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package mcp provides an MCP server that proxies to the korrel8r REST API.
package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/korrel8r/korrel8r/pkg/api"
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

const Instructions = `
Korrel8r finds correlations between observability signals and resources in a Kubernetes cluster.
It connects data from different domains (logs, metrics, alerts, traces, Kubernetes resources, etc.)
by following correlation rules to build a graph of related objects.

## Console tools

If the user refers to a console:
- Use get_console to find out what data the user is looking at, in the form of a korrel8r query.
- Use show_in_console to display results in the console. Express the results as a korrel8r query.

## Search tools

1. Use list_domains to discover available domains.
2. Use 'help' to get examples of classes and query syntax for a domain, or for all domains.
3. Search for correlated data:
   - Use create_goals_graph when the user asks about a specific signal type
     (e.g. "find logs for this pod", "what alerts fired for this deployment?").
   - Use create_neighbors_graph for open-ended exploration
     (e.g. "what is related to this pod?", "show me everything connected to these traces").

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
	client *Client
	log    logr.Logger
	tools  []*mcp.Tool
}

func (s *Server) AllTools() []*mcp.Tool { return s.tools }

// NewServer creates a new MCP server that proxies to a korrel8r REST API via the given client.
func NewServer(client *Client, version string, log logr.Logger) *Server {
	s := &Server{
		Server: mcp.NewServer(
			&mcp.Implementation{Name: "korrel8r", Title: "Korrel8r MCP Server", Version: version},
			&mcp.ServerOptions{
				Instructions: Instructions,
			}),
		client: client,
		log:    log,
	}
	s.tools = AddTools(s.Server, s.client)
	s.AddReceivingMiddleware(s.logger)
	return s
}

func addTool[In, Out any](tools *[]*mcp.Tool, server *mcp.Server, t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	*tools = append(*tools, t)
	if server != nil {
		mcp.AddTool(server, t, h)
	}
}

// AddTools adds korrel8r tools to server using client, and returns the list of tools added.
// If server is nil, returns the tool list without registering them.
func AddTools(server *mcp.Server, client *Client) []*mcp.Tool {
	var tools []*mcp.Tool

	addTool(&tools, server, &mcp.Tool{
		Name: ListDomains,
		Description: `
Returns a list of Korrel8r domains with descriptions.
A domain contains observable signals or resources that use the same query syntax and data store.
Use this first to discover available domains, then use list_domain_classes to explore a domain.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (_ *mcp.CallToolResult, out ListDomainsResult, err error) {
			domains, err := client.ListDomains(ctx)
			if err != nil {
				return nil, ListDomainsResult{}, err
			}
			return nil, ListDomainsResult{Domains: domains}, nil
		})

	addTool(&tools, server, &mcp.Tool{
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
			classes, err := client.ListDomainClasses(ctx, input.Domain)
			if err != nil {
				return nil, nil, err
			}
			return nil, &ListDomainClassesResult{Domain: input.Domain, Classes: classes}, nil
		})

	addTool(&tools, server, &mcp.Tool{
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
			doc, err := client.Help(ctx, input.Domain)
			if err != nil {
				return nil, nil, err
			}
			return nil, &HelpResult{Documentation: doc}, nil
		})

	addTool(&tools, server, &mcp.Tool{
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
			g, err := client.GraphNeighbors(ctx, input)
			if err != nil {
				return nil, nil, err
			}
			return nil, g, nil
		})

	addTool(&tools, server, &mcp.Tool{
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
			g, err := client.GraphGoals(ctx, input)
			if err != nil {
				return nil, nil, err
			}
			return nil, g, nil
		})

	addTool(&tools, server, &mcp.Tool{
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
			raw, err := client.GetObjects(ctx, input.Query, input.Constraint)
			if err != nil {
				return nil, nil, err
			}
			objects := make([]any, len(raw))
			for i, r := range raw {
				var v any
				if err := json.Unmarshal(r, &v); err != nil {
					return nil, nil, err
				}
				objects[i] = v
			}
			return nil, &ObjectsResult{Objects: objects}, nil
		})

	addTool(&tools, server, &mcp.Tool{
		Name: GetConsole,
		Description: `
If the user refers to a console, use this tool to find out what the user is looking at.
The result includes:
- view: a korrel8r query selecting data displayed in the main console view.
  Not set if the console is not displaying data.
- search: parameters for the correlation search displayed in the troubleshooting panel.
  Not set if the troubleshooting panel is not open.

Use view and search to understand what the user is looking at,
and include it as context for further planning or actions.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, *api.Console, error) {
			console, err := client.GetConsole(ctx)
			if err != nil {
				return nil, nil, err
			}
			return nil, console, nil
		})

	addTool(&tools, server, &mcp.Tool{
		Name: ShowInConsole,
		Description: `
If the user refers to a console, use this tool to update the console to display new data.

- view: setting this field to a query updates the main view of the console to display the results of the query.
- search: setting this field displays a correlation graph in the console troubleshooting panel.

Use 'help' to learn the class and query syntax for each domain.
`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, update ShowInConsoleParams) (*mcp.CallToolResult, any, error) {
			if err := client.ShowInConsole(ctx, &update); err != nil {
				return nil, nil, err
			}
			return nil, nil, nil
		})

	return tools
}

// ServeStdio runs an MCP server, it returns when the client disconnects or the context is canceled.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.Run(ctx, &mcp.StdioTransport{})
}

// HTTPHandler returns a handler for the HTTP Streamable MCP protocol.
func (s *Server) HTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(s.handler, &mcp.StreamableHTTPOptions{})
}

// handler returns the shared server for all requests.
func (s *Server) handler(*http.Request) *mcp.Server {
	return s.Server
}

// jsonValue wraps a value for JSON rendering in log output.
type jsonValue struct{ v any }

func (j jsonValue) MarshalLog() any {
	b, err := json.Marshal(j.v)
	if err != nil {
		return err.Error()
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return string(b)
	}
	return v
}

// logger is middleware to do debug logging of MCP methods
func (s *Server) logger(handler mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, tool string, req mcp.Request) (result mcp.Result, err error) {
		if s.log.V(3).Enabled() {
			start := time.Now()
			detail := s.log.V(9).Enabled()

			common := []any{"tool", tool, "parameters", jsonValue{req.GetParams()}}
			s.log.V(3).Info("MCP Request", common...)

			defer func() {
				values := append(common, "latency", time.Since(start))
				if err != nil {
					values = append(values, "error", err)
				} else if r, ok := result.(*mcp.CallToolResult); ok && r.IsError {
					values = append(values, "error", jsonValue{result})
				} else if detail {
					values = append(values, "result", jsonValue{result})
				}
				s.log.V(3).Info("MCP Response", values...)
			}()
		}
		return handler(ctx, tool, req)
	}
}
