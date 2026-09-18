# Traversal memory measurement session

## Current tooling

- `hack/profile-traversal`: server-side capture using sibling `../client/korrel8rcli`
  for neighborhood requests and curl for metrics/pprof. Gets credentials with
  `oc whoami -t`. Prints progress dots. Downloads heap profiles to temporary files
  and checks gzip integrity before publishing them. Inspect one snapshot at a time;
  passing multiple snapshots to pprof merges them.
- `hack/profile-traversal-local`: runs the same neighborhood query at depths 1–5,
  saves per-depth graphs/profiles/logs, and writes a combined `summary.csv`.
  GNU `/usr/bin/time` records peak RSS (Linux KiB converted to bytes). The profile
  parser strips pprof's `[dflt]` column suffix.
- `doc/dev/traversal-memory.md`: procedure and interpretation caveats.

Example:

```bash
hack/profile-traversal-local results/new-series \
  --query 'k8s:Pod:{"namespace":"openshift-cluster-observability-operator","name":"POD_NAME"}'
```

Requires korrel8r and Go on PATH, Python 3, GNU `/usr/bin/time`, and store access.
The configured store configuration during this session was
`etc/korrel8r/openshift-route.yaml` via `KORREL8R_CONFIG`.

## Captures (2026-09-21)

Paths are relative to the repository root; raw results are local artifacts.

| Directory | Workload |
| --- | --- |
| `results/local-fit-20260921-082517/` | Initial attempt; failed parsing `[dflt]`, fixed afterward |
| `results/local-fit-20260921-082533/` | Deployment `openshift-apiserver/apiserver`, depths 1–5 |
| `results/coo-fit-20260921-083127/` | Eight deployments in `openshift-cluster-observability-operator`, depths 1–5 |
| `results/coo-pods-rss-20260921-083848/` | Nine pods in that namespace, depths 1–5; peak RSS captured |
| `results/coo-pods-repeat-20260921-084746/` | Five repetitions each of three selected pod/depth cases |

Deployment and pod aggregate directories contain combined CSV data, saved queries,
`fit.py`, and `fit.json`. Fitting scripts require NumPy. The repetition directory
contains `cases.json`, `summary.csv`, and `variability.json`. Protect raw graphs,
logs, and cluster inventory as potentially sensitive data.

## Results

Exit-time live heap was a poor predictor of traversal work: completed traversal
state has already been released. Cumulative allocations correlated with graph
counts but cannot be used as retained-memory estimates.

For all 45 pod/depth runs, fitting peak RSS gave:

```text
peak_rss_bytes ≈ 220,324,011
               + 35,021 × node_results
               + 795,847 × query_count_entries
R² = 0.898
```

Observed RSS range: 190–345 MiB. Largest in-sample underprediction: 28.85 MiB.
Object/query correlation: 0.906. All runs succeeded without truncation; some
reported Perses API deprecation warnings. This fit has not been validated on
held-out workloads and is not a memory guarantee.

### Repetitions

Selected the lowest-, median-, and highest-RSS cases from the pod dataset and ran
five interleaved repetitions of each using the local helper's `capture` function.

| Case | Pod / depth | Objects | Visible queries | RSS range MiB | Mean MiB | Sample CV |
| --- | --- | --- | --- | --- | --- | --- |
| Small | `logging-748f5f586-lrzbg` / 1 | 43 | 6 | 194.2–204.8 | 202.5 | 2.3% |
| Medium | `troubleshooting-panel-779f779c48-lzfzp` / 3 | 456–472 | 28 | 248.2–262.8 | 258.6 | 2.3% |
| Large | `logging-748f5f586-lrzbg` / 5 | 1,745–2,417 | 77–90 | 325.2–367.4 | 339.1 | 4.8% |

No truncation or stderr messages in these repetitions. The largest RSS was at
1,887 objects, not the largest object count. Stable workload RSS varied modestly;
larger runs also changed their live-store results, preventing isolation of runtime
variation alone.

## Interpretation and next steps

- `node_results` sums node counts; `query_count_entries` counts node-level visible
  queries, not traversal-wide accepted queries. Zero-result/speculative work may
  be absent. Do not directly substitute `totalQueryLimit` into this fit.
