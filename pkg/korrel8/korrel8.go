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

import "context"

// Object represents a signal instance.
//
// Domain-specific packages manipulate their own object types.
// Must support json.Marshal and Unmarshal.
type Object = any

// Domain names a set of objects based on the same technology.
type Domain string

// Class identifies a subset of objects from the same domain with the same schema.
// For example Pod is a class in the k8s domain.
//
// Class implementations must be comparable.
type Class interface {
	Contains(Object) bool // Contains returns true if the object belongs to the class.
}

// Rule encapsulates logic to find correlated goal objects from a start object.
// Rule.Follow() returns a Query to be executed on a Store of objects in the Goal domain.
// For example a k8s GET URL or a PromQL query string.
//
// Rule implementations must be comparable.
type Rule interface {
	Start() Class                 // Class of start object
	Goal() Class                  // Class of desired result object(s)
	Follow(Object) (Query, error) // Follow the rule from the start object
}

// FIXME association between rules and stores, need to provide Domain from Class.

// Query is a query string, format depends on the store to be queried.
type Query string

// Store is a source of signals belonging to a single domain.
type Store interface {
	// Execute a query, return the resulting objects.
	Execute(context.Context, Query) ([]Object, error)
}

// Rules holds a collection of Rules forming a start/goal directed graph.
type Rules struct {
	rules       map[Rule]struct{}
	rulesByGoal map[Class][]Rule
}

// NewRuleGraph creates new RuleGraph containing some rules.
func NewRuleGraph(rules ...Rule) *Rules {
	c := &Rules{rules: map[Rule]struct{}{}, rulesByGoal: map[Class][]Rule{}}
	c.Add(rules...)
	return c
}

// Add new rules.
// Idempotent, it is safe to add the same rule twice.
func (c *Rules) Add(rules ...Rule) {
	for _, r := range rules {
		if _, ok := c.rules[r]; !ok {
			c.rules[r] = struct{}{}
			c.rulesByGoal[r.Goal()] = append(c.rulesByGoal[r.Goal()], r)
		}
	}
}

// FIXME
// RulesWithGoal returns a list of rules with the given goal.
func (c *Rules) RulesWithGoal(goal Class) []Rule { return c.rulesByGoal[goal] }

// RulesWithStartAndGoal returns a list of rules with the given start and goal.
func (c *Rules) RulesWithStartAndGoal(start, goal Class) []Rule {
	var result []Rule
	for _, r := range c.RulesWithGoal(goal) {
		if r.Start() == start {
			result = append(result, r)
		}
	}
	return result
}

// Path is a list of rules where the Goal() of each rule is the Start() of the next.
type Path []Rule

func (p Path) Execute() {
	// FIXME execute a rule chain, need mapping from Goal classes to stores.
	panic("FIXME")
}

// Paths returns chains of rules leading from start to goal.
//
// Paths be called in multiple goroutines concurrently.
// It cannot be called concurrently with Add.
func (g *Rules) Paths(start, goal Class) []Path {
	// Rules form a directed cyclic graph, with Class nodes and Rule edges.
	// Work backwards from the goal to find chains of rules from start.
	state := pathSearch{
		rulesByGoal: g.rulesByGoal,
		visited:     map[Rule]bool{},
	}
	state.dfs(start, goal)
	return state.paths
}

// pathSearch holds state for a single path search
type pathSearch struct {
	rulesByGoal map[Class][]Rule
	visited     map[Rule]bool
	current     Path
	paths       []Path
}

// dfs does depth first search for all simple edge paths treating rules as directed links from goal to start.
//
// TODO efficiency - better algorithms?
// TODO shortest paths? Weighted links or nodes?
func (ps *pathSearch) dfs(start, goal Class) {
	for _, r := range ps.rulesByGoal[goal] {
		if ps.visited[r] { // Already used this rule.
			continue
		}
		ps.visited[r] = true
		ps.current = append([]Rule{r}, ps.current...) // Add to chain
		if r.Start() == start {                       // Path has arrived at the start
			ps.paths = append(ps.paths, ps.current)
			ps.visited[r] = false       // Allow r to be re-used in a different chain.
			ps.current = ps.current[1:] // Pop and continue search.
			continue
		}
		ps.dfs(start, r.Start()) // Recursive search from r.Start
		ps.current = ps.current[1:]
		ps.visited[r] = false // Allow r to be re-used in different path
	}
}
