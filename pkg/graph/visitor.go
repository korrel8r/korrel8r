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
	if startNode := g.NodeFor(start); startNode != nil {
		n, _ := bf.Walk(g, startNode, untilFunc).(*Node)
		return n
	}
	return nil
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
