// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/traverse"
)

// Visitor used to traverse a graph.
type Visitor interface {
	// Node is called once for each node visited.
	Node(n *Node)

	// Line is called once for each line traversed.
	// Return false to indicate the line should not be traversed.
	// If [Line] returns false for all lines leading to a node, that node will not be visited.
	Line(*Line) bool
}

type FuncVisitor struct {
	NodeF func(n *Node)
	LineF func(*Line) bool
}

func (v FuncVisitor) Node(n *Node) {
	if v.NodeF != nil {
		v.NodeF(n)
	}
}

func (v FuncVisitor) Line(l *Line) bool { return v.LineF == nil || v.LineF(l) }

type CollectVisitor struct {
	Visitor Visitor // The visitor to apply
	Graph   *Graph  // The collected graph.
}

func (v *CollectVisitor) Node(n *Node) {
	v.Visitor.Node(n)
	if v.Graph.Node(n.ID()) == nil {
		v.Graph.AddNode(n)
	}
}

func (v *CollectVisitor) Line(l *Line) bool {
	ok := v.Visitor.Line(l)
	if ok {
		v.Graph.SetLine(l)
	}
	return ok
}

// BreadthFirst traversal visiting each node in g that is reachable from start.
// If until(n) returns true, the traversal stops and n is returned. Otherwise nil is returned.
func (g *Graph) BreadthFirst(start korrel8r.Class, v Visitor, until func(*Node, int) bool) *Node {
	bf := traverse.BreadthFirst{
		Traverse: traverseFunc(g, v),
		Visit:    visitFunc(v),
	}
	var untilFunc func(n graph.Node, d int) bool
	if until != nil {
		untilFunc = func(n graph.Node, d int) bool { return until(n.(*Node), d) }
	}
	n, _ := bf.Walk(g, g.NodeFor(start), untilFunc).(*Node)
	return n
}

// DepthFirst traversal visiting each node in g that is reachable from start in depth first order.
// If until(n) returns true, the traversal stops and n is returned. Otherwise nil is returned.
func (g *Graph) DepthFirst(start korrel8r.Class, v Visitor, until func(*Node) bool) *Node {
	df := traverse.DepthFirst{
		Traverse: traverseFunc(g, v),
		Visit:    visitFunc(v),
	}
	var untilFunc func(n graph.Node) bool
	if until != nil {
		untilFunc = func(n graph.Node) bool { return until(n.(*Node)) }
	}
	n, _ := df.Walk(g, g.NodeFor(start), untilFunc).(*Node)
	return n
}

// GoalSearch traverses the shortest paths from start to all goals, in breadth first order.
func (g *Graph) GoalSearch(start korrel8r.Class, goals []korrel8r.Class, v Visitor) {
	g.ShortestPaths(start, goals...).BreadthFirst(start, v, nil)
}

// Neighbours traverses a breadth-first neighbourhood of start up to depth.
// It never follows lines to already nodes of lower depth, so the traversal is acyclic.
func (g *Graph) Neighbours(start korrel8r.Class, depth int, v Visitor) {
	ranked := map[int64]int{} // Depths of nodes already ranked.
	line := func(l *Line) bool {
		from := ranked[l.From().ID()]
		if from >= depth {
			return false // Don't exceed depth.
		}
		if to, seen := ranked[l.To().ID()]; !seen {
			ranked[l.To().ID()] = from + 1
		} else if to <= from {
			return false // Lower rank
		}
		return v.Line(l)
	}
	until := func(n *Node, d int) bool {
		ranked[n.ID()] = d
		return d > depth
	}
	g.BreadthFirst(start, FuncVisitor{LineF: line, NodeF: v.Node}, until)
}

func visitFunc(v Visitor) func(graph.Node) { return func(n graph.Node) { v.Node(n.(*Node)) } }

func traverseFunc(g *Graph, v Visitor) func(graph.Edge) bool {
	return func(e graph.Edge) bool {
		start, goal := e.From().(*Node), e.To().(*Node)
		ok := false
		g.EachLineBetween(start, goal, func(l *Line) { ok = v.Line(l) || ok })
		return ok
	}
}

// TODO: can we simplify by reducing this layer and use topo.graph functions more directly?
