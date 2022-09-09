// package korrel8 correlates observable signals from different domains
//
// Generic types and interfaces to define correlation rules, and correlate objects between different domains.
//
// The main interfaces are:
//
// - Object: A set of attributes representing a signal (e.g. log record, metric time-series, ...)
// The concrete type depends on the domain, for correlation purposes it is equivalent to a JSON object.
//
// - Domain: a set of objects with a common vocabulary (e.g. k8s resources, OpenTracing spans, ...)
//
// - Class: a subset of objects in the same domain with a common schema (e.g. k8s Pod, prometheus Alert)
//
// - Rule: takes a starting object and returns a query for related goal objects.
//
// - Store: a store of objects belonging to the same domain (e.g. a Loki log store, k8s API server)
//
// Signals and resources from different domains have different representations and naming conventions.
// Domain-specific packages implement the interfaces in this package so we can do cross-domain correlation.
//
package korrel8

import (
	"context"
)

// Object represents a signal instance.
type Object interface {
	Identifier() Identifier // Identifies this object instance.
	Class() Class           // Class of the object.
}

// Domain names a set of objects based on the same technology.
type Domain string

// Identifier is a comparable value that identifies an "instance" of a signal.
//
// For example a namespace+name for a k8s resource, or a uri+labels for a metric time series.
type Identifier any

// Class identifies a subset of objects from the same domain with the same schema.
// For example Pod is a class in the k8s domain.
//
// Class implementations must be comparable.
type Class interface {
	Domain() Domain // Domain of this class.
}

// Rule encapsulates logic to find correlated goal objects from a start object.
//
type Rule interface {
	Start() Class                  // Class of start object
	Goal() Class                   // Class of desired result object(s)
	Follow(Object) (Result, error) // Follow the rule from the start object.
}

// Result holds a collection of queries for a single Domain.
// Query string format depends on the domain to be queried.
// For example a k8s GET URL or a PromQL query string.
type Result struct {
	Domain  Domain
	Queries []string
}

// Get the collection of objects returned by executing all queries against store.
// Results are de-duplicated based on Object.Identifier.
func (r Result) Get(ctx context.Context, s Store) ([]Object, error) {
	m := map[Identifier]Object{}
	for _, q := range r.Queries {
		objs, err := s.Query(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, o := range objs {
			m[o.Identifier()] = o // Keep only one object per Identifier
		}
	}
	var objs []Object
	for _, o := range m {
		objs = append(objs, o)
	}
	return objs, nil
}

// Store is a source of signals belonging to a single domain.
type Store interface {
	// Query a query, return the resulting objects.
	Query(ctx context.Context, query string) ([]Object, error)
}

// Rules holds a collection of Rules forming a start/goal directed graph.
type Rules struct {
	rules       []Rule
	rulesByGoal map[Class][]int // Index into rules so we have a comparable rule id.
}

// NewRules creates new RuleGraph containing some rules.
func NewRules(rules ...Rule) *Rules {
	c := &Rules{rulesByGoal: map[Class][]int{}}
	c.Add(rules...)
	return c
}

// Add new rules.
func (c *Rules) Add(rules ...Rule) {
	for _, r := range rules {
		c.rules = append(c.rules, r)
		i := len(c.rules) - 1 // Rule index
		c.rulesByGoal[r.Goal()] = append(c.rulesByGoal[r.Goal()], i)
	}
}

// Path is a list of rules where the Goal() of each rule is the Start() of the next.
type Path []Rule

// Follow rules in a path, using the stores map to determine the store for each Domain.
// Returns a result that may have more than one query.
func (p Path) Follow(stores map[Domain]Store) Result {
	panic("FIXME")
}

// Paths returns chains of rules leading from start to goal.
//
// Paths be called in multiple goroutines concurrently.
// It cannot be called concurrently with Add.
func (rs *Rules) Paths(start, goal Class) []Path {
	// Rules form a directed cyclic graph, with Class nodes and Rule edges.
	// Work backwards from the goal to find chains of rules from start.
	state := pathSearch{
		Rules:   rs,
		visited: map[int]bool{},
	}
	state.dfs(start, goal)
	return state.paths
}

// pathSearch holds state for a single path search
type pathSearch struct {
	*Rules
	visited map[int]bool
	current Path
	paths   []Path
}

// dfs does depth first search for all simple edge paths treating rules as directed links from goal to start.
//
// TODO efficiency - better algorithms?
// TODO shortest paths? Weighted links or nodes?
func (ps *pathSearch) dfs(start, goal Class) {
	for _, i := range ps.rulesByGoal[goal] {
		if ps.visited[i] { // Already used this rule.
			continue
		}
		r := ps.rules[i]
		ps.visited[i] = true
		ps.current = append([]Rule{r}, ps.current...) // Add to chain
		if r.Start() == start {                       // Path has arrived at the start
			ps.paths = append(ps.paths, ps.current)
			ps.visited[i] = false       // Allow r to be re-used in a different chain.
			ps.current = ps.current[1:] // Pop and continue search.
			continue
		}
		ps.dfs(start, r.Start()) // Recursive search from r.Start
		ps.current = ps.current[1:]
		ps.visited[i] = false // Allow r to be re-used in different path
	}
}
