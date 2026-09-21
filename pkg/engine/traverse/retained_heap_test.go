// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/graph"
)

// TestRuleGraphRetainedHeap is an opt-in measurement helper, not a memory-limit
// assertion. Run each sample in a fresh process with KORREL8R_TEST_RETAINED_TOPOLOGY
// set to "data" or "view". It measures incremental post-GC topology storage,
// excluding the engine, rules and classes (which remain alive in both snapshots).
// Multiple instances amortize runtime noise. This does not measure peak heap.
func TestRuleGraphRetainedHeap(t *testing.T) {
	mode := os.Getenv("KORREL8R_TEST_RETAINED_TOPOLOGY")
	if mode == "" {
		t.Skip("opt-in retained-heap measurement")
	}
	if mode != "data" && mode != "view" {
		t.Fatal("KORREL8R_TEST_RETAINED_TOPOLOGY must be data or view")
	}
	e := makeEngine(t)
	rules := e.Rules()
	// Warm construction paths before the baseline snapshot.
	_ = graph.NewData(rules...).Graph()
	const instances = 32
	data := make([]*graph.Data, instances)
	views := make([]graph.View, instances)
	runtime.GC()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := range data {
		data[i] = graph.NewData(rules...)
		if mode == "view" {
			views[i] = data[i].Graph()
		}
	}
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(data)
	runtime.KeepAlive(views)
	runtime.KeepAlive(rules)
	runtime.KeepAlive(e)
	// Signed deltas make background heap noise visible rather than underflowing.
	fmt.Printf("retained mode=%s instances=%d bytes_per_graph=%d objects_per_graph=%d\n",
		mode, instances, (int64(after.HeapAlloc)-int64(before.HeapAlloc))/instances,
		(int64(after.HeapObjects)-int64(before.HeapObjects))/instances)
}
