// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package api implements a REST API for korrel8r.
//
// Endpoints expect JSON bodies and return JSON values.
// Some endpoints accept both GET and POST.
// POST versions allow more options in the request body.
//
// # API Base path
//
// REST API paths are prefixed with
//
//	/api/v1alpha1
//
// # GET /domains
//
// List of korrel8r domain names.
//   - Response: [Domains]
//
// # GET /stores
//
// List of all store configurations objects.
//   - Response: [Stores]
//
// # GET /stores/DOMAIN
//
// List of all store configurations objects for DOMAIN.
//   - Response: [Stores]
//
// # GET /goals?start=DOMAIN+CLASS&query=QUERY&goal=DOMAIN+CLASS
//
// Execute 'query' to get objects of class 'start'.
// Search from start objects, return related queries for the goal class.
// The 'goal' parameter can be repeated for multiple goals,
// or missing to get the start object instead.
//   - Response: [Results]
//
// # POST /goals
//
// Like 'GET /goals' with parameters in the request body. Allows additional parameters.
//   - Request: [GoalsRequest]
//   - Response: [Results]
//
// # GET /graphs?start=DOMAIN+CLASS&query=QUERY&goal=DOMAIN+CLASS
//
// Like 'GET /goals' but returns the complete search graph.
//   - Response: [Graph]
//
// # POST /graphs
//
// Like 'GET /graphs' but with parameters in the request body. Allows additional parameters.
//
//   - Request: [GoalsRequest]
//   - Response: [Graph]
//
// # GET /neighbours?start=DOMAIN+CLASS&query=QUERY&depth=DEPTH
//
// Execute 'query' to get objects of class 'start'.
// Returns a graph of queries for neighbours, up to a distance of 'depth'.
//   - Response: [Graph]
//
// # POST /neighbours
//
// Like 'GET /neighbours' but with parameters in the request body. Allows additional parameters.
//
//   - Request: [NeighboursRequest]
//   - Response: [Graph]
package api

import (
	"encoding/json"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Domains is a list of domain names.
type Domains []string

// Stores is a list of store configurations.
type Stores []korrel8r.StoreConfig

// Class identifies a class in a domain.
type Class struct {
	Domain string `json:"domain"`
	Class  string `json:"class"`
}

// QueryCounts maps query strings to a result count.
type QueryCounts map[string]int

// Result provides queries and result counts for a given class.
type Result struct {
	Class   Class       `json:"class"`                 // Class of the result.
	Queries QueryCounts `json:"queryCounts,omitempty"` // QueryCounts maps query strings to result counts.
}

// Results of correlation
type Results []Result

// Start is a starting point for correlation.
type Start struct {
	Start   Class             `json:"start"`             // Class of starting objects
	Query   string            `json:"query,omitempty"`   // Query for starting objects (optional)
	Objects []json.RawMessage `json:"objects,omitempty"` // Objects lists JSON-serialized starting objects (optional)
}

// GoalsRequest body
type GoalsRequest struct {
	Start Start   `json:",inline"`         // Start of correlation search.
	Goals []Class `json:"goals,omitempty"` // Goal class for correlation.
}

// Graph is a directed graph of results.
type Graph struct {
	Nodes []Result // Result nodes.
	Edges [][2]int // Edges are (from, to) indices into Nodes.
}

// NeighboursRequest requests a neighbours graph
type NeighboursRequest struct {
	Start Start `json:",inline"` // Start of correlation search.
	Depth int   `json:"depth"`   // Max depth of neighbours graph.
}
