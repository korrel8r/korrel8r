package engine

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Result accumulates queries and objects for a single class in a correlation chain.
type Result struct {
	Class   korrel8r.Class
	Objects korrel8r.Result // Collect objects
	Queries unique.List[korrel8r.Query]
}

func NewResult(class korrel8r.Class) Result {
	return Result{Class: class, Objects: korrel8r.NewResult(class), Queries: unique.NewList[korrel8r.Query]()}
}

// Empty is true if there are no objects in this result.
func (r Result) Empty() bool { return r.Objects == nil || len(r.Objects.List()) == 0 }

// Results is a correlation sequence containing all queries and objects leading to the final result.
type Results []Result

// Results collects a Result per class.
func (rs *Results) Append(results ...Result) {
	for _, r := range results {
		p := rs.Get(r.Class)
		p.Objects.Append(r.Objects.List()...)
		p.Queries.Append(r.Queries.List...)
	}
}

func (rs Results) find(class korrel8r.Class) (int, bool) {
	for i := range rs {
		if (rs)[i].Class == class {
			return i, true
		}
	}
	return len(rs), false
}

// Get a pointer to the Result for class, one is added if necessary.
func (rs *Results) Get(class korrel8r.Class) *Result {
	i, ok := rs.find(class)
	if !ok {
		*rs = append(*rs, NewResult(class))
	}
	return &(*rs)[i]
}

// Last returns the last result or nil if empty.
func (rs Results) Last() *Result {
	if len(rs) > 0 {
		return &rs[len(rs)-1]
	}
	return nil
}
