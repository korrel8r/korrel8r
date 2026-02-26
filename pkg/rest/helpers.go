// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
)

var (
	Spec     = must.Must1(GetSwagger())
	BasePath = must.Must1(Spec.Servers.BasePath())
)

func queryCounts(gq graph.Queries, _ GraphOptions) []QueryCount {
	qcs := make([]QueryCount, 0, len(gq))
	for _, qc := range gq {
		if qc.Count <= 0 {
			continue // Omit query counts with no results
		}
		qcs = append(qcs, QueryCount{Query: qc.Query.String(), Count: &qc.Count})
	}
	slices.SortFunc(qcs, func(a, b QueryCount) int {
		if n := cmp.Compare(ptr.Deref(a.Count), ptr.Deref(b.Count)); n != 0 {
			return -n
		}
		return cmp.Compare(a.Query, b.Query)
	})
	return qcs
}

func rule(l *graph.Line, opts GraphOptions) (r Rule) {
	r.Name = l.Rule.Name()
	r.Queries = queryCounts(l.Queries, opts)
	return r
}

func node(n *graph.Node, opts GraphOptions) Node {
	node := Node{
		Class:   n.Class.String(),
		Queries: queryCounts(n.Queries, opts),
		Count:   ptr.To(len(n.Result.List())),
	}
	if ptr.Deref(opts.Results) {
		for _, o := range n.Result.List() {
			j, _ := json.Marshal(o)
			node.Result = append(node.Result, j)
		}
	}
	return node
}

func nodes(g *graph.Graph, opts GraphOptions) []Node {
	if g == nil {
		return nil
	}
	nodes := []Node{} // Want [] not null for empty in JSON.
	g.EachNode(func(n *graph.Node) {
		if !n.Empty() { // Skip empty nodes
			nodes = append(nodes, node(n, opts))
		}
	})
	return nodes
}

func edge(e *graph.Edge, opts GraphOptions) Edge {
	edge := Edge{
		Start: e.Start().Class.String(),
		Goal:  e.Goal().Class.String(),
	}
	if ptr.Deref(opts.Rules) {
		e.EachLine(func(l *graph.Line) {
			if l.Queries.Total() != 0 {
				edge.Rules = append(edge.Rules, rule(l, opts))
			}
		})
	}
	return edge
}

func edges(g *graph.Graph, opts GraphOptions) []Edge {
	var edges []Edge
	if g == nil {
		return nil
	}
	g.EachEdge(func(e *graph.Edge) {
		if !e.Goal().Empty() { // Skip edges that lead to an empty node.
			edges = append(edges, edge(e, opts))
		}
	})
	return edges
}

// Normalize API values by sorting slices in a predictable order.
// Useful for tests that need to compare actual and expected results.
func Normalize(v any) any {
	switch v := v.(type) {
	case *Graph:
		Normalize(v.Nodes)
		Normalize(v.Edges)
	case Graph:
		Normalize(&v)
	case []Node:
		slices.SortFunc(v, func(a, b Node) int { return strings.Compare(a.Class, b.Class) })
		for _, n := range v {
			Normalize(n)
		}
	case []Edge:
		slices.SortFunc(v, func(a, b Edge) int {
			if n := strings.Compare(a.Start, b.Start); n != 0 {
				return n
			} else {
				return strings.Compare(a.Goal, b.Goal)
			}
		})
		for _, e := range v {
			Normalize(e)
		}
	case Node:
		Normalize(v.Queries)
	case Edge:
		for _, r := range v.Rules {
			Normalize(r.Queries)
		}
	case []QueryCount:
		slices.SortFunc(v, func(a, b QueryCount) int { return strings.Compare(a.Query, b.Query) })
	}
	return v
}

// NewGraph returns a new rest.Graph corresponding to the internal graph.Graph.
func NewGraph(g *graph.Graph, optsPtr *GraphOptions) *Graph {
	if g == nil {
		return &Graph{}
	}
	opts := ptr.Deref(optsPtr)
	return &Graph{Nodes: nodes(g, opts), Edges: edges(g, opts)}
}

func copyBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	var buf = &bytes.Buffer{}
	_, _ = io.Copy(buf, r.Body) // Copy body data
	r.Body = io.NopCloser(buf)  // Put back the Body data.
	return buf.String()
}

// Response writer that collects the response body for logging.
type responseWriter struct {
	gin.ResponseWriter
	*bytes.Buffer
}

func newResponseWriter(rw gin.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: rw, Buffer: &bytes.Buffer{}}
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.Buffer.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w responseWriter) WriteString(s string) (int, error) {
	w.Buffer.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// TraverseStart constructs a traverse.Start from a rest.Start parameter.
func TraverseStart(e *engine.Engine, start Start) (traverse.Start, error) {
	var (
		class   korrel8r.Class
		queries []korrel8r.Query
	)
	if start.Class != "" {
		var err error
		class, err = e.Class(start.Class)
		if err != nil {
			return traverse.Start{}, err
		}
	}
	for _, q := range start.Queries {
		query, err := e.Query(q)
		if err != nil {
			return traverse.Start{}, err
		}
		if class == nil {
			class = query.Class() // Use class of first query
		}
		if query.Class() != class {
			return traverse.Start{}, fmt.Errorf("query class mismatch: expected class %v in query %v", class, q)
		}
		queries = append(queries, query)
	}
	if class == nil {
		return traverse.Start{}, fmt.Errorf("no class provided to start graph traversal")
	}
	var objects []korrel8r.Object
	for _, raw := range start.Objects {
		o, err := class.Unmarshal([]byte(raw))
		if err != nil {
			return traverse.Start{}, err
		}
		objects = append(objects, o)
	}

	return traverse.Start{
		Class:      class,
		Objects:    objects,
		Queries:    queries,
		Constraint: start.Constraint,
	}, nil
}

func ListDomains(e *engine.Engine) []Domain {
	var domains []Domain
	for _, d := range e.Domains() {
		var stores []Store
		for _, sc := range e.StoreConfigsFor(d) {
			if len(sc) > 0 {
				stores = append(stores, (Store)(sc))
			}
		}
		domains = append(domains, Domain{
			Name:        d.Name(),
			Description: d.Description(),
			Stores:      stores,
		})
	}
	return domains
}
