// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	sigsyaml "sigs.k8s.io/yaml"
)

// TestObjectRetainedHeap is an opt-in measurement helper, not a memory-limit assertion.
// It measures the post-GC heap retained by decoded k8s objects, with and without
// [stripUnused]. Run each sample in a fresh process:
//
//	KORREL8R_TEST_RETAINED_K8S=strip go test ./pkg/domains/k8s -run TestObjectRetainedHeap -v
//	KORREL8R_TEST_RETAINED_K8S=full  go test ./pkg/domains/k8s -run TestObjectRetainedHeap -v
//
// The fixture is real captured cluster data including metadata.managedFields.
// Multiple instances amortize runtime noise. This does not measure peak heap:
// the API client decodes managedFields before the store sees the object, so the
// saving here is in retained memory, not in allocation.
func TestObjectRetainedHeap(t *testing.T) {
	mode := os.Getenv("KORREL8R_TEST_RETAINED_K8S")
	if mode == "" {
		t.Skip("opt-in retained-heap measurement")
	}
	if mode != "strip" && mode != "full" {
		t.Fatal("KORREL8R_TEST_RETAINED_K8S must be strip or full")
	}
	fixtures := readFixtures(t)
	require.NotEmpty(t, fixtures, "no fixture objects")

	decode := func() []Object {
		objects := make([]Object, len(fixtures))
		for i, b := range fixtures {
			var o Object
			require.NoError(t, json.Unmarshal(b, &o))
			if mode == "strip" {
				o = stripUnused(o)
			}
			objects[i] = o
		}
		return objects
	}
	_ = decode() // Warm the decode path before the baseline snapshot.

	const instances = 32
	batches := make([][]Object, instances)
	runtime.GC()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := range batches {
		batches[i] = decode()
	}
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(batches)
	runtime.KeepAlive(fixtures)

	// Signed deltas make background heap noise visible rather than underflowing.
	total := int64(instances * len(fixtures))
	fmt.Printf("retained mode=%s objects=%d bytes_per_object=%d heap_objects_per_object=%d\n",
		mode, total,
		(int64(after.HeapAlloc)-int64(before.HeapAlloc))/total,
		(int64(after.HeapObjects)-int64(before.HeapObjects))/total)
}

// readFixtures returns the raw JSON of every object in the mock-store fixture file.
func readFixtures(t *testing.T) [][]byte {
	t.Helper()
	b, err := os.ReadFile("testdata/domain_test.yaml")
	require.NoError(t, err)
	var byQuery map[string][]json.RawMessage
	require.NoError(t, sigsyaml.Unmarshal(b, &byQuery))
	var objects [][]byte
	for _, list := range byQuery {
		for _, o := range list {
			objects = append(objects, o)
		}
	}
	return objects
}
