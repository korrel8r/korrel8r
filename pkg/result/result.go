// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package result

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Result is an Appender that stores objects in order.
type Result interface {
	korrel8r.Appender
	Add(korrel8r.Object) bool // Add returns true if the object was added.
	List() []korrel8r.Object  // List returns the objects in order added/appended.
}

// New returns a [Set] if class implements IDer, a [List] otherwise.
func New(class korrel8r.Class) Result {
	if ider, _ := class.(korrel8r.IDer); ider != nil {
		return NewSet(ider)
	}
	return NewList()
}

// List implements [Result] by simply appending to a slice.
type List []korrel8r.Object

func NewList() *List                              { return &List{} }
func (r *List) Append(objects ...korrel8r.Object) { *r = append(*r, objects...) }
func (r *List) List() []korrel8r.Object           { return []korrel8r.Object(*r) }
func (r *List) Add(o korrel8r.Object) bool        { r.Append(o); return true }

// Set implements [Result] and de-duplicates results using an IDer.
// It ignores second and subsequent objects with the same ID.
type Set struct {
	dedup unique.Deduplicator[any, korrel8r.Object]
	list  []korrel8r.Object
}

func NewSet(id korrel8r.IDer) *Set {
	return &Set{dedup: *unique.NewDeduplicator(id.ID)}
}
func (r Set) List() []korrel8r.Object { return r.list }

func (r *Set) Append(objects ...korrel8r.Object) {
	for _, o := range objects {
		r.Add(o)
	}
}
func (r *Set) Add(o korrel8r.Object) bool {
	ok := r.dedup.Unique(o)
	if ok {
		r.list = append(r.list, o)
	}
	return ok
}
