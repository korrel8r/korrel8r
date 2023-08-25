// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package api

import (
	"encoding/json"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// FIXME move to korrel8r
func classname(c korrel8r.Class) string {
	return strings.Join([]string{c.String(), c.Domain().String()}, ".")
}

//	@description	Starting point for correlation.
type Start struct {
	// Class of starting objects
	Class string `json:"class" example:"class.domain"`
	// Queries for starting objects
	Queries []string `json:"query,omitempty"`
	// Objects in JSON form
	Objects []json.RawMessage `json:"objects,omitempty" swaggertype:"object"`
}

//	@description	Starting point for a goals search.
type GoalsRequest struct {
	Start Start    `json:"start"`                                  // Start of correlation search.
	Goals []string `json:"goals,omitempty" example:"class.domain"` // Goal classes for correlation.
}

//	@description	Starting point for a neighbours search.
type NeighboursRequest struct {
	Start Start `json:"start"` // Start of correlation search.
	Depth int   `json:"depth"` // Max depth of neighbours graph.
}

// Options control the format of the graph
type Options struct {
	// WithRules if true include rules in the graph edges.
	WithRules bool `form:"withRules"`
}

//	@description	A set of query strings with counts of results found by the query.
//	@description	Value of -1 means the query was not run so result count is unknown.
//	@example		queryString:10
type Queries map[string]int

// Rule is a correlation rule with a list of queries and results counts found during navigation.
// Rules form a directed multi-graph over classes in the result graph.
type Rule struct {
	// Name is an optional descriptive name.
	Name string `json:"name,omitempty"`
	// Queries generated while following this rule.
	Queries Queries `json:"queries,omitempty" example:"querystring:10"`
}

// Node in the result graph, contains results for a single class.
type Node struct {
	// Class is the full name of the class in "CLASS.DOMAIN" form.
	Class string `json:"class" example:"class.domain"`
	// Queries yielding results for this class.
	Queries Queries `json:"queries,omitempty" example:"querystring:10"`
	// Count of results found for this class, after de-duplication.
	Count int
}

// Directed edge in the result graph, from Start to Goal classes.
type Edge struct {
	// Start is the class name of the start node.
	Start string `json:"start"`
	// Goal is the class name of the goal node.
	Goal string `json:"goal" example:"class.domain"`
	// Rules is the set of rules followed along this edge (optional).
	Rules []Rule `json:"rules,omitempty"`
}

//	@description	Graph resulting from a correlation search.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges,omitempty"`
}

func rule(l *graph.Line) (r Rule) {
	r.Name = l.Rule.String()
	r.Queries = queries(l.QueryCounts)
	return r
}

func queries(g graph.QueryCounts) Queries {
	ret := Queries{}
	for qs, qc := range g {
		if qc.Count != 0 { // No value in returning queries that are known to yield no results.
			ret[qs] = qc.Count
		}
	}
	if len(ret) == 0 {
		ret = nil
	}
	return ret
}

func newGraph(g *graph.Graph, opts *Options) *Graph {
	res := Graph{Nodes: []Node{}} // Want to show [] for empty nodes, not null
	g.EachNode(func(n *graph.Node) { res.Nodes = append(res.Nodes, node(n)) })
	g.EachEdge(func(e *graph.Edge) { res.Edges = append(res.Edges, edge(e, opts.WithRules)) })
	return &res
}

func node(n *graph.Node) Node {
	return Node{Class: classname(n.Class), Queries: queries(n.QueryCounts), Count: len(n.Result.List())}
}

func edge(e *graph.Edge, withRules bool) Edge {
	edge := Edge{
		Start: classname(e.Start().Class),
		Goal:  classname(e.Goal().Class),
	}
	if withRules {
		e.EachLine(func(l *graph.Line) {
			edge.Rules = append(edge.Rules, rule(l))
		})
	}
	return edge
}

// StoreConfig contains configuration parameters for a store.
type StoreConfig map[string]string
