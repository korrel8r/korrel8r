package korrel8

import "github.com/alanconway/korrel8/pkg/unique"

// ListResult collects objects in order.
type ListResult []Object

func (r *ListResult) Append(objects ...Object)    { *r = append(*r, objects...) }
func (r *ListResult) List() []Object              { return []Object(*r) }
func NewListResult(initial ...Object) *ListResult { r := ListResult(initial); return &r }

// SetResult collects objects in order, ignoring objects if their Identifier() is already in the set.
type SetResult = *unique.Set[Identifier, Object]

func NewSetResult(initial ...Object) SetResult { return unique.NewSet(Object.Identifier, initial...) }