- Peak RSS includes startup, stores, output serialization, and profiling overhead.
  It is not isolated traversal heap. Coefficients describe correlated work, not
  literal object/query sizes.
- Further full-matrix repetitions are lower priority than held-out validation,
  accepted-query accounting, and representative payload variation.
- Use fixed/replayed store data to isolate GC/concurrency variation, and test
  intended server concurrency before recommending memory-based count limits.
- No hard memory-budget recommendation was established in this session.

## Captures (2026-09-23, memory-reduction branch)

Re-measured after the branch changes (`b522929a`..`2af44e90`): adjacency-matrix
topology, compact int-ID topology, separated static node state, and
`stripUnused` removal of `metadata.managedFields`. Raw results are local
artifacts under `/tmp/mem-refit/`, not committed.

CLI runs against a live OpenShift cluster with `etc/korrel8r/openshift-route.yaml`,
peak RSS from GNU `/usr/bin/time`, two repetitions each of two workload families:
all deployments (`k8s:Deployment.apps:{}`) at depths 0–4, and one deployment
(`openshift-apiserver/apiserver`) at depths 1–5. No run truncated.

| Run | Objects | Visible queries | Nodes | Peak RSS MiB |
| --- | --- | --- | --- | --- |
| `all-d0-r1` | 100 | 1 | 1 | 76.0 |
| `all-d0-r2` | 100 | 1 | 1 | 81.9 |
| `all-d1-r1` | 2380 | 70 | 15 | 135.1 |
| `all-d1-r2` | 2423 | 70 | 15 | 132.7 |
| `all-d2-r1` | 2689 | 145 | 27 | 138.0 |
| `all-d2-r2` | 2761 | 141 | 27 | 140.1 |
| `all-d3-r1` | 2624 | 169 | 33 | 143.8 |
| `all-d3-r2` | 3753 | 175 | 32 | 158.5 |
| `all-d4-r1` | 3611 | 177 | 32 | 159.0 |
| `all-d4-r2` | 3084 | 168 | 32 | 143.6 |
| `one-d1-r1` | 206 | 6 | 5 | 85.3 |
| `one-d1-r2` | 206 | 6 | 5 | 86.4 |
| `one-d2-r1` | 419 | 21 | 11 | 104.3 |
| `one-d2-r2` | 421 | 23 | 12 | 104.9 |
| `one-d3-r1` | 930 | 57 | 14 | 108.5 |
| `one-d3-r2` | 798 | 56 | 15 | 112.6 |
| `one-d4-r1` | 1230 | 92 | 18 | 115.0 |
| `one-d4-r2` | 1690 | 95 | 19 | 111.0 |
| `one-d5-r1` | 1934 | 109 | 24 | 119.8 |
| `one-d5-r2` | 1795 | 99 | 19 | 113.6 |

Least-squares fits over all 20 runs:

```text
peak_rss_bytes ≈ 90,060,171 + 20,632 × objects                      R² = 0.932
peak_rss_bytes ≈ 89,866,582 + 18,157 × objects + 51,111 × queries   R² = 0.934
```

Largest in-sample underprediction 11.0 MiB. Objects and visible queries remain
collinear (r = 0.941); the second coefficient buys 0.002 of R², so the
single-coefficient form is preferred. Objects/query ratio ranged 13.4–100.

Compared with 2026-09-21 (220 MiB baseline, 35 KiB/object plus 796 KiB/query,
collapsing to about 75 KiB/object) the per-object cost is roughly a third. The
baseline also differs because these are CLI runs; 2026-09-21 measured a server
under `hack/profile-traversal`, whose smallest observed RSS was 190 MiB.
Server-mode sizing should keep using a ~200 MiB baseline until re-measured.

Supporting opt-in retained-heap measurements on the same branch:

| Measurement | Value |
| --- | --- |
| k8s object, unstripped | 57,639 B, 892 heap objects |
| k8s object, `stripUnused` | 31,597 B, 518 heap objects |
| Rule graph `Data` | 83,190 B, 14 heap objects |
| Rule graph `View` | 82,957 B, 12 heap objects |

The 45% cut in retained bytes per k8s object accounts for most of the drop in the
per-object coefficient. Topology is fixed overhead and negligible at any realistic
limit.

