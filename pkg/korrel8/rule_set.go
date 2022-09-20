package korrel8

import (
	"fmt"
)

// RuleSet holds a collection of Rules forming a directed graph from start -> goal or vice versa.
type RuleSet struct {
	rules       []Rule
	rulesByGoal map[Class][]int // Index into rules so we have a comparable rule id.
}

// NewRuleSet creates new RuleGraph containing some rules.
func NewRuleSet(rules ...Rule) *RuleSet {
	c := &RuleSet{rulesByGoal: map[Class][]int{}}
	c.Add(rules...)
	return c
}

// Add new rules.
func (c *RuleSet) Add(rules ...Rule) {
	for _, r := range rules {
		c.rules = append(c.rules, r)
		i := len(c.rules) - 1 // Rule index
		c.rulesByGoal[r.Goal()] = append(c.rulesByGoal[r.Goal()], i)
	}
}

// FindPaths returns chains of rules leading from start to goal.
//
// FindPaths be called in multiple goroutines concurrently.
// It cannot be called concurrently with Add.
func (rs *RuleSet) FindPaths(start, goal Class) []Path {
	// Rules form a directed cyclic graph, with Class nodes and Rule edges.
	// Work backwards from the goal to find chains of rules from start.
	log.Info("finding paths", "start", start, "goal", goal)
	state := pathSearch{
		RuleSet: rs,
		visited: map[int]bool{},
	}
	state.dfs(start, goal)
	// FIXME String()
	var str []string
	for _, p := range state.paths {
		str = append(str, fmt.Sprintf("%v", p))
	}
	log.Info("found paths", "paths", str)
	return state.paths
}

// pathSearch holds state for a single path search
type pathSearch struct {
	*RuleSet
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
