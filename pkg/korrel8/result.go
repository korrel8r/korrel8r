package korrel8

// ListResult collects objects in a slice, does not do any de-duplication.
type ListResult []Object

func (r *ListResult) Append(objects ...Object)    { *r = append(*r, objects...) }
func (r ListResult) List() []Object               { return []Object(r) }
func NewListResult(initial ...Object) *ListResult { r := ListResult(initial); return &r }

// SetResult collects objects in order, ignores duplicatesa
type SetResult struct {
	dedup Deduplicator
	list  []Object
}

// NewSetResult collects objects in a slice, removing duplicates.
func NewSetResult(d Deduplicator) SetResult { return SetResult{dedup: d} }
func (r *SetResult) Append(objects ...Object) {
	for _, o := range objects {
		if r.dedup.Unique(o) {
			r.list = append(r.list, o)
		}
	}
}

func (r SetResult) List() []Object { return r.list }

// NeverDeduplicator is a no-op de-duplicator that declares all values unique.
type NeverDeduplicator struct{}

func (NeverDeduplicator) Unique(Object) bool { return true }