Not done: server-mode capture with `hack/profile-traversal`, held-out validation,
and verification that the proposed limits hold at concurrency. Wall times varied
10x on repeated identical queries (0.36s to 32s), indicating live-store variation
that peak RSS may also carry.

## Server-mode captures (2026-09-23)

The CLI numbers above understate both baseline and object counts. Re-measured in
server mode, which is what `tuning` actually configures.

Setup: `korrel8r web --unsafe-shared-session --http 127.0.0.1:8080`, one fresh
server process per run, POST to `/api/v1alpha1/graphs/neighbors`. Baseline RSS is
`VmRSS` after a trivial warm-up request that creates the stores; peak is `VmHWM`
after the measured request. The stock `etc/korrel8r/openshift-route.yaml` sets
`requestTimeout: 1m`, which aborted every large traversal with
`context deadline exceeded`; runs used a copy with `requestTimeout: 15m`. That
copy needs the relative `rules/` include resolvable from its own directory.

| Run | Objects | Visible queries | Nodes | Baseline MiB | Peak RSS MiB |
| --- | --- | --- | --- | --- | --- |
| `all-d0-r1` | 100 | 1 | 1 | 76.2 | 84.4 |
| `all-d0-r2` | 100 | 1 | 1 | 79.8 | 84.7 |
| `all-d1-r1` | 11836 | 453 | 16 | 74.1 | 215.6 |
| `all-d1-r2` | 11579 | 454 | 16 | 75.8 | 219.6 |
| `all-d2-r1` | 25720 | 1199 | 30 | 80.6 | 361.6 |
| `all-d2-r2` | 25619 | 1203 | 30 | 79.3 | 362.8 |
| `all-d3-r1` | 37987 | 2089 | 42 | 75.8 | 492.2 |
| `all-d3-r2` | 37939 | 2091 | 42 | 76.0 | 483.4 |
| `all-d4-r1` | 40065 | 2396 | 42 | 77.4 | 508.8 |
| `all-d4-r2` | 40768 | 2408 | 41 | 75.9 | 523.5 |
| `one-d1-r1` | 222 | 6 | 5 | 76.1 | 90.0 |
| `one-d1-r2` | 229 | 6 | 5 | 78.9 | 95.1 |
| `one-d2-r1` | 434 | 21 | 11 | 74.9 | 107.2 |
| `one-d2-r2` | 439 | 21 | 11 | 79.4 | 114.2 |
| `one-d3-r1` | 1241 | 62 | 14 | 79.7 | 109.1 |
| `one-d3-r2` | 1241 | 62 | 14 | 75.5 | 110.5 |
| `one-d4-r1` | 1691 | 185 | 18 | 78.6 | 115.8 |
| `one-d4-r2` | 1719 | 190 | 18 | 74.2 | 115.2 |
| `one-d5-r1` | 18374 | 856 | 27 | 80.0 | 274.9 |
| `one-d5-r2` | 2184 | 215 | 18 | 79.6 | 115.5 |

Least-squares fits over all 20 runs:

```text
peak_rss_bytes ≈ 99,009,850 + 10,893 × objects                     R² = 0.998
peak_rss_bytes ≈ 99,153,053 + 10,532 × objects + 6,517 × queries   R² = 0.998
```

Largest in-sample underprediction 15.2 MiB. Objects and visible queries are
collinear (r = 0.991) and the query term adds nothing, so the single-coefficient
form is used. Measured baseline 77.4 MiB (min 74.1, max 80.6, sd 2.1); the fitted
intercept of 94 MiB is higher and is the number used for sizing.

Server object counts are much larger than the CLI runs for the same query and
depth (11,836 versus 2,380 at depth 1), so the two datasets are not comparable and
were fitted separately. The cause was not investigated.

Compared with 2026-09-21 (220 MiB baseline, effective 75 KiB/object) the baseline
is 2.3x lower and the per-object cost nearly 7x lower.

Not done: held-out validation, multi-session baseline (all runs used
`--unsafe-shared-session`, so one engine), and verification that the proposed
limits hold at concurrency. `one-d5` varied 10x in objects between repetitions
(18,374 versus 2,184), so live-store variation is large at depth.
