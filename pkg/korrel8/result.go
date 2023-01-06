package korrel8

import (
	"github.com/korrel8/korrel8/pkg/unique"
)

// Result is an Appender that stores objects in order.
type Result interface {
	Appender
	List() []Object
}

// ListResult implements Collecter by simply appending to a slice.
type ListResult []Object

func NewListResult() *ListResult               { return &ListResult{} }
func (r *ListResult) Append(objects ...Object) { *r = append(*r, objects...) }
func (r ListResult) List() []Object            { return []Object(r) }

// SetResult ignores objects with duplicate IDs when appending.
type SetResult struct {
	dedup unique.Deduplicator[any, Object]
	list  []Object
}

func NewSetResult(id IDer) *SetResult { return &SetResult{dedup: unique.NewDeduplicator(id.ID)} }
func (r SetResult) List() []Object    { return r.list }
func (r *SetResult) Append(objects ...Object) {
	for _, o := range objects {
		if r.dedup.Unique(o) {
			r.list = append(r.list, objects...)
		}
	}
}

// NewResult returns a SetResult class implements IDer, or a ListResult otherwise.
func NewResult(class Class) Result {
	if id, _ := class.(IDer); id != nil {
		return NewSetResult(id)
	}
	return NewListResult()
}
