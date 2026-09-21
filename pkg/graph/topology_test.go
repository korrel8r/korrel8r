// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestTopologyStorage(t *testing.T) {
	a := adjacency{offsets: make([]uint32, 1)}
	assert.Equal(t, uintptr(12), unsafe.Sizeof(topologyLine{}))
	assert.Equal(t, uintptr(4), unsafe.Sizeof(a.offsets[0]))
}

func TestTopology(t *testing.T) {
	lines := []topologyLine{{0, 1, 0}, {0, 1, 1}, {1, 1, 2}, {1, 0, 3}}
	topo := newTopology(3, lines)
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
	assert.Zero(t, testing.AllocsPerRun(100, func() { topo.out.each(0, func(int) {}) }))
	empty := newTopology(0, nil)
	empty.out.each(0, func(int) { t.Fatal("unexpected line") })
}
