package engine

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/korrel8r/korrel8r/pkg/uri"
)

// Result accumulates reference and object results for a single class in a correlation chain.
type Result struct {
	Class      korrel8r.Class
	Objects    korrel8r.Result // Collect objects
	References unique.List[uri.Reference]
}

func NewResult(class korrel8r.Class) Result {
	return Result{Class: class, Objects: korrel8r.NewResult(class), References: unique.NewList[uri.Reference]()}
}

// Results is a correlation sequence containing all references and objects leading to the final result.
type Results struct {
	classes map[korrel8r.Class]int
	List    []Result
}

func NewResults() *Results {
	return &Results{classes: map[korrel8r.Class]int{}}
}

// Results aggregates results into a Result per class.
// List is in order that classes are first seen.
func (rs *Results) Append(stages ...Result) {
	for _, r := range stages {
		p := rs.Get(r.Class)
		p.Objects.Append(r.Objects.List()...)
		p.References.Append(r.References.List...)
	}
}

// Get a pointer to the Result for class, one is created if necessary.
// The returned pointer becomes invalid if List is modified (e.g. by calling Append).
func (rs *Results) Get(class korrel8r.Class) *Result {
	i, ok := rs.classes[class]
	if !ok {
		i = len(rs.List)
		rs.classes[class] = i
		rs.List = append(rs.List, NewResult(class))
	}
	return &rs.List[i]
}

// FinalRefs gets the final set of references at the end of the results.
func (rs *Results) FinalRefs() []uri.Reference {
	if len(rs.List) > 0 {
		return rs.List[len(rs.List)-1].References.List
	}
	return nil
}

// Prune returns new Results containing only results with non-empty Objects.
func (rs *Results) Prune() *Results {
	pruned := NewResults()
	for _, r := range rs.List {
		if len(r.Objects.List()) != 0 {
			*pruned.Get(r.Class) = r
		}
	}
	return pruned
}
