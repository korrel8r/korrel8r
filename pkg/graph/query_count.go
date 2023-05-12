// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"golang.org/x/exp/slices"
)

// QueryCounts is a set of queries and a count of objects they returned.
// Map is indexed by the JSON form of the query.
type QueryCounts map[string]QueryCount

// QueryCount is a query and a count of the data items it returned.
type QueryCount struct {
	Query korrel8r.Query
	Count int
}

func (qcs QueryCounts) Get(q korrel8r.Query) (QueryCount, bool) {
	qc, ok := qcs[korrel8r.JSONString(q)]
	return qc, ok
}
func (qcs QueryCounts) Put(q korrel8r.Query, c int) { qcs[korrel8r.JSONString(q)] = QueryCount{q, c} }

// Total the counts
func (qcs QueryCounts) Total() int {
	total := 0
	for _, qc := range qcs {
		total += qc.Count
	}
	return total
}

// SortQueries collects incoming queries and sorts by decreasing count.
func (qcs QueryCounts) Sort() (list []QueryCount) {
	for _, qc := range qcs {
		list = append(list, qc)
	}
	slices.SortFunc(list, func(a, b QueryCount) bool { return a.Count > b.Count })
	return list
}
