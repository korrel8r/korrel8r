// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import "github.com/korrel8r/korrel8r/pkg/korrel8r"

// QueryCount records the number of objects resulting from a query.
// Count == -1 means the query has not been evaluated.
type QueryCount struct {
	Query        korrel8r.Query
	Count        int
	StatusCounts map[string]int
}

// Queries holds mutable query accounting for a Node or Line, keyed by query
// identity. It belongs to a search or result Graph, never to immutable topology.
// Concurrent access requires synchronization by the owner.
type Queries map[korrel8r.Query]QueryCount

func (qs Queries) Has(q korrel8r.Query) bool   { _, ok := qs[q]; return ok }
func (qs Queries) Set(q korrel8r.Query, n int) { qs[q] = QueryCount{Query: q, Count: n} }
func (qs Queries) Get(q korrel8r.Query) int {
	if qc, ok := qs[q]; ok {
		return qc.Count
	}
	return -1
}

// AddStatuses merges status counts into the QueryCount for q.
func (qs Queries) AddStatuses(q korrel8r.Query, statuses map[string]int) {
	qc := qs[q]
	if qc.StatusCounts == nil {
		qc.StatusCounts = map[string]int{}
	}
	for k, v := range statuses {
		qc.StatusCounts[k] += v
	}
	qs[q] = qc
}

// Total sums positive object counts, excluding unevaluated queries.
func (qs Queries) Total() (total int) {
	for _, qc := range qs {
		if qc.Count > 0 {
			total += qc.Count
		}
	}
	return total
}
