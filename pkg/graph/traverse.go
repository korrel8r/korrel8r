package graph

import (
	"fmt"

	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/topo"
)

// Traverse follows each path in a path graph from start to goal, visiting the edges on the path.
//
// A path graph must have:
//
//   - exactly one node (the "start") with no incoming edges.
//   - exactly one node (the "goal") with no outgoing edges.
//   - at least one path from start to goal
//
// The graph is traversed in topological order as far as possible.
// Follows all shortest paths in a cycle, which may mean edges may be visited twice.
func (g *Graph) Traverse(visit func(Edge)) error {
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
			sub := g.subGraphOf(append(cycles[j], u, v))
			j++
			uSub, vSub := sub.NodeFor(ClassForNode(u)), sub.NodeFor(ClassForNode(v))
			paths, _ := path.DijkstraAllFrom(uSub, sub).AllTo(vSub.ID())
			for _, path := range paths {
				visitPath(sub, path, visit)
			}
			i++ // v is now processed, skip it.
		}
	}
	return nil
}
