// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Result is an Appender that stores objects in order.
type Result interface {
	korrel8r.Appender
	Add(korrel8r.Object) bool
	List() []korrel8r.Object
}

// NewResult returns a SetResult if class implements IDer, a ListResult otherwise.
func NewResult(class korrel8r.Class) Result {
	if ider, _ := class.(korrel8r.IDer); ider != nil {
		return NewSetResult(ider)
	}
	return NewListResult()
}

// ListResult implements Collecter by simply appending to a slice.
type ListResult []korrel8r.Object

func NewListResult() *ListResult                        { return &ListResult{} }
func (r *ListResult) Append(objects ...korrel8r.Object) { *r = append(*r, objects...) }
func (r ListResult) List() []korrel8r.Object            { return []korrel8r.Object(r) }
func (r *ListResult) Add(o korrel8r.Object) bool        { r.Append(o); return true }

// SetResult de-duplicates the result using an IDer, it ignores second and subsequent objects with the same ID.
type SetResult struct {
	dedup unique.Deduplicator[any, korrel8r.Object]
	list  []korrel8r.Object
}

func NewSetResult(id korrel8r.IDer) *SetResult {
	return &SetResult{dedup: unique.NewDeduplicator(id.ID)}
}
func (r SetResult) List() []korrel8r.Object { return r.list }

func (r *SetResult) Append(objects ...korrel8r.Object) {
	for _, o := range objects {
		r.Add(o)
	}
}
func (r *SetResult) Add(o korrel8r.Object) bool {
	ok := r.dedup.Unique(o)
	if ok {
		r.list = append(r.list, o)
	}
	return ok
}
