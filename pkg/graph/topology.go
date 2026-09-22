// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"cmp"
	"slices"
)

// topology stores connectivity only: no class/rule interfaces, Gonum objects or
// mutable results. Data resolves the integer IDs to classes and rules.
// A line's index in lines is its stable identity. Parallel lines and self-loops
// remain distinct records, in creation order. IDs are uint32 to reduce retained
// memory; Data converts them to the public ID types at its boundaries.
type topology struct {
	lines   []topologyLine
	out, in adjacency
	edges   edgeIndex
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
// ruleWeight gives the weight of a rule ID, used to pre-compute edge weights.
func newTopology(nodes int, lines []topologyLine, ruleWeight func(uint32) uint32) topology {
	checkedTopologyID(nodes, "nodes")
	checkedTopologyID(len(lines), "lines")
	t := topology{
		lines: lines,
		out:   newAdjacency(nodes, lines, false),
		in:    newAdjacency(nodes, lines, true),
	}
	t.edges = newEdgeIndex(nodes, t.out, lines, ruleWeight)
	return t
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

// edgeIndex collapses parallel lines into one entry per distinct (from,to) edge,
// with the least rule weight, so weight and edge lookups do not scan adjacency.
// Node n's entries occupy entries[offsets[n]:offsets[n+1]], sorted by to.
// It costs one 8-byte entry per distinct edge, never more than one per line.
type edgeIndex struct {
	offsets []uint32
	entries []edgeEntry
}

type edgeEntry struct{ to, weight uint32 }

func newEdgeIndex(nodes int, out adjacency, lines []topologyLine, ruleWeight func(uint32) uint32) edgeIndex {
	e := edgeIndex{offsets: make([]uint32, nodes+1)}
	e.entries = make([]edgeEntry, 0, len(lines))
	for n := range nodes {
		e.offsets[n] = uint32(len(e.entries))
		for _, id := range out.lines[out.offsets[n]:out.offsets[n+1]] {
			l := lines[id]
			e.entries = append(e.entries, edgeEntry{to: l.to, weight: ruleWeight(l.rule)})
		}
		// Sort this node's entries by target, then keep only the least weight per target.
		group := e.entries[e.offsets[n]:]
		slices.SortStableFunc(group, func(a, b edgeEntry) int { return cmp.Compare(a.to, b.to) })
		kept := group[:0]
		for _, entry := range group {
			if len(kept) > 0 && kept[len(kept)-1].to == entry.to {
				kept[len(kept)-1].weight = min(kept[len(kept)-1].weight, entry.weight)
				continue
			}
			kept = append(kept, entry)
		}
		e.entries = e.entries[:int(e.offsets[n])+len(kept)]
	}
	e.offsets[nodes] = uint32(len(e.entries))
	return e
}

// find returns the least weight of the from->to edge, and whether the edge exists.
func (e edgeIndex) find(from, to int64) (uint32, bool) {
	if from < 0 || from >= int64(len(e.offsets)-1) || to < 0 || to > int64(^uint32(0)) {
		return 0, false
	}
	group := e.entries[e.offsets[from]:e.offsets[from+1]]
	i, ok := slices.BinarySearchFunc(group, uint32(to), func(a edgeEntry, to uint32) int {
		return cmp.Compare(a.to, to)
	})
	if !ok {
		return 0, false
	}
	return group[i].weight, true
}

func checkedTopologyID(id int, kind string) uint32 {
	if uint64(id) > uint64(^uint32(0)) {
		panic("graph: too many topology " + kind)
	}
	return uint32(id)
}
