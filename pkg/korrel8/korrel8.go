// package korrel8 Cross-domain correlation for k8s clusters and hetrogeneous observability signals.
//
// Types and interfaces to create correlation rules and correlate objects between different domains.
// First some definitions:
//
//  - object: a set of attributes derived from an observable signal (e.g. log record or metric time-series) or a resource (e.g. k8s resource)
//  - Domain: a set of objects with a common representation and naming conventions. (e.g. prometheus metrics, openshift logs, k8s resources, OpenTracing spans)
//  - Class: a subset of objects in a domain (e.g. k8s Pod, prometheus Alert)
//  - Rule: a function taking a context object and returning a query for related goal objects.
//  - Store: a repository of objects belonging to the same domain (e.g. a Loki log store, k8s API server)
//
// Signals and resources from different domains have different representations and naming conventions.
// Domain-specific packages implement the interfaces here so we do cross-domain correlation.
//
package korrel8

// Object can be any Go value. Domain-specific packages manipulate their own object types.
type Object = interface{}

// Domain names a set of objects based on the same technology.
// Examples of domains are: "log" (openshift log records), "prometheus" (metrics and alerts), "k8s" (kubernetes resources)
type Domain string

// Class identifies a subset of objects from the same domain with the same schema.
// For example Pod is a class in the k8s domain.
//
// Class implementations must be comparable.
type Class interface {
	Contains(Object) bool // Contains is true if the instance belongs to the class.
	String() string       // String returns the class name.
}

// Rule encapsulates logic to find correlated objects from an initial context object.
//
// Rule.Follow() does not return correlated objects directly, it returns a Query to be executed
// on a Store of objects in the Goal domain. For example a k8s GET URL or a PromQL query string.
//
// Rule implementations must be comparable.
type Rule interface {
	Context() Class              // Class of initial context object
	Goal() Class                 // Class of desired result
	Follow(context Object) Query // Function to transform context to query for results
	String() string              // Name of the rule
}

// Query is a query string, format depends on the store to be queried.
type Query string

// Store is a source of signals belonging to a single domain.
type Store interface {
	// Execute a query, return the resulting objects.
	Execute(query Query) ([]Object, error)
}

// Correlator holds a collection of Rules and can apply them to correlate objects.
type Correlator struct {
	rules       map[Rule]struct{}
	rulesByGoal map[Class][]Rule
}

// New creates a Correlator containing some rules.
func New(rules ...Rule) *Correlator {
	c := &Correlator{rules: map[Rule]struct{}{}, rulesByGoal: map[Class][]Rule{}}
	for _, r := range rules {
		c.Add(r)
	}
	return c
}

// Add adds a new rule. Idempotent, it is safe to add the same rule twice.
func (c *Correlator) Add(r Rule) {
	if _, ok := c.rules[r]; !ok {
		c.rules[r] = struct{}{}
		c.rulesByGoal[r.Goal()] = append(c.rulesByGoal[r.Goal()], r)
	}
}

// Correlate finds a path in the Rule graph from Context to Goal and follows it to produce a Query.
//
// Correlate be called in multiple goroutines concurrently.
// It cannot be called concurrently with Add.
func (c *Correlator) Correlate(context, goal Class, instance Object) (Query, error) {
	panic("FIXME")
	// Call findPaths then execute the functions on the paths, and collect the results.
}

// findPaths finds paths from context to goal.
//
// Rules form a directed cyclic graph, with Class nodes and Rule edges.
// Work backwards from the goal to find chains of rules that start at context.
func (c *Correlator) findPaths(context, goal Class) [][]Rule {
	state := correlation{
		rulesByGoal: c.rulesByGoal,
		visited:     map[Rule]bool{},
	}
	state.dfs(goal, context)
	return state.paths
	// FIXME efficiency - eliminate redundant paths, use better algorithms?
	// Other considerations - shortest paths? Weighted links or nodes?
}

// correlation holds state for a single correlation search
type correlation struct {
	rulesByGoal map[Class][]Rule
	visited     map[Rule]bool
	current     []Rule
	paths       [][]Rule
}

// dfs does depth first search for all simple edge paths from u to v,
// treating rules as directed links from Goal to Context.
func (c *correlation) dfs(goal, context Class) {
	for _, r := range c.rulesByGoal[goal] {
		if c.visited[r] { // Already used this rule.
			continue
		}
		c.visited[r] = true
		c.current = append([]Rule{r}, c.current...) // Add to chain
		// FIXME is class equality sufficient or do we need to consider IsA relationship?
		if r.Context() == context { // Path has arrived at the context
			c.paths = append(c.paths, c.current)
			c.visited[r] = false      // Allow r to be re-used in a different chain.
			c.current = c.current[1:] // Pop and continue search.
			continue
		}
		c.dfs(r.Context(), context) // Recursive search from r.Context
		c.current = c.current[1:]
		c.visited[r] = false // Allow r to be re-used in different path
	}
}
