package engine

import (
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// QueryResult holds a query and the result of evaluating that query.
type QueryResult struct {
	Query  korrel8r.Query
	Result korrel8r.Result
}

func (qr QueryResult) Empty() bool {
	return qr.Result == nil || len(qr.Result.List()) == 0
}

// Results collected from traversal of graph lines (rules)
type Results struct{ query map[int64]QueryResult } // By link id
func NewResults() *Results                         { return &Results{query: map[int64]QueryResult{}} }

func (r *Results) Get(l *graph.Line) (QueryResult, bool) { qr, ok := r.query[l.ID()]; return qr, ok }
func (r *Results) Set(l *graph.Line, qr QueryResult)     { r.query[l.ID()] = qr }

// ByClass returns all QueryResults for a Class.
func (r *Results) ByClass() map[korrel8r.Class][]QueryResult {
	m := map[korrel8r.Class][]QueryResult{}
	for _, qr := range r.query {
		m[qr.Query.Class()] = append(m[qr.Query.Class()], qr)
	}
	return m
}
