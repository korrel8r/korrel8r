# Traversal benchmarks

The benchmarks in `traverse_bench_test.go` use mock stores and do not require a
Kubernetes or OpenShift cluster. Benchmark-specific setup and what each benchmark
measures are documented next to its implementation.

## Run benchmarks

From the repository root:

```bash
# Run the standard traversal benchmark set ten times and record environment metadata.
hack/benchmark-traverse

# Run every benchmark once.
go test ./pkg/engine/traverse -run '^$' -bench . -benchmem

# Write results elsewhere or choose the number of samples.
COUNT=10 hack/benchmark-traverse /tmp/traverse-results.txt
```

## Compare changes

Capture multiple samples before and after a change with the same machine, Go
version, configuration, and background workload:

```bash
hack/benchmark-traverse before.txt
# Make the change.
hack/benchmark-traverse after.txt
benchstat before.txt after.txt
```

Install `benchstat` if necessary:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```
