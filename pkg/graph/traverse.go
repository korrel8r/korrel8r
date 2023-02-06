package graph

import (
	"fmt"

	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/topo"
)

// Traverse a path graph from u to v, visiting each line. The graph must satisfy:
//
//   - exactly one node (the "start") has no incoming edges.
//   - exactly one node (the "goal") has no outgoing edges.
//   - there is at least one path from start to goal
//   - all nodes in the graph are on a path from start to goal.
//
// The graph is traversed in topological order where possible.
// It follows all shortest paths to traverse a cycle, which may mean rules are traversed twice.
func Traverse(g *Graph, visit func(multi.Edge)) error {
	sorted, err := topo.Sort(g)
	cycles, ok := err.(topo.Unorderable)
	if err != nil && !ok {
		return err
	}
	if len(sorted) == 0 {
		return fmt.Errorf("emtpy path graph")
	}
	start, goal := sorted[0], sorted[len(sorted)-1]
	if start == nil || goal == nil {
		return fmt.Errorf("invalid path graph")
	}

	j := 0 // Cycles index
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != nil { // In topo order, process all inbound edges
			visitTo(g, sorted[i], visit)
		} else {
			// Follow all shortest paths through a cycle.
			// Note a cycle is never the first or last element in sorted path.
			u, v := sorted[i-1], sorted[i+1]
			sub := g.SubGraph(append(cycles[j], u, v))
			j++
			uSub, vSub := sub.NodeForClass(ClassForNode(u)), sub.NodeForClass(ClassForNode(v))
			paths, _ := path.DijkstraAllFrom(uSub, sub).AllTo(vSub.ID())
			for _, path := range paths {
				visitPath(sub, path, visit)
			}
			i++ // v is now processed, skip it.
		}
	}
	return nil
}
