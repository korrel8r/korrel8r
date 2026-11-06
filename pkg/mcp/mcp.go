// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package mcp provides an MCP server that proxies to the korrel8r REST API.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
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

type InstructionsResult struct {
	Instructions string `json:"instructions" jsonschema:"Korrel8r MCP server instructions"`
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

If the user refers to a console, use 'get_console' and 'show_in_console' to read and update it.
If the user asks "what am I looking at?", "what is this?" or says "show me ..." or "display ..." they may be referring to the console.

To search: use 'list_domains' to discover domains, then 'help' to learn query syntax.
Use 'create_goals_graph' for targeted queries ("find logs for this pod")
and 'create_neighbors_graph' for open-ended exploration ("what is related to this pod?").
`

const (
	InstructionsTool     = "instructions"
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
	log logr.Logger
}

// NewServer creates a new MCP server that proxies to a korrel8r REST API via the given client.
func NewServer(client *Client, version string, log logr.Logger) *Server {
	s := &Server{
		Server: mcp.NewServer(
			&mcp.Implementation{Name: "korrel8r", Title: "Korrel8r MCP Server", Version: version},
			&mcp.ServerOptions{
				Instructions: Instructions,
			}),
		log: log,
	}
	for _, name := range ToolNames {
		binding, _ := BindTool(client, name)
		s.AddTool(binding.Tool, binding.Handler)
	}
	s.AddReceivingMiddleware(s.logger)
	return s
}

// ToolNames lists all Korrel8r tools in server registration order.
var ToolNames = []string{
	InstructionsTool, ListDomains, ListDomainClasses, Help, CreateNeighborsGraph,
	CreateGoalsGraph, GetObjects, GetConsole, ShowInConsole,
}

// schemaFor returns the JSON schema for type T, treating a pointer like its element type.
func schemaFor[T any]() *jsonschema.Schema {
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	s, err := jsonschema.ForType(t, &jsonschema.ForOptions{TypeSchemas: map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[json.RawMessage](): {}, // Raw JSON is not a byte array.
	}})
	if err != nil {
		panic(err) // Tool types are fixed at compile time, a failure here is a bug.
	}
	return s
}

// newTool returns a binding for tool t, implemented by f.
// Schemas are derived from the In and Out types, arguments are validated against the input schema.
// An Out type of 'any' means the tool has no output schema or structured content.
func newTool[In, Out any](t *mcp.Tool, f func(context.Context, In) (Out, error)) Binding {
	if t.Annotations == nil {
		t.Annotations = &mcp.ToolAnnotations{ReadOnlyHint: true}
	}
	inputSchema := schemaFor[In]()
	t.InputSchema = inputSchema
	resolved, err := inputSchema.Resolve(nil)
	if err != nil {
		panic(err) // Schemas come from Go types, a failure here is a bug.
	}
	structured := reflect.TypeFor[Out]() != reflect.TypeFor[any]()
	var outputSchema *jsonschema.Resolved
	if structured {
		schema := schemaFor[Out]()
		t.OutputSchema = schema
		outputSchema, err = schema.Resolve(nil)
		if err != nil {
			panic(err) // Schemas come from Go types, a failure here is a bug.
		}
	}
	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw := req.Params.Arguments
		if len(raw) == 0 {
			raw = json.RawMessage(`{}`) // Absent arguments are an empty object.
		}
		args := map[string]any{}
		if err := json.Unmarshal(raw, &args); err != nil {
			return toolError(err), nil
		}
		if err := resolved.Validate(args); err != nil {
			return toolError(fmt.Errorf("invalid arguments: %w", err)), nil
		}
		var input In
		if err := json.Unmarshal(raw, &input); err != nil {
			return toolError(err), nil
		}
		out, err := f(ctx, input)
		if err != nil {
			return toolError(err), nil
		}
		result := &mcp.CallToolResult{Content: []mcp.Content{}} // Empty, not nil, as on the wire.
		if structured {
			b, err := json.Marshal(out)
			if err != nil {
				return nil, fmt.Errorf("marshal %v result: %w", t.Name, err)
			}
			var value any
			if err := json.Unmarshal(b, &value); err != nil {
				return nil, fmt.Errorf("decode %v result: %w", t.Name, err)
			}
			if err := outputSchema.Validate(value); err != nil {
				return nil, fmt.Errorf("invalid %v result: %w", t.Name, err)
			}
			result.StructuredContent = out
			result.Content = []mcp.Content{&mcp.TextContent{Text: string(b)}}
		}
		return result, nil
	}
	return Binding{Tool: t, Handler: handler}
}

func toolError(err error) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	result.SetError(err)
	return result
}

// Binding pairs a tool definition with its callable handler.
// Construct bindings with the client for the current caller.
type Binding struct {
	Tool    *mcp.Tool
	Handler mcp.ToolHandler
}

// BindTool returns one tool and its handler bound to client.
// A nil client can be used to inspect metadata, but the handler cannot be called.
func BindTool(client *Client, name string) (Binding, bool) {
	switch name {
	case InstructionsTool:
		return newTool(&mcp.Tool{
			Name:        InstructionsTool,
			Description: `Return the Korrel8r MCP server instructions. Use this tool if your client does not expose or use the instructions provided during MCP initialization; it returns the same guidance for using Korrel8r tools.`,
		},
			func(context.Context, struct{}) (InstructionsResult, error) {
				return InstructionsResult{Instructions: Instructions}, nil
			}), true

	case ListDomains:
		return newTool(&mcp.Tool{
			Name:        ListDomains,
			Description: `List available domains with descriptions. A domain groups signals or resources that share a query syntax and data store. Use 'list_domain_classes' or 'help' to explore a domain further.`,
		},
			func(ctx context.Context, _ struct{}) (ListDomainsResult, error) {
				domains, err := client.ListDomains(ctx)
				if domains == nil {
					domains = []api.Domain{}
				}
				return ListDomainsResult{Domains: domains}, err
			}), true

	case ListDomainClasses:
		return newTool(&mcp.Tool{
			Name:        ListDomainClasses,
			Description: `List classes in a domain. A class represents objects with a specific structure. Full class names have the form "domain:class" and are used in queries and as goal parameters. Use 'help' for details on class and query syntax.`,
		},
			func(ctx context.Context, input DomainParams) (*ListDomainClassesResult, error) {
				classes, err := client.ListDomainClasses(ctx, input.Domain)
				if err != nil {
					return nil, err
				}
				if classes == nil {
					classes = []string{}
				}
				return &ListDomainClassesResult{Domain: input.Domain, Classes: classes}, nil
			}), true

	case Help:
		return newTool(&mcp.Tool{
			Name:        Help,
			Description: `Get help on domains, classes, and query syntax. Omit the domain parameter for help on all domains. Class names have the form "domain:class". Query strings have the form "domain:class:selector". Use this before constructing queries for other tools.`,
		},
			func(ctx context.Context, input HelpParams) (*HelpResult, error) {
				doc, err := client.Help(ctx, input.Domain)
				if err != nil {
					return nil, err
				}
				return &HelpResult{Documentation: doc}, nil
			}), true

	case CreateNeighborsGraph:
		return newTool(&mcp.Tool{
			Name:        CreateNeighborsGraph,
			Description: `Follow correlation rules outward from start objects up to a given depth. Returns a graph of correlated classes with queries and result counts. Use for open-ended exploration like "what is related to this pod?" Depth 1 = direct correlations; depth 2-3 typically reaches logs, metrics, and alerts. Start queries use "domain:class:selector" format; see 'help' for syntax.`,
		},
			func(ctx context.Context, input NeighborParams) (*api.Graph, error) {
				return client.GraphNeighbors(ctx, input)
			}), true

	case CreateGoalsGraph:
		return newTool(&mcp.Tool{
			Name:        CreateGoalsGraph,
			Description: `Follow correlation paths from start objects to specific goal classes. Returns a graph of correlated classes with queries and result counts. Use for targeted queries like "find logs for this pod" or "what alerts fired for this deployment?" Start queries use "domain:class:selector" format; goals are class names like ["log:application"]. See 'help' for syntax.`,
		},
			func(ctx context.Context, input GoalParams) (*api.Graph, error) {
				return client.GraphGoals(ctx, input)
			}), true

	case GetObjects:
		return newTool(&mcp.Tool{
			Name:        GetObjects,
			Description: `Execute a query and return matching objects as self-contained JSON (all labels/fields included per object). Query format is "domain:class:selector"; see 'help' for syntax. Use the constraint parameter (limit number of objects, start/end time as RFC 3339) to control result size, especially for high-volume domains like logs, metrics, and traces.`,
		},
			func(ctx context.Context, input ObjectsParams) (*ObjectsResult, error) {
				raw, err := client.GetObjects(ctx, input.Query, input.Constraint)
				if err != nil {
					return nil, err
				}
				objects := make([]any, len(raw))
				for i, r := range raw {
					if err := json.Unmarshal(r, &objects[i]); err != nil {
						return nil, err
					}
				}
				return &ObjectsResult{Objects: objects}, nil
			}), true

	case GetConsole:
		return newTool(&mcp.Tool{
			Name:        GetConsole,
			Description: `Get what the user is looking at in the console. Returns a view query (main console view) and/or search parameters (troubleshooting panel), either may be absent. Use these as context for further actions.`,
		},
			func(ctx context.Context, _ struct{}) (*api.Console, error) {
				return client.GetConsole(ctx)
			}), true

	case ShowInConsole:
		return newTool(&mcp.Tool{
			Name: ShowInConsole,
			// Unlike the other tools, this one changes the state of the user's console.
			Annotations: &mcp.ToolAnnotations{DestructiveHint: func() *bool { v := true; return &v }()},
			Description: `Update the console to display new data. Set 'view' to a query to update the main view, and/or set 'search' to display a correlation graph in the troubleshooting panel. See 'help' for query syntax.`,
		},
			func(ctx context.Context, update ShowInConsoleParams) (any, error) {
				return nil, client.ShowInConsole(ctx, &update)
			}), true
	default:
		return Binding{}, false
	}
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
