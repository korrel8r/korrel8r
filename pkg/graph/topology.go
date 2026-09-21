// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

// topology stores connectivity only: no class/rule interfaces, Gonum objects or
// mutable results. Data resolves the integer IDs to classes and rules.
// A line's index in lines is its stable identity. Parallel lines and self-loops
// remain distinct records, in creation order. IDs are uint32 to reduce retained
// memory; Data converts them to the public ID types at its boundaries.
type topology struct {
	lines   []topologyLine
	out, in adjacency
}

type topologyLine struct {
	from, to uint32 // Node IDs: indexes into Data.classes.
	rule     uint32 // Rule ID: index into Data.rules, shared by all expansions of a rule.
}

// adjacency is a compressed sparse row index, not another set of line records.
// Node n's line IDs occupy lines[offsets[n]:offsets[n+1]], in creation order.
// Offsets has node-count + 1 entries, including the final end offset.
type adjacency struct{ offsets, lines []uint32 }

// newTopology takes ownership of lines and builds both adjacency indexes.
// The caller must not modify lines afterwards. Node endpoints must be in [0,nodes).
func newTopology(nodes int, lines []topologyLine) topology {
	checkedTopologyID(nodes, "nodes")
	checkedTopologyID(len(lines), "lines")
	return topology{
		lines: lines,
		out:   newAdjacency(nodes, lines, false),
		in:    newAdjacency(nodes, lines, true),
	}
}

func newAdjacency(nodes int, lines []topologyLine, incoming bool) adjacency {
	a := adjacency{offsets: make([]uint32, nodes+1), lines: make([]uint32, len(lines))}
	endpoint := func(l topologyLine) uint32 {
		if incoming {
			return l.to
		}
		return l.from
	}
	for _, l := range lines {
		a.offsets[endpoint(l)+1]++
	}
	for i := 1; i < len(a.offsets); i++ {
		a.offsets[i] += a.offsets[i-1]
	}
	next := append([]uint32(nil), a.offsets[:nodes]...)
	for id, l := range lines {
		n := endpoint(l)
		a.lines[next[n]] = uint32(id)
		next[n]++
	}
	return a
}

func (a adjacency) each(node int64, visit func(int)) {
	if node < 0 || node >= int64(len(a.offsets)-1) {
		return
	}
	for _, id := range a.lines[a.offsets[node]:a.offsets[node+1]] {
		visit(int(id))
	}
}

func checkedTopologyID(id int, kind string) uint32 {
	if uint64(id) > uint64(^uint32(0)) {
		panic("graph: too many topology " + kind)
	}
	return uint32(id)
}
