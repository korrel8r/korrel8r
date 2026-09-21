# Immutable rule topology and Gonum views

## Ownership and representations

- **Class identity** is a node ID and its class, stored in `Data`. It is not a
  `Node` object.
- **Topology lines** are integer `(from, to, rule)` records. Their slice indexes
  are stable line IDs. Parallel lines and self-loops remain separate records.
- **View** implements Gonum's read-only weighted directed graph interface over
  `Data`. Its nodes are lightweight IDs; it has no mutable `Node`/`Line` objects.
- **Search-owned nodes and lines** hold mutable results, queries and attributes.
  The result `Graph` reuses these objects when the search finishes.

`Data` contains private class/rule tables and compressed sparse row adjacency:
per-node offsets into incoming and outgoing line-ID arrays. Native integer IDs
avoid silent 32-bit truncation. Input order determines IDs and adjacency order.
Construction still eagerly expands each rule's Start × Goal combinations.

| File | Responsibility |
| --- | --- |
| `topology.go` | Integer connectivity and adjacency indexes. |
| `data.go` | Class/rule identity tables, topology construction and ID access. |
| `view.go` | Read-only Gonum adapter over Data, without a second graph representation. |
| `graph.go` | Mutable Gonum graph construction, selection, pruning and rendering. |
| `node.go`, `line.go` | Search/result objects, including their mutable state. |
| `queries.go` | Query accounting, synchronized by the search owner. |

## Read-only algorithms

`Data.Graph()` and `Engine.Graph()` return a lightweight `View`. Creating one
requires no graph construction or allocation. `Data.shared`, `sharedOnce`, and
`SharedGraph` are removed: there is no retained map-backed graph cache.

Gonum paths operate on edges between node pairs, not individual rule lines.
The view therefore returns each neighbor only once and uses the minimum rule
spread (`len(rule.Goal())`) across the pair's parallel lines as its weight.
Self-weight remains zero, independently of whether a self-loop exists, matching
`Graph.Weight`. An edge exists only if a topology line actually connects it.

Neighbor iterators collect, sort and deduplicate IDs on demand, avoiding another
persistent adjacency index. Each iterator owns its state and supports Reset;
concurrent searches share only immutable topology. This trades some per-search
allocation for lower retained memory. Goal search still uses Gonum's Yen
algorithm, then expands each path edge to all matching topology line IDs.

## Mutable results and rendering

`New(data)` creates an empty mutable graph. `data.FullGraph()` creates mutable
nodes and lines for all topology records. `Data.Rules()` returns a copy of the
input rule list, not one entry per expanded line. The rules CLI filters that
list directly: text output constructs no graph, and DOT output builds a graph
only from the selected rules.

`Data.NodeID`, `Class`, `NodeCount`, `LineCount`, `Endpoints`, `RuleForLine`,
`EachLineIDFrom` and `EachLineIDTo` access metadata without creating graph objects.
`Graph.NodeFor` instead returns a node present in that mutable graph, or nil.

`NewNode(id)` and `NewLine(id, from, to)` create independently owned mutable
objects, not cached lookups. Line endpoints are checked against topology IDs.
Searches lazily create nodes and a sparse map of line pointers; their result
graph reuses these without copying state or rebinding endpoints. Routing uses
scoped IDs in per-node slices, with reverse lookup preserving last-scoped-line
identity for duplicate endpoint/rule keys.

`Node.Copy`, `Line.Copy` and `Graph.Select` copy identities, not accumulated
results. `Graph.AddLine` retains the supplied line and its Queries; its endpoints
must belong to that graph. Use `AddLine` rather than the embedded Gonum `SetLine`
to maintain the ordered line index used for allocation-free iteration.

## API migration

- `Data.Nodes`/`NodeFor`: use `NodeID`, `Class`, `NodeCount`.
- `Data.Lines` and pointer-based adjacency: use ID-based topology access.
- `Data.EmptyGraph`: use `New(data)`.
- `Data.SharedGraph`: use `Data.Graph()` for read-only algorithms. Use
  `FullGraph` for mutable graphs and rendering, filtering the rule list first
  when only part of the topology is needed.
- `Engine.Graph()` now returns `View`, not `*Graph`. Its nodes are not `*Node`.

## Validation and measurement

```sh
go test ./pkg/graph ./pkg/engine/... ./cmd/korrel8r
go test -race ./pkg/graph ./pkg/engine/...
go test ./pkg/engine/traverse -run '^$' \
  -bench 'Benchmark(RuleGraph|Neighbors|NeighborScope|NewTraverser|Goals)$' \
  -benchmem -count=10
```

Compare constructor and repeated-search costs separately. For retained heap,
run `TestRuleGraphRetainedHeap` in fresh processes with
`KORREL8R_TEST_RETAINED_TOPOLOGY=data` or `view`. It reports post-GC incremental
storage per graph, not peak memory. Historical benchmark directories describe
the representations at their respective revisions, including the removed
shared graph. Tests compare the view against `FullGraph` for neighbor sets,
weights, near-shortest paths, parallel-line expansion, results and query counts.
