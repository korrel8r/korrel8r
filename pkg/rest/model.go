// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"encoding/json"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Array is a slice that serializes to JSON as '[]' not 'null' for a nil value.
type Array[T any] []T

func (a Array[T]) MarshalJSON() ([]byte, error) {
	if a == nil {
		return json.Marshal([]T{})
	}
	return json.Marshal([]T(a))
}

// NOTE: Swag oddity - fields with struct or enum types will use the @description of the type.
// They should normally NOT have a doc comment of their own or the generated schema will have a strange `allOf`
// wrapper to preserve the multiple comments.

// @description Store is a map of name:value attributes used to connect to a store.
type Store = config.Store // @name Store

// @description Constraint constrains the objects that will be included in search results.
type Constraint = korrel8r.Constraint // @name Constraint

// @description Domain configuration information.
type Domain struct {
	// Name of the domain.
	Name string `json:"name"`
	// Stores configured for the domain.
	Stores Array[Store] `json:"stores,omitempty"`
} // @name Domain

// @description Classes is a map from class names to a short description.
type Classes map[string]string // @name Classes

// Note: use json.RawMessage for and objects because we don't know the type of these values
// until the engine resolves the class name as a Class value.

// @description Start identifies a set of starting objects for correlation.
// The starting object set includes:
// - results from getting each of the [Start.Queries]
// - unmarshalled objects from [Start.Objects]
type Start struct {
	Queries    Array[string]          `json:"queries,omitempty"`                      // Queries for starting objects
	Class      string            `json:"class,omitempty"`                        // Class for `objects`
	Objects    Array[json.RawMessage] `json:"objects,omitempty" swaggertype:"object"` // Objects of `class` serialized as JSON
	Constraint *Constraint       `json:"constraint,omitempty"`
} // @name Start

// @description	Starting point for a goals search.
type Goals struct {
	Start Start    `json:"start"`
	Goals Array[string] `json:"goals,omitempty" example:"domain:class"` // Goal classes for correlation.
} // @name Goals

// @description	Starting point for a neighbours search.
type Neighbours struct {
	Start Start `json:"start"`
	Depth int   `json:"depth"` // Max depth of neighbours graph.

} // @name Neighbours

// Options control the format of the graph
type Options struct {
	Rules bool `form:"rules"` // Rules if true include rules in the graph edges.
} // @name GraphOptions

// Objects requests objects corresponding to a query.
type Objects struct {
	Query      string      `form:"query"` // Query string to execute.
	Constraint *Constraint `json:"constraint,omitempty"`
} // @name Objects

// @description Query run during a correlation with a count of results found.
type QueryCount struct {
	Query string `json:"query"` // Query for correlation data.
	Count int    `json:"count"` // Count of results or -1 if the query was not executed.
} // @name QueryCount

// Rule is a correlation rule with a list of queries and results counts found during navigation.
// Rules form a directed multi-graph over classes in the result graph.
type Rule struct {
	// Name is an optional descriptive name.
	Name string `json:"name,omitempty"`
	// Queries generated while following this rule.
	Queries Array[QueryCount] `json:"queries,omitempty"`
} // @name Rule

// Node in the result graph, contains results for a single class.
type Node struct {
	// Class is the full class name in "DOMAIN:CLASS" form.
	Class string `json:"class" example:"domain:class"`
	// Queries yielding results for this class.
	Queries Array[QueryCount] `json:"queries,omitempty"`
	// Count of results found for this class, after de-duplication.
	Count int `json:"count"`
} // @name Node

// @description Directed edge in the result graph, from Start to Goal classes.
type Edge struct {
	// Start is the class name of the start node.
	Start string `json:"start"`
	// Goal is the class name of the goal node.
	Goal string `json:"goal" example:"domain:class"`
	// Rules is the set of rules followed along this edge.
	Rules Array[Rule] `json:"rules,omitempty" extensions:"x-omitempty"`
} // @name Edge

// @description	Graph resulting from a correlation search.
type Graph struct {
	Nodes Array[Node] `json:"nodes"`
	Edges Array[Edge] `json:"edges,omitempty"`
} // @name Graph
