# Calibrating traversal limits against memory

Estimate `totalLimit` and `totalQueryLimit` from representative traversals. This
is a sizing heuristic, not a memory guarantee. Repeat after changes to Korrel8r,
Go, rules, stores, configuration, or payload sizes.

## Quick local comparison

For a simpler end-to-end comparison on Linux, with `korrel8r`, Go, and Python 3
on PATH and GNU `/usr/bin/time` installed:

```bash
hack/profile-traversal-local results/local-01 \
  --config etc/korrel8r/openshift-route.yaml \
  --query 'k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}'
```

Use a new output directory for each series. The helper runs the same query with
`korrel8r neighbors --memprofile` at depths 1–5, overriding any supplied depth.
Each `depth-N` directory contains `graph.json`, `heap.prof`, the raw profile, and
stderr. A combined `summary.csv` has one row per depth, recording:

- Graph node count and node-level `QueryCount` entry count (not edge duplicates).
- Sum of query result counts and sum of deduplicated node result counts.
- Peak process RSS from `/usr/bin/time`, converted from KiB to bytes.
- Profile `inuse_space` and `alloc_space` totals in bytes, plus truncation status.

Query result counts may count the same object in multiple queries. Visible query
entries are not traversal-wide accepted queries. The exit-time, post-GC profile
may omit released traversal state; allocated bytes measure churn, and neither
profile column measures peak memory. Use `peak_rss_bytes` for empirical process
memory fits; it includes startup, stores, serialization, and profiling overhead,
not just traversal state. This is not a hard memory bound. Preserve build and
workload details separately. For memory-budget calibration, use the procedure below.

## 1. Prepare workloads

Save `korrel8rcli neighbors` arguments for typical, large-payload, and high-fan-out
searches. Vary depth and per-query `--limit`, including searches with few or no
results, to vary the objects-per-query ratio.

Keep store data and time ranges stable. Record the server build, local changes,
Go version, configuration/rules, resource limits, and `GOGC`/`GOMEMLIMIT`. Disable
total limits in both configuration and requests for initial calibration. Exclude
truncated or error-affected runs. Protect credentials and captured payloads.

## 2. Capture measurements

Use an otherwise idle server; counters are process-wide. Expose profiling only
on a trusted interface:

```bash
korrel8r web --httpprofile --http 127.0.0.1:8080
```

Warm up each workload once, then capture at least five repetitions in separate
directories:

```bash
SAMPLE_SECONDS=0.1 hack/profile-traversal \
  http://127.0.0.1:8080 results/deployment-01 -- \
  --query 'k8s:Deployment.v1.apps:{"namespace":"openshift-apiserver","name":"apiserver"}' \
  --depth 3
```

Build `korrel8rcli` in the sibling `../client` checkout first. Arguments after `--`
are passed to its `neighbors` command. The helper obtains a bearer token with
`oc whoami -t` for both traversal and pprof requests; log in with `oc` first.
Metrics and pprof still use `curl`.

The helper saves metrics, heap profiles, and the response, but not arguments;
preserve workload arguments separately without credentials. Record the client
revision too. The capture revision identifies the checkout, not the server.

Record one CSV row per run:

```text
revision,scenario,repeat,baseline_heap_bytes,peak_heap_bytes,retained_objects,accepted_queries,truncated
```

| Value | Source |
| --- | --- |
| Baseline heap | `go_memstats_heap_inuse_bytes` in `metrics-before.prom` |
| Peak heap | Maximum of that metric across all snapshots, including before/after |
| Retained objects | After-minus-before delta of `traverse_retained_objects_total` |
| Accepted queries | After-minus-before delta of `traverse_accepted_queries_total` |
| Truncation | Presence of `truncation` in the response |

Verify metric names and sum all label series. A counter absent before first use
is zero; missing metrics afterward need investigation. Do not count visible graph
queries: accepted queries include speculative and zero-result queries.

Inspect unexpected growth with:

```bash
go tool pprof -sample_index=inuse_space -top results/deployment-01/heap-0003.prof
```

Sampling misses short peaks and profiling adds overhead. Captures are serial, so
intervals exceed `SAMPLE_SECONDS`. Verify peaks with lower-overhead process/container
monitoring. `alloc_space` measures cumulative allocations, not retained memory;
post-completion GC profiles may omit already-released traversal state.

## 3. Fit the model

Fit non-negative coefficients across the runs:

```text
peakHeap - baselineHeap ≈ F + a × retainedObjects + b × acceptedQueries
```

`F` estimates fixed traversal overhead; `a` and `b` estimate bytes per object and
query, including associated bookkeeping. They are not exact Go object sizes.

