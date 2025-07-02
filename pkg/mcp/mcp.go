// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mcp

import (
	"context"
	"encoding/json"
	"io"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/korrel8r/korrel8r/pkg/text"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

var log = logging.Log()

// Tool arguments are just structs, annotated with jsonschema tags
// More at https://mcpgolang.com/tools#schema-generation

// FIXME Re-use json schema from openapi? Re-use structs?

type listDomains struct{}

type listDomainClasses struct {
	Domain string `json:"domain" jsonschema:"required,description=Name of the domain to list"`
}

// FIXME relate to REST API
// Missing constraint
type createNeighbourGraph struct {
	Depth int `json:"depth,omitempty" jsonschema:"description=Search depth, the maximum number of relationships to traverse when creatig the neighbourhood graph"`

	Queries []string `json:"queries,omitempty" jsonschema:"description=Array of fully-qualified query strings. All queries must be for the same class'. Example: k8s:Pod:{\"namespace\":\"default\"}"`
}

// register handlers with the MCP server
func register(e *engine.Engine, s *mcp.Server) {
	// Rergistering a handler fails if the handle function is invalid, panic in that case.
	must.Must(s.RegisterTool("list_domains",
		`Returns a list of Korrel8r domains.
A domain contains obeservable signals or resources that use the same data model,
storage technology, and query syntax.
`,
		func(req listDomains) (*mcp.ToolResponse, error) {
			request(req)
			text := text.WriteString(text.NewPrinter(e).ListDomains)
			return response(text, nil)
		}))

	must.Must(s.RegisterTool("list_domain_classes",
		`Returns a list of classes in a domain.
A domain contains one or more classes, representing objects with different structures.
Some domains have only a single class, others (like the 'k8s' domain) have many.
`,
		func(req listDomainClasses) (*mcp.ToolResponse, error) {
			request(req)
			d, err := e.Domain(req.Domain)
			if err != nil {
				return nil, err // FIXME should this be in the request?
			}
			text := text.WriteString(func(w io.Writer) {
				text.NewPrinter(e).ListClasses(w, d) // FIXME better ERROR HANDLING
			})
			return response(text, nil)
		}))

	must.Must(s.RegisterTool("create_neibourhood_graph",
		`Returns a JSON graph of correlated objects.
	From a set of start objects, follow correlation rules to find relaed objects up to a maximum distance from the start.`,
		func(req createNeighbourGraph) (*mcp.ToolResponse, error) {
			request(req)
			// FIXME lots of overlap with REST. Error handling
			var queries []korrel8r.Query
			for _, q := range req.Queries {
				query, err := e.Query(q)
				if err != nil {
					return nil, err // FIXME
				}
				queries = append(queries, query)
			}
			// FIXME error handling
			start := traverse.Start{Queries: queries, Class: queries[0].Class()}
			// FIXME request context?
			g, err := traverse.New(e, e.Graph()).Neighbours(context.Background(), start, req.Depth)
			data, _ := json.Marshal(rest.NewGraph(g, false))
			return response(string(data), err)
		}))

	// FIXME register prompts, resources etc.
}

// ServeStdio is a blocking function that listens on stdin and responds on stdout.
func ServeStdio(e *engine.Engine) (err error) {
	transport := stdio.NewStdioServerTransport()
	done := make(chan error, 1)
	transport.SetErrorHandler(func(err error) { done <- err })
	transport.SetCloseHandler(func() { close(done) })
	server := mcp.NewServer(transport)
	register(e, server)
	if err := server.Serve(); err != nil {
		return err
	}
	err = <-done
	return err
}

// GinHandler returns gin.HandlerFunc for the Streamaable MCP protocol.
func GinHandler(e *engine.Engine) (gin.HandlerFunc, error) {
	transport := http.NewGinTransport()
	server := mcp.NewServer(transport)
	register(e, server)
	return transport.Handler(), nil
}

func response(text string, err error) (*mcp.ToolResponse, error) {
	log.V(3).Info("MCP response", "text", text)
	return mcp.NewToolResponse(mcp.NewTextContent(text)), traverse.IgnorePartialError(err)
}

func request(v any) {
	log.V(3).Info("MCP Request", "name", reflect.TypeOf(v).Name(), "data", v)
}
