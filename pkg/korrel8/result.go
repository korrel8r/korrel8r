package korrel8

import (
	"github.com/korrel8/korrel8/pkg/unique"
)

// ListResult implements Result to append to a slice []Object.
type ListResult []Object

func NewListResult(objects ...Object) *ListResult { var r ListResult; r.Append(objects...); return &r }
func (r *ListResult) Append(objects ...Object)    { *r = append(*r, objects...) }
func (r ListResult) List() []Object               { return []Object(r) }

// SetResult implements result to ignore objects with duplicate keys.
type SetResult struct {
	class Class
	dedup unique.Deduplicator[any, Object]
	list  []Object
}

// NewSetResult for objects in class.
func NewSetResult(class Class, objects ...Object) *SetResult {
	r := &SetResult{class: class, dedup: unique.NewDeduplicator(class.Key)}
	r.Append(objects...)
	return r
}

func (r SetResult) List() []Object { return r.list }
func (r SetResult) Class() Class   { return r.class }
func (r *SetResult) Append(objects ...Object) {
	for _, o := range objects {
		if r.dedup.Unique(o) {
			r.list = append(r.list, objects...)
		}
	}
}