- Do not divide the same memory measurement by both counts: that counts bytes twice.
- If object and query counts vary together, add workloads with different ratios;
  the two coefficients cannot otherwise be separated reliably.
- Validate on held-out scenarios and repetitions. Increase reserves or coefficients
  to cover underprediction; unstable coefficients indicate an inadequate model.

Save requests, raw captures, CSV, fitting method, coefficients, and validation
results together for comparisons across revisions.

## 4. Calculate and validate limits

For process memory target `P`, reserve `B` for baseline and non-traversal memory,
and allow at most `N` concurrent traversals. Choose safety factor `s` (initially
0.6) and object-budget fraction `f` (initially 0.75):

```text
X = (P - B) / N
U = s × X - F

totalLimit      = floor(f × U / a)
totalQueryLimit = floor((1 - f) × U / b)
```

Do not subtract baseline again from `X`. Require positive `U`, coefficients, and
limits: **zero limits mean unlimited**. Adjust the split for observed workload
ratios to avoid premature truncation.

Heap in-use is not RSS. Reserve space for stacks, runtime/native memory, caches,
store buffers, and GC/allocator slack. In-flight decoding may continue after a
limit is reached, so counts cannot enforce a hard process memory ceiling.

Repeat the workloads with the proposed limits, then at concurrency `N`, monitoring
process/container peaks. Check truncation metadata, useful partial results, and
headroom. If peaks exceed the target, lower limits or concurrency, reduce
per-query limits, or increase reserve. Mock allocation benchmarks alone cannot
validate real-store payloads and buffering.

## 5. Quick sizing from a container memory limit

When there is no local fit, use the coefficients measured on 2026-09-23 against a
live OpenShift cluster on the memory-reduction branch
(`doc/dev/traversal-memory-session.md`, 20 server runs, peak RSS):

```text
peakRSS ≈ B + c × retainedObjects
```

| Parameter | Value | Note |
| --- | --- | --- |
| `B` | 95 MiB | baseline server RSS after store warm-up, single session |
| `c` | 11 KiB per retained object | includes that object's share of query cost |
| `s` | 0.8 | safety factor |
| `N` | — | concurrent traversals to allow |

For a container memory limit `M`:

```text
totalLimit      = floor((s × M - B) / (N × c))
totalQueryLimit = floor(totalLimit / 10)
```

| `M` | `N` | `totalLimit` | `totalQueryLimit` |
| --- | --- | --- | --- |
| 256 MiB | 1 | 10000 | 1000 |
| 512 MiB | 1 | 29000 | 2900 |
| 512 MiB | 2 | 14000 | 1400 |
| 1 GiB | 1 | 67000 | 6700 |
| 2 GiB | 1 | 143000 | 14000 |

The fit gave R² = 0.998 with a largest in-sample underprediction of 15 MiB, over
100–40,768 objects and 84–524 MiB RSS. Adding a separate query term
(`B` + 10.5 KiB × objects + 6.5 KiB × queries) did not improve R², and objects and
queries were collinear (r = 0.991). Use the single-coefficient form. The query
limit is a backstop derived from the observed 17–100 objects per visible query,
halved again because accepted queries include speculative and zero-result queries
that never appear in the graph.

Measured baseline was 77.4 MiB (sd 2.1) with one shared session; 95 MiB is the
fitted intercept, which is the safer number to size against. A server with many
concurrent user sessions holds one engine per session, so re-measure the baseline
for that deployment shape.

### Change from the 2026-09-21 coefficients

The earlier session measured a 220 MiB baseline and an effective 75 KiB per object.
Both were far too pessimistic for this branch: the baseline is roughly 95 MiB and
the per-object cost roughly 11 KiB. Most of the per-object drop is `stripUnused`
removing `metadata.managedFields`: the opt-in retained-heap test
(`pkg/domains/k8s`, real captured fixtures) measures 57,639 B/object unstripped
versus 31,597 B/object stripped, a 45% cut in retained bytes and 42% fewer heap
objects. Topology changes affect only fixed overhead: `TestRuleGraphRetainedHeap`
measures about 83 KB per rule graph, negligible at any realistic limit.

Re-run both opt-in measurements after further allocation changes:

```bash
KORREL8R_TEST_RETAINED_K8S=strip go test ./pkg/domains/k8s -run TestObjectRetainedHeap -v
KORREL8R_TEST_RETAINED_TOPOLOGY=data go test ./pkg/engine/traverse -run TestRuleGraphRetainedHeap -v -count=1
```

These numbers come from one cluster and one revision. They are a starting point
for the validation in section 4, not a memory guarantee; re-measure with sections
1–4 for other payload sizes, domains, or store configurations.
