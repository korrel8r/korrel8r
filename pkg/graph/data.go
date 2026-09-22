// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"slices"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gonum.org/v1/gonum/graph/multi"
)

// Data is the shared, immutable graph of all rules.
// It assigns immutable integer IDs to nodes (classes), rules, and lines,
// and provides fast, allocation-free iteration via an adjacency list.
//
// Mutable [Node] and [Line] values are created and managed separately to hold
// results and queries.
// A [View] provides a fast read-only Gonum graph of the Data.
type Data struct {
	classes  []korrel8r.Class // Immutable class table, indexed by node ID.
	nodeID   map[korrel8r.Class]int64
	rules    []korrel8r.Rule // One reference per input rule, not per expanded line.
	topology topology
}

// NewData builds a rule graph definition. Input order determines node and line
// IDs; parallel lines, self-loops and repeated input rules are preserved.
func NewData(rules ...korrel8r.Rule) *Data {
	d := &Data{nodeID: make(map[korrel8r.Class]int64), rules: slices.Clone(rules)}
	var lines []topologyLine
	for ruleID, r := range rules {
		starts, goals := r.Start(), r.Goal()
		for _, start := range starts {
			for _, goal := range goals {
				from, to := d.classID(start), d.classID(goal)
				lines = append(lines, topologyLine{
					from: checkedTopologyID(int(from), "nodes"),
					to:   checkedTopologyID(int(to), "nodes"),
					rule: checkedTopologyID(ruleID, "rules"),
				})
			}
		}
	}
	d.topology = newTopology(len(d.classes), lines, d.ruleWeight)
	return d
}

// ruleWeight is a rule's edge weight: its spread, the number of goal classes.
func (d *Data) ruleWeight(ruleID uint32) uint32 {
	return checkedTopologyID(len(d.rules[ruleID].Goal()), "goals")
}

// EdgeWeight returns the least weight of any line from->to, and whether the edge exists.
func (d *Data) EdgeWeight(from, to int64) (float64, bool) {
	w, ok := d.topology.edges.find(from, to)
	return float64(w), ok
}

// classID returns the existing class ID or assigns a new one.
func (d *Data) classID(c korrel8r.Class) int64 {
	if id, ok := d.NodeID(c); ok {
		return id
	}
	id := int64(len(d.classes))
	d.classes = append(d.classes, c)
	d.nodeID[c] = id
	return id
}

// NodeID returns the class's node ID and whether it exists in the topology.
func (d *Data) NodeID(c korrel8r.Class) (int64, bool) { id, ok := d.nodeID[c]; return id, ok }

// NodeCount returns the number of topology classes.
func (d *Data) NodeCount() int { return len(d.classes) }

// Class returns the class associated with a node ID. Invalid IDs panic.
func (d *Data) Class(id int64) korrel8r.Class { return d.classes[id] }

// NewNode creates a search-owned or graph-owned mutable node for a class identity.
// Each call creates independent state; Data retains no reference to the node.
func (d *Data) NewNode(id int64) *Node {
	return (&Node{Node: multi.Node(id), Class: d.Class(id)}).Copy()
}

// LineCount returns the number of expanded topology lines.
func (d *Data) LineCount() int { return len(d.topology.lines) }

// Endpoints returns the start and goal node IDs of a line. Invalid IDs panic.
func (d *Data) Endpoints(id int) (int64, int64) {
	l := d.topology.lines[id]
	return int64(l.from), int64(l.to)
}

// RuleForLine returns the rule associated with a line. Invalid IDs panic.
func (d *Data) RuleForLine(id int) korrel8r.Rule { return d.rules[int(d.topology.lines[id].rule)] }

// EachLineIDFrom visits outgoing line IDs in creation order without allocating.
// Invalid node IDs are treated as empty adjacency lists.
func (d *Data) EachLineIDFrom(node int64, visit func(int)) { d.topology.out.each(node, visit) }

// EachLineIDTo visits incoming line IDs in creation order without allocating.
// Invalid node IDs are treated as empty adjacency lists.
func (d *Data) EachLineIDTo(node int64, visit func(int)) { d.topology.in.each(node, visit) }

// NewLine creates independently owned mutable line state between explicit
// search/graph-owned nodes. Endpoint IDs must match the topology line or it panics.
// Retain the returned pointer to accumulate Queries; each call creates fresh state.
func (d *Data) NewLine(id int, from, to *Node) *Line {
	l := d.topology.lines[id]
	if from == nil || to == nil || from.ID() != int64(l.from) || to.ID() != int64(l.to) {
		panic("graph: line endpoints do not match topology")
	}
	return &Line{Line: multi.Line{F: from, T: to, UID: int64(id)}, Rule: d.rules[int(l.rule)], Queries: Queries{}, Attrs: Attrs{}}
}

// Rules returns a copy of the input rule list, in input order. Each input entry
// appears once, regardless of how many topology lines it generates. Rules with
// empty start/goal sets and explicitly repeated input entries are preserved.
func (d *Data) Rules() []korrel8r.Rule { return slices.Clone(d.rules) }

// Classes returns the classes in node-ID order, in an independent slice.
func (d *Data) Classes() []korrel8r.Class { return slices.Clone(d.classes) }
