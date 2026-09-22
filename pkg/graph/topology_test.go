// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestTopologyStorage(t *testing.T) {
	assert.Equal(t, uintptr(12), unsafe.Sizeof(topologyLine{}))
	assert.Equal(t, uintptr(8), unsafe.Sizeof(edgeEntry{}))
	assert.Equal(t, uintptr(4), unsafe.Sizeof(adjacency{}.offsets[0]))
}

func TestTopology(t *testing.T) {
	lines := []topologyLine{{0, 1, 0}, {0, 1, 1}, {1, 1, 2}, {1, 0, 3}}
	topo := newTopology(3, lines, func(rule uint32) uint32 { return rule + 1 })
	for _, tc := range []struct {
		node    int64
		out, in []int
	}{
		{0, []int{0, 1}, []int{3}},
		{1, []int{2, 3}, []int{0, 1, 2}},
		{2, nil, nil}, {-1, nil, nil}, {3, nil, nil},
	} {
		var out, in []int
		topo.out.each(tc.node, func(id int) { out = append(out, id) })
		topo.in.each(tc.node, func(id int) { in = append(in, id) })
		assert.Equal(t, tc.out, out)
		assert.Equal(t, tc.in, in)
	}
	assert.Equal(t, lines, topo.lines)
	// Parallel lines 0 and 1 collapse to one 0->1 edge with the least rule weight.
	for _, tc := range []struct {
		from, to int64
		weight   uint32
		ok       bool
	}{
		{0, 1, 1, true}, {1, 1, 3, true}, {1, 0, 4, true},
		{0, 0, 0, false}, {2, 0, 0, false}, {-1, 0, 0, false}, {3, 0, 0, false},
	} {
		w, ok := topo.edges.find(tc.from, tc.to)
		assert.Equal(t, tc.ok, ok, "%v->%v", tc.from, tc.to)
		assert.Equal(t, tc.weight, w, "%v->%v", tc.from, tc.to)
	}
	assert.Zero(t, testing.AllocsPerRun(100, func() { topo.edges.find(0, 1) }))
	assert.Zero(t, testing.AllocsPerRun(100, func() { topo.out.each(0, func(int) {}) }))
	empty := newTopology(0, nil, func(uint32) uint32 { return 1 })
	empty.out.each(0, func(int) { t.Fatal("unexpected line") })
}
