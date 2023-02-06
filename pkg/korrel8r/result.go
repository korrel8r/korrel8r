package korrel8r

import (
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Result is an Appender that stores objects in order.
type Result interface {
	Appender
	List() []Object
}

// NewResult returns a SetResult if class implements IDer, a ListResult otherwise.
func NewResult(class Class) Result {
	if id, _ := class.(IDer); id != nil {
		return NewSetResult(id)
	}
	return NewListResult()
}

// ListResult implements Collecter by simply appending to a slice.
type ListResult []Object

func NewListResult() *ListResult               { return &ListResult{} }
func (r *ListResult) Append(objects ...Object) { *r = append(*r, objects...) }
func (r ListResult) List() []Object            { return []Object(r) }

// SetResult de-duplicates the result using an IDer, it ignores second and subsequent objects with the same ID.
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

type CountResult struct {
	Count int
	Result
}

func NewCountResult(r Result) *CountResult { return &CountResult{Result: r} }
func (c *CountResult) Append(objects ...Object) {
	c.Count += len(objects)
	c.Result.Append(objects...)
}
