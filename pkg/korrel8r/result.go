// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Result is an Appender that stores objects in order.
type Result interface {
	Appender
	List() []Object
}

// GetID returns the object ID using if class is an IDer, "" otherwise.
func GetID(class Class, object Object) string {
	if ider, _ := class.(IDer); ider != nil {
		return fmt.Sprintf("%v", ider.ID(object))
	}
	return ""
}

// NewResult returns a SetResult if class implements IDer, a ListResult otherwise.
func NewResult(class Class) Result {
	if ider, _ := class.(IDer); ider != nil {
		return NewSetResult(ider)
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
			r.list = append(r.list, o)
		}
	}
}

// FuncAppender turns makes a function conform to the Appender interface.
type FuncAppender func(Object)

func (f FuncAppender) Append(objects ...Object) {
	for _, o := range objects {
		(func(Object))(f)(o)
	}
}
