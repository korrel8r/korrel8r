// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package mcp provides an MCP server that proxies to the korrel8r REST API.
package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/jsonschema-go/jsonschema"
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

const Instructions = `Korrel8r finds correlations between observability signals and resources in a Kubernetes cluster.
It connects data from different domains (logs, metrics, alerts, traces, Kubernetes resources, etc.)
by following correlation rules to build a graph of related objects.

If the user refers to a console, use get_console and show_in_console to read and update it.

To search: use list_domains to discover domains, then 'help' to learn query syntax.
Use create_goals_graph for targeted queries ("find logs for this pod")
and create_neighbors_graph for open-ended exploration ("what is related to this pod?").
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
	if t.InputSchema == nil {
		if s, err := jsonschema.ForType(reflect.TypeFor[In](), nil); err == nil {
			t.InputSchema = s
		}
	}
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
		Name:        ListDomains,
		Description: `List available domains with descriptions. A domain groups signals or resources that share a query syntax and data store. Use 'list_domain_classes' or 'help' to explore a domain further.`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (_ *mcp.CallToolResult, out ListDomainsResult, err error) {
			domains, err := client.ListDomains(ctx)
			if err != nil {
				return nil, ListDomainsResult{}, err
			}
			return nil, ListDomainsResult{Domains: domains}, nil
		})

	addTool(&tools, server, &mcp.Tool{
		Name:        ListDomainClasses,
		Description: `List classes in a domain. A class represents objects with a specific structure. Full class names have the form "domain:class" and are used in queries and as goal parameters. Use 'help' for details on class and query syntax.`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input DomainParams) (*mcp.CallToolResult, *ListDomainClassesResult, error) {
			classes, err := client.ListDomainClasses(ctx, input.Domain)
			if err != nil {
				return nil, nil, err
			}
			return nil, &ListDomainClassesResult{Domain: input.Domain, Classes: classes}, nil
		})

	addTool(&tools, server, &mcp.Tool{
		Name:        Help,
		Description: `Get help on domains, classes, and query syntax. Omit the domain parameter for help on all domains. Class names have the form "domain:class". Query strings have the form "domain:class:selector". Use this before constructing queries for other tools.`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input HelpParams) (*mcp.CallToolResult, *HelpResult, error) {
			doc, err := client.Help(ctx, input.Domain)
			if err != nil {
				return nil, nil, err
			}
			return nil, &HelpResult{Documentation: doc}, nil
		})

	addTool(&tools, server, &mcp.Tool{
		Name:        CreateNeighborsGraph,
		Description: `Follow correlation rules outward from start objects up to a given depth. Returns a graph of correlated classes with queries and result counts. Use for open-ended exploration like "what is related to this pod?" Depth 1 = direct correlations; depth 2-3 typically reaches logs, metrics, and alerts. Start queries use "domain:class:selector" format; see 'help' for syntax.`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input NeighborParams) (*mcp.CallToolResult, *api.Graph, error) {
			g, err := client.GraphNeighbors(ctx, input)
			if err != nil {
				return nil, nil, err
			}
			return nil, g, nil
		})

	addTool(&tools, server, &mcp.Tool{
		Name:        CreateGoalsGraph,
		Description: `Follow correlation paths from start objects to specific goal classes. Returns a graph of correlated classes with queries and result counts. Use for targeted queries like "find logs for this pod" or "what alerts fired for this deployment?" Start queries use "domain:class:selector" format; goals are class names like ["log:application"]. See 'help' for syntax.`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input GoalParams) (*mcp.CallToolResult, *api.Graph, error) {
			g, err := client.GraphGoals(ctx, input)
			if err != nil {
				return nil, nil, err
			}
			return nil, g, nil
		})

	addTool(&tools, server, &mcp.Tool{
		Name:        GetObjects,
		Description: `Execute a query and return matching objects as self-contained JSON (all labels/fields included per object). Query format is "domain:class:selector"; see 'help' for syntax. Use the constraint parameter (limit number of objects, start/end time as RFC 3339) to control result size, especially for high-volume domains like logs, metrics, and traces.`,
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
		Name:        GetConsole,
		Description: `Get what the user is looking at in the console. Returns a view query (main console view) and/or search parameters (troubleshooting panel), either may be absent. Use these as context for further actions.`,
	},
		func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, *api.Console, error) {
			console, err := client.GetConsole(ctx)
			if err != nil {
				return nil, nil, err
			}
			return nil, console, nil
		})

	addTool(&tools, server, &mcp.Tool{
		Name:        ShowInConsole,
		Description: `Update the console to display new data. Set 'view' to a query to update the main view, and/or set 'search' to display a correlation graph in the troubleshooting panel. See 'help' for query syntax.`,
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
