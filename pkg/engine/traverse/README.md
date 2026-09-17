# Traversal benchmarks

The benchmarks in `traverse_bench_test.go` measure the main allocation and runtime costs of neighborhood and goal traversal. They use mock stores and do not require a Kubernetes or OpenShift cluster.

Run commands from the repository root unless noted otherwise.

## Scenarios

- `BenchmarkNeighbors` measures a complete depth-3 neighborhood search.
- `BenchmarkNeighborScope` measures selection of the depth-limited rule scope.
- `BenchmarkDataNodeFor` measures lookup by stable class identity.
- `BenchmarkDataLinesFrom` measures direct iteration over the immutable adjacency index.
- `BenchmarkNewTraverser` measures allocation and initialization of traversal state.
- `BenchmarkRuleGraph/Data` measures construction of immutable rule graph data.
- `BenchmarkRuleGraph/SharedGraph` measures rule data plus construction of the shared Gonum graph.
- `BenchmarkGoals` measures a complete goal-directed search.

The benchmarks use this start query:

```text
k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}
```

`BenchmarkNeighbors` performs an untimed warm-up and verifies that the mock scenario produces non-empty node, edge, and query results. Detailed traversal semantics are covered by the package unit tests.

## Reproduce the stored baseline

Run the repository helper from any directory:

```bash
hack/benchmark-traverse
```

It runs 10 samples and writes metadata and results to `pkg/engine/traverse/benchmark-results.txt`. Pass another output path as the first argument or override the sample count with `COUNT`:

```bash
COUNT=5 hack/benchmark-traverse /tmp/traverse-results.txt
```

The checked-in initial result is `benchmark-baseline.txt`. Do not overwrite it for routine comparisons; store new before and after result files separately.

## Run all traversal benchmarks

```bash
go test ./pkg/engine/traverse -run '^$' -bench . -benchmem
```

`-run '^$'` skips unit tests, `-bench .` selects all benchmarks, and `-benchmem` reports allocated bytes and allocation counts per operation.

Run only the memory-optimization benchmarks with multiple samples:

```bash
go test ./pkg/engine/traverse -run '^$' \
  -bench 'Benchmark(Neighbors|NeighborScope|DataNodeFor|DataLinesFrom|NewTraverser|RuleGraph)$' \
  -benchmem -count=5
```

For a quick check that each benchmark works, force one iteration:

```bash
go test ./pkg/engine/traverse -run '^$' \
  -bench 'Benchmark(Neighbors|NeighborScope|NewTraverser|RuleGraph)$' \
  -benchmem -benchtime=1x
```

Do not use a one-iteration run for performance conclusions.

## Compare changes with benchstat

Capture several samples before and after a change:

```bash
go test ./pkg/engine/traverse -run '^$' \
  -bench 'Benchmark(Neighbors|NeighborScope|NewTraverser|RuleGraph)$' \
  -benchmem -count=10 > before.txt

# Make the optimization, then run the same command.
go test ./pkg/engine/traverse -run '^$' \
  -bench 'Benchmark(Neighbors|NeighborScope|NewTraverser|RuleGraph)$' \
  -benchmem -count=10 > after.txt

benchstat before.txt after.txt
```

Install `benchstat` if necessary:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

Use the same machine, power settings, Go version, benchmark arguments, and background workload for both runs.

## Capture an allocation profile

Use a fixed iteration count so before and after profiles perform equal work:

```bash
go test ./pkg/engine/traverse -run '^$' \
  -bench '^BenchmarkNeighbors$' \
  -benchtime=10x \
  -memprofile=neighbors.prof
```

Inspect total allocated space:

```bash
go tool pprof -sample_index=alloc_space -top neighbors.prof
```

Inspect allocation count or memory still live when the profile was written:

```bash
go tool pprof -sample_index=alloc_objects -top neighbors.prof
go tool pprof -sample_index=inuse_space -top neighbors.prof
```

Open the interactive web view:

```bash
go tool pprof -http=:8080 neighbors.prof
```

`alloc_space` measures allocation churn over the benchmark. `inuse_space` measures objects still live at profile capture. Neither is automatically equivalent to peak RSS.

## Capture a CPU profile

```bash
go test ./pkg/engine/traverse -run '^$' \
  -bench '^BenchmarkNeighbors$' \
  -benchtime=10x \
  -cpuprofile=neighbors-cpu.prof

go tool pprof -top neighbors-cpu.prof
```

Increase `-benchtime` if the CPU profile contains too few samples.

## Measure individual phases

Run scope selection independently:

```bash
go test ./pkg/engine/traverse -run '^$' \
  -bench '^BenchmarkNeighborScope$' -benchmem -count=5
```

Run traverser initialization independently:

```bash
go test ./pkg/engine/traverse -run '^$' \
  -bench '^BenchmarkNewTraverser$' -benchmem -count=5
```

Compare immutable graph data with the shared Gonum graph:

```bash
go test ./pkg/engine/traverse -run '^$' \
  -bench '^BenchmarkRuleGraph' -benchmem -count=5
```

## Notes

- Engine construction and mock-store configuration happen before benchmark timing for traversal benchmarks.
- `BenchmarkRuleGraph` obtains the production rule set before timing, then measures graph construction explicitly.
- Mock stores have a small artificial delay to exercise concurrent traversal without requiring external services.
- Run `go test ./pkg/engine/traverse` after code changes to execute the package's unit tests.
- End-to-end cluster profiling remains necessary to measure actual Loki, Prometheus, Kubernetes, and network behavior.
